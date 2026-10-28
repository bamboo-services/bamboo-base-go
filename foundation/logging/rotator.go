package logging

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// ErrClosed 表示写入器已经通过 [RotatingWriter.Close] 关闭。
	// 关闭之后的 Write 返回该错误，调用方可通过 errors.Is 识别
	// 并停止写入。
	ErrClosed = errors.New("rotating writer is closed")
)

// RotatorConfig 是 [NewRotatingWriter] 的构造配置。
//
// 字段为零值或非法值时由构造器填充默认值：Dir 默认 ".logs"，
// BaseName 默认 "log"，Ext 默认 ".log"，MaxSize 小于等于 0 时
// 默认 10MB。
type RotatorConfig struct {
	Dir      string // 日志目录，构造时自动创建（权限 0755）。
	BaseName string // 基础文件名（如 "log"），当前文件名为 BaseName+Ext。
	Ext      string // 扩展名（如 ".log"），直接拼接到文件名末尾。
	MaxSize  int64  // 单个日志文件的最大字节数，超出后触发切割，默认 10MB。
}

// RotatingWriter 是支持按大小与日期自动切割、并按天归档的日志写入器，
// 实现 io.Writer 与 io.Closer 接口。
//
// 当前写入文件名为 BaseName+Ext（如 "log.log"）；按大小切割后的
// 文件命名为 BaseName.索引.Ext（如 "log.0.log"），索引递增，
// 数字越大越新；跨天后当日文件按 BaseName-日期.Ext 重命名保留。
// 历史文件归档统一命名为 logger-日期.tar.gz（前缀固定为 "logger"，
// 与 BaseName 无关），归档完成后原始日志文件被删除。
//
// 多个 goroutine 可并发调用 Write 与 Close；零值不可用，
// 须通过 [NewRotatingWriter] 构造。
type RotatingWriter struct {
	fileMu      sync.Mutex // 仅保护底层文件句柄、currentSize、currentDate
	stateMu     sync.Mutex // 保护 closed 标志与内部状态切换
	file        *os.File   // 当前写入的文件
	dir         string     // 日志目录
	baseName    string     // 基础文件名
	ext         string     // 扩展名
	maxSize     int64      // 最大文件大小
	currentSize int64      // 当前文件大小
	currentDate string     // 当前日期 (用于判断是否跨天)

	closed   bool
	closeErr error

	closeDone     chan struct{}
	schedulerStop chan struct{}
	schedulerDone chan struct{}

	archiveWg sync.WaitGroup
	bgErrors  []error

	// 私有测试 hook
	beforeArchiveFileHook func(filePath string)
}

// NewRotatingWriter 创建日志切割写入器，并启动每日归档调度器。
//
// config 的零值字段按 [RotatorConfig] 文档填充默认值。构造过程会
// 创建日志目录（0755）并完成启动检查：已有的跨天文件按日期重命名、
// 超限文件立即切割、历史文件尝试归档；前两类失败时返回错误，
// 启动阶段的归档失败不使构造失败，而是推迟到 Close 时一并返回。
// 返回的写入器可被多个 goroutine 并发使用，由调用方负责 Close。
func NewRotatingWriter(config RotatorConfig) (*RotatingWriter, error) {
	if config.Dir == "" {
		config.Dir = ".logs"
	}
	if config.BaseName == "" {
		config.BaseName = "log"
	}
	if config.Ext == "" {
		config.Ext = ".log"
	}
	if config.MaxSize <= 0 {
		config.MaxSize = 10 * 1024 * 1024 // 默认 10MB
	}

	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	w := &RotatingWriter{
		dir:           config.Dir,
		baseName:      config.BaseName,
		ext:           config.Ext,
		maxSize:       config.MaxSize,
		currentDate:   time.Now().Format("2006-01-02"),
		closeDone:     make(chan struct{}),
		schedulerStop: make(chan struct{}),
		schedulerDone: make(chan struct{}),
	}

	if err := w.checkStartupState(); err != nil {
		return nil, fmt.Errorf("启动检查失败: %w", err)
	}

	go w.startArchiveScheduler()

	return w, nil
}

