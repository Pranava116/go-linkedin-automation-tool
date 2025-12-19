package errors

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-rod/rod"
)

// ErrorType represents different categories of errors
type ErrorType int

const (
	ErrorTypeTransient ErrorType = iota // Temporary errors that can be retried
	ErrorTypePermanent                  // Permanent errors that should not be retried
	ErrorTypeTimeout                    // Timeout-related errors
	ErrorTypeRateLimit                  // Rate limiting errors
	ErrorTypeNetwork                    // Network connectivity errors
	ErrorTypeAuthentication             // Authentication/authorization errors
	ErrorTypeConfiguration              // Configuration errors
)

// LinkedInError represents a structured error with context
type LinkedInError struct {
	Type      ErrorType
	Operation string
	Message   string
	Cause     error
	Context   map[string]interface{}
	Timestamp time.Time
}

// Error implements the error interface
func (e *LinkedInError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Operation, e.Message)
}

// Unwrap returns the underlying error
func (e *LinkedInError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns true if the error can be retried
func (e *LinkedInError) IsRetryable() bool {
	return e.Type == ErrorTypeTransient || e.Type == ErrorTypeTimeout || e.Type == ErrorTypeNetwork
}

// NewError creates a new LinkedInError
func NewError(errorType ErrorType, operation, message string, cause error) *LinkedInError {
	return &LinkedInError{
		Type:      errorType,
		Operation: operation,
		Message:   message,
		Cause:     cause,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// WithContext adds context information to the error
func (e *LinkedInError) WithContext(key string, value interface{}) *LinkedInError {
	e.Context[key] = value
	return e
}

// RetryConfig defines retry behavior configuration
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors []ErrorType
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []ErrorType{
			ErrorTypeTransient,
			ErrorTypeTimeout,
			ErrorTypeNetwork,
		},
	}
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation func(ctx context.Context, attempt int) error

// RetryWithBackoff executes an operation with exponential backoff retry logic
func RetryWithBackoff(ctx context.Context, config RetryConfig, operation RetryableOperation) error {
	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute the operation
		err := operation(ctx, attempt)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry
		if !shouldRetry(err, config.RetryableErrors) {
			return err // Don't retry permanent errors
		}

		// Don't sleep after the last attempt
		if attempt == config.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateBackoffDelay(attempt, config)

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return lastErr
}

// shouldRetry determines if an error should be retried
func shouldRetry(err error, retryableTypes []ErrorType) bool {
	linkedInErr, ok := err.(*LinkedInError)
	if !ok {
		// For non-LinkedInError types, assume they might be retryable
		return true
	}

	for _, retryableType := range retryableTypes {
		if linkedInErr.Type == retryableType {
			return true
		}
	}

	return false
}

// calculateBackoffDelay calculates the delay for exponential backoff
func calculateBackoffDelay(attempt int, config RetryConfig) time.Duration {
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1))
	
	// Cap the delay at MaxDelay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	return time.Duration(delay)
}

// RodErrorHandler provides Rod-specific error handling utilities
type RodErrorHandler struct {
	defaultTimeout time.Duration
}

// NewRodErrorHandler creates a new Rod error handler
func NewRodErrorHandler(defaultTimeout time.Duration) *RodErrorHandler {
	return &RodErrorHandler{
		defaultTimeout: defaultTimeout,
	}
}

// SafeElementOperation performs a Rod element operation with proper error handling
func (reh *RodErrorHandler) SafeElementOperation(ctx context.Context, page *rod.Page, selector string, operation func(*rod.Element) error) error {
	if page == nil {
		return NewError(ErrorTypeConfiguration, "SafeElementOperation", "page cannot be nil", nil)
	}

	if selector == "" {
		return NewError(ErrorTypeConfiguration, "SafeElementOperation", "selector cannot be empty", nil)
	}

	// Create a timeout context if none provided
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), reh.defaultTimeout)
		defer cancel()
	}

	// Find the element with timeout
	element, err := page.Timeout(reh.defaultTimeout).Element(selector)
	if err != nil {
		return NewError(ErrorTypeTransient, "SafeElementOperation", 
			fmt.Sprintf("failed to find element with selector: %s", selector), err)
	}

	// Check if element is visible
	visible, err := element.Visible()
	if err != nil {
		return NewError(ErrorTypeTransient, "SafeElementOperation", 
			"failed to check element visibility", err)
	}

	if !visible {
		return NewError(ErrorTypeTransient, "SafeElementOperation", 
			fmt.Sprintf("element not visible: %s", selector), nil)
	}

	// Perform the operation
	err = operation(element)
	if err != nil {
		return NewError(ErrorTypeTransient, "SafeElementOperation", 
			"element operation failed", err)
	}

	return nil
}

