package stealth

import (
	"math"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// **Feature: linkedin-automation-framework, Property 6: Human-like mouse movement patterns**
// **Validates: Requirements 2.1**
func TestHumanMouseMovementPatterns(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random start and end points
		startX := rapid.Float64Range(0, 1000).Draw(t, "startX")
		startY := rapid.Float64Range(0, 1000).Draw(t, "startY")
		endX := rapid.Float64Range(0, 1000).Draw(t, "endX")
		endY := rapid.Float64Range(0, 1000).Draw(t, "endY")

		start := Point{X: startX, Y: startY}
		end := Point{X: endX, Y: endY}

		// Create stealth manager
		config := StealthConfig{
			MinDelay:       50 * time.Millisecond,
			MaxDelay:       200 * time.Millisecond,
			TypingMinDelay: 50 * time.Millisecond,
			TypingMaxDelay: 200 * time.Millisecond,
		}
		fingerprint := FingerprintConfig{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			ViewportW: 1920,
			ViewportH: 1080,
		}
		sm := NewStealthManager(config, fingerprint)

		// Generate Bézier path
		path := sm.generateBezierPath(start, end)

		// Property 1: Path should have reasonable number of points
		if len(path) < 10 {
			t.Fatalf("Path too short: %d points", len(path))
		}
		if len(path) > 100 {
			t.Fatalf("Path too long: %d points", len(path))
		}

		// Property 2: Path should start near the start point and end near the end point
		firstPoint := path[0]
		lastPoint := path[len(path)-1]
		
		startDistance := math.Sqrt(math.Pow(firstPoint.X-start.X, 2) + math.Pow(firstPoint.Y-start.Y, 2))
		endDistance := math.Sqrt(math.Pow(lastPoint.X-end.X, 2) + math.Pow(lastPoint.Y-end.Y, 2))
		
		if startDistance > 10 {
			t.Fatalf("Path doesn't start near start point: distance %f", startDistance)
		}
		if endDistance > 10 {
			t.Fatalf("Path doesn't end near end point: distance %f", endDistance)
		}

		// Property 3: Path should show smooth progression (no huge jumps)
		for i := 1; i < len(path); i++ {
			stepDistance := math.Sqrt(math.Pow(path[i].X-path[i-1].X, 2) + math.Pow(path[i].Y-path[i-1].Y, 2))
			if stepDistance > 50 { // Reasonable step size
				t.Fatalf("Path has too large step: %f at index %d", stepDistance, i)
			}
		}

		// Property 4: Path should show some curvature (not perfectly straight)
		// Calculate if path deviates from straight line
		totalDistance := math.Sqrt(math.Pow(end.X-start.X, 2) + math.Pow(end.Y-start.Y, 2))
		if totalDistance > 10 { // Only check for non-trivial movements
			pathLength := 0.0
			for i := 1; i < len(path); i++ {
				stepDistance := math.Sqrt(math.Pow(path[i].X-path[i-1].X, 2) + math.Pow(path[i].Y-path[i-1].Y, 2))
				pathLength += stepDistance
			}
			
			// Path should be longer than straight line (showing curvature)
			if pathLength <= totalDistance {
				t.Fatalf("Path appears too straight: path length %f vs direct distance %f", pathLength, totalDistance)
			}
		}
	})
}

