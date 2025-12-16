package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Browser   BrowserConfig   `yaml:"browser"`
	Stealth   StealthConfig   `yaml:"stealth"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Storage   StorageConfig   `yaml:"storage"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// BrowserConfig contains browser-specific settings
type BrowserConfig struct {
	Headless    bool     `yaml:"headless"`
	UserAgent   string   `yaml:"user_agent"`
	ViewportW   int      `yaml:"viewport_width"`
	ViewportH   int      `yaml:"viewport_height"`
	Flags       []string `yaml:"flags"`
	CookiePath  string   `yaml:"cookie_path"`
}

// StealthConfig contains stealth behavior parameters
type StealthConfig struct {
	MinDelay        time.Duration `yaml:"min_delay"`
	MaxDelay        time.Duration `yaml:"max_delay"`
	TypingMinDelay  time.Duration `yaml:"typing_min_delay"`
	TypingMaxDelay  time.Duration `yaml:"typing_max_delay"`
	ScrollMinDelay  time.Duration `yaml:"scroll_min_delay"`
	ScrollMaxDelay  time.Duration `yaml:"scroll_max_delay"`
	BusinessHours   bool          `yaml:"respect_business_hours"`
	CooldownPeriod  time.Duration `yaml:"cooldown_period"`
}

// RateLimitConfig contains rate limiting parameters
type RateLimitConfig struct {
	ConnectionsPerHour int           `yaml:"connections_per_hour"`
	MessagesPerHour    int           `yaml:"messages_per_hour"`
	SearchesPerHour    int           `yaml:"searches_per_hour"`
	CooldownBetween    time.Duration `yaml:"cooldown_between"`
}

