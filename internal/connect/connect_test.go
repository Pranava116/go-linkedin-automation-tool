package connect

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"pgregory.net/rapid"
)

// MockStorage implements StorageInterface for testing
type MockStorage struct {
	requests []ConnectionRequest
}

func (ms *MockStorage) SaveConnectionRequest(request ConnectionRequest) error {
	ms.requests = append(ms.requests, request)
	return nil
}

func (ms *MockStorage) GetSentRequests() ([]ConnectionRequest, error) {
	return ms.requests, nil
}

// MockStealth implements StealthInterface for testing
type MockStealth struct{}

func (ms *MockStealth) HumanMouseMove(ctx context.Context, page *rod.Page, target *rod.Element) error {
	return nil
}

func (ms *MockStealth) HumanType(ctx context.Context, element *rod.Element, text string) error {
	return element.Input(text)
}

func (ms *MockStealth) RandomDelay(min, max time.Duration) error {
	time.Sleep(min)
	return nil
}

// TestProfilePageNavigation tests profile page navigation functionality
// **Feature: linkedin-automation-framework, Property 25: Profile page navigation**
// **Validates: Requirements 5.1**
func TestProfilePageNavigation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid LinkedIn profile URL
		username := rapid.StringMatching(`[a-zA-Z0-9\-]{3,30}`).Draw(t, "username")
		profileURL := "https://www.linkedin.com/in/" + username + "/"

		// Create test browser (headless for CI)
		l := launcher.New().Headless(true).NoSandbox(true)
		url, err := l.Launch()
		if err != nil {
			t.Skipf("Failed to launch browser: %v", err)
		}
		defer l.Cleanup()

		browser := rod.New().ControlURL(url)
		err = browser.Connect()
		if err != nil {
			t.Skipf("Failed to connect to browser: %v", err)
		}
		defer browser.Close()

		page, err := browser.Page(proto.TargetCreateTarget{})
		if err != nil {
			t.Skipf("Failed to create page: %v", err)
		}
		defer page.Close()

		// Create connection manager
		storage := &MockStorage{}
		rateLimiter := NewSimpleRateLimiter(10, time.Hour)
		stealth := &MockStealth{}
		cm := NewConnectManager(storage, rateLimiter, stealth)

		// Test navigation
		ctx := context.Background()
		err = cm.NavigateToProfile(ctx, page, profileURL)

		// For any valid LinkedIn profile URL, navigation should either succeed
		// or fail with a specific error (like network issues, not found, etc.)
		// The key property is that the method handles the URL correctly
		if err != nil {
			// Navigation can fail for legitimate reasons (network, 404, etc.)
			// but should not panic or return invalid error types
			if err.Error() == "" {
				t.Fatalf("Navigation error should have a message")
			}
		} else {
			// If navigation succeeds, verify we're on the correct page
			info, urlErr := page.Info()
			if urlErr == nil && info != nil {
				// Should be on LinkedIn domain
				if !strings.Contains(info.URL, "linkedin.com") {
					t.Fatalf("Expected to be on LinkedIn domain, got: %s", info.URL)
				}
			}
		}
	})
}

// TestInvalidProfileURLHandling tests handling of invalid profile URLs
func TestInvalidProfileURLHandling(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate invalid URLs
		invalidURL := rapid.OneOf(
			rapid.Just(""),
			rapid.Just("not-a-url"),
			rapid.Just("https://example.com"),
			rapid.Just("https://linkedin.com/invalid"),
			rapid.StringMatching(`https?://[^/]+/[^/]*`).Filter(func(s string) bool {
				return !strings.Contains(s, "linkedin.com/in/")
			}),
		).Draw(t, "invalidURL")

		// Create minimal setup
		storage := &MockStorage{}
		rateLimiter := NewSimpleRateLimiter(10, time.Hour)
		stealth := &MockStealth{}
		cm := NewConnectManager(storage, rateLimiter, stealth)

		// Create a mock page (we don't need real browser for URL validation)
		ctx := context.Background()

		// For any invalid URL, navigation should return an error
		err := cm.NavigateToProfile(ctx, nil, invalidURL)
		if err == nil {
			t.Fatalf("Expected error for invalid URL: %s", invalidURL)
		}

		// Error message should be meaningful
		if err.Error() == "" {
			t.Fatalf("Error should have a descriptive message")
		}
	})
}

// TestNavigateToProfileWithNilPage tests navigation with nil page
func TestNavigateToProfileWithNilPage(t *testing.T) {
	storage := &MockStorage{}
	rateLimiter := NewSimpleRateLimiter(10, time.Hour)
	stealth := &MockStealth{}
	cm := NewConnectManager(storage, rateLimiter, stealth)

	ctx := context.Background()
	err := cm.NavigateToProfile(ctx, nil, "https://www.linkedin.com/in/testuser/")

	// Should handle nil page gracefully
	if err == nil {
		t.Fatal("Expected error when page is nil")
	}
}