package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	
	"linkedin-automation-framework/internal/errors"
)

// Authenticator interface for LinkedIn authentication
type Authenticator interface {
	Login(ctx context.Context, page *rod.Page) error
	IsLoggedIn(ctx context.Context, page *rod.Page) (bool, error)
	HandleChallenge(ctx context.Context, page *rod.Page) error
	LoadCredentials() error
	SaveSession(path string) error
	LoadSession(path string) error
}

// Credentials represents login credentials
type Credentials struct {
	Username string
	Password string
}

// StealthTyper interface for human-like typing
type StealthTyper interface {
	HumanType(ctx context.Context, element *rod.Element, text string) error
	RandomDelay(min, max time.Duration) error
}

// AuthManager implements Authenticator interface
type AuthManager struct {
	credentials   Credentials
	stealthTyper  StealthTyper
	cookieManager CookieManager
	errorHandler  *errors.RodErrorHandler
	recovery      *errors.GracefulErrorRecovery
}

// CookieManager interface for cookie persistence
type CookieManager interface {
	SaveCookies(path string) error
	LoadCookies(path string) error
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(stealthTyper StealthTyper, cookieManager CookieManager) *AuthManager {
	return &AuthManager{
		stealthTyper:  stealthTyper,
		cookieManager: cookieManager,
		errorHandler:  errors.NewRodErrorHandler(30 * time.Second),
		recovery:      errors.NewGracefulErrorRecovery(nil),
	}
}

// LoadCredentials loads credentials from environment variables
func (am *AuthManager) LoadCredentials() error {
	username := os.Getenv("LINKEDIN_USERNAME")
	password := os.Getenv("LINKEDIN_PASSWORD")

	if username == "" {
		return fmt.Errorf("LINKEDIN_USERNAME environment variable not set")
	}
	if password == "" {
		return fmt.Errorf("LINKEDIN_PASSWORD environment variable not set")
	}

	am.credentials = Credentials{
		Username: username,
		Password: password,
	}

	return nil
}

// Login performs LinkedIn login using Rod navigation and stealth typing
func (am *AuthManager) Login(ctx context.Context, page *rod.Page) error {
	return am.recovery.SafeExecute("login", func() error {
		if page == nil {
			return errors.NewError(errors.ErrorTypeConfiguration, "login", 
				"page cannot be nil", nil)
		}

		retryConfig := errors.DefaultRetryConfig()
		retryConfig.MaxAttempts = 2 // Login should not be retried too many times
		retryConfig.InitialDelay = 5 * time.Second
		
		return errors.RetryWithBackoff(ctx, retryConfig, func(ctx context.Context, attempt int) error {
			// Navigate to LinkedIn login page
			err := am.errorHandler.SafeNavigation(ctx, page, "https://www.linkedin.com/login")
			if err != nil {
				return err
			}

			// Find and fill username field
			err = am.errorHandler.SafeElementOperation(ctx, page, "#username", func(element *rod.Element) error {
				if am.stealthTyper != nil {
					return am.stealthTyper.HumanType(ctx, element, am.credentials.Username)
				}
				return element.Input(am.credentials.Username)
			})
			if err != nil {
				return err
			}

			// Small delay between fields
			if am.stealthTyper != nil {
				am.stealthTyper.RandomDelay(500*time.Millisecond, 1500*time.Millisecond)
			}

			// Find and fill password field
			err = am.errorHandler.SafeElementOperation(ctx, page, "#password", func(element *rod.Element) error {
				if am.stealthTyper != nil {
					return am.stealthTyper.HumanType(ctx, element, am.credentials.Password)
				}
				return element.Input(am.credentials.Password)
			})
			if err != nil {
				return err
			}

			// Small delay before clicking submit
			if am.stealthTyper != nil {
				am.stealthTyper.RandomDelay(500*time.Millisecond, 1500*time.Millisecond)
			}

			// Find and click submit button
			err = am.errorHandler.SafeElementOperation(ctx, page, "button[type='submit']", func(element *rod.Element) error {
				return element.Click(proto.InputMouseButtonLeft, 1)
			})
			if err != nil {
				return err
			}

			// Wait for navigation after login
			time.Sleep(3 * time.Second)

			// Check if login was successful
			loggedIn, err := am.IsLoggedIn(ctx, page)
			if err != nil {
				return errors.NewError(errors.ErrorTypeTransient, "login", 
					"failed to check login state", err)
			}

			if !loggedIn {
				// Check for security challenges
				hasChallenge, err := am.detectChallenge(ctx, page)
				if err != nil {
					return errors.NewError(errors.ErrorTypeTransient, "login", 
						"failed to detect security challenge", err)
				}
				if hasChallenge {
					return errors.NewError(errors.ErrorTypePermanent, "login", 
						"security challenge detected (captcha or 2FA required)", nil)
				}
				return errors.NewError(errors.ErrorTypePermanent, "login", 
					"login failed - credentials may be incorrect", nil)
			}

			return nil
		})
	})
}

// IsLoggedIn detects login state via DOM analysis
func (am *AuthManager) IsLoggedIn(ctx context.Context, page *rod.Page) (bool, error) {
	// Check for common LinkedIn logged-in indicators
	// Method 1: Check for feed or home page elements
	feedElement, err := page.Timeout(5 * time.Second).Element(".feed-identity-module")
	if err == nil && feedElement != nil {
		return true, nil
	}

	// Method 2: Check for navigation bar with profile
	navProfile, err := page.Timeout(5 * time.Second).Element(".global-nav__me")
	if err == nil && navProfile != nil {
		return true, nil
	}

	// Method 3: Check URL - if we're on feed or home, we're logged in
	info, err := page.Info()
	if err == nil && info != nil {
		url := info.URL
		if url == "https://www.linkedin.com/feed/" || 
		   url == "https://www.linkedin.com/feed" ||
		   url == "https://www.linkedin.com/in/" {
			return true, nil
		}
	}

	// Method 4: Check for presence of login form (if present, not logged in)
	loginForm, err := page.Timeout(2 * time.Second).Element("#username")
	if err == nil && loginForm != nil {
		return false, nil
	}

	// Default to not logged in if we can't determine
	return false, nil
}

// HandleChallenge detects and handles security challenges
func (am *AuthManager) HandleChallenge(ctx context.Context, page *rod.Page) error {
	hasChallenge, err := am.detectChallenge(ctx, page)
	if err != nil {
		return fmt.Errorf("failed to detect challenge: %w", err)
	}

	if hasChallenge {
		return fmt.Errorf("security challenge detected - manual intervention required")
	}

	return nil
}

// detectChallenge detects security challenges (captcha, 2FA) without bypassing
func (am *AuthManager) detectChallenge(ctx context.Context, page *rod.Page) (bool, error) {
	// Check for CAPTCHA indicators
	captchaElement, err := page.Timeout(2 * time.Second).Element("#captcha-internal")
	if err == nil && captchaElement != nil {
		return true, nil
	}

	// Check for reCAPTCHA
	recaptcha, err := page.Timeout(2 * time.Second).Element(".g-recaptcha")
	if err == nil && recaptcha != nil {
		return true, nil
	}

	// Check for 2FA/verification code input
	verificationInput, err := page.Timeout(2 * time.Second).Element("input[name='pin']")
	if err == nil && verificationInput != nil {
		return true, nil
	}

	// Check for security verification page
	securityCheck, err := page.Timeout(2 * time.Second).Element(".security-verification")
	if err == nil && securityCheck != nil {
		return true, nil
	}

	// Check for challenge text in page content
	pageText, err := page.Timeout(2 * time.Second).Element("body")
	if err == nil && pageText != nil {
		text, err := pageText.Text()
		if err == nil {
			// Look for common challenge phrases
			if containsAny(text, []string{
				"verify",
				"verification",
				"security check",
				"unusual activity",
				"confirm your identity",
			}) {
				return true, nil
			}
		}
	}

	return false, nil
}

// SaveSession saves the current session cookies
func (am *AuthManager) SaveSession(path string) error {
	if am.cookieManager == nil {
		return fmt.Errorf("cookie manager not configured")
	}
	return am.cookieManager.SaveCookies(path)
}

// LoadSession loads a saved session from cookies
func (am *AuthManager) LoadSession(path string) error {
	if am.cookieManager == nil {
		return fmt.Errorf("cookie manager not configured")
	}
	return am.cookieManager.LoadCookies(path)
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