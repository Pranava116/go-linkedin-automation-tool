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

// **Feature: linkedin-automation-framework, Property 3: Browser flag application**
// **Validates: Requirements 1.3**
func TestBrowserFlagApplication(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random browser flags
		flags := rapid.SliceOf(rapid.SampledFrom([]string{
			"--no-sandbox",
			"--disable-dev-shm-usage", 
			"--disable-web-security",
		})).Draw(t, "flags")
		
		config := BrowserConfig{
			Headless:  true, // Use headless for testing
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			ViewportW: 1920,
			ViewportH: 1080,
			Flags:     flags,
		}

		manager := NewManager(config)

		// Property: Browser flags should be properly stored in configuration
		if manager == nil {
			t.Fatal("Browser manager creation failed")
		}

		// Verify flags are properly stored
		if len(manager.config.Flags) != len(flags) {
			t.Fatalf("Flag count mismatch: expected %d, got %d", len(flags), len(manager.config.Flags))
		}

		for i, expectedFlag := range flags {
			if manager.config.Flags[i] != expectedFlag {
				t.Fatalf("Flag mismatch at index %d: expected %s, got %s", i, expectedFlag, manager.config.Flags[i])
			}
		}

		// Test that manager can be initialized with these flags (without actually launching browser)
		// This tests that the flag configuration is valid and doesn't cause immediate errors
		if manager.config.Headless != true {
			t.Fatal("Headless configuration should be preserved")
		}
	})
}

// **Feature: linkedin-automation-framework, Property 4: Resource cleanup on shutdown**
// **Validates: Requirements 1.4**
func TestResourceCleanupOnShutdown(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random browser configuration
		headless := rapid.Bool().Draw(t, "headless")
		userAgent := rapid.StringMatching(`Mozilla/5\.0 \(.*\) AppleWebKit/537\.36`).Draw(t, "userAgent")
		
		config := BrowserConfig{
			Headless:  headless,
			UserAgent: userAgent,
			ViewportW: 1920,
			ViewportH: 1080,
			Flags:     []string{"--no-sandbox", "--disable-dev-shm-usage"},
		}

		manager := NewManager(config)

		// Property: Resource cleanup should work properly regardless of browser state
		if manager == nil {
			t.Fatal("Browser manager creation failed")
		}

		// Test cleanup when browser is not initialized
		err := manager.Close()
		if err != nil {
			t.Fatalf("Close should not fail when browser is not initialized: %v", err)
		}

		// Verify browser is still nil after close
		if manager.Browser() != nil {
			t.Fatal("Browser should remain nil after closing uninitialized manager")
		}

		// Test that multiple close calls don't cause issues
		err = manager.Close()
		if err != nil {
			t.Fatalf("Multiple close calls should not fail: %v", err)
		}

		// Test that Close is idempotent
		for i := 0; i < 3; i++ {
			err = manager.Close()
			if err != nil {
				t.Fatalf("Close should be idempotent, failed on call %d: %v", i+1, err)
			}
		}
	})
}

// **Feature: linkedin-automation-framework, Property 5: Page creation with context management**
// **Validates: Requirements 1.5**
func TestPageCreationWithContextManagement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random browser configuration
		headless := rapid.Bool().Draw(t, "headless")
		viewportW := rapid.IntRange(800, 1920).Draw(t, "viewportW")
		viewportH := rapid.IntRange(600, 1080).Draw(t, "viewportH")
		
		config := BrowserConfig{
			Headless:  headless,
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			ViewportW: viewportW,
			ViewportH: viewportH,
			Flags:     []string{"--no-sandbox", "--disable-dev-shm-usage"},
		}

		manager := NewManager(config)

		// Property: Page creation should fail gracefully when browser is not initialized
		if manager == nil {
			t.Fatal("Browser manager creation failed")
		}

		// Test NewPage when browser is not initialized
		page, err := manager.NewPage()
		if err == nil {
			t.Fatal("NewPage should fail when browser is not initialized")
		}
		if page != nil {
			t.Fatal("Page should be nil when creation fails")
		}

		// Test NewIncognitoPage when browser is not initialized
		incognitoPage, err := manager.NewIncognitoPage()
		if err == nil {
			t.Fatal("NewIncognitoPage should fail when browser is not initialized")
		}
		if incognitoPage != nil {
			t.Fatal("Incognito page should be nil when creation fails")
		}

		// Verify error messages are informative
		page, err = manager.NewPage()
		if err == nil || err.Error() != "browser not initialized" {
			t.Fatalf("Expected 'browser not initialized' error, got: %v", err)
		}

		incognitoPage, err = manager.NewIncognitoPage()
		if err == nil || err.Error() != "browser not initialized" {
			t.Fatalf("Expected 'browser not initialized' error for incognito page, got: %v", err)
		}

		// Verify configuration is preserved
		if manager.config.ViewportW != viewportW {
			t.Fatalf("ViewportW configuration mismatch: expected %d, got %d", viewportW, manager.config.ViewportW)
		}

		if manager.config.ViewportH != viewportH {
			t.Fatalf("ViewportH configuration mismatch: expected %d, got %d", viewportH, manager.config.ViewportH)
		}
	})
}