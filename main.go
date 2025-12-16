package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"linkedin-automation-framework/internal/browser"
	"linkedin-automation-framework/internal/config"
	"linkedin-automation-framework/internal/logger"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal, cleaning up...")
		cancel()
	}()

	// Initialize configuration
	cfg := &config.Config{
		Browser: config.BrowserConfig{
			Headless:  true,
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			ViewportW: 1920,
			ViewportH: 1080,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}

	// Initialize logger
	loggerConfig := logger.LoggingConfig{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	}
	appLogger := logger.NewLogger(loggerConfig)

	// Initialize browser manager
	browserConfig := browser.BrowserConfig{
		Headless:  cfg.Browser.Headless,
		UserAgent: cfg.Browser.UserAgent,
		ViewportW: cfg.Browser.ViewportW,
		ViewportH: cfg.Browser.ViewportH,
	}
	browserManager := browser.NewManager(browserConfig)

	appLogger.Info(ctx, "LinkedIn Automation Framework starting", 
		logger.F("version", "1.0.0"),
		logger.F("mode", "development"))

	// Initialize browser
	if err := browserManager.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize browser: %v", err)
	}
	defer browserManager.Close()

	appLogger.Info(ctx, "Application initialized successfully")

	// Wait for shutdown signal
	<-ctx.Done()
	appLogger.Info(ctx, "Application shutting down gracefully")
}