package stealth

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
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
	IsWithinBusinessHours(t time.Time) bool
	ShouldRateLimit(actionCount int, timeWindow time.Duration, maxActions int) bool
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
	BusinessStart   int // Hour of day (0-23)
	BusinessEnd     int // Hour of day (0-23)
	CooldownPeriod  time.Duration
	MaxActionsPerWindow int
	RateLimitWindow time.Duration
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

// Point represents a 2D coordinate
type Point struct {
	X, Y float64
}

// HumanMouseMove implements human-like mouse movement with Bézier curves and micro-corrections
func (sm *StealthManager) HumanMouseMove(ctx context.Context, page *rod.Page, target *rod.Element) error {
	// Get target element position
	box, err := target.Shape()
	if err != nil {
		return fmt.Errorf("failed to get target element shape: %w", err)
	}

	if len(box.Quads) == 0 {
		return fmt.Errorf("target element has no visible area")
	}

	// Calculate target center with slight randomization
	quad := box.Quads[0]
	targetX := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
	targetY := (quad[1] + quad[3] + quad[5] + quad[7]) / 4
	
	// Add small random offset to make movement more natural
	targetX += (rand.Float64() - 0.5) * 10
	targetY += (rand.Float64() - 0.5) * 10

	// Get current mouse position (Rod doesn't provide this directly, so we'll use a reasonable default)
	start := Point{X: 100, Y: 100} // Default starting position
	end := Point{X: targetX, Y: targetY}

	// Generate Bézier curve path with overshoot and micro-corrections
	path := sm.generateBezierPath(start, end)

	// Move along the path with human-like timing
	for i, point := range path {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Move to point using proto.Point
		err := page.Mouse.MoveTo(proto.Point{X: point.X, Y: point.Y})
		if err != nil {
			return fmt.Errorf("failed to move mouse to point: %w", err)
		}

		// Add micro-delays between movements
		if i < len(path)-1 {
			delay := time.Duration(rand.Intn(5)+1) * time.Millisecond
			time.Sleep(delay)
		}
	}

	return nil
}

// generateBezierPath creates a human-like mouse movement path using Bézier curves
func (sm *StealthManager) generateBezierPath(start, end Point) []Point {
	distance := math.Sqrt(math.Pow(end.X-start.X, 2) + math.Pow(end.Y-start.Y, 2))
	steps := int(distance / 5) // Adjust step size based on distance
	if steps < 10 {
		steps = 10
	}
	if steps > 100 {
		steps = 100
	}

	// Create control points for Bézier curve with some randomness
	cp1X := start.X + (end.X-start.X)*0.25 + (rand.Float64()-0.5)*50
	cp1Y := start.Y + (end.Y-start.Y)*0.25 + (rand.Float64()-0.5)*50
	cp2X := start.X + (end.X-start.X)*0.75 + (rand.Float64()-0.5)*50
	cp2Y := start.Y + (end.Y-start.Y)*0.75 + (rand.Float64()-0.5)*50

	cp1 := Point{X: cp1X, Y: cp1Y}
	cp2 := Point{X: cp2X, Y: cp2Y}

	path := make([]Point, steps)
	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps-1)
		point := sm.cubicBezier(start, cp1, cp2, end, t)
		
		// Add micro-corrections (small random variations)
		point.X += (rand.Float64() - 0.5) * 2
		point.Y += (rand.Float64() - 0.5) * 2
		
		path[i] = point
	}

	return path
}

// cubicBezier calculates a point on a cubic Bézier curve
func (sm *StealthManager) cubicBezier(p0, p1, p2, p3 Point, t float64) Point {
	u := 1 - t
	tt := t * t
	uu := u * u
	uuu := uu * u
	ttt := tt * t

	x := uuu*p0.X + 3*uu*t*p1.X + 3*u*tt*p2.X + ttt*p3.X
	y := uuu*p0.Y + 3*uu*t*p1.Y + 3*u*tt*p2.Y + ttt*p3.Y

	return Point{X: x, Y: y}
}

// HumanType implements human typing simulation with realistic delays and mistakes
func (sm *StealthManager) HumanType(ctx context.Context, element *rod.Element, text string) error {
	// Clear existing text first
	err := element.SelectAllText()
	if err != nil {
		return fmt.Errorf("failed to select existing text: %w", err)
	}

	// Type each character with human-like delays
	for i, char := range text {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Simulate occasional typing mistakes (5% chance)
		if rand.Float64() < 0.05 && i > 0 {
			// Type a wrong character, then backspace and correct
			wrongChar := rune('a' + rand.Intn(26))
			err := element.Input(string(wrongChar))
			if err != nil {
				return fmt.Errorf("failed to input wrong character: %w", err)
			}
			
			// Delay before correction
			delay := time.Duration(rand.Intn(200)+100) * time.Millisecond
			time.Sleep(delay)
			
			// Backspace
			keyActions, err := element.KeyActions()
			if err != nil {
				return fmt.Errorf("failed to get key actions: %w", err)
			}
			err = keyActions.Press(input.Backspace).Do()
			if err != nil {
				return fmt.Errorf("failed to press backspace: %w", err)
			}
			
			// Small delay before typing correct character
			time.Sleep(time.Duration(rand.Intn(100)+50) * time.Millisecond)
		}

		// Type the actual character
		err := element.Input(string(char))
		if err != nil {
			return fmt.Errorf("failed to input character: %w", err)
		}

		// Add realistic delay between keystrokes
		if i < len(text)-1 {
			minDelay := sm.config.TypingMinDelay
			maxDelay := sm.config.TypingMaxDelay
			if minDelay == 0 {
				minDelay = 50 * time.Millisecond
			}
			if maxDelay == 0 {
				maxDelay = 200 * time.Millisecond
			}
			
			delay := minDelay + time.Duration(rand.Int63n(int64(maxDelay-minDelay)))
			time.Sleep(delay)
		}
	}

	return nil
}

