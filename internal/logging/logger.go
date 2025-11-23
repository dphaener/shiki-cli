package logging

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

// Level represents log severity level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger provides structured JSON logging
type Logger struct {
	level  Level
	output io.Writer
}

// NewLogger creates a new logger with specified level and output
func NewLogger(level Level, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}

	return &Logger{
		level:  level,
		output: output,
	}
}

// logEntry represents a single log entry in JSON format
type logEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// log writes a log entry if it meets the minimum level
func (l *Logger) log(level Level, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := logEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     levelString(level),
		Message:   msg,
		Fields:    fields,
	}

	data, _ := json.Marshal(entry)
	_, _ = l.output.Write(append(data, '\n')) // Ignore write errors (best effort logging)
}

// Debug logs a debug-level message
func (l *Logger) Debug(msg string, fields map[string]interface{}) {
	l.log(LevelDebug, msg, fields)
}

// Info logs an info-level message
func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.log(LevelInfo, msg, fields)
}

// Warn logs a warning-level message
func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.log(LevelWarn, msg, fields)
}

// Error logs an error-level message
func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.log(LevelError, msg, fields)
}

// levelString converts Level to string representation
func levelString(level Level) string {
	switch level {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}