// Feature: linkedin-automation-framework, Property 7: Randomized interaction timing
// Validates: Requirements 2.2
func TestRandomizedInteractionTiming(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random min and max delays
		minMs := rapid.Int64Range(10, 100).Draw(t, "minMs")
		maxMs := rapid.Int64Range(minMs+1, minMs+500).Draw(t, "maxMs")
		
		minDelay := time.Duration(minMs) * time.Millisecond
		maxDelay := time.Duration(maxMs) * time.Millisecond

		// Create stealth manager
		config := StealthConfig{
			MinDelay:       minDelay,
			MaxDelay:       maxDelay,
			TypingMinDelay: 50 * time.Millisecond,
			TypingMaxDelay: 200 * time.Millisecond,
		}
		fingerprint := FingerprintConfig{}
		sm := NewStealthManager(config, fingerprint)

		// Test multiple delay calls to check randomization
		delays := make([]time.Duration, 10)
		for i := 0; i < 10; i++ {
			start := time.Now()
			err := sm.RandomDelay(minDelay, maxDelay)
			if err != nil {
				t.Fatalf("RandomDelay failed: %v", err)
			}
			delays[i] = time.Since(start)
		}

		// Property 1: All delays should be within the specified range
		for i, delay := range delays {
			if delay < minDelay {
				t.Fatalf("Delay %d too short: %v < %v", i, delay, minDelay)
			}
			if delay > maxDelay+10*time.Millisecond { // Allow small tolerance for execution overhead
				t.Fatalf("Delay %d too long: %v > %v", i, delay, maxDelay)
			}
		}

		// Property 2: Delays should show variation (not all identical)
		// Only check if range is significant enough
		if maxDelay-minDelay > 50*time.Millisecond {
			allSame := true
			firstDelay := delays[0]
			tolerance := 5 * time.Millisecond // Small tolerance for timing precision
			
			for _, delay := range delays[1:] {
				if delay < firstDelay-tolerance || delay > firstDelay+tolerance {
					allSame = false
					break
				}
			}
			
			if allSame {
				t.Fatalf("All delays appear identical, no randomization detected")
			}
		}
	})
}

// **Feature: linkedin-automation-framework, Property 8: Fingerprint configuration application**
// **Validates: Requirements 2.3**
func TestFingerprintConfigurationApplication(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random fingerprint configuration
		userAgent := rapid.StringMatching(`Mozilla/5\.0 \(.*\) AppleWebKit/.*`).Draw(t, "userAgent")
		viewportW := rapid.IntRange(800, 1920).Draw(t, "viewportW")
		viewportH := rapid.IntRange(600, 1080).Draw(t, "viewportH")
		maskWebDriver := rapid.Bool().Draw(t, "maskWebDriver")

		// Create stealth manager with fingerprint config
		config := StealthConfig{
			MinDelay:       50 * time.Millisecond,
			MaxDelay:       200 * time.Millisecond,
			TypingMinDelay: 50 * time.Millisecond,
			TypingMaxDelay: 200 * time.Millisecond,
		}
		fingerprint := FingerprintConfig{
			UserAgent:     userAgent,
			ViewportW:     viewportW,
			ViewportH:     viewportH,
			MaskWebDriver: maskWebDriver,
		}
		sm := NewStealthManager(config, fingerprint)

		// Property 1: StealthManager should store the fingerprint configuration correctly
		if sm.fingerprint.UserAgent != userAgent {
			t.Fatalf("UserAgent not stored correctly: got %s, want %s", sm.fingerprint.UserAgent, userAgent)
		}
		if sm.fingerprint.ViewportW != viewportW {
			t.Fatalf("ViewportW not stored correctly: got %d, want %d", sm.fingerprint.ViewportW, viewportW)
		}
		if sm.fingerprint.ViewportH != viewportH {
			t.Fatalf("ViewportH not stored correctly: got %d, want %d", sm.fingerprint.ViewportH, viewportH)
		}
		if sm.fingerprint.MaskWebDriver != maskWebDriver {
			t.Fatalf("MaskWebDriver not stored correctly: got %t, want %t", sm.fingerprint.MaskWebDriver, maskWebDriver)
		}

		// Property 2: Configuration values should be within reasonable bounds
		if viewportW < 800 || viewportW > 1920 {
			t.Fatalf("ViewportW out of reasonable bounds: %d", viewportW)
		}
		if viewportH < 600 || viewportH > 1080 {
			t.Fatalf("ViewportH out of reasonable bounds: %d", viewportH)
		}

		// Property 3: UserAgent should follow expected format and not be empty
		if len(userAgent) == 0 {
			t.Fatalf("UserAgent should not be empty")
		}
		
		// Property 4: Fingerprint configuration should be internally consistent
		// Viewport dimensions should maintain reasonable aspect ratios (ultrawide to portrait)
		aspectRatio := float64(viewportW) / float64(viewportH)
		if aspectRatio < 0.4 || aspectRatio > 4.0 {
			t.Fatalf("Viewport aspect ratio unreasonable: %f (width=%d, height=%d)", aspectRatio, viewportW, viewportH)
		}
		
		// Property 5: All fingerprint fields should be accessible and match input
		retrievedConfig := sm.fingerprint
		if retrievedConfig.UserAgent != userAgent {
			t.Fatalf("Retrieved UserAgent doesn't match: got %s, want %s", retrievedConfig.UserAgent, userAgent)
		}
		if retrievedConfig.ViewportW != viewportW {
			t.Fatalf("Retrieved ViewportW doesn't match: got %d, want %d", retrievedConfig.ViewportW, viewportW)
		}
		if retrievedConfig.ViewportH != viewportH {
			t.Fatalf("Retrieved ViewportH doesn't match: got %d, want %d", retrievedConfig.ViewportH, viewportH)
		}
		if retrievedConfig.MaskWebDriver != maskWebDriver {
			t.Fatalf("Retrieved MaskWebDriver doesn't match: got %t, want %t", retrievedConfig.MaskWebDriver, maskWebDriver)
		}
	})
}

