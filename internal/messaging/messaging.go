package messaging

import (
	"context"
	"time"
)

// MessageSender interface for LinkedIn messaging
type MessageSender interface {
	SendMessage(ctx context.Context, connection AcceptedConnection, template MessageTemplate) error
	DetectAcceptedConnections(ctx context.Context) ([]AcceptedConnection, error)
	TrackMessage(message SentMessage) error
}

// AcceptedConnection represents an accepted connection
type AcceptedConnection struct {
	ProfileURL  string
	ProfileName string
	AcceptedAt  time.Time
}

// MessageTemplate represents a message template
type MessageTemplate struct {
	Name      string
	Subject   string
	Body      string
	Variables map[string]string
}

// SentMessage represents a sent message
type SentMessage struct {
	RecipientURL string
	Template     string
	Content      string
	SentAt       time.Time
	Response     string
}

// MessagingManager implements MessageSender interface
type MessagingManager struct {
	storage     StorageInterface
	rateLimiter RateLimiterInterface
}

// StorageInterface defines storage operations needed by messaging
type StorageInterface interface {
	SaveMessage(message SentMessage) error
	GetMessageHistory() ([]SentMessage, error)
}

// RateLimiterInterface defines rate limiting operations
type RateLimiterInterface interface {
	CanSendMessage() bool
	RecordMessage()
}

// NewMessagingManager creates a new messaging manager
func NewMessagingManager(storage StorageInterface, rateLimiter RateLimiterInterface) *MessagingManager {
	return &MessagingManager{
		storage:     storage,
		rateLimiter: rateLimiter,
	}
}