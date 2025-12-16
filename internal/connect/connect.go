package connect

import (
	"context"
	"time"
	"github.com/go-rod/rod"
)

// ConnectionManager interface for LinkedIn connection requests
type ConnectionManager interface {
	SendConnectionRequest(ctx context.Context, profile ProfileResult, note string) error
	DetectConnectButton(ctx context.Context, page *rod.Page) (*rod.Element, error)
	TrackSentRequest(request ConnectionRequest) error
}

// ProfileResult represents a profile to connect with
type ProfileResult struct {
	URL         string
	Name        string
	Title       string
	Company     string
	Location    string
	Mutual      int
	Premium     bool
	Timestamp   time.Time
}

// ConnectionRequest represents a sent connection request
type ConnectionRequest struct {
	ProfileURL  string
	ProfileName string
	Note        string
	SentAt      time.Time
	Status      string // pending, accepted, declined
}

// ConnectManager implements ConnectionManager interface
type ConnectManager struct {
	storage     StorageInterface
	rateLimiter RateLimiterInterface
}

// StorageInterface defines storage operations needed by connect
type StorageInterface interface {
	SaveConnectionRequest(request ConnectionRequest) error
	GetSentRequests() ([]ConnectionRequest, error)
}

// RateLimiterInterface defines rate limiting operations
type RateLimiterInterface interface {
	CanSendConnection() bool
	RecordConnection()
}

// NewConnectManager creates a new connection manager
func NewConnectManager(storage StorageInterface, rateLimiter RateLimiterInterface) *ConnectManager {
	return &ConnectManager{
		storage:     storage,
		rateLimiter: rateLimiter,
	}
}