// RandomDelay implements randomized timing for interactions
func (sm *StealthManager) RandomDelay(min, max time.Duration) error {
	if min > max {
		min, max = max, min
	}
	
	if min == max {
		time.Sleep(min)
		return nil
	}
	
	delay := min + time.Duration(rand.Int63n(int64(max-min)))
	time.Sleep(delay)
	return nil
}

// ConfigureFingerprint implements browser fingerprint configuration
func (sm *StealthManager) ConfigureFingerprint(browser *rod.Browser) error {
	// Get pages to configure
	pages, err := browser.Pages()
	if err != nil {
		return fmt.Errorf("failed to get pages: %w", err)
	}

	// If no pages exist, create one temporarily for configuration
	if len(pages) == 0 {
		page, err := browser.Page(proto.TargetCreateTarget{})
		if err != nil {
			return fmt.Errorf("failed to create page for fingerprint configuration: %w", err)
		}
		pages = []*rod.Page{page}
	}

	// Configure each page
	for _, page := range pages {
		// Set User-Agent
		if sm.fingerprint.UserAgent != "" {
			err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
				UserAgent: sm.fingerprint.UserAgent,
			})
			if err != nil {
				return fmt.Errorf("failed to set user agent: %w", err)
			}
		}

		// Set viewport size
		if sm.fingerprint.ViewportW > 0 && sm.fingerprint.ViewportH > 0 {
			err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
				Width:  sm.fingerprint.ViewportW,
				Height: sm.fingerprint.ViewportH,
			})
			if err != nil {
				return fmt.Errorf("failed to set viewport: %w", err)
			}
		}

		// Mask webdriver property
		if sm.fingerprint.MaskWebDriver {
			_, err := page.Eval(`() => {
				Object.defineProperty(navigator, 'webdriver', {
					get: () => undefined,
				});
				
				// Also mask other automation indicators
				window.chrome = {
					runtime: {},
				};
				
				Object.defineProperty(navigator, 'plugins', {
					get: () => [1, 2, 3, 4, 5],
				});
				
				Object.defineProperty(navigator, 'languages', {
					get: () => ['en-US', 'en'],
				});
			}`)
			if err != nil {
				return fmt.Errorf("failed to mask webdriver property: %w", err)
			}
		}
	}

	return nil
}

// ScrollNaturally implements natural scrolling behavior
func (sm *StealthManager) ScrollNaturally(ctx context.Context, page *rod.Page) error {
	// Random scroll direction and distance
	scrollDown := rand.Float64() < 0.7 // 70% chance to scroll down
	scrollDistance := rand.Intn(300) + 100 // 100-400 pixels

	if !scrollDown {
		scrollDistance = -scrollDistance
	}

	// Perform scroll with multiple small movements for naturalness
	steps := rand.Intn(5) + 3 // 3-7 steps
	stepSize := scrollDistance / steps

	for i := 0; i < steps; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := page.Mouse.Scroll(0, float64(stepSize), steps)
		if err != nil {
			return fmt.Errorf("failed to scroll: %w", err)
		}

		// Small delay between scroll steps
		delay := time.Duration(rand.Intn(50)+20) * time.Millisecond
		time.Sleep(delay)
	}

	return nil
}

// IdleBehavior implements mouse hovering and idle movement simulation
func (sm *StealthManager) IdleBehavior(ctx context.Context, page *rod.Page) error {
	// Perform 2-5 small random movements
	movements := rand.Intn(4) + 2
	
	for i := 0; i < movements; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Generate random position within reasonable viewport bounds
		newX := rand.Float64() * 800 + 100 // 100-900 range
		newY := rand.Float64() * 600 + 100 // 100-700 range

		err := page.Mouse.MoveTo(proto.Point{X: newX, Y: newY})
		if err != nil {
			continue // Skip if movement fails
		}

		// Random pause between movements
		delay := time.Duration(rand.Intn(1000)+500) * time.Millisecond
		time.Sleep(delay)
	}

	return nil
}

// EnforceCooldown implements cooldown period enforcement
func (sm *StealthManager) EnforceCooldown(lastAction time.Time, cooldownPeriod time.Duration) error {
	elapsed := time.Since(lastAction)
	if elapsed < cooldownPeriod {
		remaining := cooldownPeriod - elapsed
		time.Sleep(remaining)
	}
	return nil
}

// IsWithinBusinessHours checks if the given time is within configured business hours
func (sm *StealthManager) IsWithinBusinessHours(t time.Time) bool {
	// If business hours are not enforced, always return true
	if !sm.config.BusinessHours {
		return true
	}

	// Get the hour of the day
	hour := t.Hour()

	// Check if within business hours
	// Handle cases where business hours span midnight
	if sm.config.BusinessStart <= sm.config.BusinessEnd {
		// Normal case: e.g., 9 AM to 5 PM
		return hour >= sm.config.BusinessStart && hour < sm.config.BusinessEnd
	} else {
		// Spans midnight: e.g., 10 PM to 6 AM
		return hour >= sm.config.BusinessStart || hour < sm.config.BusinessEnd
	}
}

// ShouldRateLimit determines if rate limiting should be applied based on action count
func (sm *StealthManager) ShouldRateLimit(actionCount int, timeWindow time.Duration, maxActions int) bool {
	// If no rate limit is configured, don't limit
	if maxActions <= 0 {
		return false
	}

	// If action count exceeds max actions, rate limit should be applied
	return actionCount >= maxActions
}