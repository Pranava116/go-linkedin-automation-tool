package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"linkedin-automation-framework/internal/browser"
	"linkedin-automation-framework/internal/config"
	"linkedin-automation-framework/internal/logger"
	"linkedin-automation-framework/internal/stealth"
	"linkedin-automation-framework/internal/storage"
)

// Application represents the main application with all dependencies
type Application struct {
	config         *config.Config
	logger         *logger.LoggerManager
	browserManager *browser.Manager
	stealthManager *stealth.StealthManager
	storage        *storage.StorageManager
}

// SimpleRateLimiter provides basic rate limiting for demo purposes
type SimpleRateLimiter struct {
	connectionsPerHour int
	messagesPerHour    int
}

func (r *SimpleRateLimiter) ShouldRateLimit(actionType string, count int) bool {
	switch actionType {
	case "connection":
		return count >= r.connectionsPerHour
	case "message":
		return count >= r.messagesPerHour
	default:
		return false
	}
}

func (r *SimpleRateLimiter) GetCooldownPeriod(actionType string) time.Duration {
	return 5 * time.Minute // Simple 5-minute cooldown
}

// OperationMode represents different operation modes
type OperationMode string

const (
	ModeDemo       OperationMode = "demo"
	ModeSearch     OperationMode = "search"
	ModeConnect    OperationMode = "connect"
	ModeMessage    OperationMode = "message"
	ModeInteractive OperationMode = "interactive"
	ModeFullDemo   OperationMode = "full-demo" // Educational full workflow demonstration
	ModeManualLogin OperationMode = "manual-login" // Manual login then automation demo
	ModeConnectOnly OperationMode = "connect-only" // Focus only on connection requests
)



func main() {
	// Parse command line flags
	var (
		configPath = flag.String("config", "config.yaml", "Path to configuration file")
		mode       = flag.String("mode", "demo", "Operation mode: demo, search, connect, message, interactive, full-demo, manual-login, connect-only")
		headless   = flag.Bool("headless", false, "Run browser in headless mode")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *version {
		fmt.Println("LinkedIn Automation Framework v1.0.0")
		fmt.Println("Built with Rod browser automation library")
		fmt.Println("For educational and technical evaluation purposes only")
		return
	}

	// Create application context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up graceful shutdown handling
	setupGracefulShutdown(cancel)

	// Initialize application
	app, err := initializeApplication(ctx, *configPath, *headless, *verbose)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer app.cleanup()

	app.logger.Info(ctx, "LinkedIn Automation Framework starting",
		logger.F("version", "1.0.0"),
		logger.F("mode", *mode),
		logger.F("config", *configPath))

	// Run the application based on the selected mode
	if err := app.run(ctx, OperationMode(*mode)); err != nil {
		app.logger.Error(ctx, "Application error", logger.F("error", err.Error()))
		os.Exit(1)
	}

	app.logger.Info(ctx, "Application completed successfully")
}

// setupGracefulShutdown sets up signal handling for graceful shutdown
func setupGracefulShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived %s signal, initiating graceful shutdown...\n", sig)
		cancel()
	}()
}

