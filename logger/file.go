package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileWriter is a writer that writes to a file.
type FileWriter struct {
	// Path is the path to the log file.
	Path string
	// MaxSize is the maximum size of the log file in bytes.
	MaxSize int64
	// MaxBackups is the maximum number of old log files to retain.
	MaxBackups int
	// BufferSize is the size of the buffer in bytes.
	BufferSize int
	// FlushInterval is the interval to flush the buffer.
	FlushInterval time.Duration

	mu         sync.Mutex
	file       *os.File
	size       int64
	buffer     []byte
	lastFlush  time.Time
	flushTimer *time.Timer
}

// NewFileWriter creates a new file writer.
func NewFileWriter(path string) *FileWriter {
	return &FileWriter{
		Path:          path,
		MaxSize:       100 * 1024 * 1024, // 100MB
		MaxBackups:    10,
		BufferSize:    4096, // 4KB
		FlushInterval: time.Second,
		buffer:        make([]byte, 0, 4096),
		lastFlush:     time.Now(),
	}
}

// Write writes data to the file.
func (w *FileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Open the file if it's not open
	if w.file == nil {
		if err := w.openFile(); err != nil {
			return 0, err
		}
	}

	// Check if the file needs to be rotated
	if w.size+int64(len(p)) > w.MaxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	// Add to buffer
	w.buffer = append(w.buffer, p...)
	w.size += int64(len(p))

	// Flush if buffer is full or it's been a while since the last flush
	if len(w.buffer) >= w.BufferSize || time.Since(w.lastFlush) >= w.FlushInterval {
		if err := w.flush(); err != nil {
			return 0, err
		}
	} else if w.flushTimer == nil {
		// Start a timer to flush the buffer after the flush interval
		w.flushTimer = time.AfterFunc(w.FlushInterval, func() {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.flush()
		})
	}

	return len(p), nil
}

// Close closes the file.
func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.flushTimer != nil {
		w.flushTimer.Stop()
		w.flushTimer = nil
	}

	if err := w.flush(); err != nil {
		return err
	}

	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}

	return nil
}

// openFile opens the log file.
func (w *FileWriter) openFile() error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(w.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Open the file
	file, err := os.OpenFile(w.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Get the file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}

	w.file = file
	w.size = info.Size()
	return nil
}

// flush flushes the buffer to the file.
func (w *FileWriter) flush() error {
	if len(w.buffer) == 0 {
		return nil
	}

	if w.file == nil {
		if err := w.openFile(); err != nil {
			return err
		}
	}

	if _, err := w.file.Write(w.buffer); err != nil {
		return err
	}

	w.buffer = w.buffer[:0]
	w.lastFlush = time.Now()

	if w.flushTimer != nil {
		w.flushTimer.Stop()
		w.flushTimer = nil
	}

	return nil
}

// rotate rotates the log file.
func (w *FileWriter) rotate() error {
	// Flush the buffer
	if err := w.flush(); err != nil {
		return err
	}

	// Close the current file
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}

	// Rotate the log files
	for i := w.MaxBackups - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", w.Path, i)
		newPath := fmt.Sprintf("%s.%d", w.Path, i+1)
		os.Rename(oldPath, newPath)
	}

	// Rename the current log file
	os.Rename(w.Path, fmt.Sprintf("%s.1", w.Path))

	// Open a new log file
	return w.openFile()
}

// RotatingFileWriter is a writer that writes to a rotating file.
type RotatingFileWriter struct {
	// Path is the path to the log file.
	Path string
	// MaxSize is the maximum size of the log file in bytes.
	MaxSize int64
	// MaxBackups is the maximum number of old log files to retain.
	MaxBackups int
	// MaxAge is the maximum age of old log files in days.
	MaxAge int
	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.
	LocalTime bool
	// Compress determines if the rotated log files should be compressed
	// using gzip.
	Compress bool

	mu   sync.Mutex
	file *os.File
	size int64
}

// NewRotatingFileWriter creates a new rotating file writer.
func NewRotatingFileWriter(path string) *RotatingFileWriter {
	return &RotatingFileWriter{
		Path:       path,
		MaxSize:    100 * 1024 * 1024, // 100MB
		MaxBackups: 10,
		MaxAge:     30,
		LocalTime:  true,
		Compress:   false,
	}
}

// Write writes data to the file.
func (w *RotatingFileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Open the file if it's not open
	if w.file == nil {
		if err := w.openFile(); err != nil {
			return 0, err
		}
	}

	// Check if the file needs to be rotated
	if w.size+int64(len(p)) > w.MaxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	// Write to the file
	n, err = w.file.Write(p)
	w.size += int64(n)
	return n, err
}

// Close closes the file.
func (w *RotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}

	return nil
}

// openFile opens the log file.
func (w *RotatingFileWriter) openFile() error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(w.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Open the file
	file, err := os.OpenFile(w.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Get the file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}

	w.file = file
	w.size = info.Size()
	return nil
}

// rotate rotates the log file.
func (w *RotatingFileWriter) rotate() error {
	// Close the current file
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}

	// Generate the timestamp
	var timestamp string
	if w.LocalTime {
		timestamp = time.Now().Format("2006-01-02T15-04-05")
	} else {
		timestamp = time.Now().UTC().Format("2006-01-02T15-04-05")
	}

	// Rename the current log file
	backupPath := fmt.Sprintf("%s.%s", w.Path, timestamp)
	os.Rename(w.Path, backupPath)

	// Compress the backup file if needed
	if w.Compress {
		// TODO: Implement compression
	}

	// Remove old backup files
	if w.MaxBackups > 0 || w.MaxAge > 0 {
		w.removeOldBackups()
	}

	// Open a new log file
	return w.openFile()
}

// removeOldBackups removes old backup files.
func (w *RotatingFileWriter) removeOldBackups() {
	// Get the directory and pattern
	dir := filepath.Dir(w.Path)
	pattern := filepath.Base(w.Path) + ".*"

	// Find all backup files
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}

	// Sort the backup files by modification time
	type backupFile struct {
		Path    string
		ModTime time.Time
	}
	var backups []backupFile
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		backups = append(backups, backupFile{Path: match, ModTime: info.ModTime()})
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(backups); i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].ModTime.Before(backups[j].ModTime) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// Remove old backups by count
	if w.MaxBackups > 0 && len(backups) > w.MaxBackups {
		for i := w.MaxBackups; i < len(backups); i++ {
			os.Remove(backups[i].Path)
		}
		backups = backups[:w.MaxBackups]
	}

	// Remove old backups by age
	if w.MaxAge > 0 {
		cutoff := time.Now().Add(-time.Duration(w.MaxAge) * 24 * time.Hour)
		for _, backup := range backups {
			if backup.ModTime.Before(cutoff) {
				os.Remove(backup.Path)
			}
		}
	}
}
