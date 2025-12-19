package connect

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

// ConnectionManager interface for LinkedIn connection requests
type ConnectionManager interface {
	SendConnectionRequest(ctx context.Context, page *rod.Page, profile ProfileResult, note string) error
	DetectConnectButton(ctx context.Context, page *rod.Page) (*rod.Element, error)
	TrackSentRequest(request ConnectionRequest) error
	NavigateToProfile(ctx context.Context, page *rod.Page, profileURL string) error
}

// ProfileResult represents a profile to connect with
type ProfileResult struct {
	URL         string
	Name        string
	Title       string
	Company     string
	Location    string
	Mutual      int
	Premium     bool
	Timestamp   time.Time
}

// ConnectionRequest represents a sent connection request
type ConnectionRequest struct {
	ProfileURL  string
	ProfileName string
	Note        string
	SentAt      time.Time
	Status      string // pending, accepted, declined
}

// ConnectManager implements ConnectionManager interface
type ConnectManager struct {
	storage     StorageInterface
	rateLimiter RateLimiterInterface
	stealth     StealthInterface
}

// StorageInterface defines storage operations needed by connect
type StorageInterface interface {
	SaveConnectionRequest(request ConnectionRequest) error
	GetSentRequests() ([]ConnectionRequest, error)
}

// RateLimiterInterface defines rate limiting operations
type RateLimiterInterface interface {
	CanSendConnection() bool
	RecordConnection()
}

// StealthInterface defines stealth operations needed by connect
type StealthInterface interface {
	HumanMouseMove(ctx context.Context, page *rod.Page, target *rod.Element) error
	HumanType(ctx context.Context, element *rod.Element, text string) error
	RandomDelay(min, max time.Duration) error
}

// NewConnectManager creates a new connection manager
func NewConnectManager(storage StorageInterface, rateLimiter RateLimiterInterface, stealth StealthInterface) *ConnectManager {
	return &ConnectManager{
		storage:     storage,
		rateLimiter: rateLimiter,
		stealth:     stealth,
	}
}

// NavigateToProfile navigates to a LinkedIn profile page using Rod methods
func (cm *ConnectManager) NavigateToProfile(ctx context.Context, page *rod.Page, profileURL string) error {
	if profileURL == "" {
		return fmt.Errorf("profile URL cannot be empty")
	}

	// Validate that this looks like a LinkedIn profile URL
	if !strings.Contains(profileURL, "linkedin.com/in/") {
		return fmt.Errorf("invalid LinkedIn profile URL: %s", profileURL)
	}

	// Navigate to the profile page
	err := page.Navigate(profileURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to profile %s: %w", profileURL, err)
	}

	// Wait for page to load
	err = page.WaitLoad()
	if err != nil {
		return fmt.Errorf("failed to wait for profile page to load: %w", err)
	}

	// Add a small delay to ensure page is fully rendered
	if cm.stealth != nil {
		err = cm.stealth.RandomDelay(1*time.Second, 3*time.Second)
		if err != nil {
			return fmt.Errorf("failed to add navigation delay: %w", err)
		}
	}

	return nil
}

// DetectConnectButton detects Connect buttons using Rod selectors
func (cm *ConnectManager) DetectConnectButton(ctx context.Context, page *rod.Page) (*rod.Element, error) {
	// Common LinkedIn Connect button selectors
	selectors := []string{
		`button[aria-label*="Connect"]`,
		`button[data-control-name="connect"]`,
		`button:has-text("Connect")`,
		`.pv-s-profile-actions button:has-text("Connect")`,
		`button[data-test-id="connect-cta"]`,
		`.artdeco-button--primary:has-text("Connect")`,
	}

	// Try each selector to find the Connect button
	for _, selector := range selectors {
		element, err := page.Element(selector)
		if err == nil && element != nil {
			// Verify the element is visible and clickable
			visible, err := element.Visible()
			if err == nil && visible {
				return element, nil
			}
		}
	}

	// If no Connect button found with standard selectors, try a more general approach
	buttons, err := page.Elements("button")
	if err != nil {
		return nil, fmt.Errorf("failed to find any buttons on page: %w", err)
	}

	for _, button := range buttons {
		text, err := button.Text()
		if err != nil {
			continue
		}

		// Check if button text contains "Connect" (case insensitive)
		if strings.Contains(strings.ToLower(text), "connect") {
			visible, err := button.Visible()
			if err == nil && visible {
				return button, nil
			}
		}

		// Also check aria-label
		ariaLabel, err := button.Attribute("aria-label")
		if err == nil && ariaLabel != nil && strings.Contains(strings.ToLower(*ariaLabel), "connect") {
			visible, err := button.Visible()
			if err == nil && visible {
				return button, nil
			}
		}
	}

	return nil, fmt.Errorf("no Connect button found on the page")
}

