package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// BrowserManager interface for Rod browser lifecycle management
type BrowserManager interface {
	Initialize(ctx context.Context) error
	Browser() *rod.Browser
	NewPage() (*rod.Page, error)
	NewIncognitoPage() (*rod.Page, error)
	SaveCookies(path string) error
	LoadCookies(path string) error
	Close() error
}

// Manager implements BrowserManager interface
type Manager struct {
	browser *rod.Browser
	config  BrowserConfig
}

// BrowserConfig contains browser configuration options
type BrowserConfig struct {
	Headless   bool
	UserAgent  string
	ViewportW  int
	ViewportH  int
	Flags      []string
	CookiePath string
}

// NewManager creates a new browser manager instance
func NewManager(config BrowserConfig) *Manager {
	return &Manager{
		config: config,
	}
}

// Implement BrowserManager interface methods
func (m *Manager) Initialize(ctx context.Context) error {
	// Create launcher with configuration
	l := launcher.New()
	
	// Configure headless mode
	if m.config.Headless {
		l = l.Headless(true)
	} else {
		l = l.Headless(false)
	}
	
	// Apply common browser flags using available methods
	for _, flag := range m.config.Flags {
		switch flag {
		case "--no-sandbox":
			l = l.NoSandbox(true)
		case "--disable-dev-shm-usage":
			// This flag will be handled by Rod automatically in most cases
		case "--disable-web-security":
			// This flag will be handled by Rod automatically in most cases
		}
	}
	
	// User agent and viewport will be set later via page API
	
	// Launch browser
	url, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}
	
	// Connect to browser
	browser := rod.New().ControlURL(url)
	err = browser.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}
	
	m.browser = browser
	
	// Configure fingerprint settings
	err = m.configureFingerprint(ctx)
	if err != nil {
		return fmt.Errorf("failed to configure fingerprint: %w", err)
	}
	
	return nil
}

func (m *Manager) Browser() *rod.Browser {
	return m.browser
}

func (m *Manager) NewPage() (*rod.Page, error) {
	if m.browser == nil {
		return nil, fmt.Errorf("browser not initialized")
	}
	
	page, err := m.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	
	// Set viewport if configured
	if m.config.ViewportW > 0 && m.config.ViewportH > 0 {
		err = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width:  m.config.ViewportW,
			Height: m.config.ViewportH,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to set viewport: %w", err)
		}
	}
	
	return page, nil
}

func (m *Manager) NewIncognitoPage() (*rod.Page, error) {
	if m.browser == nil {
		return nil, fmt.Errorf("browser not initialized")
	}
	
	// Create incognito browser context
	incognito, err := m.browser.Incognito()
	if err != nil {
		return nil, fmt.Errorf("failed to create incognito context: %w", err)
	}
	
	page, err := incognito.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create incognito page: %w", err)
	}
	
	// Set viewport if configured
	if m.config.ViewportW > 0 && m.config.ViewportH > 0 {
		err = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width:  m.config.ViewportW,
			Height: m.config.ViewportH,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to set viewport: %w", err)
		}
	}
	
	return page, nil
}

func (m *Manager) SaveCookies(path string) error {
	if m.browser == nil {
		return fmt.Errorf("browser not initialized")
	}
	
	// Get all cookies from all pages
	pages, err := m.browser.Pages()
	if err != nil {
		return fmt.Errorf("failed to get pages: %w", err)
	}
	
	if len(pages) == 0 {
		return fmt.Errorf("no pages available to get cookies from")
	}
	
	// Use the first page to get cookies
	cookies, err := pages[0].Cookies([]string{})
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}
	
	// Marshal cookies to JSON
	data, err := json.Marshal(cookies)
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}
	
	// Write to file
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write cookies file: %w", err)
	}
	
	return nil
}

func (m *Manager) LoadCookies(path string) error {
	if m.browser == nil {
		return fmt.Errorf("browser not initialized")
	}
	
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("cookies file does not exist: %s", path)
	}
	
	// Read cookies file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read cookies file: %w", err)
	}
	
	// Unmarshal cookies
	var cookies []*proto.NetworkCookie
	err = json.Unmarshal(data, &cookies)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cookies: %w", err)
	}
	
	// Get pages to set cookies
	pages, err := m.browser.Pages()
	if err != nil {
		return fmt.Errorf("failed to get pages: %w", err)
	}
	
	if len(pages) == 0 {
		return fmt.Errorf("no pages available to set cookies")
	}
	
	// Convert cookies to the correct type for SetCookies
	cookieParams := make([]*proto.NetworkCookieParam, len(cookies))
	for i, cookie := range cookies {
		cookieParams[i] = &proto.NetworkCookieParam{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HTTPOnly,
			SameSite: cookie.SameSite,
		}
	}
	
	// Set cookies on the first page
	err = pages[0].SetCookies(cookieParams)
	if err != nil {
		return fmt.Errorf("failed to set cookies: %w", err)
	}
	
	return nil
}

func (m *Manager) Close() error {
	if m.browser == nil {
		return nil // Already closed or never initialized
	}
	
	// Close all pages first
	pages, err := m.browser.Pages()
	if err == nil {
		for _, page := range pages {
			_ = page.Close() // Ignore individual page close errors
		}
	}
	
	// Close browser
	err = m.browser.Close()
	if err != nil {
		return fmt.Errorf("failed to close browser: %w", err)
	}
	
	m.browser = nil
	return nil
}

// configureFingerprint applies fingerprint settings to mask browser automation
func (m *Manager) configureFingerprint(ctx context.Context) error {
	if m.browser == nil {
		return fmt.Errorf("browser not initialized")
	}
	
	// Get all pages to configure
	pages, err := m.browser.Pages()
	if err != nil {
		return fmt.Errorf("failed to get pages: %w", err)
	}
	
	// If no pages exist, create one temporarily for configuration
	if len(pages) == 0 {
		page, err := m.browser.Page(proto.TargetCreateTarget{})
		if err != nil {
			return fmt.Errorf("failed to create page for fingerprint configuration: %w", err)
		}
		pages = []*rod.Page{page}
	}
	
	// Configure each page
	for _, page := range pages {
		// Mask webdriver property
		_, err := page.Eval(`() => {
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined,
			});
		}`)
		if err != nil {
			return fmt.Errorf("failed to mask webdriver property: %w", err)
		}
		
		// Set user agent if configured
		if m.config.UserAgent != "" {
			err = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
				UserAgent: m.config.UserAgent,
			})
			if err != nil {
				return fmt.Errorf("failed to set user agent: %w", err)
			}
		}
		
		// Configure additional fingerprint properties
		_, err = page.Eval(`() => {
			// Override plugins
			Object.defineProperty(navigator, 'plugins', {
				get: () => [1, 2, 3, 4, 5],
			});
			
			// Override languages
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en'],
			});
			
			// Override permissions
			const originalQuery = window.navigator.permissions.query;
			window.navigator.permissions.query = (parameters) => (
				parameters.name === 'notifications' ?
					Promise.resolve({ state: Notification.permission }) :
					originalQuery(parameters)
			);
		}`)
		if err != nil {
			return fmt.Errorf("failed to configure fingerprint properties: %w", err)
		}
	}
	
	return nil
}