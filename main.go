package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

// OperationMode represents different operation modes
type OperationMode string

const (
	ModeDemo       OperationMode = "demo"
	ModeSearch     OperationMode = "search"
	ModeConnect    OperationMode = "connect"
	ModeMessage    OperationMode = "message"
	ModeInteractive OperationMode = "interactive"
)



func main() {
	// Parse command line flags
	var (
		configPath = flag.String("config", "config.yaml", "Path to configuration file")
		mode       = flag.String("mode", "demo", "Operation mode: demo, search, connect, message, interactive")
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
	default:
		return fmt.Errorf("unsupported operation mode: %s", mode)
	}
}

// runDemo runs a demonstration of all framework capabilities
func (app *Application) runDemo(ctx context.Context) error {
	app.logger.Info(ctx, "Starting demonstration mode")

	// Create a new page
	page, err := app.browserManager.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Demonstrate basic browser functionality
	app.logger.Info(ctx, "Demonstrating browser navigation...")
	if err := page.Navigate("https://www.linkedin.com"); err != nil {
		app.logger.Warn(ctx, "Navigation failed", logger.F("error", err.Error()))
		return fmt.Errorf("navigation failed: %w", err)
	}

	// Demonstrate stealth behaviors
	app.logger.Info(ctx, "Demonstrating stealth behaviors...")
	if err := app.stealthManager.IdleBehavior(ctx, page); err != nil {
		app.logger.Warn(ctx, "Idle behavior failed", logger.F("error", err.Error()))
	}

	if err := app.stealthManager.ScrollNaturally(ctx, page); err != nil {
		app.logger.Warn(ctx, "Natural scrolling failed", logger.F("error", err.Error()))
	}

	// Demonstrate random delay
	app.logger.Info(ctx, "Demonstrating random delays...")
	if err := app.stealthManager.RandomDelay(app.config.Stealth.MinDelay, app.config.Stealth.MaxDelay); err != nil {
		app.logger.Warn(ctx, "Random delay failed", logger.F("error", err.Error()))
	}

	app.logger.Info(ctx, "Demo completed successfully")
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
	
	fmt.Println("LinkedIn Automation Framework - Interactive Mode")
	fmt.Println("===============================================")
	fmt.Println("This mode allows you to interactively control the automation.")
	fmt.Println("Available commands: search, connect, message, demo, quit")
	
	// Implementation would include interactive command processing
	// For now, just run demo mode
	return app.runDemo(ctx)
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