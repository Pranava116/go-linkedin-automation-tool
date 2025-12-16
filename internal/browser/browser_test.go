package browser

import (
	"testing"

	"pgregory.net/rapid"
)

// **Feature: linkedin-automation-framework, Property 1: Browser initialization consistency**
// **Validates: Requirements 1.1**
func TestBrowserInitializationConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random browser configuration
		headless := rapid.Bool().Draw(t, "headless")
		userAgent := rapid.StringMatching(`Mozilla/5\.0 \(.*\) AppleWebKit/537\.36`).Draw(t, "userAgent")
		viewportW := rapid.IntRange(800, 1920).Draw(t, "viewportW")
		viewportH := rapid.IntRange(600, 1080).Draw(t, "viewportH")
		
		config := BrowserConfig{
			Headless:  headless,
			UserAgent: userAgent,
			ViewportW: viewportW,
			ViewportH: viewportH,
			Flags:     []string{"--no-sandbox", "--disable-dev-shm-usage"},
		}

		manager := NewManager(config)

		// Property: Browser manager should be created consistently with valid configuration
		if manager == nil {
			t.Fatal("Browser manager creation failed")
		}

		// Verify configuration is properly stored
		if manager.config.Headless != headless {
			t.Fatalf("Headless configuration mismatch: expected %v, got %v", headless, manager.config.Headless)
		}

		if manager.config.UserAgent != userAgent {
			t.Fatalf("UserAgent configuration mismatch: expected %s, got %s", userAgent, manager.config.UserAgent)
		}

		if manager.config.ViewportW != viewportW {
			t.Fatalf("ViewportW configuration mismatch: expected %d, got %d", viewportW, manager.config.ViewportW)
		}

		if manager.config.ViewportH != viewportH {
			t.Fatalf("ViewportH configuration mismatch: expected %d, got %d", viewportH, manager.config.ViewportH)
		}
	})
}

// **Feature: linkedin-automation-framework, Property 2: Mode configuration support**
// **Validates: Requirements 1.2**
func TestModeConfigurationSupport(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random mode configuration
		headless := rapid.Bool().Draw(t, "headless")
		
		config := BrowserConfig{
			Headless:  headless,
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			ViewportW: 1920,
			ViewportH: 1080,
			Flags:     []string{"--no-sandbox", "--disable-dev-shm-usage"},
		}

		manager := NewManager(config)

		// Property: Browser manager should support both headless and non-headless mode configuration
		if manager == nil {
			t.Fatal("Browser manager creation failed")
		}

		// Verify mode configuration is properly stored
		if manager.config.Headless != headless {
			t.Fatalf("Mode configuration failed: expected headless=%v, got %v", headless, manager.config.Headless)
		}

		// Verify that configuration is consistent
		expectedUserAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
		if manager.config.UserAgent != expectedUserAgent {
			t.Fatalf("UserAgent configuration mismatch: expected %s, got %s", expectedUserAgent, manager.config.UserAgent)
		}

		// Verify viewport configuration
		if manager.config.ViewportW != 1920 || manager.config.ViewportH != 1080 {
			t.Fatalf("Viewport configuration mismatch: expected 1920x1080, got %dx%d", 
				manager.config.ViewportW, manager.config.ViewportH)
		}
	})
}

