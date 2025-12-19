package errors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"pgregory.net/rapid"
)

// **Feature: linkedin-automation-framework, Property 42: Graceful error handling**
// **Validates: Requirements 8.3**
func TestGracefulErrorHandling(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random error scenarios
		errorType := rapid.SampledFrom([]ErrorType{
			ErrorTypeTransient,
			ErrorTypePermanent,
			ErrorTypeTimeout,
			ErrorTypeRateLimit,
			ErrorTypeNetwork,
			ErrorTypeAuthentication,
			ErrorTypeConfiguration,
		}).Draw(t, "errorType")

		operation := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]*`).Draw(t, "operation")
		message := rapid.String().Draw(t, "message")
		
		// Create a LinkedInError
		linkedInErr := NewError(errorType, operation, message, nil)
		
		// Test that error creation doesn't panic
		if linkedInErr == nil {
			t.Fatalf("NewError returned nil")
		}
		
		// Test that error implements error interface properly
		errorString := linkedInErr.Error()
		if errorString == "" {
			t.Fatalf("Error() returned empty string")
		}
		
		// Test that error contains operation and message
		if !contains(errorString, operation) {
			t.Fatalf("Error string doesn't contain operation: %s", errorString)
		}
		
		// Test graceful error recovery
		recovery := NewGracefulErrorRecovery(nil)
		
		// Test SafeExecute with normal operation (should not panic)
		err := recovery.SafeExecute("test_operation", func() error {
			return linkedInErr
		})
		
		if err != linkedInErr {
			t.Fatalf("SafeExecute should return the original error, got: %v", err)
		}
		
		// Test SafeExecute with panicking operation (should recover gracefully)
		err = recovery.SafeExecute("panic_operation", func() error {
			panic("test panic")
		})
		
		if err == nil {
			t.Fatalf("SafeExecute should return error when panic occurs")
		}
		
		// Verify the panic was converted to a proper error
		linkedInPanicErr, ok := err.(*LinkedInError)
		if !ok {
			t.Fatalf("Panic should be converted to LinkedInError")
		}
		
		if linkedInPanicErr.Type != ErrorTypePermanent {
			t.Fatalf("Panic errors should be permanent, got: %v", linkedInPanicErr.Type)
		}
	})
}

// Mock logger for testing
type mockLogger struct {
	errorCalls []logCall
	warnCalls  []logCall
	infoCalls  []logCall
}

type logCall struct {
	msg    string
	fields map[string]interface{}
}

func (ml *mockLogger) Error(msg string, fields map[string]interface{}) {
	ml.errorCalls = append(ml.errorCalls, logCall{msg: msg, fields: fields})
}

func (ml *mockLogger) Warn(msg string, fields map[string]interface{}) {
	ml.warnCalls = append(ml.warnCalls, logCall{msg: msg, fields: fields})
}

func (ml *mockLogger) Info(msg string, fields map[string]interface{}) {
	ml.infoCalls = append(ml.infoCalls, logCall{msg: msg, fields: fields})
}

// Test graceful error handling with different error types
func TestErrorTypeClassification(t *testing.T) {
	testCases := []struct {
		errorType ErrorType
		retryable bool
	}{
		{ErrorTypeTransient, true},
		{ErrorTypePermanent, false},
		{ErrorTypeTimeout, true},
		{ErrorTypeRateLimit, false},
		{ErrorTypeNetwork, true},
		{ErrorTypeAuthentication, false},
		{ErrorTypeConfiguration, false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("ErrorType_%d", tc.errorType), func(t *testing.T) {
			err := NewError(tc.errorType, "test_op", "test message", nil)
			
			if err.IsRetryable() != tc.retryable {
				t.Errorf("Expected retryable=%v for error type %v, got %v", 
					tc.retryable, tc.errorType, err.IsRetryable())
			}
		})
	}
}

// Test Rod error handling
func TestRodErrorHandling(t *testing.T) {
	handler := NewRodErrorHandler(5 * time.Second)
	
	testCases := []struct {
		name          string
		rodError      error
		expectedType  ErrorType
	}{
		{
			name:         "timeout error",
			rodError:     fmt.Errorf("context deadline exceeded"),
			expectedType: ErrorTypeTimeout,
		},
		{
			name:         "network error",
			rodError:     fmt.Errorf("connection refused"),
			expectedType: ErrorTypeNetwork,
		},
		{
			name:         "element not found",
			rodError:     fmt.Errorf("element not found"),
			expectedType: ErrorTypeTransient,
		},
		{
			name:         "generic error",
			rodError:     fmt.Errorf("some other error"),
			expectedType: ErrorTypeTransient,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := handler.HandleRodError("test_operation", tc.rodError)
			
			linkedInErr, ok := err.(*LinkedInError)
			if !ok {
				t.Fatalf("Expected LinkedInError, got %T", err)
			}
			
			if linkedInErr.Type != tc.expectedType {
				t.Errorf("Expected error type %v, got %v", tc.expectedType, linkedInErr.Type)
			}
		})
	}
}

// Test panic recovery with logger
func TestPanicRecoveryWithLogger(t *testing.T) {
	logger := &mockLogger{}
	recovery := NewGracefulErrorRecovery(logger)
	
	err := recovery.SafeExecute("test_panic", func() error {
		panic("test panic message")
	})
	
	if err == nil {
		t.Fatal("Expected error from panic recovery")
	}
	
	if len(logger.errorCalls) != 1 {
		t.Fatalf("Expected 1 error log call, got %d", len(logger.errorCalls))
	}
	
	logCall := logger.errorCalls[0]
	if logCall.msg != "Panic recovered" {
		t.Errorf("Expected log message 'Panic recovered', got '%s'", logCall.msg)
	}
	
	if logCall.fields["operation"] != "test_panic" {
		t.Errorf("Expected operation 'test_panic' in log fields, got %v", logCall.fields["operation"])
	}
}

// Test error context
func TestErrorContext(t *testing.T) {
	err := NewError(ErrorTypeTransient, "test_op", "test message", nil)
	
	err.WithContext("profile_url", "https://linkedin.com/in/test")
	err.WithContext("attempt", 2)
	
	if err.Context["profile_url"] != "https://linkedin.com/in/test" {
		t.Errorf("Expected profile_url context, got %v", err.Context["profile_url"])
	}
	
	if err.Context["attempt"] != 2 {
		t.Errorf("Expected attempt context, got %v", err.Context["attempt"])
	}
}

// **Feature: linkedin-automation-framework, Property 43: Exponential backoff retry**
// **Validates: Requirements 8.4**
func TestExponentialBackoffRetry(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random retry configuration
		maxAttempts := rapid.IntRange(1, 10).Draw(t, "maxAttempts")
		initialDelayMs := rapid.IntRange(10, 1000).Draw(t, "initialDelayMs")
		maxDelayMs := rapid.IntRange(initialDelayMs*2, 30000).Draw(t, "maxDelayMs")
		backoffFactor := rapid.Float64Range(1.1, 5.0).Draw(t, "backoffFactor")
		
		config := RetryConfig{
			MaxAttempts:   maxAttempts,
			InitialDelay:  time.Duration(initialDelayMs) * time.Millisecond,
			MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
			BackoffFactor: backoffFactor,
			RetryableErrors: []ErrorType{ErrorTypeTransient, ErrorTypeTimeout, ErrorTypeNetwork},
		}
		
		// Test successful operation (should not retry)
		attemptCount := 0
		err := RetryWithBackoff(context.Background(), config, func(ctx context.Context, attempt int) error {
			attemptCount++
			return nil // Success on first attempt
		})
		
		if err != nil {
			t.Fatalf("Expected no error for successful operation, got: %v", err)
		}
		
		if attemptCount != 1 {
			t.Fatalf("Expected 1 attempt for successful operation, got: %d", attemptCount)
		}
		
		// Test permanent error (should not retry)
		attemptCount = 0
		permanentErr := NewError(ErrorTypePermanent, "test", "permanent error", nil)
		err = RetryWithBackoff(context.Background(), config, func(ctx context.Context, attempt int) error {
			attemptCount++
			return permanentErr
		})
		
		if err != permanentErr {
			t.Fatalf("Expected permanent error to be returned, got: %v", err)
		}
		
		if attemptCount != 1 {
			t.Fatalf("Expected 1 attempt for permanent error, got: %d", attemptCount)
		}
		
		// Test retryable error that eventually succeeds (only if maxAttempts > 1)
		if maxAttempts > 1 {
			attemptCount = 0
			successAfter := rapid.IntRange(2, maxAttempts).Draw(t, "successAfter")
			transientErr := NewError(ErrorTypeTransient, "test", "transient error", nil)
			
			err = RetryWithBackoff(context.Background(), config, func(ctx context.Context, attempt int) error {
				attemptCount++
				if attempt < successAfter {
					return transientErr
				}
				return nil // Success after retries
			})
			
			if err != nil {
				t.Fatalf("Expected success after retries, got: %v", err)
			}
			
			if attemptCount != successAfter {
				t.Fatalf("Expected %d attempts, got: %d", successAfter, attemptCount)
			}
		}
		
		// Test retryable error that exhausts all attempts
		attemptCount = 0
		transientErr := NewError(ErrorTypeTransient, "test", "transient error", nil)
		
		err = RetryWithBackoff(context.Background(), config, func(ctx context.Context, attempt int) error {
			attemptCount++
			return transientErr // Always fail
		})
		
		if err != transientErr {
			t.Fatalf("Expected transient error after exhausting attempts, got: %v", err)
		}
		
		if attemptCount != maxAttempts {
			t.Fatalf("Expected %d attempts, got: %d", maxAttempts, attemptCount)
		}
		
		if attemptCount != maxAttempts {
			t.Fatalf("Expected %d attempts, got: %d", maxAttempts, attemptCount)
		}
	})
}

// Test exponential backoff delay calculation
func TestBackoffDelayCalculation(t *testing.T) {
	config := RetryConfig{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}
	
	testCases := []struct {
		attempt      int
		expectedMin  time.Duration
		expectedMax  time.Duration
	}{
		{1, 100 * time.Millisecond, 100 * time.Millisecond},   // 100ms * 2^0
		{2, 200 * time.Millisecond, 200 * time.Millisecond},   // 100ms * 2^1
		{3, 400 * time.Millisecond, 400 * time.Millisecond},   // 100ms * 2^2
		{4, 800 * time.Millisecond, 800 * time.Millisecond},   // 100ms * 2^3
		{10, 5 * time.Second, 5 * time.Second},                // Capped at MaxDelay
	}
	
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("attempt_%d", tc.attempt), func(t *testing.T) {
			delay := calculateBackoffDelay(tc.attempt, config)
			
			if delay < tc.expectedMin || delay > tc.expectedMax {
				t.Errorf("Attempt %d: expected delay between %v and %v, got %v", 
					tc.attempt, tc.expectedMin, tc.expectedMax, delay)
			}
		})
	}
}

// Test context cancellation during retry
func TestRetryContextCancellation(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxAttempts = 10
	config.InitialDelay = 100 * time.Millisecond
	
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	attemptCount := 0
	transientErr := NewError(ErrorTypeTransient, "test", "transient error", nil)
	
	err := RetryWithBackoff(ctx, config, func(ctx context.Context, attempt int) error {
		attemptCount++
		return transientErr // Always fail to trigger retries
	})
	
	// Should return context error when cancelled
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
	}
	
	// Should have attempted at least once but not all attempts
	if attemptCount == 0 {
		t.Error("Expected at least one attempt")
	}
	
	if attemptCount >= config.MaxAttempts {
		t.Errorf("Expected fewer than %d attempts due to context cancellation, got %d", 
			config.MaxAttempts, attemptCount)
	}
}

// **Feature: linkedin-automation-framework, Property 44: Rod timeout and context usage**
// **Validates: Requirements 8.5**
func TestRodTimeoutUsage(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random timeout values
		timeoutMs := rapid.IntRange(100, 5000).Draw(t, "timeoutMs")
		timeout := time.Duration(timeoutMs) * time.Millisecond
		
		// Test RodErrorHandler with different timeout values
		handler := NewRodErrorHandler(timeout)
		
		if handler.defaultTimeout != timeout {
			t.Fatalf("Expected default timeout %v, got %v", timeout, handler.defaultTimeout)
		}
		
		// Test SafeNavigation with nil context (should create timeout context)
		err := handler.SafeNavigation(nil, nil, "")
		if err == nil {
			t.Fatal("Expected error for nil page")
		}
		
		linkedInErr, ok := err.(*LinkedInError)
		if !ok {
			t.Fatalf("Expected LinkedInError, got %T", err)
		}
		
		if linkedInErr.Type != ErrorTypeConfiguration {
			t.Fatalf("Expected configuration error for nil page, got %v", linkedInErr.Type)
		}
		
		// Test SafeElementOperation with nil context (should create timeout context)
		err = handler.SafeElementOperation(nil, nil, "test-selector", func(element *rod.Element) error {
			return nil
		})
		
		if err == nil {
			t.Fatal("Expected error for nil page")
		}
		
		linkedInErr, ok = err.(*LinkedInError)
		if !ok {
			t.Fatalf("Expected LinkedInError, got %T", err)
		}
		
		if linkedInErr.Type != ErrorTypeConfiguration {
			t.Fatalf("Expected configuration error for nil page, got %v", linkedInErr.Type)
		}
		
		// Test empty selector validation
		err = handler.SafeElementOperation(context.Background(), nil, "", func(element *rod.Element) error {
			return nil
		})
		
		if err == nil {
			t.Fatal("Expected error for empty selector")
		}
		
		linkedInErr, ok = err.(*LinkedInError)
		if !ok {
			t.Fatalf("Expected LinkedInError, got %T", err)
		}
		
		if linkedInErr.Type != ErrorTypeConfiguration {
			t.Fatalf("Expected configuration error for empty selector, got %v", linkedInErr.Type)
		}
		
		// Test empty URL validation
		err = handler.SafeNavigation(context.Background(), nil, "")
		
		if err == nil {
			t.Fatal("Expected error for empty URL")
		}
		
		linkedInErr, ok = err.(*LinkedInError)
		if !ok {
			t.Fatalf("Expected LinkedInError, got %T", err)
		}
		
		if linkedInErr.Type != ErrorTypeConfiguration {
			t.Fatalf("Expected configuration error for empty URL, got %v", linkedInErr.Type)
		}
	})
}

// Test Rod error categorization
func TestRodErrorCategorization(t *testing.T) {
	handler := NewRodErrorHandler(5 * time.Second)
	
	testCases := []struct {
		name         string
		errorMessage string
		expectedType ErrorType
	}{
		{
			name:         "timeout error",
			errorMessage: "context deadline exceeded",
			expectedType: ErrorTypeTimeout,
		},
		{
			name:         "timeout error variant",
			errorMessage: "operation timeout",
			expectedType: ErrorTypeTimeout,
		},
		{
			name:         "network error",
			errorMessage: "connection refused",
			expectedType: ErrorTypeNetwork,
		},
		{
			name:         "dns error",
			errorMessage: "dns resolution failed",
			expectedType: ErrorTypeNetwork,
		},
		{
			name:         "element not found",
			errorMessage: "element not found",
			expectedType: ErrorTypeTransient,
		},
		{
			name:         "no such element",
			errorMessage: "no such element",
			expectedType: ErrorTypeTransient,
		},
		{
			name:         "page error",
			errorMessage: "page crashed",
			expectedType: ErrorTypeTransient,
		},
		{
			name:         "browser error",
			errorMessage: "browser disconnected",
			expectedType: ErrorTypeTransient,
		},
		{
			name:         "generic error",
			errorMessage: "unknown error occurred",
			expectedType: ErrorTypeTransient,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalErr := fmt.Errorf("%s", tc.errorMessage)
			err := handler.HandleRodError("test_operation", originalErr)
			
			linkedInErr, ok := err.(*LinkedInError)
			if !ok {
				t.Fatalf("Expected LinkedInError, got %T", err)
			}
			
			if linkedInErr.Type != tc.expectedType {
				t.Errorf("Expected error type %v for message '%s', got %v", 
					tc.expectedType, tc.errorMessage, linkedInErr.Type)
			}
			
			if linkedInErr.Operation != "test_operation" {
				t.Errorf("Expected operation 'test_operation', got '%s'", linkedInErr.Operation)
			}
			
			if linkedInErr.Cause != originalErr {
				t.Errorf("Expected cause to be original error, got %v", linkedInErr.Cause)
			}
		})
	}
}

// Test timeout context creation and usage
func TestTimeoutContextCreation(t *testing.T) {
	handler := NewRodErrorHandler(2 * time.Second)
	
	// Test that operations respect timeout
	start := time.Now()
	
	// This should fail quickly due to nil page, not due to timeout
	err := handler.SafeNavigation(nil, nil, "https://example.com")
	
	elapsed := time.Since(start)
	
	// Should fail immediately due to nil page validation, not timeout
	if elapsed > 100*time.Millisecond {
		t.Errorf("Expected quick failure for nil page, took %v", elapsed)
	}
	
	if err == nil {
		t.Fatal("Expected error for nil page")
	}
	
	linkedInErr, ok := err.(*LinkedInError)
	if !ok {
		t.Fatalf("Expected LinkedInError, got %T", err)
	}
	
	if linkedInErr.Type != ErrorTypeConfiguration {
		t.Fatalf("Expected configuration error, got %v", linkedInErr.Type)
	}
}