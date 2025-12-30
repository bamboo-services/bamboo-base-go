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

	// 打开或创建日志文件
	if err := w.openFile(); err != nil {
		return nil, err
	}

	// 启动归档调度器
	go w.startArchiveScheduler()

	return w, nil
}

// Write 写入数据到日志文件
//
// 实现 io.Writer 接口。当文件大小超过阈值时自动切割。
func (w *RotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查是否需要切割
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
