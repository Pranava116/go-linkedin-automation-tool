package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Logger interface for structured logging
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	WithModule(module string) Logger
	WithAction(action string) Logger
	WithProfile(profileURL string) Logger
}

// Field represents a log field
type Field struct {
	Key   string
	Value interface{}
}

// LogLevel represents logging levels
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  LogLevel
	Format string // "json" or "text"
	Output string // "stdout", "stderr", or file path
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Module    string                 `json:"module,omitempty"`
	Action    string                 `json:"action,omitempty"`
	Profile   string                 `json:"profile,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// LoggerManager implements Logger interface
type LoggerManager struct {
	config  LoggingConfig
	module  string
	action  string
	profile string
	writer  io.Writer
}

// NewLogger creates a new logger instance
func NewLogger(config LoggingConfig) *LoggerManager {
	writer := getWriter(config.Output)
	return &LoggerManager{
		config: config,
		writer: writer,
	}
}

// getWriter returns the appropriate writer based on output configuration
func getWriter(output string) io.Writer {
	switch output {
	case "stdout", "":
		return os.Stdout
	case "stderr":
		return os.Stderr
	default:
		// For file paths, we'd open a file, but for simplicity using stdout
		// In production, this would handle file creation and rotation
		return os.Stdout
	}
}

// F creates a new log field
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// shouldLog checks if the message should be logged based on configured level
func (l *LoggerManager) shouldLog(level LogLevel) bool {
	return level >= l.config.Level
}

// log handles the actual logging with structured format
func (l *LoggerManager) log(level LogLevel, msg string, fields ...Field) {
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Message:   msg,
		Module:    l.module,
		Action:    l.action,
		Profile:   l.profile,
	}

	// Add custom fields
	if len(fields) > 0 {
		entry.Fields = make(map[string]interface{})
		for _, field := range fields {
			entry.Fields[field.Key] = field.Value
		}
	}

	// Format and write log entry
	switch l.config.Format {
	case "json":
		l.writeJSON(entry)
	default:
		l.writeText(entry)
	}
}

// writeJSON writes log entry in JSON format
func (l *LoggerManager) writeJSON(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple text if JSON marshaling fails
		fmt.Fprintf(l.writer, "[ERROR] Failed to marshal log entry: %v\n", err)
		return
	}
	fmt.Fprintf(l.writer, "%s\n", data)
}

// writeText writes log entry in human-readable text format
func (l *LoggerManager) writeText(entry LogEntry) {
	output := fmt.Sprintf("[%s] %s %s", entry.Timestamp, entry.Level, entry.Message)
	
	if entry.Module != "" {
		output += fmt.Sprintf(" module=%s", entry.Module)
	}
	if entry.Action != "" {
		output += fmt.Sprintf(" action=%s", entry.Action)
	}
	if entry.Profile != "" {
		output += fmt.Sprintf(" profile=%s", entry.Profile)
	}
	
	if entry.Fields != nil {
		for key, value := range entry.Fields {
			output += fmt.Sprintf(" %s=%v", key, value)
		}
	}
	
	fmt.Fprintf(l.writer, "%s\n", output)
}

// Implement Logger interface methods
func (l *LoggerManager) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log(DebugLevel, msg, fields...)
}

func (l *LoggerManager) Info(ctx context.Context, msg string, fields ...Field) {
	l.log(InfoLevel, msg, fields...)
}

func (l *LoggerManager) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log(WarnLevel, msg, fields...)
}

func (l *LoggerManager) Error(ctx context.Context, msg string, fields ...Field) {
	l.log(ErrorLevel, msg, fields...)
}

func (l *LoggerManager) WithModule(module string) Logger {
	newLogger := *l
	newLogger.module = module
	return &newLogger
}

func (l *LoggerManager) WithAction(action string) Logger {
	newLogger := *l
	newLogger.action = action
	return &newLogger
}

func (l *LoggerManager) WithProfile(profileURL string) Logger {
	newLogger := *l
	newLogger.profile = profileURL
	return &newLogger
}