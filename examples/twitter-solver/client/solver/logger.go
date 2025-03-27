package solver

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Logger represents a logger for the Twitter solver
type Logger struct {
	*log.Logger
	file *os.File
}

var logger *Logger

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	logFile := filepath.Join(logsDir, fmt.Sprintf("twitter_solver_%s.log", time.Now().Format("20060102")))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Create a multi-writer to write to both file and console
	multiWriter := io.MultiWriter(os.Stdout, file)

	return &Logger{
		Logger: log.New(multiWriter, "", log.LstdFlags),
		file:   file,
	}
}

// Log logs a message with the given format and arguments
func (l *Logger) Log(format string, v ...interface{}) {
	l.Printf(format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.Printf("ERROR: "+format, v...)
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.Printf("INFO: "+format, v...)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.Printf("DEBUG: "+format, v...)
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