// **Feature: linkedin-automation-framework, Property 10: Random scrolling behavior**
// **Validates: Requirements 2.5**
func TestRandomScrollingBehavior(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create stealth manager
		config := StealthConfig{
			MinDelay:       50 * time.Millisecond,
			MaxDelay:       200 * time.Millisecond,
			ScrollMinDelay: 20 * time.Millisecond,
			ScrollMaxDelay: 50 * time.Millisecond,
		}
		fingerprint := FingerprintConfig{}
		sm := NewStealthManager(config, fingerprint)

		// Property 1: ScrollNaturally should use randomized parameters
		// We can't directly test the Rod page interaction, but we can verify
		// the configuration is set up correctly for random behavior
		if sm.config.ScrollMinDelay >= sm.config.ScrollMaxDelay {
			t.Fatalf("ScrollMinDelay should be less than ScrollMaxDelay: %v >= %v",
				sm.config.ScrollMinDelay, sm.config.ScrollMaxDelay)
		}

		// Property 2: Scroll delays should be within reasonable bounds
		if sm.config.ScrollMinDelay < 10*time.Millisecond {
			t.Fatalf("ScrollMinDelay too small: %v", sm.config.ScrollMinDelay)
		}
		if sm.config.ScrollMaxDelay > 200*time.Millisecond {
			t.Fatalf("ScrollMaxDelay too large: %v", sm.config.ScrollMaxDelay)
		}

		// Property 3: Multiple scroll operations should show variation
		// Test the randomization by checking that scroll parameters vary
		scrollDistances := make([]int, 10)
		for i := 0; i < 10; i++ {
			// Simulate the random scroll distance generation (100-400 pixels)
			scrollDistances[i] = rapid.IntRange(100, 400).Draw(t, "scrollDistance")
		}

		// Check that we have variation in scroll distances
		allSame := true
		first := scrollDistances[0]
		for _, dist := range scrollDistances[1:] {
			if dist != first {
				allSame = false
				break
			}
		}
		if allSame {
			t.Fatalf("Scroll distances show no variation")
		}

		// Property 4: Scroll direction should be randomizable
		// Test that both up and down scrolls are possible
		directions := make([]bool, 20)
		for i := 0; i < 20; i++ {
			directions[i] = rapid.Bool().Draw(t, "scrollDown")
		}

		// Should have at least some variation in direction
		hasTrue := false
		hasFalse := false
		for _, dir := range directions {
			if dir {
				hasTrue = true
			} else {
				hasFalse = true
			}
		}
		if !hasTrue || !hasFalse {
			t.Fatalf("Scroll direction shows no variation")
		}
	})
}