// StorageConfig contains storage settings
type StorageConfig struct {
	Type     string `yaml:"type"` // "sqlite" or "json"
	Path     string `yaml:"path"`
	Database string `yaml:"database"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// ConfigManager interface for configuration management
type ConfigManager interface {
	Load(path string) (*Config, error)
	LoadWithEnvOverrides(path string) (*Config, error)
	Validate(config *Config) error
	GetDefaults() *Config
}

// Manager implements ConfigManager interface
type Manager struct{}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{}
}

// Load loads configuration from YAML file
func (m *Manager) Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return config, nil
}

// LoadWithEnvOverrides loads configuration from YAML and applies environment variable overrides
func (m *Manager) LoadWithEnvOverrides(path string) (*Config, error) {
	config, err := m.Load(path)
	if err != nil {
		// If file doesn't exist, start with defaults
		if os.IsNotExist(err) {
			config = m.GetDefaults()
		} else {
			return nil, err
		}
	}

	// Apply environment variable overrides
	m.applyEnvOverrides(config)

	// Validate the final configuration
	if err := m.Validate(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// applyEnvOverrides applies environment variable overrides to configuration
func (m *Manager) applyEnvOverrides(config *Config) {
	// Browser configuration overrides
	if val := os.Getenv("BROWSER_HEADLESS"); val != "" {
		if headless, err := strconv.ParseBool(val); err == nil {
			config.Browser.Headless = headless
		}
	}
	if val := os.Getenv("BROWSER_USER_AGENT"); val != "" {
		config.Browser.UserAgent = val
	}
	if val := os.Getenv("BROWSER_VIEWPORT_WIDTH"); val != "" {
		if width, err := strconv.Atoi(val); err == nil {
			config.Browser.ViewportW = width
		}
	}
	if val := os.Getenv("BROWSER_VIEWPORT_HEIGHT"); val != "" {
		if height, err := strconv.Atoi(val); err == nil {
			config.Browser.ViewportH = height
		}
	}
	if val := os.Getenv("BROWSER_FLAGS"); val != "" {
		config.Browser.Flags = strings.Split(val, ",")
	}
	if val := os.Getenv("BROWSER_COOKIE_PATH"); val != "" {
		config.Browser.CookiePath = val
	}

	// Stealth configuration overrides
	if val := os.Getenv("STEALTH_MIN_DELAY"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Stealth.MinDelay = duration
		}
	}
	if val := os.Getenv("STEALTH_MAX_DELAY"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Stealth.MaxDelay = duration
		}
	}
	if val := os.Getenv("STEALTH_TYPING_MIN_DELAY"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Stealth.TypingMinDelay = duration
		}
	}
	if val := os.Getenv("STEALTH_TYPING_MAX_DELAY"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Stealth.TypingMaxDelay = duration
		}
	}
	if val := os.Getenv("STEALTH_SCROLL_MIN_DELAY"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Stealth.ScrollMinDelay = duration
		}
	}
	if val := os.Getenv("STEALTH_SCROLL_MAX_DELAY"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Stealth.ScrollMaxDelay = duration
		}
	}
	if val := os.Getenv("STEALTH_BUSINESS_HOURS"); val != "" {
		if businessHours, err := strconv.ParseBool(val); err == nil {
			config.Stealth.BusinessHours = businessHours
		}
	}
	if val := os.Getenv("STEALTH_COOLDOWN_PERIOD"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Stealth.CooldownPeriod = duration
		}
	}

	// Rate limit configuration overrides
	if val := os.Getenv("RATE_LIMIT_CONNECTIONS_PER_HOUR"); val != "" {
		if rate, err := strconv.Atoi(val); err == nil {
			config.RateLimit.ConnectionsPerHour = rate
		}
	}
	if val := os.Getenv("RATE_LIMIT_MESSAGES_PER_HOUR"); val != "" {
		if rate, err := strconv.Atoi(val); err == nil {
			config.RateLimit.MessagesPerHour = rate
		}
	}
	if val := os.Getenv("RATE_LIMIT_SEARCHES_PER_HOUR"); val != "" {
		if rate, err := strconv.Atoi(val); err == nil {
			config.RateLimit.SearchesPerHour = rate
		}
	}
	if val := os.Getenv("RATE_LIMIT_COOLDOWN_BETWEEN"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.RateLimit.CooldownBetween = duration
		}
	}

	// Storage configuration overrides
	if val := os.Getenv("STORAGE_TYPE"); val != "" {
		config.Storage.Type = val
	}
	if val := os.Getenv("STORAGE_PATH"); val != "" {
		config.Storage.Path = val
	}
	if val := os.Getenv("STORAGE_DATABASE"); val != "" {
		config.Storage.Database = val
	}

	// Logging configuration overrides
	if val := os.Getenv("LOGGING_LEVEL"); val != "" {
		config.Logging.Level = val
	}
	if val := os.Getenv("LOGGING_FORMAT"); val != "" {
		config.Logging.Format = val
	}
	if val := os.Getenv("LOGGING_OUTPUT"); val != "" {
		config.Logging.Output = val
	}
}

// Validate validates the configuration and applies defaults where necessary
func (m *Manager) Validate(config *Config) error {
	// Apply defaults if values are missing or invalid
	defaults := m.GetDefaults()

	// Browser validation and defaults
	if config.Browser.UserAgent == "" {
		config.Browser.UserAgent = defaults.Browser.UserAgent
	}
	if config.Browser.ViewportW <= 0 {
		config.Browser.ViewportW = defaults.Browser.ViewportW
	}
	if config.Browser.ViewportH <= 0 {
		config.Browser.ViewportH = defaults.Browser.ViewportH
	}
	if config.Browser.CookiePath == "" {
		config.Browser.CookiePath = defaults.Browser.CookiePath
	}

	// Stealth validation and defaults
	if config.Stealth.MinDelay <= 0 {
		config.Stealth.MinDelay = defaults.Stealth.MinDelay
	}
	if config.Stealth.MaxDelay <= 0 {
		config.Stealth.MaxDelay = defaults.Stealth.MaxDelay
	}
	if config.Stealth.MaxDelay < config.Stealth.MinDelay {
		return fmt.Errorf("stealth max_delay (%v) must be greater than min_delay (%v)", config.Stealth.MaxDelay, config.Stealth.MinDelay)
	}
	if config.Stealth.TypingMinDelay <= 0 {
		config.Stealth.TypingMinDelay = defaults.Stealth.TypingMinDelay
	}
	if config.Stealth.TypingMaxDelay <= 0 {
		config.Stealth.TypingMaxDelay = defaults.Stealth.TypingMaxDelay
	}
	if config.Stealth.TypingMaxDelay < config.Stealth.TypingMinDelay {
		return fmt.Errorf("stealth typing_max_delay (%v) must be greater than typing_min_delay (%v)", config.Stealth.TypingMaxDelay, config.Stealth.TypingMinDelay)
	}
	if config.Stealth.ScrollMinDelay <= 0 {
		config.Stealth.ScrollMinDelay = defaults.Stealth.ScrollMinDelay
	}
	if config.Stealth.ScrollMaxDelay <= 0 {
		config.Stealth.ScrollMaxDelay = defaults.Stealth.ScrollMaxDelay
	}
	if config.Stealth.ScrollMaxDelay < config.Stealth.ScrollMinDelay {
		return fmt.Errorf("stealth scroll_max_delay (%v) must be greater than scroll_min_delay (%v)", config.Stealth.ScrollMaxDelay, config.Stealth.ScrollMinDelay)
	}
	if config.Stealth.CooldownPeriod <= 0 {
		config.Stealth.CooldownPeriod = defaults.Stealth.CooldownPeriod
	}

	// Rate limit validation and defaults
	if config.RateLimit.ConnectionsPerHour <= 0 {
		config.RateLimit.ConnectionsPerHour = defaults.RateLimit.ConnectionsPerHour
	}
	if config.RateLimit.MessagesPerHour <= 0 {
		config.RateLimit.MessagesPerHour = defaults.RateLimit.MessagesPerHour
	}
	if config.RateLimit.SearchesPerHour <= 0 {
		config.RateLimit.SearchesPerHour = defaults.RateLimit.SearchesPerHour
	}
	if config.RateLimit.CooldownBetween <= 0 {
		config.RateLimit.CooldownBetween = defaults.RateLimit.CooldownBetween
	}

	// Storage validation and defaults
	if config.Storage.Type == "" {
		config.Storage.Type = defaults.Storage.Type
	}
	if config.Storage.Type != "sqlite" && config.Storage.Type != "json" {
		return fmt.Errorf("storage type must be 'sqlite' or 'json', got: %s", config.Storage.Type)
	}
	if config.Storage.Path == "" {
		config.Storage.Path = defaults.Storage.Path
	}
	if config.Storage.Database == "" {
		config.Storage.Database = defaults.Storage.Database
	}

	// Logging validation and defaults
	if config.Logging.Level == "" {
		config.Logging.Level = defaults.Logging.Level
	}
	validLevels := []string{"debug", "info", "warn", "error"}
	levelValid := false
	for _, level := range validLevels {
		if strings.ToLower(config.Logging.Level) == level {
			levelValid = true
			config.Logging.Level = level // normalize case
			break
		}
	}
	if !levelValid {
		return fmt.Errorf("logging level must be one of %v, got: %s", validLevels, config.Logging.Level)
	}
	if config.Logging.Format == "" {
		config.Logging.Format = defaults.Logging.Format
	}
	if config.Logging.Output == "" {
		config.Logging.Output = defaults.Logging.Output
	}

	return nil
}

// GetDefaults returns default configuration values
func (m *Manager) GetDefaults() *Config {
	return &Config{
		Browser: BrowserConfig{
			Headless:   true,
			UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			ViewportW:  1920,
			ViewportH:  1080,
			Flags:      []string{"--no-sandbox", "--disable-blink-features=AutomationControlled"},
			CookiePath: "./cookies.json",
		},
		Stealth: StealthConfig{
			MinDelay:        500 * time.Millisecond,
			MaxDelay:        2 * time.Second,
			TypingMinDelay:  50 * time.Millisecond,
			TypingMaxDelay:  200 * time.Millisecond,
			ScrollMinDelay:  100 * time.Millisecond,
			ScrollMaxDelay:  500 * time.Millisecond,
			BusinessHours:   true,
			CooldownPeriod:  5 * time.Minute,
		},
		RateLimit: RateLimitConfig{
			ConnectionsPerHour: 10,
			MessagesPerHour:    5,
			SearchesPerHour:    20,
			CooldownBetween:    30 * time.Second,
		},
		Storage: StorageConfig{
			Type:     "sqlite",
			Path:     "./data",
			Database: "linkedin_automation.db",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}