// SendConnectionRequest sends a connection request with optional personalized note
func (cm *ConnectManager) SendConnectionRequest(ctx context.Context, page *rod.Page, profile ProfileResult, note string) error {
	// Check rate limiting first
	if cm.rateLimiter != nil && !cm.rateLimiter.CanSendConnection() {
		return fmt.Errorf("rate limit exceeded, cannot send connection request")
	}

	if page == nil {
		return fmt.Errorf("page cannot be nil")
	}

	// Navigate to the profile
	err := cm.NavigateToProfile(ctx, page, profile.URL)
	if err != nil {
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	// Find the Connect button
	connectButton, err := cm.DetectConnectButton(ctx, page)
	if err != nil {
		return fmt.Errorf("failed to detect Connect button: %w", err)
	}

	// Use stealth behavior to move to and click the button
	if cm.stealth != nil {
		err = cm.stealth.HumanMouseMove(ctx, page, connectButton)
		if err != nil {
			return fmt.Errorf("failed to move mouse to Connect button: %w", err)
		}

		// Add a small delay before clicking
		err = cm.stealth.RandomDelay(500*time.Millisecond, 1500*time.Millisecond)
		if err != nil {
			return fmt.Errorf("failed to add pre-click delay: %w", err)
		}
	}

	// Click the Connect button
	err = connectButton.Click("left", 1)
	if err != nil {
		return fmt.Errorf("failed to click Connect button: %w", err)
	}

	// Wait for potential modal or note dialog
	time.Sleep(2 * time.Second)

	// If a note is provided, try to find and fill the note field
	if note != "" {
		err = cm.handleConnectionNote(ctx, page, note)
		if err != nil {
			// Don't fail the entire operation if note handling fails
			// Just log and continue
			fmt.Printf("Warning: failed to add note to connection request: %v\n", err)
		}
	}

	// Look for and click the final Send button
	err = cm.confirmConnectionRequest(ctx, page)
	if err != nil {
		return fmt.Errorf("failed to confirm connection request: %w", err)
	}

	// Record the connection request
	request := ConnectionRequest{
		ProfileURL:  profile.URL,
		ProfileName: profile.Name,
		Note:        note,
		SentAt:      time.Now(),
		Status:      "pending",
	}

	err = cm.TrackSentRequest(request)
	if err != nil {
		return fmt.Errorf("failed to track sent request: %w", err)
	}

	// Record with rate limiter
	if cm.rateLimiter != nil {
		cm.rateLimiter.RecordConnection()
	}

	return nil
}

// handleConnectionNote handles adding a personalized note to the connection request
func (cm *ConnectManager) handleConnectionNote(ctx context.Context, page *rod.Page, note string) error {
	// Common selectors for note input fields
	noteSelectors := []string{
		`textarea[name="message"]`,
		`textarea[aria-label*="message"]`,
		`textarea[placeholder*="message"]`,
		`.send-invite__custom-message textarea`,
		`#custom-message`,
	}

	var noteField *rod.Element
	var err error

	// Try to find the note input field
	for _, selector := range noteSelectors {
		noteField, err = page.Element(selector)
		if err == nil && noteField != nil {
			visible, err := noteField.Visible()
			if err == nil && visible {
				break
			}
		}
		noteField = nil
	}

	if noteField == nil {
		return fmt.Errorf("could not find note input field")
	}

	// Use stealth typing to fill the note
	if cm.stealth != nil {
		err = cm.stealth.HumanType(ctx, noteField, note)
		if err != nil {
			return fmt.Errorf("failed to type note: %w", err)
		}
	} else {
		err = noteField.Input(note)
		if err != nil {
			return fmt.Errorf("failed to input note: %w", err)
		}
	}

	return nil
}

// confirmConnectionRequest finds and clicks the final Send button
func (cm *ConnectManager) confirmConnectionRequest(ctx context.Context, page *rod.Page) error {
	// Common selectors for Send/Confirm buttons
	sendSelectors := []string{
		`button[aria-label*="Send"]`,
		`button:has-text("Send invitation")`,
		`button:has-text("Send")`,
		`.send-invite__actions button[type="submit"]`,
		`button[data-control-name="send_invite"]`,
	}

	var sendButton *rod.Element
	var err error

	// Try to find the Send button
	for _, selector := range sendSelectors {
		sendButton, err = page.Element(selector)
		if err == nil && sendButton != nil {
			visible, err := sendButton.Visible()
			if err == nil && visible {
				break
			}
		}
		sendButton = nil
	}

	if sendButton == nil {
		return fmt.Errorf("could not find Send button")
	}

	// Use stealth behavior to click the Send button
	if cm.stealth != nil {
		err = cm.stealth.HumanMouseMove(ctx, page, sendButton)
		if err != nil {
			return fmt.Errorf("failed to move mouse to Send button: %w", err)
		}

		err = cm.stealth.RandomDelay(500*time.Millisecond, 1000*time.Millisecond)
		if err != nil {
			return fmt.Errorf("failed to add pre-send delay: %w", err)
		}
	}

	// Click the Send button
	err = sendButton.Click("left", 1)
	if err != nil {
		return fmt.Errorf("failed to click Send button: %w", err)
	}

	// Wait for the request to be processed
	time.Sleep(2 * time.Second)

	return nil
}

// TrackSentRequest persists sent request data using storage module
func (cm *ConnectManager) TrackSentRequest(request ConnectionRequest) error {
	if cm.storage == nil {
		return fmt.Errorf("storage interface not configured")
	}

	err := cm.storage.SaveConnectionRequest(request)
	if err != nil {
		return fmt.Errorf("failed to save connection request: %w", err)
	}

	return nil
}