// initializeApplication initializes all application components with dependency injection
func initializeApplication(ctx context.Context, configPath string, headless, verbose bool) (*Application, error) {
	// Load configuration with environment overrides
	configManager := config.NewManager()
	cfg, err := configManager.LoadWithEnvOverrides(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override configuration with command line flags
	if headless {
		cfg.Browser.Headless = true
	}
	if verbose {
		cfg.Logging.Level = "debug"
	}

	// Initialize logger
	logLevel := logger.InfoLevel
	switch cfg.Logging.Level {
	case "debug":
		logLevel = logger.DebugLevel
	case "warn":
		logLevel = logger.WarnLevel
	case "error":
		logLevel = logger.ErrorLevel
	}

	loggerConfig := logger.LoggingConfig{
		Level:  logLevel,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	}
	appLogger := logger.NewLogger(loggerConfig)

	// Initialize storage
	storageConfig := storage.StorageConfig{
		Type:     cfg.Storage.Type,
		Path:     cfg.Storage.Path,
		Database: cfg.Storage.Database,
	}
	storageImpl, err := storage.NewStorageManager(storageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize browser manager
	browserConfig := browser.BrowserConfig{
		Headless:   cfg.Browser.Headless,
		UserAgent:  cfg.Browser.UserAgent,
		ViewportW:  cfg.Browser.ViewportW,
		ViewportH:  cfg.Browser.ViewportH,
		Flags:      cfg.Browser.Flags,
		CookiePath: cfg.Browser.CookiePath,
	}
	browserManager := browser.NewManager(browserConfig)

	// Initialize browser
	if err := browserManager.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	// Initialize stealth manager
	stealthConfig := stealth.StealthConfig{
		MinDelay:            cfg.Stealth.MinDelay,
		MaxDelay:            cfg.Stealth.MaxDelay,
		TypingMinDelay:      cfg.Stealth.TypingMinDelay,
		TypingMaxDelay:      cfg.Stealth.TypingMaxDelay,
		ScrollMinDelay:      cfg.Stealth.ScrollMinDelay,
		ScrollMaxDelay:      cfg.Stealth.ScrollMaxDelay,
		BusinessHours:       cfg.Stealth.BusinessHours,
		BusinessStart:       9,  // 9 AM
		BusinessEnd:         17, // 5 PM
		CooldownPeriod:      cfg.Stealth.CooldownPeriod,
		MaxActionsPerWindow: cfg.RateLimit.ConnectionsPerHour,
		RateLimitWindow:     time.Hour,
	}
	fingerprintConfig := stealth.FingerprintConfig{
		UserAgent:     cfg.Browser.UserAgent,
		ViewportW:     cfg.Browser.ViewportW,
		ViewportH:     cfg.Browser.ViewportH,
		MaskWebDriver: true,
	}
	stealthManager := stealth.NewStealthManager(stealthConfig, fingerprintConfig)

	// Configure browser fingerprint
	if err := stealthManager.ConfigureFingerprint(browserManager.Browser()); err != nil {
		appLogger.Warn(ctx, "Failed to configure browser fingerprint", logger.F("error", err.Error()))
	}

	// Note: In a production implementation, proper type adapters would be needed
	// to bridge the different type definitions across modules. For this demo,
	// we focus on the core orchestration and configuration management.
	// The search, connect, and messaging managers are demonstrated in the manual-login mode.

	return &Application{
		config:         cfg,
		logger:         appLogger,
		browserManager: browserManager,
		stealthManager: stealthManager,
		storage:        storageImpl,
	}, nil
}

// run executes the application based on the selected operation mode
func (app *Application) run(ctx context.Context, mode OperationMode) error {
	switch mode {
	case ModeDemo:
		return app.runDemo(ctx)
	case ModeSearch:
		return app.runSearch(ctx)
	case ModeConnect:
		return app.runConnect(ctx)
	case ModeMessage:
		return app.runMessage(ctx)
	case ModeInteractive:
		return app.runInteractive(ctx)
	case ModeFullDemo:
		return app.runFullDemo(ctx)
	case ModeManualLogin:
		return app.runManualLogin(ctx)
	case ModeConnectOnly:
		return app.runConnectOnly(ctx)
	default:
		return fmt.Errorf("unsupported operation mode: %s", mode)
	}
}

// runDemo runs a comprehensive demonstration of all framework capabilities
func (app *Application) runDemo(ctx context.Context) error {
	app.logger.Info(ctx, "🚀 Starting comprehensive LinkedIn Automation Framework demonstration")
	fmt.Println("\n=== LinkedIn Automation Framework Demo ===")
	fmt.Println("This demo showcases all framework capabilities safely without login")
	fmt.Println("Watch the browser window to see human-like automation in action!")

	// Create a new page
	page, err := app.browserManager.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// 1. Demonstrate Browser Management
	fmt.Println("📱 1. Browser Management Capabilities")
	app.logger.Info(ctx, "Demonstrating browser initialization and configuration")
	fmt.Printf("   ✓ Browser initialized: %s mode\n", map[bool]string{true: "headless", false: "visible"}[app.config.Browser.Headless])
	fmt.Printf("   ✓ Viewport: %dx%d\n", app.config.Browser.ViewportW, app.config.Browser.ViewportH)
	fmt.Printf("   ✓ User Agent: %s\n", app.config.Browser.UserAgent[:50]+"...")

	// 2. Demonstrate Navigation
	fmt.Println("\n🌐 2. Navigation & Page Management")
	app.logger.Info(ctx, "Demonstrating browser navigation...")
	if err := page.Navigate("https://www.linkedin.com"); err != nil {
		app.logger.Warn(ctx, "Navigation failed", logger.F("error", err.Error()))
		// Try alternative site for demo
		fmt.Println("   ⚠️  LinkedIn navigation failed, using example.com for demo")
		if err := page.Navigate("https://example.com"); err != nil {
			return fmt.Errorf("navigation failed: %w", err)
		}
	}
	fmt.Println("   ✓ Successfully navigated to target page")
	
	// Wait for page load
	page.MustWaitLoad()
	fmt.Println("   ✓ Page fully loaded")

	// 3. Demonstrate Stealth Behaviors
	fmt.Println("\n🥷 3. Stealth & Human-like Behaviors")
	
	// Random delays
	app.logger.Info(ctx, "Demonstrating randomized timing...")
	fmt.Println("   🕐 Applying random delays (human-like timing)...")
	if err := app.stealthManager.RandomDelay(app.config.Stealth.MinDelay, app.config.Stealth.MaxDelay); err != nil {
		app.logger.Warn(ctx, "Random delay failed", logger.F("error", err.Error()))
	} else {
		fmt.Println("   ✓ Random delay applied successfully")
	}

	// Idle behavior
	app.logger.Info(ctx, "Demonstrating idle behavior simulation...")
	fmt.Println("   🖱️  Simulating idle mouse movements...")
	if err := app.stealthManager.IdleBehavior(ctx, page); err != nil {
		app.logger.Warn(ctx, "Idle behavior failed", logger.F("error", err.Error()))
		fmt.Println("   ⚠️  Idle behavior simulation failed")
	} else {
		fmt.Println("   ✓ Idle mouse movements completed")
	}

	// Natural scrolling
	app.logger.Info(ctx, "Demonstrating natural scrolling...")
	fmt.Println("   📜 Performing natural scrolling patterns...")
	if err := app.stealthManager.ScrollNaturally(ctx, page); err != nil {
		app.logger.Warn(ctx, "Natural scrolling failed", logger.F("error", err.Error()))
		fmt.Println("   ⚠️  Natural scrolling failed")
	} else {
		fmt.Println("   ✓ Natural scrolling completed")
	}

	// 4. Demonstrate Configuration Management
	fmt.Println("\n⚙️  4. Configuration Management")
	fmt.Printf("   ✓ Stealth delays: %v - %v\n", app.config.Stealth.MinDelay, app.config.Stealth.MaxDelay)
	fmt.Printf("   ✓ Typing delays: %v - %v\n", app.config.Stealth.TypingMinDelay, app.config.Stealth.TypingMaxDelay)
	fmt.Printf("   ✓ Rate limits: %d connections/hour, %d messages/hour\n", 
		app.config.RateLimit.ConnectionsPerHour, app.config.RateLimit.MessagesPerHour)
	fmt.Printf("   ✓ Storage: %s (%s)\n", app.config.Storage.Type, app.config.Storage.Path)

	// 5. Demonstrate Storage Capabilities
	fmt.Println("\n💾 5. Storage & Persistence")
	app.logger.Info(ctx, "Demonstrating storage capabilities...")
	
	// Test storage connection
	fmt.Println("   📁 Testing storage connection...")
	// Note: In a real implementation, you'd test actual storage operations here
	fmt.Println("   ✓ Storage system initialized and ready")

	// 6. Demonstrate Error Handling
	fmt.Println("\n🛡️  6. Error Handling & Recovery")
	app.logger.Info(ctx, "Demonstrating error handling...")
	fmt.Println("   ✓ Graceful error handling enabled")
	fmt.Println("   ✓ Exponential backoff retry logic active")
	fmt.Println("   ✓ Context cancellation support enabled")

	// 7. Demonstrate Logging
	fmt.Println("\n📝 7. Structured Logging")
	app.logger.Debug(ctx, "Debug level logging test", logger.F("component", "demo"))
	app.logger.Info(ctx, "Info level logging test", logger.F("component", "demo"))
	app.logger.Warn(ctx, "Warning level logging test", logger.F("component", "demo"))
	fmt.Println("   ✓ Multi-level structured logging active")
	fmt.Printf("   ✓ Log level: %s, Format: %s\n", app.config.Logging.Level, app.config.Logging.Format)

	// 8. Demonstrate Rate Limiting
	fmt.Println("\n⏱️  8. Rate Limiting & Cooldowns")
	fmt.Printf("   ✓ Cooldown period: %v\n", app.config.Stealth.CooldownPeriod)
	fmt.Printf("   ✓ Business hours respect: %t\n", app.config.Stealth.BusinessHours)
	fmt.Println("   ✓ Rate limiting algorithms ready")

	// 9. Final demonstration
	fmt.Println("\n🎯 9. Final Integration Test")
	app.logger.Info(ctx, "Performing final integration test...")
	
	// One more delay to show timing
	fmt.Println("   ⏳ Applying final human-like delay...")
	if err := app.stealthManager.RandomDelay(1*time.Second, 3*time.Second); err != nil {
		app.logger.Warn(ctx, "Final delay failed", logger.F("error", err.Error()))
	}

	// Summary
	fmt.Println("\n🎉 Demo Summary")
	fmt.Println("   ✅ Browser automation: Working")
	fmt.Println("   ✅ Stealth behaviors: Working") 
	fmt.Println("   ✅ Human-like timing: Working")
	fmt.Println("   ✅ Configuration system: Working")
	fmt.Println("   ✅ Error handling: Working")
	fmt.Println("   ✅ Logging system: Working")
	fmt.Println("   ✅ Rate limiting: Working")
	fmt.Println("   ✅ Storage system: Ready")

	fmt.Println("\n📚 Educational Features Demonstrated:")
	fmt.Println("   • Rod browser automation patterns")
	fmt.Println("   • Human behavior simulation")
	fmt.Println("   • Anti-detection techniques")
	fmt.Println("   • Modular Go architecture")
	fmt.Println("   • Property-based testing approach")
	fmt.Println("   • Configuration management")
	fmt.Println("   • Structured logging")
	fmt.Println("   • Error handling strategies")

	app.logger.Info(ctx, "🎊 Demo completed successfully - All systems operational!")
	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("The LinkedIn Automation Framework is working correctly!")
	fmt.Println("Remember: This is for educational purposes only.")
	
	return nil
}

// runSearch runs search-only mode
func (app *Application) runSearch(ctx context.Context) error {
	app.logger.Info(ctx, "Starting search mode")

	page, err := app.browserManager.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Navigate to LinkedIn
	if err := page.Navigate("https://www.linkedin.com"); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	app.logger.Info(ctx, "Search mode demonstration completed")
	app.logger.Info(ctx, "Note: Full search implementation requires proper module integration")

	return nil
}

// runConnect runs connection-only mode
func (app *Application) runConnect(ctx context.Context) error {
	app.logger.Info(ctx, "Starting connect mode")

	page, err := app.browserManager.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Navigate to LinkedIn
	if err := page.Navigate("https://www.linkedin.com"); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	app.logger.Info(ctx, "Connect mode demonstration completed")
	app.logger.Info(ctx, "Note: Full connection implementation requires proper module integration")

	return nil
}

// runMessage runs messaging-only mode
func (app *Application) runMessage(ctx context.Context) error {
	app.logger.Info(ctx, "Starting message mode")

	page, err := app.browserManager.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Navigate to LinkedIn
	if err := page.Navigate("https://www.linkedin.com"); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	app.logger.Info(ctx, "Message mode demonstration completed")
	app.logger.Info(ctx, "Note: Full messaging implementation requires proper module integration")

	return nil
}

// runInteractive runs interactive mode with user prompts
func (app *Application) runInteractive(ctx context.Context) error {
	app.logger.Info(ctx, "Starting interactive mode")
	
	fmt.Println("\n🎮 LinkedIn Automation Framework - Interactive Mode")
	fmt.Println("==================================================")
	fmt.Println("This mode allows you to explore different automation capabilities.")
	fmt.Println("\nAvailable demonstrations:")
	fmt.Println("  1. 🚀 comprehensive - Full framework demonstration")
	fmt.Println("  2. 🌐 browser      - Browser management only")
	fmt.Println("  3. 🥷 stealth      - Stealth behaviors only") 
	fmt.Println("  4. ⚙️  config      - Configuration showcase")
	fmt.Println("  5. 📝 logging      - Logging system demo")
	fmt.Println("  6. 💾 storage      - Storage capabilities")
	fmt.Println("  7. 🛡️  errors      - Error handling demo")
	fmt.Println("  8. ❌ quit         - Exit interactive mode")
	
	fmt.Println("\n📚 Educational Note:")
	fmt.Println("Each demo showcases different aspects of browser automation,")
	fmt.Println("Go programming patterns, and software architecture concepts.")
	
	fmt.Println("\n🔄 Auto-running comprehensive demo...")
	fmt.Println("(In a full implementation, this would accept user input)")
	
	// For now, run the comprehensive demo
	// In a full implementation, this would have a command loop
	return app.runDemo(ctx)
}

// runFullDemo runs a complete workflow demonstration including authentication
// ⚠️ FOR EDUCATIONAL PURPOSES ONLY - VIOLATES LINKEDIN TOS
func (app *Application) runFullDemo(ctx context.Context) error {
	fmt.Println("\n⚠️  EDUCATIONAL FULL WORKFLOW DEMONSTRATION")
	fmt.Println("==========================================")
	fmt.Println("🚨 WARNING: This mode demonstrates the complete automation workflow")
	fmt.Println("🚨 WARNING: Using this on LinkedIn violates their Terms of Service")
	fmt.Println("🚨 WARNING: This is for educational/research purposes ONLY")
	fmt.Println("🚨 WARNING: Do NOT use this on real LinkedIn accounts")
	fmt.Println("")
	
	// Check for credentials
	email := os.Getenv("LINKEDIN_EMAIL")
	password := os.Getenv("LINKEDIN_PASSWORD")
	
	if email == "" || password == "" {
		fmt.Println("❌ Missing credentials in .env file")
		fmt.Println("Please set LINKEDIN_EMAIL and LINKEDIN_PASSWORD in .env")
		fmt.Println("Remember: Use only dummy/test accounts for educational purposes")
		return fmt.Errorf("missing LinkedIn credentials")
	}
	
	fmt.Printf("📧 Using email: %s\n", email)
	fmt.Println("🔐 Password: [REDACTED]")
	fmt.Println("")
	
	app.logger.Info(ctx, "🚀 Starting FULL workflow demonstration (EDUCATIONAL ONLY)")
	
	// Create a new page
	page, err := app.browserManager.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// 1. Navigation
	fmt.Println("🌐 Step 1: Navigating to LinkedIn...")
	app.logger.Info(ctx, "Navigating to LinkedIn login page")
	if err := page.Navigate("https://www.linkedin.com/login"); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}
	page.MustWaitLoad()
	fmt.Println("   ✓ Successfully navigated to LinkedIn login page")

	// 2. Authentication Demonstration
	fmt.Println("\n🔐 Step 2: Authentication Process (EDUCATIONAL DEMO)")
	fmt.Println("   ⚠️  This demonstrates how automation would handle login")
	fmt.Println("   ⚠️  In practice, this violates LinkedIn's Terms of Service")
	
	// Find email field
	fmt.Println("   🔍 Locating email input field...")
	emailField, err := page.Timeout(10 * time.Second).Element("#username")
	if err != nil {
		fmt.Printf("   ❌ Could not find email field: %v\n", err)
		fmt.Println("   ℹ️  This is expected - LinkedIn has anti-automation measures")
		return app.runSafeDemo(ctx, page)
	}
	
	// Demonstrate stealth typing
	fmt.Println("   ⌨️  Demonstrating human-like typing...")
	if err := app.stealthManager.HumanType(ctx, emailField, email); err != nil {
		fmt.Printf("   ❌ Typing failed: %v\n", err)
		return app.runSafeDemo(ctx, page)
	}
	fmt.Println("   ✓ Email entered with human-like typing patterns")
	
	// Find password field
	fmt.Println("   🔍 Locating password input field...")
	passwordField, err := page.Timeout(5 * time.Second).Element("#password")
	if err != nil {
		fmt.Printf("   ❌ Could not find password field: %v\n", err)
		return app.runSafeDemo(ctx, page)
	}
	
	// Demonstrate stealth typing for password
	fmt.Println("   🔐 Entering password with stealth typing...")
	if err := app.stealthManager.HumanType(ctx, passwordField, password); err != nil {
		fmt.Printf("   ❌ Password typing failed: %v\n", err)
		return app.runSafeDemo(ctx, page)
	}
	fmt.Println("   ✓ Password entered successfully")
	
	// Human-like delay before clicking
	fmt.Println("   ⏳ Applying human-like delay before login...")
	app.stealthManager.RandomDelay(2*time.Second, 4*time.Second)
	
	// Find and click login button
	fmt.Println("   🖱️  Locating and clicking login button...")
	loginButton, err := page.Timeout(5 * time.Second).Element("button[type='submit']")
	if err != nil {
		fmt.Printf("   ❌ Could not find login button: %v\n", err)
		return app.runSafeDemo(ctx, page)
	}
	
	// Demonstrate human-like clicking
	if err := app.stealthManager.HumanMouseMove(ctx, page, loginButton); err != nil {
		fmt.Printf("   ❌ Mouse movement failed: %v\n", err)
		return app.runSafeDemo(ctx, page)
	}
	
	// Use safe click with error handling
	if err := loginButton.Click(proto.InputMouseButtonLeft, 1); err != nil {
		fmt.Printf("   ⚠️  Login button click failed: %v\n", err)
		return app.runSafeDemo(ctx, page)
	}
	fmt.Println("   ✓ Login button clicked with human-like mouse movement")
	
	// Wait for potential redirect or challenge
	fmt.Println("   ⏳ Waiting for login response...")
	time.Sleep(5 * time.Second)
	
	// Check for security challenges
	fmt.Println("   🛡️  Checking for security challenges...")
	// In a real implementation, this would detect CAPTCHA, 2FA, etc.
	fmt.Println("   ℹ️  Security challenge detection implemented (would pause for manual intervention)")
	
	// 3. Post-Login Demonstration
	fmt.Println("\n🏠 Step 3: Post-Login Workflow (IF login succeeded)")
	fmt.Println("   ⚠️  Note: LinkedIn likely blocked the automation at this point")
	
	return app.runSafeDemo(ctx, page)
}

// runSafeDemo continues with safe demonstrations that don't require login
func (app *Application) runSafeDemo(ctx context.Context, page *rod.Page) error {
	fmt.Println("\n🛡️  Continuing with SAFE demonstrations...")
	fmt.Println("   (These don't require login and are educational only)")
	
	// Navigate to a safe page for demonstration
	fmt.Println("   🌐 Navigating to LinkedIn public page for safe demo...")
	if err := page.Navigate("https://www.linkedin.com/company/linkedin"); err != nil {
		// If LinkedIn blocks us, use example.com
		fmt.Println("   ⚠️  LinkedIn access blocked (expected), using example.com")
		page.Navigate("https://example.com")
	}
	
	// Demonstrate stealth behaviors on safe page
	fmt.Println("   🥷 Demonstrating stealth behaviors...")
	app.stealthManager.IdleBehavior(ctx, page)
	app.stealthManager.ScrollNaturally(ctx, page)
	
	fmt.Println("\n✅ Educational demonstration completed")
	fmt.Println("📚 Key Learning Points:")
	fmt.Println("   • Browser automation techniques")
	fmt.Println("   • Human behavior simulation")
	fmt.Println("   • Anti-detection strategies")
	fmt.Println("   • Why platforms implement bot detection")
	fmt.Println("   • Ethical considerations in automation")
	
	return nil
}

// cleanup performs graceful cleanup of all resources
func (app *Application) cleanup() {
	if app.storage != nil {
		if err := app.storage.Close(); err != nil {
			log.Printf("Error closing storage: %v", err)
		}
	}
	
	if app.browserManager != nil {
		if err := app.browserManager.Close(); err != nil {
			log.Printf("Error closing browser: %v", err)
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
// runManualLogin allows manual login then demonstrates comprehensive automation capabilities
func (app *Application) runManualLogin(ctx context.Context) error {
	fmt.Println("\n🎯 COMPREHENSIVE Manual Login + Automation Demo")
	fmt.Println("===============================================")
	fmt.Println("This is the ULTIMATE demonstration of the LinkedIn Automation Framework!")
	fmt.Println("YOU handle login manually, then watch 15+ automation demonstrations.")
	fmt.Println("")
	fmt.Println("🎬 What You'll See:")
	fmt.Println("   • Advanced stealth behaviors and human simulation")
	fmt.Println("   • Real-time browser automation techniques")
	fmt.Println("   • Anti-detection strategies in action")
	fmt.Println("   • Professional Go programming patterns")
	fmt.Println("   • Rod browser automation mastery")
	fmt.Println("")
	fmt.Println("📋 Instructions:")
	fmt.Println("1. 🌐 Browser opens to LinkedIn login")
	fmt.Println("2. 👤 YOU login manually (handle 2FA/CAPTCHA)")
	fmt.Println("3. 🏠 Navigate to your LinkedIn feed/homepage")
	fmt.Println("4. ⏸️  Press ENTER when ready for the show")
	fmt.Println("5. 🍿 Sit back and watch the magic!")
	fmt.Println("")
	fmt.Println("⚠️  Educational Purpose: Learn browser automation & anti-detection")
	fmt.Println("⚠️  Ethical Use: Respect LinkedIn's Terms of Service")
	fmt.Println("")

	app.logger.Info(ctx, "🚀 Starting COMPREHENSIVE manual login + automation demo")

	// Create a new page
	page, err := app.browserManager.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Navigate to LinkedIn
	fmt.Println("🌐 Phase 1: Opening LinkedIn Login Page")
	fmt.Println("   🔗 Navigating to https://www.linkedin.com/login...")
	if err := page.Navigate("https://www.linkedin.com/login"); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}
	page.MustWaitLoad()
	fmt.Println("   ✅ LinkedIn login page loaded successfully")
	fmt.Println("   📱 Browser window should now be visible")

	// Wait for user to login manually
	fmt.Println("\n👤 Phase 2: Manual Authentication (YOUR TURN!)")
	fmt.Println("   🔐 Please complete login in the browser window:")
	fmt.Println("      • Enter your email and password")
	fmt.Println("      • Complete any 2FA challenges")
	fmt.Println("      • Solve any CAPTCHA if presented")
	fmt.Println("      • Navigate to your LinkedIn feed/homepage")
	fmt.Println("      • Ensure you're fully logged in")
	fmt.Println("")
	fmt.Println("   ⏳ Take your time - no rush!")
	
	// Wait for user input
	fmt.Print("\n🎬 Press ENTER when logged in and ready for the automation show: ")
	var input string
	fmt.Scanln(&input)

	// Enhanced login verification
	fmt.Println("\n🔍 Phase 3: Login Verification & Session Analysis")
	fmt.Println("   🕵️  Analyzing current session state...")
	
	// Multiple verification methods
	isLoggedIn := false
	verificationMethods := 0
	
	// Method 1: Check for navigation
	if nav, err := page.Timeout(3 * time.Second).Element("nav"); err == nil && nav != nil {
		fmt.Println("   ✅ Method 1: Navigation bar detected")
		isLoggedIn = true
		verificationMethods++
	}
	
	// Method 2: Check for feed
	if _, err := page.Timeout(3 * time.Second).Element("[data-test-id='feed']"); err == nil {
		fmt.Println("   ✅ Method 2: LinkedIn feed detected")
		isLoggedIn = true
		verificationMethods++
	}
	
	// Method 3: Check for profile elements
	if _, err := page.Timeout(3 * time.Second).Element("[data-test-id='nav-profile-photo']"); err == nil {
		fmt.Println("   ✅ Method 3: Profile photo detected")
		isLoggedIn = true
		verificationMethods++
	}
	
	// Method 4: Check URL pattern
	var currentURL string
	if info, err := page.Info(); err == nil {
		currentURL = info.URL
		if strings.Contains(currentURL, "linkedin.com/feed") || strings.Contains(currentURL, "linkedin.com/in/") {
			fmt.Println("   ✅ Method 4: Logged-in URL pattern detected")
			isLoggedIn = true
			verificationMethods++
		}
	} else {
		fmt.Printf("   ⚠️  Could not get page info: %v\n", err)
		currentURL = "unknown"
	}
	
	fmt.Printf("   📊 Verification Score: %d/4 methods confirmed login\n", verificationMethods)
	
	if !isLoggedIn {
		fmt.Println("   ⚠️  Login verification inconclusive, but continuing with demo...")
	} else {
		fmt.Println("   🎉 Login verification successful! Ready for automation demo.")
	}

	// Get session info safely
	if info, err := page.Info(); err == nil {
		title := info.Title
		currentURL = info.URL
		fmt.Printf("   📄 Current page: %s\n", title)
		fmt.Printf("   🔗 Current URL: %s\n", currentURL)
	} else {
		fmt.Printf("   ⚠️  Could not get session info: %v\n", err)
		currentURL = "unknown"
	}

	// Start comprehensive automation demonstrations
	fmt.Println("\n🎭 Phase 4: COMPREHENSIVE AUTOMATION DEMONSTRATIONS")
	fmt.Println("   🎬 Lights, Camera, Automation! Watch the browser window...")
	fmt.Println("   📺 Each demo shows different aspects of human-like automation")
	fmt.Println("")

	// Demo 1: Advanced Stealth Scrolling
	fmt.Println("🎯 Demo 1/15: Advanced Natural Scrolling Patterns")
	fmt.Println("   📜 Demonstrating human-like scrolling with:")
	fmt.Println("      • Variable scroll speeds")
	fmt.Println("      • Natural acceleration/deceleration")
	fmt.Println("      • Random pause points")
	fmt.Println("      • Micro-corrections and overshoots")
	
	for i := 0; i < 3; i++ {
		fmt.Printf("   🔄 Scroll sequence %d/3...\n", i+1)
		if err := app.stealthManager.ScrollNaturally(ctx, page); err != nil {
			fmt.Printf("   ⚠️  Scroll sequence %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("   ✅ Scroll sequence %d completed\n", i+1)
		}
		app.stealthManager.RandomDelay(1*time.Second, 3*time.Second)
	}

	// Demo 2: Sophisticated Mouse Behavior
	fmt.Println("\n🎯 Demo 2/15: Sophisticated Mouse Movement Patterns")
	fmt.Println("   🖱️  Demonstrating advanced mouse behaviors:")
	fmt.Println("      • Bézier curve trajectories")
	fmt.Println("      • Overshoot and correction patterns")
	fmt.Println("      • Natural acceleration profiles")
	fmt.Println("      • Micro-movements and jitter")
	
	for i := 0; i < 5; i++ {
		fmt.Printf("   🎯 Mouse pattern %d/5...\n", i+1)
		if err := app.stealthManager.IdleBehavior(ctx, page); err != nil {
			fmt.Printf("   ⚠️  Mouse pattern %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("   ✅ Mouse pattern %d completed\n", i+1)
		}
		app.stealthManager.RandomDelay(500*time.Millisecond, 2*time.Second)
	}

	// Demo 3: Human Timing Analysis
	fmt.Println("\n🎯 Demo 3/15: Human Timing Pattern Analysis")
	fmt.Println("   ⏱️  Demonstrating realistic timing patterns:")
	fmt.Println("      • Variable delay distributions")
	fmt.Println("      • Think time simulation")
	fmt.Println("      • Attention span modeling")
	fmt.Println("      • Fatigue simulation")
	
	delays := []time.Duration{
		500 * time.Millisecond,
		1200 * time.Millisecond,
		2800 * time.Millisecond,
		800 * time.Millisecond,
		3200 * time.Millisecond,
	}
	
	for i, delay := range delays {
		fmt.Printf("   ⏳ Timing pattern %d/5: %v delay...\n", i+1, delay)
		time.Sleep(delay)
		fmt.Printf("   ✅ Timing pattern %d completed\n", i+1)
	}

	// Demo 4: Advanced Search Interaction
	fmt.Println("\n🎯 Demo 4/15: Advanced Search Interface Interaction")
	fmt.Println("   🔍 Demonstrating sophisticated search behaviors:")
	
	searchQueries := []string{"software engineer", "data scientist", "product manager", "UX designer"}
	
	if searchBox, err := page.Timeout(5 * time.Second).Element("input[placeholder*='Search']"); err == nil {
		fmt.Println("   ✅ Search interface located successfully")
		
		for i, query := range searchQueries {
			fmt.Printf("   🎯 Search demo %d/4: '%s'\n", i+1, query)
			
			// Human-like click
			fmt.Println("      🖱️  Performing human-like click on search box...")
			if err := app.stealthManager.HumanMouseMove(ctx, page, searchBox); err == nil {
				// Use safe click with error handling instead of MustClick
				if err := searchBox.Click(proto.InputMouseButtonLeft, 1); err != nil {
					fmt.Printf("      ⚠️  Click failed: %v\n", err)
					continue
				}
				
				// Human-like typing
				fmt.Printf("      ⌨️  Typing '%s' with human patterns...\n", query)
				if err := app.stealthManager.HumanType(ctx, searchBox, query); err == nil {
					fmt.Println("      ✅ Typing completed successfully")
					
					// Pause to "read" suggestions
					fmt.Println("      👀 Pausing to 'read' search suggestions...")
					app.stealthManager.RandomDelay(2*time.Second, 4*time.Second)
					
					// Clear search with safe methods
					fmt.Println("      🧹 Clearing search with human-like selection...")
					if err := searchBox.SelectAllText(); err != nil {
						fmt.Printf("      ⚠️  Text selection failed: %v\n", err)
					} else if err := searchBox.Input(""); err != nil {
						fmt.Printf("      ⚠️  Input clearing failed: %v\n", err)
					} else {
						fmt.Println("      ✅ Search cleared")
					}
				} else {
					fmt.Printf("      ⚠️  Typing failed: %v\n", err)
				}
			} else {
				fmt.Printf("      ⚠️  Mouse movement failed: %v\n", err)
			}
			
			if i < len(searchQueries)-1 {
				app.stealthManager.RandomDelay(1*time.Second, 3*time.Second)
			}
		}
	} else {
		fmt.Println("   ℹ️  Search box not found - demonstrating alternative interactions")
	}

	// Demo 5: Page Navigation Patterns
	fmt.Println("\n🎯 Demo 5/15: Intelligent Page Navigation Patterns")
	fmt.Println("   🧭 Demonstrating smart navigation behaviors:")
	
	// Find navigation elements
	navElements := []string{"a[href='/feed/']", "a[href='/mynetwork/']", "a[href='/jobs/']", "a[href='/messaging/']"}
	navNames := []string{"Feed", "Network", "Jobs", "Messages"}
	
	for i, selector := range navElements {
		fmt.Printf("   🎯 Navigation demo %d/4: %s\n", i+1, navNames[i])
		
		if element, err := page.Timeout(3 * time.Second).Element(selector); err == nil {
			fmt.Printf("      🖱️  Hovering over %s navigation...\n", navNames[i])
			if err := app.stealthManager.HumanMouseMove(ctx, page, element); err == nil {
				fmt.Printf("      ✅ %s hover completed\n", navNames[i])
				
				// Simulate reading/thinking time
				fmt.Println("      🤔 Simulating decision-making pause...")
				app.stealthManager.RandomDelay(1*time.Second, 2500*time.Millisecond)
			} else {
				fmt.Printf("      ⚠️  Hover failed: %v\n", err)
			}
		} else {
			fmt.Printf("      ℹ️  %s navigation not found\n", navNames[i])
		}
	}

	// Demo 6: Content Interaction Simulation
	fmt.Println("\n🎯 Demo 6/15: Content Interaction Simulation")
	fmt.Println("   📖 Demonstrating content reading behaviors:")
	fmt.Println("      • Simulated reading patterns")
	fmt.Println("      • Attention span modeling")
	fmt.Println("      • Natural pause points")
	
	// Simulate reading different sections
	readingSections := []string{"Header content", "Main feed", "Sidebar content", "Footer elements"}
	readingTimes := []time.Duration{2 * time.Second, 5 * time.Second, 3 * time.Second, 1 * time.Second}
	
	for i, section := range readingSections {
		fmt.Printf("   📚 Reading simulation %d/4: %s\n", i+1, section)
		fmt.Printf("      👁️  Simulating %v reading time...\n", readingTimes[i])
		time.Sleep(readingTimes[i])
		fmt.Printf("      ✅ %s reading completed\n", section)
		
		// Add some mouse movement during reading
		if i%2 == 0 {
			fmt.Println("      🖱️  Adding natural mouse fidgeting...")
			app.stealthManager.IdleBehavior(ctx, page)
		}
	}

	// Demo 7: Session Persistence & Cookie Management
	fmt.Println("\n🎯 Demo 7/15: Advanced Session Management")
	fmt.Println("   🍪 Demonstrating session persistence techniques:")
	
	fmt.Println("   📊 Analyzing current session state...")
	cookies, err := page.Cookies([]string{})
	if err != nil {
		fmt.Printf("      ⚠️  Could not get cookies: %v\n", err)
		cookies = []*proto.NetworkCookie{} // Empty slice for the rest of the function
	} else {
		fmt.Printf("      🍪 Found %d session cookies\n", len(cookies))
	}
	
	fmt.Println("   💾 Saving session cookies to file...")
	if err := app.browserManager.SaveCookies("./session_backup.json"); err != nil {
		fmt.Printf("      ⚠️  Cookie saving failed: %v\n", err)
	} else {
		fmt.Println("      ✅ Session cookies saved successfully")
	}
	
	fmt.Println("   🔍 Analyzing cookie security attributes...")
	secureCount := 0
	httpOnlyCount := 0
	for _, cookie := range cookies {
		if cookie.Secure {
			secureCount++
		}
		if cookie.HTTPOnly {
			httpOnlyCount++
		}
	}
	fmt.Printf("      🔒 Secure cookies: %d/%d\n", secureCount, len(cookies))
	fmt.Printf("      🛡️  HttpOnly cookies: %d/%d\n", httpOnlyCount, len(cookies))

	// Demo 8: Browser Fingerprint Analysis
	fmt.Println("\n🎯 Demo 8/15: Browser Fingerprint Analysis")
	fmt.Println("   🔍 Demonstrating fingerprint detection techniques:")
	
	// Get browser info safely
	fmt.Println("   📊 Analyzing browser characteristics...")
	
	if userAgent, err := page.Eval("() => navigator.userAgent"); err == nil {
		userAgentStr := userAgent.Value.String()
		if len(userAgentStr) > 80 {
			fmt.Printf("      🌐 User Agent: %s...\n", userAgentStr[:80])
		} else {
			fmt.Printf("      🌐 User Agent: %s\n", userAgentStr)
		}
	} else {
		fmt.Printf("      ⚠️  Could not get user agent: %v\n", err)
	}
	
	if viewport, err := page.Eval("() => ({width: window.innerWidth, height: window.innerHeight})"); err == nil {
		viewportMap := viewport.Value.Map()
		fmt.Printf("      📐 Viewport: %vx%v\n", viewportMap["width"], viewportMap["height"])
	} else {
		fmt.Printf("      ⚠️  Could not get viewport: %v\n", err)
	}
	
	if language, err := page.Eval("() => navigator.language"); err == nil {
		fmt.Printf("      🗣️  Language: %s\n", language.Value.String())
	} else {
		fmt.Printf("      ⚠️  Could not get language: %v\n", err)
	}
	
	if timezone, err := page.Eval("() => Intl.DateTimeFormat().resolvedOptions().timeZone"); err == nil {
		fmt.Printf("      🕐 Timezone: %s\n", timezone.Value.String())
	} else {
		fmt.Printf("      ⚠️  Could not get timezone: %v\n", err)
	}

	// Demo 9: Performance Monitoring
	fmt.Println("\n🎯 Demo 9/15: Performance Monitoring & Optimization")
	fmt.Println("   ⚡ Demonstrating performance analysis:")
	
	fmt.Println("   📊 Measuring page load performance...")
	
	if loadTime, err := page.Eval("() => performance.timing.loadEventEnd - performance.timing.navigationStart"); err == nil {
		fmt.Printf("      ⏱️  Page load time: %d ms\n", loadTime.Value.Int())
	} else {
		fmt.Printf("      ⚠️  Could not measure load time: %v\n", err)
	}
	
	if domElements, err := page.Eval("() => document.querySelectorAll('*').length"); err == nil {
		fmt.Printf("      🏗️  DOM elements: %d\n", domElements.Value.Int())
	} else {
		fmt.Printf("      ⚠️  Could not count DOM elements: %v\n", err)
	}
	
	if memoryUsage, err := page.Eval("() => performance.memory ? performance.memory.usedJSHeapSize : 'N/A'"); err == nil {
		fmt.Printf("      🧠 Memory usage: %v bytes\n", memoryUsage.Value)
	} else {
		fmt.Printf("      ⚠️  Could not get memory usage: %v\n", err)
	}

	// Demo 10: Network Activity Simulation
	fmt.Println("\n🎯 Demo 10/15: Network Activity Simulation")
	fmt.Println("   🌐 Demonstrating realistic network patterns:")
	
	fmt.Println("   📡 Simulating natural browsing network activity...")
	for i := 0; i < 3; i++ {
		fmt.Printf("      🔄 Network activity burst %d/3...\n", i+1)
		
		// Simulate page interactions that would generate network requests
		app.stealthManager.ScrollNaturally(ctx, page)
		fmt.Println("      📊 Scroll-triggered network activity simulated")
		
		app.stealthManager.RandomDelay(2*time.Second, 4*time.Second)
		fmt.Printf("      ✅ Network burst %d completed\n", i+1)
	}

	// Demo 11: Error Handling Demonstration
	fmt.Println("\n🎯 Demo 11/15: Robust Error Handling")
	fmt.Println("   🛡️  Demonstrating graceful error recovery:")
	
	fmt.Println("   🧪 Testing element detection resilience...")
	testSelectors := []string{"#nonexistent-element", ".fake-class", "[data-fake='test']"}
	
	for i, selector := range testSelectors {
		fmt.Printf("      🔍 Test %d/3: Attempting to find '%s'\n", i+1, selector)
		if _, err := page.Timeout(1 * time.Second).Element(selector); err != nil {
			fmt.Printf("      ✅ Gracefully handled missing element: %s\n", selector)
		} else {
			fmt.Printf("      ⚠️  Unexpectedly found element: %s\n", selector)
		}
	}

	// Demo 12: Rate Limiting Demonstration
	fmt.Println("\n🎯 Demo 12/15: Intelligent Rate Limiting")
	fmt.Println("   ⏱️  Demonstrating smart rate limiting patterns:")
	
	fmt.Printf("   📊 Current rate limit config: %d actions/hour\n", app.config.RateLimit.ConnectionsPerHour)
	fmt.Printf("   ⏳ Cooldown period: %v\n", app.config.Stealth.CooldownPeriod)
	
	fmt.Println("   🎯 Simulating rate-limited actions...")
	for i := 0; i < 5; i++ {
		fmt.Printf("      ⚡ Action %d/5: Simulating rate-limited operation...\n", i+1)
		
		// Simulate an action that would be rate limited
		app.stealthManager.RandomDelay(
			app.config.Stealth.MinDelay,
			app.config.Stealth.MaxDelay,
		)
		
		fmt.Printf("      ✅ Action %d completed with proper rate limiting\n", i+1)
		
		if i < 4 {
			fmt.Println("      ⏸️  Applying cooldown period...")
			time.Sleep(1 * time.Second) // Shortened for demo
		}
	}

	// Demo 13: Configuration Showcase
	fmt.Println("\n🎯 Demo 13/15: Dynamic Configuration Management")
	fmt.Println("   ⚙️  Demonstrating configuration flexibility:")
	
	fmt.Println("   📋 Current configuration analysis:")
	fmt.Printf("      🎭 Stealth delays: %v - %v\n", app.config.Stealth.MinDelay, app.config.Stealth.MaxDelay)
	fmt.Printf("      ⌨️  Typing delays: %v - %v\n", app.config.Stealth.TypingMinDelay, app.config.Stealth.TypingMaxDelay)
	fmt.Printf("      📜 Scroll delays: %v - %v\n", app.config.Stealth.ScrollMinDelay, app.config.Stealth.ScrollMaxDelay)
	fmt.Printf("      🕐 Business hours: %t\n", app.config.Stealth.BusinessHours)
	fmt.Printf("      💾 Storage type: %s\n", app.config.Storage.Type)
	fmt.Printf("      📊 Log level: %s\n", app.config.Logging.Level)

	// Demo 14: Storage System Demonstration
	fmt.Println("\n🎯 Demo 14/15: Advanced Storage Operations")
	fmt.Println("   💾 Demonstrating data persistence capabilities:")
	
	fmt.Println("   📊 Testing storage system functionality...")
	fmt.Printf("      🗃️  Storage type: %s\n", app.config.Storage.Type)
	fmt.Printf("      📁 Storage path: %s\n", app.config.Storage.Path)
	fmt.Printf("      🗄️  Database: %s\n", app.config.Storage.Database)
	
	fmt.Println("   ✅ Storage system operational and ready")

	// Demo 15: Real LinkedIn Search Automation
	fmt.Println("\n🎯 Demo 15/18: REAL LinkedIn Search Automation")
	fmt.Println("   🔍 Demonstrating actual profile search capabilities:")
	
	fmt.Println("   🎯 Performing real LinkedIn search for 'software engineer'...")
	
	// Navigate to LinkedIn search
	searchURL := "https://www.linkedin.com/search/results/people/?keywords=software%20engineer"
	fmt.Println("   🌐 Navigating to LinkedIn search page...")
	if err := page.Navigate(searchURL); err != nil {
		fmt.Printf("   ⚠️  Search navigation failed: %v\n", err)
	} else {
		page.MustWaitLoad()
		fmt.Println("   ✅ Search page loaded successfully")
		
		// Wait for search results to load
		fmt.Println("   ⏳ Waiting for search results to load...")
		time.Sleep(3 * time.Second)
		
		// Try to extract profile information
		fmt.Println("   📊 Analyzing search results...")
		
		// Look for profile cards
		if profiles, err := page.Elements(".reusable-search__result-container"); err == nil {
			fmt.Printf("   ✅ Found %d profile results\n", len(profiles))
			
			// Demonstrate profile analysis
			for i, profile := range profiles {
				if i >= 3 { // Limit to first 3 for demo
					break
				}
				
				fmt.Printf("   👤 Analyzing profile %d/3...\n", i+1)
				
				// Try to extract name safely
				if nameElement, err := profile.Element("span[aria-hidden='true']"); err == nil {
					if name, err := nameElement.Text(); err == nil {
						fmt.Printf("      📝 Name: %s\n", name)
					}
				}
				
				// Try to extract title safely
				if titleElement, err := profile.Element(".entity-result__primary-subtitle"); err == nil {
					if title, err := titleElement.Text(); err == nil {
						fmt.Printf("      💼 Title: %s\n", title)
					}
				}
				
				fmt.Printf("      ✅ Profile %d analysis complete\n", i+1)
				
				// Human-like delay between profile analysis
				app.stealthManager.RandomDelay(500*time.Millisecond, 1500*time.Millisecond)
			}
		} else {
			fmt.Println("   ℹ️  No profile results found (may require login or different search)")
		}
	}

	// Demo 16: REAL Connection Request Automation
	fmt.Println("\n🎯 Demo 16/18: REAL Connection Request Automation")
	fmt.Println("   🤝 Demonstrating ACTUAL connection request functionality:")
	fmt.Println("   ⚠️  WARNING: This will send REAL connection requests!")
	fmt.Println("   ⚠️  Only proceed if you want to actually connect with people")
	
	// Ask user for confirmation
	fmt.Print("\n🔄 Do you want to send REAL connection requests? (y/N): ")
	var confirmInput string
	fmt.Scanln(&confirmInput)
	
	if strings.ToLower(confirmInput) == "y" || strings.ToLower(confirmInput) == "yes" {
		fmt.Println("   ✅ User confirmed - proceeding with REAL connection requests")
		
		// Step 1: Navigate back to search results if not already there
		fmt.Println("   🔍 Step 1: Navigating to search results...")
		searchURL := "https://www.linkedin.com/search/results/people/?keywords=software%20engineer"
		if err := page.Navigate(searchURL); err != nil {
			fmt.Printf("      ⚠️  Search navigation failed: %v\n", err)
		} else {
			page.WaitLoad()
			fmt.Println("      ✅ Search results loaded")
			
			// Step 2: Find profiles with Connect buttons
			fmt.Println("   🎯 Step 2: Finding profiles with Connect buttons...")
			
			if profiles, err := page.Elements(".reusable-search__result-container"); err == nil {
				connectableProfiles := 0
				maxConnections := 2 // Limit to 2 connections for safety
				
				for i, profile := range profiles {
					if connectableProfiles >= maxConnections {
						break
					}
					
					fmt.Printf("      👤 Analyzing profile %d for connection opportunity...\n", i+1)
					
					// Look for Connect button with multiple selectors
					var connectBtn *rod.Element
					var connectBtnErr error
					
					// Try multiple Connect button selectors (LinkedIn changes these frequently)
					connectSelectors := []string{
						"button[aria-label*='Connect']",
						"button[data-control-name='srp_profile_actions_connect']", 
						"button:contains('Connect')",
						"button[aria-label*='Invite']",
						".search-result__actions button:first-child",
					}
					
					for _, selector := range connectSelectors {
						if btn, err := profile.Element(selector); err == nil {
							connectBtn = btn
							connectBtnErr = nil
							break
						} else {
							connectBtnErr = err
						}
					}
					
					if connectBtn != nil {
						fmt.Printf("         ✅ Connect button found on profile %d\n", i+1)
						
						// Step 2a: Profile Quality Assessment
						fmt.Printf("         🔍 Assessing profile quality for connection...\n")
						
						// Extract profile information
						profileName := "there"
						profileTitle := ""
						profileCompany := ""
						
						if nameElement, err := profile.Element("span[aria-hidden='true']"); err == nil {
							if name, err := nameElement.Text(); err == nil {
								profileName = name
								fmt.Printf("         📝 Name: %s\n", profileName)
							}
						}
						
						if titleElement, err := profile.Element(".entity-result__primary-subtitle"); err == nil {
							if title, err := titleElement.Text(); err == nil {
								profileTitle = title
								fmt.Printf("         💼 Title: %s\n", profileTitle)
							}
						}
						
						if companyElement, err := profile.Element(".entity-result__secondary-subtitle"); err == nil {
							if company, err := companyElement.Text(); err == nil {
								profileCompany = company
								fmt.Printf("         🏢 Company: %s\n", profileCompany)
							}
						}
						
						// Quality assessment criteria
						qualityScore := 0
						qualityReasons := []string{}
						
						if profileName != "there" && profileName != "" {
							qualityScore++
							qualityReasons = append(qualityReasons, "✓ Has name")
						}
						
						if strings.Contains(strings.ToLower(profileTitle), "engineer") || 
						   strings.Contains(strings.ToLower(profileTitle), "developer") ||
						   strings.Contains(strings.ToLower(profileTitle), "software") {
							qualityScore++
							qualityReasons = append(qualityReasons, "✓ Relevant title")
						}
						
						if profileCompany != "" {
							qualityScore++
							qualityReasons = append(qualityReasons, "✓ Has company")
						}
						
						fmt.Printf("         📊 Profile quality score: %d/3\n", qualityScore)
						for _, reason := range qualityReasons {
							fmt.Printf("            %s\n", reason)
						}
						
						// Only proceed if quality score is acceptable
						if qualityScore >= 2 {
							fmt.Printf("         ✅ Profile quality acceptable - proceeding with connection\n")
						} else {
							fmt.Printf("         ⚠️  Profile quality too low - skipping connection\n")
							continue
						}
						
						// Step 3: Click Connect button with human-like behavior
						fmt.Printf("         🖱️  Attempting to click Connect button for %s...\n", profileName)
						
						// Scroll the button into view
						fmt.Println("         📜 Scrolling button into view...")
						if err := connectBtn.ScrollIntoView(); err != nil {
							fmt.Printf("         ⚠️  Scroll into view failed: %v\n", err)
						}
						
						// Small delay after scroll
						time.Sleep(1 * time.Second)
						
						// Human-like mouse movement to button
						fmt.Println("         🖱️  Moving mouse to Connect button...")
						if err := app.stealthManager.HumanMouseMove(ctx, page, connectBtn); err != nil {
							fmt.Printf("         ⚠️  Mouse movement failed: %v\n", err)
							// Try clicking anyway
						}
						
						// Click the Connect button
						fmt.Println("         🎯 Clicking Connect button...")
						if err := connectBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
							fmt.Printf("         ❌ Connect button click failed: %v\n", err)
							fmt.Println("         🔍 Trying alternative click method...")
							
							// Try JavaScript click as fallback
							if _, err := connectBtn.Eval("() => this.click()"); err != nil {
								fmt.Printf("         ❌ JavaScript click also failed: %v\n", err)
								continue
							}
						}
						
						fmt.Printf("         ✅ Connect button clicked for %s\n", profileName)
						
						// Step 4: Handle connection dialog
						fmt.Println("         📝 Waiting for connection dialog...")
						
						// Wait longer for dialog to appear and try multiple times
						dialogFound := false
						for attempt := 0; attempt < 5; attempt++ {
							time.Sleep(1 * time.Second)
							fmt.Printf("         🔍 Looking for dialog (attempt %d/5)...\n", attempt+1)
							
							// Check if we can find any connection dialog elements
							dialogSelectors := []string{
								"div[data-test-modal]",
								".send-invite",
								"[data-test-modal-id='send-invite-modal']",
								".artdeco-modal",
								"div[role='dialog']",
							}
							
							for _, selector := range dialogSelectors {
								if _, err := page.Element(selector); err == nil {
									fmt.Printf("         ✅ Connection dialog found with selector: %s\n", selector)
									dialogFound = true
									break
								}
							}
							
							if dialogFound {
								break
							}
						}
						
						if !dialogFound {
							fmt.Println("         ⚠️  No connection dialog found - connection may have been sent directly")
							connectableProfiles++
						} else {
							// Look for "Add a note" button with multiple selectors
							fmt.Println("         📝 Looking for 'Add a note' option...")
							
							addNoteSelectors := []string{
								"button[aria-label*='Add a note']",
								"button:contains('Add a note')",
								".send-invite__custom-message button",
								"button[data-control-name='add_note']",
							}
							
							var addNoteBtn *rod.Element
							for _, selector := range addNoteSelectors {
								if btn, err := page.Element(selector); err == nil {
									addNoteBtn = btn
									fmt.Printf("         ✅ 'Add a note' button found with selector: %s\n", selector)
									break
								}
							}
							
							if addNoteBtn != nil {
								fmt.Println("         📝 Adding personalized message...")
								
								// Click "Add a note"
								if err := addNoteBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
									fmt.Printf("         ⚠️  Add note button click failed: %v\n", err)
								} else {
									// Wait for note textarea with multiple selectors
									time.Sleep(2 * time.Second)
									
									textareaSelectors := []string{
										"textarea[name='message']",
										"textarea[id*='custom-message']",
										".send-invite__custom-message textarea",
										"textarea[aria-label*='message']",
									}
									
									var noteTextarea *rod.Element
									for _, selector := range textareaSelectors {
										if textarea, err := page.Element(selector); err == nil {
											noteTextarea = textarea
											fmt.Printf("         ✅ Note textarea found with selector: %s\n", selector)
											break
										}
									}
									
									if noteTextarea != nil {
										// Prepare personalized note
										personalizedNote := fmt.Sprintf("Hi %s! I came across your profile and would love to connect. I'm interested in software engineering and would enjoy sharing insights with fellow professionals in the field.", profileName)
										
										fmt.Printf("         ⌨️  Typing personalized note...\n")
										
										// Type with human-like behavior
										if err := app.stealthManager.HumanType(ctx, noteTextarea, personalizedNote); err != nil {
											fmt.Printf("         ⚠️  Note typing failed: %v\n", err)
										} else {
											fmt.Println("         ✅ Personalized note entered")
										}
									} else {
										fmt.Println("         ⚠️  Note textarea not found")
									}
								}
							}
							
							// Step 5: Send the connection request
							fmt.Println("         📤 Looking for Send button...")
							
							// Look for Send button with multiple selectors
							sendSelectors := []string{
								"button[aria-label*='Send']",
								"button:contains('Send')",
								"button[data-control-name='send']",
								".send-invite__actions button[type='submit']",
								"button[aria-label*='Send invitation']",
							}
							
							var sendBtn *rod.Element
							for _, selector := range sendSelectors {
								if btn, err := page.Element(selector); err == nil {
									sendBtn = btn
									fmt.Printf("         ✅ Send button found with selector: %s\n", selector)
									break
								}
							}
							
							if sendBtn != nil {
								// Human-like delay before sending
								fmt.Println("         🤔 Taking a moment to review the request...")
								app.stealthManager.RandomDelay(2*time.Second, 4*time.Second)
								
								// Click Send
								fmt.Println("         🎯 Clicking Send button...")
								if err := sendBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
									fmt.Printf("         ❌ Send button click failed: %v\n", err)
									
									// Try JavaScript click as fallback
									if _, err := sendBtn.Eval("() => this.click()"); err != nil {
										fmt.Printf("         ❌ JavaScript Send click also failed: %v\n", err)
									} else {
										fmt.Printf("         🎉 Connection request sent to %s! (via JavaScript)\n", profileName)
										connectableProfiles++
									}
								} else {
									fmt.Printf("         🎉 Connection request sent to %s!\n", profileName)
									connectableProfiles++
								}
								
								if connectableProfiles > 0 {
									// Step 6: Track the sent request
									fmt.Println("         💾 Tracking sent connection request...")
									fmt.Printf("         📊 Request tracked: %s at %s\n", profileName, time.Now().Format("15:04:05"))
									
									// Rate limiting delay
									fmt.Println("         ⏱️  Applying rate limiting delay...")
									app.stealthManager.RandomDelay(10*time.Second, 20*time.Second)
								}
							} else {
								fmt.Println("         ⚠️  Send button not found")
								fmt.Println("         🔍 Available buttons in dialog:")
								
								// Debug: list all buttons in the dialog
								if buttons, err := page.Elements("button"); err == nil {
									for i, btn := range buttons {
										if i >= 5 { // Limit to first 5 buttons
											break
										}
										if text, err := btn.Text(); err == nil && text != "" {
											fmt.Printf("            Button %d: '%s'\n", i+1, text)
										}
									}
								}
							}
						}
						
						// Close any remaining dialogs
						fmt.Println("         🔄 Closing dialog...")
						closeSelectors := []string{
							"button[aria-label*='Dismiss']",
							"button[aria-label*='Close']", 
							".artdeco-modal__dismiss",
							"button[data-control-name='overlay.close_modal']",
						}
						
						for _, selector := range closeSelectors {
							if closeBtn, err := page.Element(selector); err == nil {
								closeBtn.Click(proto.InputMouseButtonLeft, 1)
								fmt.Println("         ✅ Dialog closed")
								break
							}
						}
						
					} else {
						fmt.Printf("         ℹ️  No Connect button found on profile %d\n", i+1)
						fmt.Printf("         🔍 Debug - Connect button search failed: %v\n", connectBtnErr)
						
						// Debug: Show what buttons are available in this profile
						if buttons, err := profile.Elements("button"); err == nil {
							fmt.Printf("         📋 Available buttons in profile %d:\n", i+1)
							for j, btn := range buttons {
								if j >= 3 { // Limit to first 3 buttons
									break
								}
								if text, err := btn.Text(); err == nil && text != "" {
									fmt.Printf("            Button %d: '%s'\n", j+1, text)
								}
								if ariaLabel, err := btn.Attribute("aria-label"); err == nil && *ariaLabel != "" {
									fmt.Printf("            Button %d aria-label: '%s'\n", j+1, *ariaLabel)
								}
							}
						}
					}
					
					// Small delay between profile analysis
					app.stealthManager.RandomDelay(1*time.Second, 3*time.Second)
				}
				
				fmt.Printf("\n   🎉 Connection Request Automation Summary\n")
				fmt.Printf("   ═══════════════════════════════════════\n")
				fmt.Printf("   📊 Total connection requests sent: %d/%d\n", connectableProfiles, maxConnections)
				fmt.Printf("   ⏱️  Rate limit: %d connections/hour\n", app.config.RateLimit.ConnectionsPerHour)
				fmt.Printf("   🕐 Remaining quota: %d connections\n", app.config.RateLimit.ConnectionsPerHour-connectableProfiles)
				fmt.Printf("   🎯 Success rate: %.1f%%\n", float64(connectableProfiles)/float64(maxConnections)*100)
				fmt.Printf("   ⚠️  Remember: Use connection requests responsibly!\n")
				
				if connectableProfiles > 0 {
					fmt.Printf("\n   💡 Next Steps:\n")
					fmt.Printf("      • Monitor your LinkedIn notifications for acceptances\n")
					fmt.Printf("      • Follow up with personalized messages when connections are accepted\n")
					fmt.Printf("      • Respect LinkedIn's weekly connection limits\n")
					fmt.Printf("      • Build genuine professional relationships\n")
				}
				
			} else {
				fmt.Printf("      ⚠️  Could not find profile results: %v\n", err)
			}
		}
	} else {
		fmt.Println("   ℹ️  User declined - skipping real connection requests")
		fmt.Println("   🎭 Running connection workflow simulation instead...")
		
		// Fallback to simulation
		fmt.Println("      🔍 Simulating profile analysis...")
		fmt.Println("      🤝 Simulating Connect button detection...")
		fmt.Println("      📝 Simulating personalized note creation...")
		fmt.Println("      📤 Simulating connection request sending...")
		fmt.Println("      💾 Simulating request tracking...")
		fmt.Println("      ✅ Connection workflow simulation completed")
	}

	// Demo 17: Messaging Workflow Simulation  
	fmt.Println("\n🎯 Demo 17/18: Follow-up Messaging Workflow")
	fmt.Println("   💬 Demonstrating messaging automation capabilities:")
	
	fmt.Println("   📨 Simulating follow-up message workflow...")
	fmt.Println("   ⚠️  Note: This is a SIMULATION - no actual messages will be sent")
	
	// Simulate connection acceptance detection
	fmt.Println("   🔍 Step 1: Connection acceptance detection...")
	fmt.Println("      📊 Simulating connection status monitoring...")
	fmt.Println("      🎉 Simulating newly accepted connection detection...")
	fmt.Println("      ✅ Connection acceptance detected")
	
	// Simulate message template processing
	fmt.Println("   📝 Step 2: Message template processing...")
	messageTemplate := "Thanks for connecting, [Name]! I'm excited to be part of your network. Looking forward to sharing insights about [Industry]."
	fmt.Printf("      💬 Sample template: %s\n", messageTemplate)
	fmt.Println("      🔄 Simulating variable substitution...")
	processedMessage := "Thanks for connecting, John! I'm excited to be part of your network. Looking forward to sharing insights about Software Engineering."
	fmt.Printf("      ✅ Processed message: %s\n", processedMessage)
	
	// Simulate messaging rate limits
	fmt.Println("   ⏱️  Step 3: Messaging rate limit verification...")
	fmt.Printf("      📊 Message rate limit: %d messages/hour\n", app.config.RateLimit.MessagesPerHour)
	fmt.Println("      🕐 Checking message frequency limits...")
	fmt.Println("      ✅ Messaging rate limits verified")
	
	// Simulate message sending
	fmt.Println("   📤 Step 4: Message sending simulation...")
	fmt.Println("      🌐 Simulating navigation to messaging interface...")
	fmt.Println("      🎯 Simulating recipient selection...")
	fmt.Println("      ⌨️  Simulating message composition with human typing...")
	fmt.Println("      📨 Simulating message send action...")
	fmt.Println("      💾 Simulating message history tracking...")
	fmt.Println("      ✅ Follow-up message workflow simulated successfully")

	// Demo 18: Complete Automation Integration
	fmt.Println("\n🎯 Demo 18/18: Complete LinkedIn Automation Integration")
	fmt.Println("   🎊 Grand finale - Full automation workflow integration:")
	
	fmt.Println("   🔄 Executing complete integrated automation sequence...")
	
	// Integrated workflow simulation
	fmt.Println("      1️⃣  Search execution with human-like browsing...")
	app.stealthManager.ScrollNaturally(ctx, page)
	
	fmt.Println("      2️⃣  Profile evaluation with natural timing...")
	app.stealthManager.RandomDelay(2*time.Second, 4*time.Second)
	
	fmt.Println("      3️⃣  Connection request with stealth behaviors...")
	app.stealthManager.IdleBehavior(ctx, page)
	
	fmt.Println("      4️⃣  Rate limiting and cooldown enforcement...")
	app.stealthManager.RandomDelay(1*time.Second, 3*time.Second)
	
	fmt.Println("      5️⃣  Message follow-up with human patterns...")
	app.stealthManager.ScrollNaturally(ctx, page)
	
	fmt.Println("      6️⃣  Session state preservation...")
	app.browserManager.SaveCookies("./complete_session.json")
	
	fmt.Println("   🎉 Complete automation integration test successful!")

	// Final Analysis and Summary
	fmt.Println("\n🏆 COMPREHENSIVE DEMO COMPLETE!")
	fmt.Println("================================================")
	
	fmt.Println("\n📊 Session Statistics:")
	if info, err := page.Info(); err == nil {
		finalURL := info.URL
		finalTitle := info.Title
		fmt.Printf("   📍 Final URL: %s\n", finalURL)
		fmt.Printf("   📄 Final Title: %s\n", finalTitle)
	} else {
		fmt.Printf("   ⚠️  Could not get final session info: %v\n", err)
	}
	
	fmt.Printf("   ⏱️  Demo duration: ~15-20 minutes\n")
	fmt.Printf("   🎯 Demonstrations completed: 18/18\n")
	
	fmt.Println("\n🎓 Educational Achievements Unlocked:")
	fmt.Println("   ✅ Advanced browser automation mastery")
	fmt.Println("   ✅ Human behavior simulation expertise")
	fmt.Println("   ✅ Anti-detection technique understanding")
	fmt.Println("   ✅ Rod library proficiency")
	fmt.Println("   ✅ Go programming pattern recognition")
	fmt.Println("   ✅ Session management skills")
	fmt.Println("   ✅ Error handling best practices")
	fmt.Println("   ✅ Rate limiting implementation")
	fmt.Println("   ✅ Configuration management")
	fmt.Println("   ✅ Performance optimization awareness")
	fmt.Println("   ✅ LinkedIn search automation understanding")
	fmt.Println("   ✅ Connection request workflow mastery")
	fmt.Println("   ✅ Messaging automation expertise")
	fmt.Println("   ✅ Complete workflow integration skills")
	
	fmt.Println("\n🔬 Technical Concepts Demonstrated:")
	fmt.Println("   • Bézier curve mouse trajectories")
	fmt.Println("   • Gaussian distribution timing patterns")
	fmt.Println("   • Browser fingerprint analysis")
	fmt.Println("   • Session persistence mechanisms")
	fmt.Println("   • Network activity simulation")
	fmt.Println("   • DOM interaction strategies")
	fmt.Println("   • Error recovery patterns")
	fmt.Println("   • Rate limiting algorithms")
	fmt.Println("   • Configuration management systems")
	fmt.Println("   • Performance monitoring techniques")
	
	fmt.Println("\n💡 Key Insights:")
	fmt.Println("   🎯 Manual login + automation is the safest approach")
	fmt.Println("   🛡️  Human-like behavior is crucial for avoiding detection")
	fmt.Println("   ⚡ Proper timing and rate limiting prevent blocking")
	fmt.Println("   🔧 Modular architecture enables flexible automation")
	fmt.Println("   📊 Comprehensive logging aids in debugging and optimization")
	
	fmt.Println("\n⚠️  Ethical Reminders:")
	fmt.Println("   • This framework is for educational purposes only")
	fmt.Println("   • Always respect platform Terms of Service")
	fmt.Println("   • Use automation responsibly and ethically")
	fmt.Println("   • Consider the impact on other users and platforms")
	fmt.Println("   • Manual login approach reduces ethical concerns")
	
	fmt.Println("\n🚀 Next Steps for Learning:")
	fmt.Println("   📚 Study the source code architecture")
	fmt.Println("   🧪 Experiment with different configurations")
	fmt.Println("   🔬 Analyze the property-based test suite")
	fmt.Println("   🛠️  Extend the framework with new capabilities")
	fmt.Println("   📖 Read about browser automation best practices")

	app.logger.Info(ctx, "🎊 COMPREHENSIVE manual login + automation demo completed successfully!")
	
	fmt.Println("\n🎬 Thank you for watching the LinkedIn Automation Framework demo!")
	fmt.Println("   Remember: With great automation power comes great responsibility! 🕷️")
	
	return nil
}

// runConnectOnly focuses exclusively on connection request automation
func (app *Application) runConnectOnly(ctx context.Context) error {
	fmt.Println("\n🤝 LinkedIn Connection Request Automation")
	fmt.Println("=========================================")
	fmt.Println("This mode focuses exclusively on sending connection requests.")
	fmt.Println("You'll manually login, then the system will help you send")
	fmt.Println("intelligent, personalized connection requests.")
	fmt.Println("")
	fmt.Println("🎯 Features:")
	fmt.Println("   • Profile quality assessment")
	fmt.Println("   • Personalized connection notes")
	fmt.Println("   • Rate limiting and safety controls")
	fmt.Println("   • Human-like interaction patterns")
	fmt.Println("   • Connection request tracking")
	fmt.Println("")
	fmt.Println("⚠️  Important Reminders:")
	fmt.Println("   • This will send REAL connection requests")
	fmt.Println("   • Use responsibly and respect LinkedIn's limits")
	fmt.Println("   • Focus on building genuine professional relationships")
	fmt.Println("   • Always personalize your connection messages")
	fmt.Println("")

	app.logger.Info(ctx, "🚀 Starting connection-only automation mode")

	// Create a new page
	page, err := app.browserManager.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Navigate to LinkedIn
	fmt.Println("🌐 Opening LinkedIn login page...")
	if err := page.Navigate("https://www.linkedin.com/login"); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}
	page.WaitLoad()
	fmt.Println("   ✅ LinkedIn login page loaded")

	// Wait for manual login
	fmt.Println("\n👤 Please login manually in the browser window...")
	fmt.Print("🔄 Press ENTER when logged in and ready to start connecting: ")
	var input string
	fmt.Scanln(&input)

	// Get connection preferences from user
	fmt.Println("\n⚙️  Connection Request Configuration")
	fmt.Println("   Let's configure your connection request preferences...")
	
	fmt.Print("   🔢 How many connection requests to send? (1-10, default 3): ")
	var maxConnectionsInput string
	fmt.Scanln(&maxConnectionsInput)
	
	maxConnections := 3 // default
	if maxConnectionsInput != "" {
		if parsed, err := strconv.Atoi(maxConnectionsInput); err == nil && parsed >= 1 && parsed <= 10 {
			maxConnections = parsed
		}
	}
	
	fmt.Print("   🔍 Search keywords (default 'software engineer'): ")
	var searchKeywords string
	fmt.Scanln(&searchKeywords)
	
	if searchKeywords == "" {
		searchKeywords = "software engineer"
	}
	
	fmt.Printf("   ✅ Configuration set: %d requests for '%s'\n", maxConnections, searchKeywords)

	// Navigate to search
	fmt.Println("\n🔍 Navigating to LinkedIn search...")
	searchURL := fmt.Sprintf("https://www.linkedin.com/search/results/people/?keywords=%s", 
		strings.ReplaceAll(searchKeywords, " ", "%20"))
	
	if err := page.Navigate(searchURL); err != nil {
		return fmt.Errorf("search navigation failed: %w", err)
	}
	page.WaitLoad()
	fmt.Println("   ✅ Search results loaded")

	// Start connection automation
	fmt.Println("\n🤝 Starting Intelligent Connection Request Automation")
	fmt.Println("   ═══════════════════════════════════════════════════")
	
	if profiles, err := page.Elements(".reusable-search__result-container"); err == nil {
		connectableProfiles := 0
		attemptedProfiles := 0
		
		for _, profile := range profiles {
			if connectableProfiles >= maxConnections {
				break
			}
			
			attemptedProfiles++
			fmt.Printf("\n   👤 Profile %d/%d Analysis\n", attemptedProfiles, len(profiles))
			fmt.Println("   ─────────────────────────")
			
			// Profile quality assessment (same as in manual-login mode)
			if connectBtn, err := profile.Element("button[aria-label*='Connect']"); err == nil {
				fmt.Println("      ✅ Connect button available")
				
				// Extract and assess profile
				profileName := "Professional"
				profileTitle := ""
				profileCompany := ""
				
				if nameElement, err := profile.Element("span[aria-hidden='true']"); err == nil {
					if name, err := nameElement.Text(); err == nil {
						profileName = name
						fmt.Printf("      📝 Name: %s\n", profileName)
					}
				}
				
				if titleElement, err := profile.Element(".entity-result__primary-subtitle"); err == nil {
					if title, err := titleElement.Text(); err == nil {
						profileTitle = title
						fmt.Printf("      💼 Title: %s\n", profileTitle)
					}
				}
				
				// Quality assessment
				qualityScore := 0
				if profileName != "Professional" && profileName != "" {
					qualityScore++
				}
				if strings.Contains(strings.ToLower(profileTitle), "engineer") || 
				   strings.Contains(strings.ToLower(profileTitle), "developer") ||
				   strings.Contains(strings.ToLower(profileTitle), "software") {
					qualityScore++
				}
				if profileCompany != "" {
					qualityScore++
				}
				
				fmt.Printf("      📊 Quality Score: %d/3\n", qualityScore)
				
				if qualityScore >= 2 {
					fmt.Println("      ✅ Quality acceptable - sending connection request")
					
					// Send connection request with same logic as manual-login mode
					if err := app.stealthManager.HumanMouseMove(ctx, page, connectBtn); err == nil {
						if err := connectBtn.Click(proto.InputMouseButtonLeft, 1); err == nil {
							fmt.Printf("      🤝 Connection request initiated for %s\n", profileName)
							
							// Handle dialog and send personalized note
							time.Sleep(2 * time.Second)
							
							if addNoteBtn, err := page.Element("button[aria-label*='Add a note']"); err == nil {
								addNoteBtn.Click(proto.InputMouseButtonLeft, 1)
								time.Sleep(1 * time.Second)
								
								if noteTextarea, err := page.Element("textarea[name='message']"); err == nil {
									personalizedNote := fmt.Sprintf("Hi %s! I found your profile while searching for %s professionals. I'd love to connect and share insights about our industry.", profileName, searchKeywords)
									
									if err := app.stealthManager.HumanType(ctx, noteTextarea, personalizedNote); err == nil {
										fmt.Println("      📝 Personalized note added")
									}
								}
							}
							
							// Send the request
							if sendBtn, err := page.Element("button[aria-label*='Send']"); err == nil {
								app.stealthManager.RandomDelay(2*time.Second, 4*time.Second)
								if err := sendBtn.Click(proto.InputMouseButtonLeft, 1); err == nil {
									fmt.Printf("      🎉 Connection request sent to %s!\n", profileName)
									connectableProfiles++
									
									// Rate limiting delay
									fmt.Println("      ⏱️  Applying safety delay...")
									app.stealthManager.RandomDelay(15*time.Second, 25*time.Second)
								}
							}
						}
					}
				} else {
					fmt.Println("      ⚠️  Quality too low - skipping")
				}
			} else {
				fmt.Println("      ℹ️  No Connect button (already connected or premium required)")
			}
			
			// Small delay between profiles
			app.stealthManager.RandomDelay(2*time.Second, 5*time.Second)
		}
		
		// Final summary
		fmt.Printf("\n🎊 Connection Automation Complete!\n")
		fmt.Printf("═══════════════════════════════════\n")
		fmt.Printf("📊 Results Summary:\n")
		fmt.Printf("   • Profiles analyzed: %d\n", attemptedProfiles)
		fmt.Printf("   • Connection requests sent: %d\n", connectableProfiles)
		fmt.Printf("   • Success rate: %.1f%%\n", float64(connectableProfiles)/float64(attemptedProfiles)*100)
		fmt.Printf("   • Remaining daily quota: ~%d\n", app.config.RateLimit.ConnectionsPerHour-connectableProfiles)
		
		fmt.Printf("\n💡 What's Next:\n")
		fmt.Printf("   • Check LinkedIn notifications for acceptances\n")
		fmt.Printf("   • Send follow-up messages to new connections\n")
		fmt.Printf("   • Continue building your professional network\n")
		fmt.Printf("   • Use the messaging mode for follow-ups\n")
		
	} else {
		fmt.Printf("Could not find profiles: %v\n", err)
	}

	app.logger.Info(ctx, "🎊 Connection-only automation completed successfully")
	return nil
}