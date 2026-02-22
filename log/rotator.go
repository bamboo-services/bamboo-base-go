package xLog

import (
	"archive/tar"
	"compress/gzip"
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

// RotatorConfig 日志切割器配置
type RotatorConfig struct {
	Dir      string // 日志目录
	BaseName string // 基础文件名 (如 "log")
	Ext      string // 扩展名 (如 ".log")
	MaxSize  int64  // 最大文件大小 (字节)，默认 10MB
}

// RotatingWriter 支持自动切割的日志写入器
//
// 实现 io.Writer 接口，当文件大小超过阈值时自动切割。
// 切割后的文件命名格式: log.0.log, log.1.log, log.2.log ... (索引递增，数字越大越新)
// 每天 00:00:05 自动将前一天的日志打包为 logger-yyyy-MM-dd.tar.gz
type RotatingWriter struct {
	mu          sync.Mutex
	file        *os.File // 当前写入的文件
	dir         string   // 日志目录
	baseName    string   // 基础文件名
	ext         string   // 扩展名
	maxSize     int64    // 最大文件大小
	currentSize int64    // 当前文件大小
	currentDate string   // 当前日期 (用于判断是否跨天)
}

// NewRotatingWriter 创建日志切割写入器
//
// 参数说明:
//   - config: 切割器配置
//
// 返回值:
//   - *RotatingWriter: 切割写入器实例
//   - error: 创建错误
func NewRotatingWriter(config RotatorConfig) (*RotatingWriter, error) {
	// 设置默认值
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

	// 创建日志目录
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	w := &RotatingWriter{
		dir:         config.Dir,
		baseName:    config.BaseName,
		ext:         config.Ext,
		maxSize:     config.MaxSize,
		currentDate: time.Now().Format("2006-01-02"),
	}

	// 执行启动状态检查（包含打开/创建文件）
	if err := w.checkStartupState(); err != nil {
		return nil, fmt.Errorf("启动检查失败: %w", err)
	}

	// 启动归档调度器
	go w.startArchiveScheduler()

	return w, nil
}

// Write 写入数据到日志文件
//
// 实现 io.Writer 接口。支持以下自动切割条件:
//  1. 检测到日期变化（跨天）
//  2. 文件大小超过阈值
func (w *RotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查是否跨天
	today := time.Now().Format("2006-01-02")
	if w.currentDate != today {
		if err := w.rotateForNewDay(today); err != nil {
			return 0, fmt.Errorf("跨天切割日志失败: %w", err)
		}
	}

	// 检查是否需要切割（大小）
	if w.currentSize+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, fmt.Errorf("日志切割失败: %w", err)
		}
	}

	// 写入数据
	n, err = w.file.Write(p)
	w.currentSize += int64(n)
	return n, err
}

// Close 关闭日志文件
func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// openFile 打开或创建当前日志文件
func (w *RotatingWriter) openFile() error {
	filePath := w.currentFilePath()

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	// 获取当前文件大小
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	w.file = file
	w.currentSize = info.Size()
	return nil
}

// currentFilePath 获取当前日志文件路径
func (w *RotatingWriter) currentFilePath() string {
	return filepath.Join(w.dir, w.baseName+w.ext)
}

// rotatedFilePath 获取切割后的日志文件路径
func (w *RotatingWriter) rotatedFilePath(index int) string {
	return filepath.Join(w.dir, fmt.Sprintf("%s.%d%s", w.baseName, index, w.ext))
}

// datedFilePath 获取带日期的日志文件路径
//
// 格式: {dir}/{basename}-{date}{ext}
// 示例: .logs/log-2024-01-15.log
func (w *RotatingWriter) datedFilePath(date string) string {
	return filepath.Join(w.dir, fmt.Sprintf("%s-%s%s", w.baseName, date, w.ext))
}

// datedFilePathWithIndex 获取带日期和序号的日志文件路径（避免冲突）
//
// 格式: {dir}/{basename}-{date}.{index}{ext}
// 示例: .logs/log-2024-01-15.1.log
func (w *RotatingWriter) datedFilePathWithIndex(date string, index int) string {
	return filepath.Join(w.dir, fmt.Sprintf("%s-%s.%d%s", w.baseName, date, index, w.ext))
}

