package utility

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides logging capabilities with file rotation
type Logger struct {
	level      LogLevel
	logDir     string
	currentLog *os.File
	mu         sync.Mutex
	mode       string // "file", "cli", "journal"
}

var (
	instance *Logger
	once     sync.Once
)

// GetLogger returns the singleton logger instance
func GetLogger() *Logger {
	once.Do(func() {
		instance = &Logger{
			level:  INFO,
			logDir: "log",
			mode:   "file",
		}
		instance.init()
	})
	return instance
}

// NewLogger creates a new logger with the specified mode
func NewLogger(mode string, level LogLevel) *Logger {
	logger := &Logger{
		level:  level,
		logDir: "log",
		mode:   mode,
	}
	if mode == "file" {
		logger.init()
	}
	return logger
}

// init initializes the logger and performs log rotation
func (l *Logger) init() {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(l.logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		return
	}

	// Rotate existing logs
	l.rotateLogs()

	// Open current log file
	currentLogPath := filepath.Join(l.logDir, "current.log")
	file, err := os.OpenFile(currentLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		return
	}

	l.currentLog = file
}

// rotateLogs rotates existing log files
func (l *Logger) rotateLogs() {
	archiveDir := filepath.Join(l.logDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return
	}

	currentLogPath := filepath.Join(l.logDir, "current.log")

	// Check if current.log exists
	if _, err := os.Stat(currentLogPath); err == nil {
		// Move bot-7.log to bot-8.log (delete bot-8.log)
		for i := 7; i >= 1; i-- {
			oldPath := filepath.Join(archiveDir, fmt.Sprintf("bot-%d.log", i))
			newPath := filepath.Join(archiveDir, fmt.Sprintf("bot-%d.log", i+1))

			if i == 7 {
				// Delete the oldest log
				os.Remove(newPath)
			}

			// Rename if file exists
			if _, err := os.Stat(oldPath); err == nil {
				os.Rename(oldPath, newPath)
			}
		}

		// Move current.log to bot-1.log
		os.Rename(currentLogPath, filepath.Join(archiveDir, "bot-1.log"))
	}
}

// log writes a log message
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05.000")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level.String(), message)

	switch l.mode {
	case "file":
		if l.currentLog != nil {
			l.currentLog.WriteString(logLine)
		} else {
			fmt.Fprint(os.Stderr, logLine)
		}
	case "cli":
		l.printColoredLog(level, timestamp, message)
	case "journal":
		// For systemd journal, we'll use simple stdout
		fmt.Print(logLine)
	default:
		fmt.Print(logLine)
	}
}

// printColoredLog prints a colored log message to the console
func (l *Logger) printColoredLog(level LogLevel, timestamp, message string) {
	const (
		colorReset  = "\033[0m"
		colorBlue   = "\033[0;34m"
		colorGreen  = "\033[0;32m"
		colorYellow = "\033[1;33m"
		colorRed    = "\033[0;31m"
	)

	var color string
	switch level {
	case DEBUG:
		color = colorBlue
	case INFO:
		color = colorGreen
	case WARN:
		color = colorYellow
	case ERROR:
		color = colorRed
	default:
		color = colorReset
	}

	fmt.Printf("%s[%s] [%s]%s %s\n", color, timestamp, level.String(), colorReset, message)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Raw logs a message without timestamp or level
func (l *Logger) Raw(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentLog != nil {
		l.currentLog.WriteString(message + "\n")
	} else {
		fmt.Println(message)
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Close closes the log file
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentLog != nil {
		return l.currentLog.Close()
	}
	return nil
}

// GetWriter returns an io.Writer for the logger
func (l *Logger) GetWriter() io.Writer {
	if l.currentLog != nil {
		return l.currentLog
	}
	return os.Stdout
}

// ListLogFiles returns a list of all log files
func (l *Logger) ListLogFiles() []string {
	files := []string{}

	// Add current log
	currentLogPath := filepath.Join(l.logDir, "current.log")
	if _, err := os.Stat(currentLogPath); err == nil {
		files = append(files, currentLogPath)
	}

	// Add archived logs
	archiveDir := filepath.Join(l.logDir, "archive")
	if entries, err := os.ReadDir(archiveDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				files = append(files, filepath.Join(archiveDir, entry.Name()))
			}
		}
	}

	return files
}
