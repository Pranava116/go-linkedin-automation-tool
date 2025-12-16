package storage

import (
	"time"
)

// Storage interface for persistent data management
type Storage interface {
	SaveConnectionRequest(request ConnectionRequest) error
	GetSentRequests() ([]ConnectionRequest, error)
	SaveMessage(message SentMessage) error
	GetMessageHistory() ([]SentMessage, error)
	SaveSearchResults(results []ProfileResult) error
	GetSearchResults() ([]ProfileResult, error)
	Close() error
}

// ConnectionRequest represents a sent connection request
type ConnectionRequest struct {
	ProfileURL  string
	ProfileName string
	Note        string
	SentAt      time.Time
	Status      string // pending, accepted, declined
}

// SentMessage represents a sent message
type SentMessage struct {
	RecipientURL string
	Template     string
	Content      string
	SentAt       time.Time
	Response     string
}

// ProfileResult represents a discovered profile
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

// StorageConfig contains storage configuration
type StorageConfig struct {
	Type     string // "sqlite" or "json"
	Path     string
	Database string
}

// StorageManager implements Storage interface
type StorageManager struct {
	config StorageConfig
}

// NewStorageManager creates a new storage manager
func NewStorageManager(config StorageConfig) *StorageManager {
	return &StorageManager{
		config: config,
	}
}