// SafeNavigation performs Rod navigation with proper error handling and timeout
func (reh *RodErrorHandler) SafeNavigation(ctx context.Context, page *rod.Page, url string) error {
	if page == nil {
		return NewError(ErrorTypeConfiguration, "SafeNavigation", "page cannot be nil", nil)
	}

	if url == "" {
		return NewError(ErrorTypeConfiguration, "SafeNavigation", "URL cannot be empty", nil)
	}

	// Create a timeout context if none provided
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), reh.defaultTimeout)
		defer cancel()
	}

	// Navigate with timeout
	err := page.Timeout(reh.defaultTimeout).Navigate(url)
	if err != nil {
		return NewError(ErrorTypeNetwork, "SafeNavigation", 
			fmt.Sprintf("failed to navigate to URL: %s", url), err)
	}

	// Wait for page load with timeout
	err = page.Timeout(reh.defaultTimeout).WaitLoad()
	if err != nil {
		return NewError(ErrorTypeTimeout, "SafeNavigation", 
			"page load timeout", err)
	}

	return nil
}

// HandleRodError converts Rod errors to LinkedInError with appropriate categorization
func (reh *RodErrorHandler) HandleRodError(operation string, err error) error {
	if err == nil {
		return nil
	}

	// Categorize Rod errors
	errorMessage := err.Error()
	
	// Timeout errors
	if containsAny(errorMessage, []string{"timeout", "deadline", "context deadline exceeded"}) {
		return NewError(ErrorTypeTimeout, operation, "operation timed out", err)
	}

	// Network errors
	if containsAny(errorMessage, []string{"connection", "network", "dns", "resolve"}) {
		return NewError(ErrorTypeNetwork, operation, "network error", err)
	}

	// Element not found (usually retryable)
	if containsAny(errorMessage, []string{"element not found", "no such element", "cannot find"}) {
		return NewError(ErrorTypeTransient, operation, "element not found", err)
	}

	// Browser/page errors (usually retryable)
	if containsAny(errorMessage, []string{"page", "browser", "target"}) {
		return NewError(ErrorTypeTransient, operation, "browser/page error", err)
	}

	// Default to transient error
	return NewError(ErrorTypeTransient, operation, "rod operation failed", err)
}

// GracefulErrorRecovery provides a framework for graceful error recovery
type GracefulErrorRecovery struct {
	logger Logger
}

// Logger interface for error logging
type Logger interface {
	Error(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
}

// NewGracefulErrorRecovery creates a new graceful error recovery handler
func NewGracefulErrorRecovery(logger Logger) *GracefulErrorRecovery {
	return &GracefulErrorRecovery{
		logger: logger,
	}
}

// RecoverFromPanic recovers from panics and converts them to errors
func (ger *GracefulErrorRecovery) RecoverFromPanic(operation string) error {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic recovered: %v", r)
		
		if ger.logger != nil {
			ger.logger.Error("Panic recovered", map[string]interface{}{
				"operation": operation,
				"panic":     r,
			})
		}

		return NewError(ErrorTypePermanent, operation, "panic occurred", err)
	}
	return nil
}

// SafeExecute executes an operation with panic recovery
func (ger *GracefulErrorRecovery) SafeExecute(operation string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("panic recovered: %v", r)
			
			if ger.logger != nil {
				ger.logger.Error("Panic recovered", map[string]interface{}{
					"operation": operation,
					"panic":     r,
				})
			}

			err = NewError(ErrorTypePermanent, operation, "panic occurred", panicErr)
		}
	}()

	return fn()
}

// containsAny checks if a string contains any of the given substrings (case-insensitive)
func containsAny(text string, substrings []string) bool {
	lowerText := toLower(text)
	for _, substr := range substrings {
		if contains(lowerText, toLower(substr)) {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase
func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}