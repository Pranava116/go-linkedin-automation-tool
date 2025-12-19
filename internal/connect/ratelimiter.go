package connect

import (
	"sync"
	"time"
)

// SimpleRateLimiter implements basic rate limiting for connection requests
type SimpleRateLimiter struct {
	maxConnections int
	timeWindow     time.Duration
	connections    []time.Time
	mutex          sync.Mutex
}

// NewSimpleRateLimiter creates a new rate limiter
func NewSimpleRateLimiter(maxConnections int, timeWindow time.Duration) *SimpleRateLimiter {
	return &SimpleRateLimiter{
		maxConnections: maxConnections,
		timeWindow:     timeWindow,
		connections:    make([]time.Time, 0),
	}
}

// CanSendConnection checks if a connection can be sent based on rate limits
func (rl *SimpleRateLimiter) CanSendConnection() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.timeWindow)

	// Remove old connections outside the time window
	validConnections := make([]time.Time, 0)
	for _, connTime := range rl.connections {
		if connTime.After(cutoff) {
			validConnections = append(validConnections, connTime)
		}
	}
	rl.connections = validConnections

	// Check if we can send another connection
	return len(rl.connections) < rl.maxConnections
}

// RecordConnection records a new connection request
func (rl *SimpleRateLimiter) RecordConnection() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.connections = append(rl.connections, time.Now())
}