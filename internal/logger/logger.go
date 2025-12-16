package logger

import (
	"context"
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

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  LogLevel
	Format string // "json" or "text"
	Output string // "stdout", "stderr", or file path
}

// LoggerManager implements Logger interface
type LoggerManager struct {
	config  LoggingConfig
	module  string
	action  string
	profile string
}

// NewLogger creates a new logger instance
func NewLogger(config LoggingConfig) *LoggerManager {
	return &LoggerManager{
		config: config,
	}
}

// F creates a new log field
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Implement Logger interface methods
func (l *LoggerManager) Debug(ctx context.Context, msg string, fields ...Field) {
	// Implementation placeholder
}

func (l *LoggerManager) Info(ctx context.Context, msg string, fields ...Field) {
	// Implementation placeholder - would normally log to configured output
}

func (l *LoggerManager) Warn(ctx context.Context, msg string, fields ...Field) {
	// Implementation placeholder
}

func (l *LoggerManager) Error(ctx context.Context, msg string, fields ...Field) {
	// Implementation placeholder
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