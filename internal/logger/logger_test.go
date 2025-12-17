package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// **Feature: linkedin-automation-framework, Property 40: Structured logging levels**
func TestStructuredLoggingLevels(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random log level
		level := rapid.SampledFrom([]LogLevel{DebugLevel, InfoLevel, WarnLevel, ErrorLevel}).Draw(t, "level")
		
		// Generate random non-empty message
		message := rapid.StringMatching(`[a-zA-Z0-9 ]+`).Filter(func(s string) bool {
			return len(strings.TrimSpace(s)) > 0
		}).Draw(t, "message")
		
		// Generate random format
		format := rapid.SampledFrom([]string{"json", "text"}).Draw(t, "format")
		
		// Create logger with captured output
		var buf bytes.Buffer
		config := LoggingConfig{
			Level:  DebugLevel, // Set to debug to capture all levels
			Format: format,
			Output: "stdout",
		}
		
		logger := NewLogger(config)
		logger.writer = &buf // Override writer to capture output
		
		ctx := context.Background()
		
		// Test each logging level
		switch level {
		case DebugLevel:
			logger.Debug(ctx, message)
		case InfoLevel:
			logger.Info(ctx, message)
		case WarnLevel:
			logger.Warn(ctx, message)
		case ErrorLevel:
			logger.Error(ctx, message)
		}
		
		output := buf.String()
		
		// Verify output is not empty
		if output == "" {
			t.Fatalf("Expected log output, got empty string")
		}
		
		// Verify the level appears in output
		expectedLevel := level.String()
		if !strings.Contains(output, expectedLevel) {
			t.Fatalf("Expected level %q to appear in output %q", expectedLevel, output)
		}
		
		// If JSON format, verify it's valid JSON and contains expected fields
		if format == "json" {
			lines := strings.Split(strings.TrimSpace(output), "\n")
			for _, line := range lines {
				if line != "" {
					var entry LogEntry
					if err := json.Unmarshal([]byte(line), &entry); err != nil {
						t.Fatalf("Expected valid JSON output, got error: %v for line: %q", err, line)
					}
					
					// Verify required fields are present
					if entry.Timestamp == "" {
						t.Fatalf("Expected timestamp in JSON output")
					}
					if entry.Level == "" {
						t.Fatalf("Expected level in JSON output")
					}
					// Message should match what we sent
					if entry.Message != message {
						t.Fatalf("Expected message %q in JSON output, got %q", message, entry.Message)
					}
				}
			}
		} else {
			// For text format, verify the message appears in output
			if !strings.Contains(output, message) {
				t.Fatalf("Expected message %q to appear in output %q", message, output)
			}
		}
	})
}

// **Feature: linkedin-automation-framework, Property 41: Contextual log information**
func TestContextualLogInformation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random contextual information using safe characters
		module := rapid.StringMatching(`[a-zA-Z0-9_-]*`).Draw(t, "module")
		action := rapid.StringMatching(`[a-zA-Z0-9_-]*`).Draw(t, "action")
		profile := rapid.StringMatching(`[a-zA-Z0-9_.-]*`).Draw(t, "profile")
		message := rapid.StringMatching(`[a-zA-Z0-9 ]*`).Draw(t, "message")
		
		// Generate random format
		format := rapid.SampledFrom([]string{"json", "text"}).Draw(t, "format")
		
		// Create logger with captured output
		var buf bytes.Buffer
		config := LoggingConfig{
			Level:  DebugLevel,
			Format: format,
			Output: "stdout",
		}
		
		logger := NewLogger(config)
		logger.writer = &buf // Override writer to capture output
		
		// Add contextual information
		contextualLogger := logger.WithModule(module).WithAction(action).WithProfile(profile)
		
		ctx := context.Background()
		contextualLogger.Info(ctx, message)
		
		output := buf.String()
		
		// Verify output is not empty
		if output == "" {
			t.Fatalf("Expected log output, got empty string")
		}
		
		// If JSON format, verify contextual fields are properly structured
		if format == "json" {
			lines := strings.Split(strings.TrimSpace(output), "\n")
			for _, line := range lines {
				if line != "" {
					var entry LogEntry
					if err := json.Unmarshal([]byte(line), &entry); err != nil {
						t.Fatalf("Expected valid JSON output, got error: %v", err)
					}
					
					// Verify contextual fields match what was set
					if module != "" && entry.Module != module {
						t.Fatalf("Expected module %q in JSON, got %q", module, entry.Module)
					}
					if action != "" && entry.Action != action {
						t.Fatalf("Expected action %q in JSON, got %q", action, entry.Action)
					}
					if profile != "" && entry.Profile != profile {
						t.Fatalf("Expected profile %q in JSON, got %q", profile, entry.Profile)
					}
				}
			}
		} else {
			// For text format, verify contextual information appears in output
			if module != "" && !strings.Contains(output, module) {
				t.Fatalf("Expected module %q to appear in output %q", module, output)
			}
			if action != "" && !strings.Contains(output, action) {
				t.Fatalf("Expected action %q to appear in output %q", action, output)
			}
			if profile != "" && !strings.Contains(output, profile) {
				t.Fatalf("Expected profile %q to appear in output %q", profile, output)
			}
		}
	})
}

// Unit test for log level filtering
func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	config := LoggingConfig{
		Level:  WarnLevel, // Only warn and error should be logged
		Format: "text",
		Output: "stdout",
	}
	
	logger := NewLogger(config)
	logger.writer = &buf
	
	ctx := context.Background()
	
	// These should not be logged
	logger.Debug(ctx, "debug message")
	logger.Info(ctx, "info message")
	
	// These should be logged
	logger.Warn(ctx, "warn message")
	logger.Error(ctx, "error message")
	
	output := buf.String()
	
	// Verify debug and info are not in output
	if strings.Contains(output, "debug message") {
		t.Errorf("Debug message should not be logged when level is WarnLevel")
	}
	if strings.Contains(output, "info message") {
		t.Errorf("Info message should not be logged when level is WarnLevel")
	}
	
	// Verify warn and error are in output
	if !strings.Contains(output, "warn message") {
		t.Errorf("Warn message should be logged when level is WarnLevel")
	}
	if !strings.Contains(output, "error message") {
		t.Errorf("Error message should be logged when level is WarnLevel")
	}
}

// Unit test for custom fields
func TestCustomFields(t *testing.T) {
	var buf bytes.Buffer
	config := LoggingConfig{
		Level:  InfoLevel,
		Format: "json",
		Output: "stdout",
	}
	
	logger := NewLogger(config)
	logger.writer = &buf
	
	ctx := context.Background()
	logger.Info(ctx, "test message", F("key1", "value1"), F("key2", 42))
	
	output := buf.String()
	
	var entry LogEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	
	if entry.Fields == nil {
		t.Fatalf("Expected fields to be present")
	}
	
	if entry.Fields["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", entry.Fields["key1"])
	}
	
	if entry.Fields["key2"] != float64(42) { // JSON unmarshals numbers as float64
		t.Errorf("Expected key2=42, got %v", entry.Fields["key2"])
	}
}