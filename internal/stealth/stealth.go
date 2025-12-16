package stealth

import (
	"context"
	"time"
	"github.com/go-rod/rod"
)

// StealthBehavior interface for human-like behavior simulation
type StealthBehavior interface {
	HumanMouseMove(ctx context.Context, page *rod.Page, target *rod.Element) error
	HumanType(ctx context.Context, element *rod.Element, text string) error
	RandomDelay(min, max time.Duration) error
	ScrollNaturally(ctx context.Context, page *rod.Page) error
	ConfigureFingerprint(browser *rod.Browser) error
	IdleBehavior(ctx context.Context, page *rod.Page) error
	EnforceCooldown(lastAction time.Time, cooldownPeriod time.Duration) error
}

// StealthConfig contains stealth behavior parameters
type StealthConfig struct {
	MinDelay        time.Duration
	MaxDelay        time.Duration
	TypingMinDelay  time.Duration
	TypingMaxDelay  time.Duration
	ScrollMinDelay  time.Duration
	ScrollMaxDelay  time.Duration
	BusinessHours   bool
	CooldownPeriod  time.Duration
}

// FingerprintConfig contains browser fingerprint settings
type FingerprintConfig struct {
	UserAgent   string
	ViewportW   int
	ViewportH   int
	MaskWebDriver bool
}

// StealthManager implements StealthBehavior interface
type StealthManager struct {
	config      StealthConfig
	fingerprint FingerprintConfig
}

// NewStealthManager creates a new stealth manager
func NewStealthManager(config StealthConfig, fingerprint FingerprintConfig) *StealthManager {
	return &StealthManager{
		config:      config,
		fingerprint: fingerprint,
	}
}