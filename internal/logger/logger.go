package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	instance *Logger
	once     sync.Once
)

// Logger handles dual logging to console and file
type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	logFile     *os.File
	mu          sync.Mutex
}

// Init initializes the global logger instance
func Init(logDir string) error {
	var initErr error
	once.Do(func() {
		instance, initErr = newLogger(logDir)
	})
	return initErr
}

// newLogger creates a new logger that writes to both console and file
func newLogger(logDir string) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	logFileName := fmt.Sprintf("oubliette-%s.log", time.Now().Format("2006-01-02"))
	logFilePath := filepath.Join(logDir, logFileName)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create multi-writers for both console and file
	infoWriter := io.MultiWriter(os.Stdout, logFile)
	errorWriter := io.MultiWriter(os.Stderr, logFile)

	return &Logger{
		infoLogger:  log.New(infoWriter, "", log.LstdFlags),
		errorLogger: log.New(errorWriter, "ERROR: ", log.LstdFlags),
		logFile:     logFile,
	}, nil
}

// Close closes the log file
func Close() error {
	if instance != nil && instance.logFile != nil {
		return instance.logFile.Close()
	}
	return nil
}

// Info logs an informational message
func Info(format string, v ...interface{}) {
	if instance != nil {
		instance.mu.Lock()
		defer instance.mu.Unlock()
		instance.infoLogger.Printf(format, v...)
	}
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	if instance != nil {
		instance.mu.Lock()
		defer instance.mu.Unlock()
		instance.errorLogger.Printf(format, v...)
	}
}

// Println logs a simple message
func Println(v ...interface{}) {
	if instance != nil {
		instance.mu.Lock()
		defer instance.mu.Unlock()
		instance.infoLogger.Println(v...)
	}
}

// Printf logs a formatted message
func Printf(format string, v ...interface{}) {
	if instance != nil {
		instance.mu.Lock()
		defer instance.mu.Unlock()
		instance.infoLogger.Printf(format, v...)
	}
}

// Fatal logs a fatal error and exits
func Fatal(v ...interface{}) {
	if instance != nil {
		instance.mu.Lock()
		instance.errorLogger.Fatal(v...)
		instance.mu.Unlock()
	} else {
		log.Fatal(v...)
	}
}

// Fatalf logs a formatted fatal error and exits
func Fatalf(format string, v ...interface{}) {
	if instance != nil {
		instance.mu.Lock()
		instance.errorLogger.Fatalf(format, v...)
		instance.mu.Unlock()
	} else {
		log.Fatalf(format, v...)
	}
}