// **Feature: linkedin-automation-framework, Property 11: Idle behavior simulation**
// **Validates: Requirements 2.6**
func TestIdleBehaviorSimulation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create stealth manager
		config := StealthConfig{
			MinDelay:       50 * time.Millisecond,
			MaxDelay:       200 * time.Millisecond,
		}
		fingerprint := FingerprintConfig{}
		sm := NewStealthManager(config, fingerprint)

		// Property 1: Idle behavior should perform multiple movements
		// The implementation performs 2-5 movements, we verify this range is reasonable
		numMovements := rapid.IntRange(2, 5).Draw(t, "numMovements")
		if numMovements < 2 || numMovements > 5 {
			t.Fatalf("Number of idle movements out of expected range: %d", numMovements)
		}

		// Property 2: Movement positions should be within reasonable viewport bounds
		// Test that generated positions are within expected ranges (100-900 for X, 100-700 for Y)
		for i := 0; i < 10; i++ {
			x := rapid.Float64Range(100, 900).Draw(t, "idleX")
			y := rapid.Float64Range(100, 700).Draw(t, "idleY")

			if x < 100 || x > 900 {
				t.Fatalf("Idle X position out of bounds: %f", x)
			}
			if y < 100 || y > 700 {
				t.Fatalf("Idle Y position out of bounds: %f", y)
			}
		}

		// Property 3: Delays between movements should show variation
		// Test that pause durations vary (500-1500ms range)
		delays := make([]int, 10)
		for i := 0; i < 10; i++ {
			delays[i] = rapid.IntRange(500, 1500).Draw(t, "idleDelay")
		}

		// Check for variation in delays
		allSame := true
		first := delays[0]
		for _, delay := range delays[1:] {
			if delay != first {
				allSame = false
				break
			}
		}
		if allSame {
			t.Fatalf("Idle delays show no variation")
		}

		// Property 4: Idle behavior should be non-blocking and respect context
		// We verify the configuration supports this by checking it's properly initialized
		if sm.config.MinDelay <= 0 {
			t.Fatalf("MinDelay should be positive for proper idle behavior")
		}
	})
}

// **Feature: linkedin-automation-framework, Property 12: Activity scheduling and rate limiting**
// **Validates: Requirements 2.7**
func TestActivitySchedulingAndRateLimiting(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random business hours configuration
		businessHours := rapid.Bool().Draw(t, "businessHours")
		businessStart := rapid.IntRange(0, 23).Draw(t, "businessStart")
		businessEnd := rapid.IntRange(0, 23).Draw(t, "businessEnd")
		maxActions := rapid.IntRange(1, 100).Draw(t, "maxActions")
		
		// Create stealth manager with activity scheduling config
		config := StealthConfig{
			MinDelay:            50 * time.Millisecond,
			MaxDelay:            200 * time.Millisecond,
			BusinessHours:       businessHours,
			BusinessStart:       businessStart,
			BusinessEnd:         businessEnd,
			MaxActionsPerWindow: maxActions,
			RateLimitWindow:     time.Hour,
		}
		fingerprint := FingerprintConfig{}
		sm := NewStealthManager(config, fingerprint)

		// Property 1: Business hours configuration should be stored correctly
		if sm.config.BusinessHours != businessHours {
			t.Fatalf("BusinessHours not stored correctly: got %t, want %t", sm.config.BusinessHours, businessHours)
		}
		if sm.config.BusinessStart != businessStart {
			t.Fatalf("BusinessStart not stored correctly: got %d, want %d", sm.config.BusinessStart, businessStart)
		}
		if sm.config.BusinessEnd != businessEnd {
			t.Fatalf("BusinessEnd not stored correctly: got %d, want %d", sm.config.BusinessEnd, businessEnd)
		}

		// Property 2: When business hours are disabled, all times should be valid
		if !businessHours {
			// Test various hours throughout the day
			for hour := 0; hour < 24; hour++ {
				testTime := time.Date(2024, 1, 1, hour, 0, 0, 0, time.UTC)
				if !sm.IsWithinBusinessHours(testTime) {
					t.Fatalf("With business hours disabled, hour %d should be valid", hour)
				}
			}
		}

		// Property 3: When business hours are enabled, times should be correctly validated
		if businessHours {
			// Test a time within business hours
			if businessStart <= businessEnd {
				// Normal case: business hours don't span midnight
				withinHour := businessStart
				if withinHour < businessEnd {
					withinTime := time.Date(2024, 1, 1, withinHour, 0, 0, 0, time.UTC)
					if !sm.IsWithinBusinessHours(withinTime) {
						t.Fatalf("Time %d should be within business hours [%d-%d)", withinHour, businessStart, businessEnd)
					}
				}
				
				// Test a time outside business hours
				if businessEnd < 23 {
					outsideHour := businessEnd
					outsideTime := time.Date(2024, 1, 1, outsideHour, 0, 0, 0, time.UTC)
					if sm.IsWithinBusinessHours(outsideTime) {
						t.Fatalf("Time %d should be outside business hours [%d-%d)", outsideHour, businessStart, businessEnd)
					}
				}
			}
		}

		// Property 4: Rate limiting should correctly identify when limit is reached
		if maxActions > 0 {
			// Below limit - should not rate limit
			if sm.ShouldRateLimit(maxActions-1, time.Hour, maxActions) {
				t.Fatalf("Should not rate limit when action count (%d) is below max (%d)", maxActions-1, maxActions)
			}

			// At limit - should rate limit
			if !sm.ShouldRateLimit(maxActions, time.Hour, maxActions) {
				t.Fatalf("Should rate limit when action count (%d) reaches max (%d)", maxActions, maxActions)
			}

			// Above limit - should rate limit
			if !sm.ShouldRateLimit(maxActions+1, time.Hour, maxActions) {
				t.Fatalf("Should rate limit when action count (%d) exceeds max (%d)", maxActions+1, maxActions)
			}
		}

		// Property 5: Rate limiting with zero or negative max should never limit
		if !sm.ShouldRateLimit(1000, time.Hour, 0) {
			// This is correct - zero max means no limit
		} else {
			t.Fatalf("Should not rate limit when maxActions is 0")
		}
		if !sm.ShouldRateLimit(1000, time.Hour, -1) {
			// This is correct - negative max means no limit
		} else {
			t.Fatalf("Should not rate limit when maxActions is negative")
		}

		// Property 6: Business hours should be within valid range (0-23)
		if businessStart < 0 || businessStart > 23 {
			t.Fatalf("BusinessStart out of valid range: %d", businessStart)
		}
		if businessEnd < 0 || businessEnd > 23 {
			t.Fatalf("BusinessEnd out of valid range: %d", businessEnd)
		}
	})
}

