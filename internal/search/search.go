package search

import (
	"context"
	"time"
	"github.com/go-rod/rod"
)

// ProfileSearcher interface for LinkedIn profile discovery
type ProfileSearcher interface {
	Search(ctx context.Context, criteria SearchCriteria) ([]ProfileResult, error)
	ExtractProfiles(ctx context.Context, page *rod.Page) ([]ProfileResult, error)
	HandlePagination(ctx context.Context, page *rod.Page) error
}

// SearchCriteria represents search parameters
type SearchCriteria struct {
	Keywords    []string
	Location    string
	Industry    string
	Company     string
	Title       string
	Connections string
	MaxResults  int
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

// SearchManager implements ProfileSearcher interface
type SearchManager struct {
	storage StorageInterface
}

// StorageInterface defines storage operations needed by search
type StorageInterface interface {
	SaveSearchResults(results []ProfileResult) error
	GetSearchResults() ([]ProfileResult, error)
}

// NewSearchManager creates a new search manager
func NewSearchManager(storage StorageInterface) *SearchManager {
	return &SearchManager{
		storage: storage,
	}
}