// Write 将数据追加写入当前日志文件，必要时先执行切割。
//
// 写入前按序处理：已关闭时返回 [ErrClosed]；跨天时先将当前文件按
// 日期重命名并开启新文件，随后异步登记归档；当前文件剩余空间
// 不足以容纳本次写入时按索引切割。返回底层文件写入的结果，
// n 为成功写入的字节数。本方法可被多个 goroutine 并发调用。
func (w *RotatingWriter) Write(p []byte) (n int, err error) {
	w.stateMu.Lock()
	if w.closed {
		w.stateMu.Unlock()
		return 0, ErrClosed
	}
	w.stateMu.Unlock()

	w.fileMu.Lock()
	defer w.fileMu.Unlock()

	// 再次快速检查（避免竞争）
	w.stateMu.Lock()
	if w.closed {
		w.stateMu.Unlock()
		return 0, ErrClosed
	}
	w.stateMu.Unlock()

	today := time.Now().Format("2006-01-02")
	if w.currentDate != today {
		if err := w.rotateForNewDayLocked(today); err != nil {
			return 0, fmt.Errorf("跨天切割日志失败: %w", err)
		}
	}

	if w.currentSize+int64(len(p)) > w.maxSize {
		if err := w.rotateLocked(); err != nil {
			return 0, fmt.Errorf("日志切割失败: %w", err)
		}
	}

	if w.file == nil {
		return 0, ErrClosed
	}

	n, err = w.file.Write(p)
	w.currentSize += int64(n)
	return n, err
}

// Close 关闭日志文件与归档调度器，等待已登记归档全部完成。
//
// 停止顺序为：停止每日调度器、等待在途归档协程退出、关闭当前文件
// 句柄；返回值以 errors.Join 汇总文件关闭错误与后台归档记录的错误。
// 本方法幂等：并发或重复调用都会等待首次关闭完成并返回同一结果；
// 关闭之后的 Write 返回 [ErrClosed]。
func (w *RotatingWriter) Close() error {
	w.stateMu.Lock()
	if w.closed {
		closeDone := w.closeDone
		w.stateMu.Unlock()
		<-closeDone
		w.stateMu.Lock()
		err := w.closeErr
		w.stateMu.Unlock()
		return err
	}
	w.closed = true

	// 1. 停止定时调度器并等待退出
	close(w.schedulerStop)
	w.stateMu.Unlock()

	<-w.schedulerDone

	// 2. 等待所有在途/已登记归档协程真实退出（此时绝不持 fileMu）
	w.archiveWg.Wait()

	// 3. 关闭文件句柄
	w.fileMu.Lock()
	var closeErr error
	if w.file != nil {
		closeErr = w.file.Close()
		w.file = nil
	}
	w.fileMu.Unlock()

	w.stateMu.Lock()
	var allErrs []error
	if closeErr != nil {
		allErrs = append(allErrs, closeErr)
	}
	allErrs = append(allErrs, w.bgErrors...)
	w.closeErr = errors.Join(allErrs...)
	close(w.closeDone)
	finalErr := w.closeErr
	w.stateMu.Unlock()

	return finalErr
}

func (w *RotatingWriter) openFileLocked() error {
	filePath := w.currentFilePath()
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	w.file = file
	w.currentSize = info.Size()
	return nil
}

func (w *RotatingWriter) currentFilePath() string {
	return filepath.Join(w.dir, w.baseName+w.ext)
}

func (w *RotatingWriter) rotatedFilePath(index int) string {
	return filepath.Join(w.dir, fmt.Sprintf("%s.%d%s", w.baseName, index, w.ext))
}

func (w *RotatingWriter) datedFilePath(date string) string {
	return filepath.Join(w.dir, fmt.Sprintf("%s-%s%s", w.baseName, date, w.ext))
}

func (w *RotatingWriter) datedFilePathWithIndex(date string, index int) string {
	return filepath.Join(w.dir, fmt.Sprintf("%s-%s.%d%s", w.baseName, date, index, w.ext))
}