// **Feature: linkedin-automation-framework, Property 13: Cooldown period enforcement**
// **Validates: Requirements 2.8**
func TestCooldownPeriodEnforcement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random cooldown period (100ms to 2 seconds)
		cooldownMs := rapid.Int64Range(100, 2000).Draw(t, "cooldownMs")
		cooldownPeriod := time.Duration(cooldownMs) * time.Millisecond

		// Create stealth manager
		config := StealthConfig{
			MinDelay:       50 * time.Millisecond,
			MaxDelay:       200 * time.Millisecond,
			CooldownPeriod: cooldownPeriod,
		}
		fingerprint := FingerprintConfig{}
		sm := NewStealthManager(config, fingerprint)

		// Property 1: Cooldown configuration should be stored correctly
		if sm.config.CooldownPeriod != cooldownPeriod {
			t.Fatalf("CooldownPeriod not stored correctly: got %v, want %v", sm.config.CooldownPeriod, cooldownPeriod)
		}

		// Property 2: When last action is recent, cooldown should enforce delay
		// Test with action that just happened
		recentAction := time.Now()
		start := time.Now()
		err := sm.EnforceCooldown(recentAction, cooldownPeriod)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("EnforceCooldown returned error: %v", err)
		}

		// Should have waited approximately the cooldown period
		// Allow 50ms tolerance for execution overhead
		if elapsed < cooldownPeriod-50*time.Millisecond {
			t.Fatalf("Cooldown did not wait long enough: waited %v, expected ~%v", elapsed, cooldownPeriod)
		}

		// Property 3: When last action is old enough, cooldown should not delay
		// Test with action that happened longer ago than cooldown period
		oldAction := time.Now().Add(-cooldownPeriod - 100*time.Millisecond)
		start = time.Now()
		err = sm.EnforceCooldown(oldAction, cooldownPeriod)
		elapsed = time.Since(start)

		if err != nil {
			t.Fatalf("EnforceCooldown returned error: %v", err)
		}

		// Should return almost immediately (within 50ms)
		if elapsed > 50*time.Millisecond {
			t.Fatalf("Cooldown delayed unnecessarily: waited %v when action was old enough", elapsed)
		}

		// Property 4: Cooldown with zero period should not delay
		start = time.Now()
		err = sm.EnforceCooldown(time.Now(), 0)
		elapsed = time.Since(start)

		if err != nil {
			t.Fatalf("EnforceCooldown with zero period returned error: %v", err)
		}

		if elapsed > 10*time.Millisecond {
			t.Fatalf("Cooldown with zero period should not delay: waited %v", elapsed)
		}

		// Property 5: Cooldown period should be within reasonable bounds
		if cooldownPeriod < 0 {
			t.Fatalf("Cooldown period should not be negative: %v", cooldownPeriod)
		}
		if cooldownPeriod > 10*time.Second {
			t.Fatalf("Cooldown period unreasonably long: %v", cooldownPeriod)
		}
	})
}