// renameCurrentLogByDate 将当前日志文件按日期重命名
//
// 重命名规则: log.log → log-YYYY-MM-DD.log
// 如果目标文件已存在，追加序号避免覆盖: log-YYYY-MM-DD.1.log
//
// 注意: 此函数会关闭当前文件句柄，调用后需要重新打开文件
func (w *RotatingWriter) renameCurrentLogByDate(date string) error {
	currentPath := w.currentFilePath()

	// 检查当前文件是否存在
	if _, err := os.Stat(currentPath); os.IsNotExist(err) {
		return nil // 文件不存在，无需重命名
	}

	// 关闭当前文件句柄
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("关闭日志文件失败: %w", err)
		}
		w.file = nil
	}

	// 构建目标文件名
	targetPath := w.datedFilePath(date)

	// 如果目标已存在，追加序号避免覆盖
	counter := 1
	for {
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			break
		}
		// 防止无限循环
		if counter > 1000 {
			return fmt.Errorf("无法生成唯一的日期日志文件名")
		}
		targetPath = w.datedFilePathWithIndex(date, counter)
		counter++
	}

	// 执行重命名
	if err := os.Rename(currentPath, targetPath); err != nil {
		return fmt.Errorf("重命名日志文件失败: %w", err)
	}

	return nil
}

// rotate 执行日志文件切割
//
// 切割流程:
//  1. 关闭当前文件
//  2. 重命名: log.log → log.N.log (N = maxIndex + 1)
//  3. 创建新的 log.log
//
// 文件顺序: log.0.log (最早) → log.1.log → log.2.log (最新切割)
func (w *RotatingWriter) rotate() error {
	// 关闭当前文件
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("关闭日志文件失败: %w", err)
		}
		w.file = nil
	}

	// 查找现有的切割文件，获取最大索引
	maxIndex := w.findMaxRotatedIndex()
	nextIndex := maxIndex + 1

	// 将当前日志文件重命名为下一个索引
	currentPath := w.currentFilePath()
	if _, err := os.Stat(currentPath); err == nil {
		if err := os.Rename(currentPath, w.rotatedFilePath(nextIndex)); err != nil {
			return fmt.Errorf("重命名当前日志文件失败: %w", err)
		}
	}

	// 创建新的日志文件
	w.currentSize = 0
	return w.openFile()
}

// rotateForNewDay 跨天时切割日志
//
// 流程:
//  1. 将当前 log.log 按日期重命名: log-YYYY-MM-DD.log
//  2. 创建新的 log.log
//  3. 更新 currentDate
//  4. 异步触发归档
//
// 注意: 此方法在 Write() 中调用，已持有锁
func (w *RotatingWriter) rotateForNewDay(newDate string) error {
	// 按旧日期重命名当前日志
	currentPath := w.currentFilePath()
	if _, err := os.Stat(currentPath); err == nil {
		// 关闭当前文件句柄
		if w.file != nil {
			if err := w.file.Close(); err != nil {
				return fmt.Errorf("关闭日志文件失败: %w", err)
			}
			w.file = nil
		}

		datedPath := w.datedFilePath(w.currentDate)

		// 避免文件名冲突
		counter := 1
		for {
			if _, err := os.Stat(datedPath); os.IsNotExist(err) {
				break
			}
			// 防止无限循环
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

	// 更新日期
	w.currentDate = newDate

	// 创建新文件
	w.currentSize = 0
	if err := w.openFile(); err != nil {
		return err
	}

	// 触发归档（异步执行，避免阻塞写入）
	go w.archiveYesterday()

	return nil
}

// findMaxRotatedIndex 查找现有切割文件的最大索引
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

// startArchiveScheduler 启动定时归档调度器
//
// 每天 00:00:05 检查并归档前一天的日志
func (w *RotatingWriter) startArchiveScheduler() {
	for {
		now := time.Now()
		// 计算下一个 00:00:05
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 5, 0, now.Location())
		duration := next.Sub(now)

		time.Sleep(duration)

		// 执行归档
		w.mu.Lock()
		w.archiveYesterday()
		w.mu.Unlock()
	}
}

// archiveYesterday 归档前一天的日志文件
//
// 将所有切割文件打包为 logger-yyyy-MM-dd.tar.gz
func (w *RotatingWriter) archiveYesterday() {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	archiveName := fmt.Sprintf("logger-%s.tar.gz", yesterday)
	archivePath := filepath.Join(w.dir, archiveName)

	// 检查归档文件是否已存在
	if _, err := os.Stat(archivePath); err == nil {
		return // 已归档，跳过
	}

	// 收集需要归档的文件
	files := w.collectFilesToArchive()
	if len(files) == 0 {
		return
	}

	// 创建 tar.gz 归档
	if err := w.createTarGz(archivePath, files); err != nil {
		fmt.Fprintf(os.Stderr, "[LOG] 创建归档失败: %v\n", err)
		return
	}

	// 删除已归档的文件
	for _, file := range files {
		os.Remove(file)
	}
}

// collectFilesToArchive 收集需要归档的日志文件
//
// 收集所有切割后的日志文件 (log.0.log, log.1.log, ...)
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

	// 按文件名排序
	sort.Strings(files)
	return files
}