func (w *RotatingWriter) renameCurrentLogByDateLocked(date string) error {
	currentPath := w.currentFilePath()

	if _, err := os.Stat(currentPath); os.IsNotExist(err) {
		return nil
	}

	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("关闭日志文件失败: %w", err)
		}
		w.file = nil
	}

	targetPath := w.datedFilePath(date)
	counter := 1
	for {
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			break
		}
		if counter > 1000 {
			return fmt.Errorf("无法生成唯一的日期日志文件名")
		}
		targetPath = w.datedFilePathWithIndex(date, counter)
		counter++
	}

	if err := os.Rename(currentPath, targetPath); err != nil {
		return fmt.Errorf("重命名日志文件失败: %w", err)
	}

	return nil
}

func (w *RotatingWriter) rotateLocked() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("关闭日志文件失败: %w", err)
		}
		w.file = nil
	}

	maxIndex := w.findMaxRotatedIndex()
	nextIndex := maxIndex + 1

	currentPath := w.currentFilePath()
	if _, err := os.Stat(currentPath); err == nil {
		if err := os.Rename(currentPath, w.rotatedFilePath(nextIndex)); err != nil {
			return fmt.Errorf("重命名当前日志文件失败: %w", err)
		}
	}

	w.currentSize = 0
	return w.openFileLocked()
}

func (w *RotatingWriter) rotateForNewDayLocked(newDate string) error {
	currentPath := w.currentFilePath()
	if _, err := os.Stat(currentPath); err == nil {
		if w.file != nil {
			if err := w.file.Close(); err != nil {
				return fmt.Errorf("关闭日志文件失败: %w", err)
			}
			w.file = nil
		}

		datedPath := w.datedFilePath(w.currentDate)
		counter := 1
		for {
			if _, err := os.Stat(datedPath); os.IsNotExist(err) {
				break
			}
			if counter > 1000 {
				return fmt.Errorf("无法生成唯一的日期日志文件名")
			}
			datedPath = w.datedFilePathWithIndex(w.currentDate, counter)
			counter++
		}

		if err := os.Rename(currentPath, datedPath); err != nil {
			return fmt.Errorf("跨天重命名日志失败: %w", err)
		}
	}

	w.currentDate = newDate
	w.currentSize = 0
	if err := w.openFileLocked(); err != nil {
		return err
	}

	w.triggerArchiveAsync()
	return nil
}

func (w *RotatingWriter) findMaxRotatedIndex() int {
	pattern := regexp.MustCompile(fmt.Sprintf(`^%s\.(\d+)%s$`, regexp.QuoteMeta(w.baseName), regexp.QuoteMeta(w.ext)))
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return -1
	}

	maxIndex := -1
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := pattern.FindStringSubmatch(entry.Name())
		if len(matches) == 2 {
			if index, err := strconv.Atoi(matches[1]); err == nil && index > maxIndex {
				maxIndex = index
			}
		}
	}
	return maxIndex
}

func (w *RotatingWriter) startArchiveScheduler() {
	defer close(w.schedulerDone)

	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 5, 0, now.Location())
		duration := next.Sub(now)

		timer := time.NewTimer(duration)
		select {
		case <-timer.C:
			w.triggerArchiveAsync()
		case <-w.schedulerStop:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

func (w *RotatingWriter) triggerArchiveAsync() {
	w.stateMu.Lock()
	if w.closed {
		w.stateMu.Unlock()
		return
	}
	w.archiveWg.Add(1)
	w.stateMu.Unlock()

	go func() {
		defer w.archiveWg.Done()
		w.archiveYesterday()
	}()
}

func (w *RotatingWriter) archiveYesterday() {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	archiveName := fmt.Sprintf("logger-%s.tar.gz", yesterday)
	archivePath := filepath.Join(w.dir, archiveName)

	if _, err := os.Stat(archivePath); err == nil {
		return
	}

	files := w.collectFilesToArchive()
	if len(files) == 0 {
		return
	}

	if err := w.createTarGz(archivePath, files); err != nil {
		w.recordBgError(err)
		return
	}

	for _, file := range files {
		_ = os.Remove(file)
	}
}

func (w *RotatingWriter) collectFilesToArchive() []string {
	pattern := regexp.MustCompile(fmt.Sprintf(`^%s\.\d+%s$`, regexp.QuoteMeta(w.baseName), regexp.QuoteMeta(w.ext)))
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if pattern.MatchString(entry.Name()) {
			files = append(files, filepath.Join(w.dir, entry.Name()))
		}
	}

	sort.Strings(files)
	return files
}