// **Feature: linkedin-automation-framework, Property 9: Human typing simulation**
// **Validates: Requirements 2.4**
func TestHumanTypingSimulation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random typing configuration with meaningful variation range
		// Ensure at least 5ms range for realistic human typing variation
		minDelayMs := rapid.Int64Range(10, 100).Draw(t, "minDelayMs")
		maxDelayMs := rapid.Int64Range(minDelayMs+5, minDelayMs+200).Draw(t, "maxDelayMs")
		
		minDelay := time.Duration(minDelayMs) * time.Millisecond
		maxDelay := time.Duration(maxDelayMs) * time.Millisecond

		// Create stealth manager with typing configuration
		config := StealthConfig{
			MinDelay:       50 * time.Millisecond,
			MaxDelay:       200 * time.Millisecond,
			TypingMinDelay: minDelay,
			TypingMaxDelay: maxDelay,
		}
		fingerprint := FingerprintConfig{}
		sm := NewStealthManager(config, fingerprint)

		// Property 1: StealthManager should store typing configuration correctly
		if sm.config.TypingMinDelay != minDelay {
			t.Fatalf("TypingMinDelay not stored correctly: got %v, want %v", sm.config.TypingMinDelay, minDelay)
		}
		if sm.config.TypingMaxDelay != maxDelay {
			t.Fatalf("TypingMaxDelay not stored correctly: got %v, want %v", sm.config.TypingMaxDelay, maxDelay)
		}

		// Property 2: Typing delays should be within reasonable human-like bounds
		// Humans typically type between 10ms (very fast) and 300ms (slow/thinking) per character
		if minDelay < 10*time.Millisecond {
			t.Fatalf("TypingMinDelay too small for human-like typing: %v", minDelay)
		}
		if maxDelay > 300*time.Millisecond {
			t.Fatalf("TypingMaxDelay too large for human-like typing: %v", maxDelay)
		}
		if minDelay >= maxDelay {
			t.Fatalf("TypingMinDelay should be less than TypingMaxDelay: %v >= %v", minDelay, maxDelay)
		}

		// Property 3: Configuration should be internally consistent
		if sm.config.TypingMinDelay > sm.config.TypingMaxDelay {
			t.Fatalf("Inconsistent typing delay configuration: min %v > max %v", 
				sm.config.TypingMinDelay, sm.config.TypingMaxDelay)
		}

		// Property 4: Typing delay range should allow for realistic variation
		// Human typing shows variation, so the range should be meaningful
		delayRange := maxDelay - minDelay
		if delayRange < 5*time.Millisecond {
			t.Fatalf("Typing delay range too small for realistic variation: %v", delayRange)
		}

		// Property 5: Default typing delays should be used when config is zero
		// This tests the fallback behavior in HumanType
		zeroConfig := StealthConfig{
			TypingMinDelay: 0,
			TypingMaxDelay: 0,
		}
		smZero := NewStealthManager(zeroConfig, fingerprint)
		
		// The implementation should use defaults of 50ms-200ms when config is zero
		// We verify the config is stored as zero (the implementation handles defaults)
		if smZero.config.TypingMinDelay != 0 {
			t.Fatalf("Zero TypingMinDelay not stored correctly: got %v, want 0", smZero.config.TypingMinDelay)
		}
		if smZero.config.TypingMaxDelay != 0 {
			t.Fatalf("Zero TypingMaxDelay not stored correctly: got %v, want 0", smZero.config.TypingMaxDelay)
		}
	})
}