// createTarGz 创建 tar.gz 归档文件
func (w *RotatingWriter) createTarGz(archivePath string, files []string) error {
	// 创建归档文件
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("创建归档文件失败: %w", err)
	}
	defer func(archiveFile *os.File) {
		err := archiveFile.Close()
		if err != nil {
			SugarWarn(nil, "关闭归档文件失败", "file", archivePath, "error", err)
		}
	}(archiveFile)

	// 创建 gzip 写入器
	gzWriter := gzip.NewWriter(archiveFile)
	defer func(gzWriter *gzip.Writer) {
		err := gzWriter.Close()
		if err != nil {
			SugarWarn(nil, "关闭 gzip 写入器失败", "error", err)
		}
	}(gzWriter)

	// 创建 tar 写入器
	tarWriter := tar.NewWriter(gzWriter)
	defer func(tarWriter *tar.Writer) {
		err := tarWriter.Close()
		if err != nil {
			SugarWarn(nil, "关闭 tar 写入器失败", "error", err)
		}
	}(tarWriter)

	// 写入文件
	for _, file := range files {
		if err := w.addFileToTar(tarWriter, file); err != nil {
			return fmt.Errorf("添加文件 %s 到归档失败: %w", file, err)
		}
	}

	return nil
}

// addFileToTar 将文件添加到 tar 归档
func (w *RotatingWriter) addFileToTar(tarWriter *tar.Writer, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			SugarWarn(nil, "关闭文件失败", "file", filePath, "error", err)
		}
	}(file)

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

// checkStartupState 启动时检查日志文件状态
//
// 检查内容:
//  1. 当前日志文件大小是否超过限制 → 立即切割
//  2. 当前日志文件日期是否为今天 → 按日期重命名后创建新文件
//  3. 目录中是否存在旧日期日志文件 → 批量归档
func (w *RotatingWriter) checkStartupState() error {
	currentPath := w.currentFilePath()

	// 检查当前日志文件是否存在
	info, err := os.Stat(currentPath)
	if os.IsNotExist(err) {
		// 文件不存在，无需检查，直接创建新文件
		return w.openFile()
	}
	if err != nil {
		return fmt.Errorf("获取日志文件信息失败: %w", err)
	}

	// 获取文件修改时间，判断是否为当天
	modTime := info.ModTime()
	today := time.Now().Format("2006-01-02")
	fileDate := modTime.Format("2006-01-02")

	// 如果不是今天的日志，按日期重命名
	if fileDate != today {
		if err := w.renameCurrentLogByDate(fileDate); err != nil {
			return fmt.Errorf("按日期重命名日志失败: %w", err)
		}
		// 重命名后创建新文件
		if err := w.openFile(); err != nil {
			return err
		}
	} else {
		// 今天的日志，检查大小是否超限
		if info.Size() >= w.maxSize {
			if err := w.rotate(); err != nil {
				return fmt.Errorf("启动时切割日志失败: %w", err)
			}
		} else {
			// 文件正常，直接打开
			if err := w.openFile(); err != nil {
				return err
			}
		}
	}

	// 归档所有旧日期的日志文件
	if err := w.archiveOldFiles(); err != nil {
		// 归档失败仅记录警告，不阻断启动
		fmt.Fprintf(os.Stderr, "[LOG] 启动时归档旧日志失败: %v\n", err)
	}

	return nil
}

// archiveOldFiles 批量归档所有非当天的日志文件
//
// 处理范围:
//   - log.N.log 格式的切割文件
//   - log-YYYY-MM-DD.log 格式的日期文件
//
// 归档规则:
//   - 按文件修改日期分组
//   - 每个日期创建一个 tar.gz 归档
//   - 已存在归档的日期跳过
func (w *RotatingWriter) archiveOldFiles() error {
	today := time.Now().Format("2006-01-02")

	// 收集所有需要归档的文件，按日期分组
	filesByDate := make(map[string][]string)

	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return err
	}

	// 匹配模式
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

		// 跳过当前日志文件和归档文件
		if name == w.baseName+w.ext || strings.HasSuffix(name, ".tar.gz") {
			continue
		}

		// 获取文件信息
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 按修改日期分组
		fileDate := info.ModTime().Format("2006-01-02")

		// 只归档非当天的文件
		if fileDate == today {
			continue
		}

		// 匹配需要归档的文件格式
		if rotatedPattern.MatchString(name) || datedPattern.MatchString(name) {
			filesByDate[fileDate] = append(filesByDate[fileDate],
				filepath.Join(w.dir, name))
		}
	}

	// 按日期批量归档
	for date, files := range filesByDate {
		archiveName := fmt.Sprintf("logger-%s.tar.gz", date)
		archivePath := filepath.Join(w.dir, archiveName)

		// 跳过已存在的归档
		if _, err := os.Stat(archivePath); err == nil {
			continue
		}

		if err := w.createTarGz(archivePath, files); err != nil {
			return fmt.Errorf("创建 %s 归档失败: %w", date, err)
		}

		// 删除已归档文件
		for _, file := range files {
			os.Remove(file)
		}
	}

	return nil
}