func (w *RotatingWriter) createTarGz(archivePath string, files []string) (err error) {
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("创建归档文件失败: %w", err)
	}
	defer func() {
		if cErr := archiveFile.Close(); cErr != nil && err == nil {
			err = cErr
		}
	}()

	gzWriter := gzip.NewWriter(archiveFile)
	defer func() {
		if cErr := gzWriter.Close(); cErr != nil && err == nil {
			err = cErr
		}
	}()

	tarWriter := tar.NewWriter(gzWriter)
	defer func() {
		if cErr := tarWriter.Close(); cErr != nil && err == nil {
			err = cErr
		}
	}()

	for _, file := range files {
		if w.beforeArchiveFileHook != nil {
			w.beforeArchiveFileHook(file)
		}
		if err := w.addFileToTar(tarWriter, file); err != nil {
			return fmt.Errorf("添加文件 %s 到归档失败: %w", file, err)
		}
	}

	return nil
}

func (w *RotatingWriter) addFileToTar(tarWriter *tar.Writer, filePath string) (err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if cErr := file.Close(); cErr != nil && err == nil {
			err = cErr
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = filepath.Base(filePath)

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)
	return err
}

func (w *RotatingWriter) checkStartupState() error {
	currentPath := w.currentFilePath()

	info, err := os.Stat(currentPath)
	if os.IsNotExist(err) {
		w.fileMu.Lock()
		defer w.fileMu.Unlock()
		return w.openFileLocked()
	}
	if err != nil {
		return fmt.Errorf("获取日志文件信息失败: %w", err)
	}

	modTime := info.ModTime()
	today := time.Now().Format("2006-01-02")
	fileDate := modTime.Format("2006-01-02")

	w.fileMu.Lock()
	defer w.fileMu.Unlock()

	if fileDate != today {
		if err := w.renameCurrentLogByDateLocked(fileDate); err != nil {
			return fmt.Errorf("按日期重命名日志失败: %w", err)
		}
		if err := w.openFileLocked(); err != nil {
			return err
		}
	} else {
		if info.Size() >= w.maxSize {
			if err := w.rotateLocked(); err != nil {
				return fmt.Errorf("启动时切割日志失败: %w", err)
			}
		} else {
			if err := w.openFileLocked(); err != nil {
				return err
			}
		}
	}

	if err := w.archiveOldFiles(); err != nil {
		w.recordBgError(err)
	}
	return nil
}

func (w *RotatingWriter) recordBgError(err error) {
	if err == nil {
		return
	}
	w.stateMu.Lock()
	w.bgErrors = append(w.bgErrors, err)
	w.stateMu.Unlock()
}

func (w *RotatingWriter) archiveOldFiles() error {
	today := time.Now().Format("2006-01-02")
	filesByDate := make(map[string][]string)

	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return err
	}

	rotatedPattern := regexp.MustCompile(
		fmt.Sprintf(`^%s\.(\d+)%s$`, regexp.QuoteMeta(w.baseName), regexp.QuoteMeta(w.ext)),
	)
	datedPattern := regexp.MustCompile(
		fmt.Sprintf(`^%s-(\d{4}-\d{2}-\d{2})(\.\d+)?%s$`,
			regexp.QuoteMeta(w.baseName), regexp.QuoteMeta(w.ext)),
	)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == w.baseName+w.ext || strings.HasSuffix(name, ".tar.gz") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileDate := info.ModTime().Format("2006-01-02")
		if fileDate == today {
			continue
		}

		if rotatedPattern.MatchString(name) || datedPattern.MatchString(name) {
			filesByDate[fileDate] = append(filesByDate[fileDate], filepath.Join(w.dir, name))
		}
	}

	for date, files := range filesByDate {
		archiveName := fmt.Sprintf("logger-%s.tar.gz", date)
		archivePath := filepath.Join(w.dir, archiveName)

		if _, err := os.Stat(archivePath); err == nil {
			continue
		}

		if err := w.createTarGz(archivePath, files); err != nil {
			w.recordBgError(err)
			continue
		}

		for _, file := range files {
			_ = os.Remove(file)
		}
	}

	return nil
}
