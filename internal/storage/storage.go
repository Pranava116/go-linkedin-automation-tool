package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
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
	config  StorageConfig
	db      *sql.DB
	jsonMux sync.RWMutex
}

// NewStorageManager creates a new storage manager
func NewStorageManager(config StorageConfig) (*StorageManager, error) {
	sm := &StorageManager{
		config: config,
	}

	// Ensure directory exists
	if err := os.MkdirAll(config.Path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	if config.Type == "sqlite" {
		if err := sm.initSQLite(); err != nil {
			return nil, err
		}
	}

	return sm, nil
}

// initSQLite initializes SQLite database
func (sm *StorageManager) initSQLite() error {
	dbPath := filepath.Join(sm.config.Path, sm.config.Database)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	sm.db = db

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS connection_requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		profile_url TEXT NOT NULL,
		profile_name TEXT,
		note TEXT,
		sent_at DATETIME NOT NULL,
		status TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sent_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		recipient_url TEXT NOT NULL,
		template TEXT,
		content TEXT NOT NULL,
		sent_at DATETIME NOT NULL,
		response TEXT
	);

	CREATE TABLE IF NOT EXISTS search_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL UNIQUE,
		name TEXT,
		title TEXT,
		company TEXT,
		location TEXT,
		mutual INTEGER,
		premium BOOLEAN,
		timestamp DATETIME NOT NULL
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// SaveConnectionRequest saves a connection request
func (sm *StorageManager) SaveConnectionRequest(request ConnectionRequest) error {
	if sm.config.Type == "sqlite" {
		return sm.saveConnectionRequestSQLite(request)
	}
	return sm.saveConnectionRequestJSON(request)
}

func (sm *StorageManager) saveConnectionRequestSQLite(request ConnectionRequest) error {
	query := `INSERT INTO connection_requests (profile_url, profile_name, note, sent_at, status) 
	          VALUES (?, ?, ?, ?, ?)`
	_, err := sm.db.Exec(query, request.ProfileURL, request.ProfileName, request.Note, request.SentAt, request.Status)
	if err != nil {
		return fmt.Errorf("failed to save connection request: %w", err)
	}
	return nil
}

func (sm *StorageManager) saveConnectionRequestJSON(request ConnectionRequest) error {
	sm.jsonMux.Lock()
	defer sm.jsonMux.Unlock()

	requests, err := sm.loadConnectionRequestsJSON()
	if err != nil {
		requests = []ConnectionRequest{}
	}

	requests = append(requests, request)
	return sm.writeConnectionRequestsJSON(requests)
}

// GetSentRequests retrieves all sent connection requests
func (sm *StorageManager) GetSentRequests() ([]ConnectionRequest, error) {
	if sm.config.Type == "sqlite" {
		return sm.getSentRequestsSQLite()
	}
	return sm.loadConnectionRequestsJSON()
}

func (sm *StorageManager) getSentRequestsSQLite() ([]ConnectionRequest, error) {
	query := `SELECT profile_url, profile_name, note, sent_at, status FROM connection_requests ORDER BY sent_at DESC`
	rows, err := sm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query connection requests: %w", err)
	}
	defer rows.Close()

	var requests []ConnectionRequest
	for rows.Next() {
		var req ConnectionRequest
		if err := rows.Scan(&req.ProfileURL, &req.ProfileName, &req.Note, &req.SentAt, &req.Status); err != nil {
			return nil, fmt.Errorf("failed to scan connection request: %w", err)
		}
		requests = append(requests, req)
	}

	return requests, nil
}

func (sm *StorageManager) loadConnectionRequestsJSON() ([]ConnectionRequest, error) {
	filePath := filepath.Join(sm.config.Path, "connection_requests.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ConnectionRequest{}, nil
		}
		return nil, fmt.Errorf("failed to read connection requests: %w", err)
	}

	var requests []ConnectionRequest
	if err := json.Unmarshal(data, &requests); err != nil {
		return nil, fmt.Errorf("failed to unmarshal connection requests: %w", err)
	}

	return requests, nil
}

func (sm *StorageManager) writeConnectionRequestsJSON(requests []ConnectionRequest) error {
	filePath := filepath.Join(sm.config.Path, "connection_requests.json")
	data, err := json.MarshalIndent(requests, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal connection requests: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write connection requests: %w", err)
	}

	return nil
}

// SaveMessage saves a sent message
func (sm *StorageManager) SaveMessage(message SentMessage) error {
	if sm.config.Type == "sqlite" {
		return sm.saveMessageSQLite(message)
	}
	return sm.saveMessageJSON(message)
}

func (sm *StorageManager) saveMessageSQLite(message SentMessage) error {
	query := `INSERT INTO sent_messages (recipient_url, template, content, sent_at, response) 
	          VALUES (?, ?, ?, ?, ?)`
	_, err := sm.db.Exec(query, message.RecipientURL, message.Template, message.Content, message.SentAt, message.Response)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	return nil
}

func (sm *StorageManager) saveMessageJSON(message SentMessage) error {
	sm.jsonMux.Lock()
	defer sm.jsonMux.Unlock()

	messages, err := sm.loadMessagesJSON()
	if err != nil {
		messages = []SentMessage{}
	}

	messages = append(messages, message)
	return sm.writeMessagesJSON(messages)
}

// GetMessageHistory retrieves all sent messages
func (sm *StorageManager) GetMessageHistory() ([]SentMessage, error) {
	if sm.config.Type == "sqlite" {
		return sm.getMessageHistorySQLite()
	}
	return sm.loadMessagesJSON()
}

func (sm *StorageManager) getMessageHistorySQLite() ([]SentMessage, error) {
	query := `SELECT recipient_url, template, content, sent_at, response FROM sent_messages ORDER BY sent_at DESC`
	rows, err := sm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []SentMessage
	for rows.Next() {
		var msg SentMessage
		if err := rows.Scan(&msg.RecipientURL, &msg.Template, &msg.Content, &msg.SentAt, &msg.Response); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (sm *StorageManager) loadMessagesJSON() ([]SentMessage, error) {
	filePath := filepath.Join(sm.config.Path, "sent_messages.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []SentMessage{}, nil
		}
		return nil, fmt.Errorf("failed to read messages: %w", err)
	}

	var messages []SentMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messages, nil
}

func (sm *StorageManager) writeMessagesJSON(messages []SentMessage) error {
	filePath := filepath.Join(sm.config.Path, "sent_messages.json")
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write messages: %w", err)
	}

	return nil
}

// SaveSearchResults saves search results
func (sm *StorageManager) SaveSearchResults(results []ProfileResult) error {
	if sm.config.Type == "sqlite" {
		return sm.saveSearchResultsSQLite(results)
	}
	return sm.saveSearchResultsJSON(results)
}

func (sm *StorageManager) saveSearchResultsSQLite(results []ProfileResult) error {
	tx, err := sm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO search_results 
		(url, name, title, company, location, mutual, premium, timestamp) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, result := range results {
		_, err := stmt.Exec(result.URL, result.Name, result.Title, result.Company,
			result.Location, result.Mutual, result.Premium, result.Timestamp)
		if err != nil {
			return fmt.Errorf("failed to save search result: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (sm *StorageManager) saveSearchResultsJSON(results []ProfileResult) error {
	sm.jsonMux.Lock()
	defer sm.jsonMux.Unlock()

	existing, err := sm.loadSearchResultsJSON()
	if err != nil {
		existing = []ProfileResult{}
	}

	// Deduplicate by URL
	urlMap := make(map[string]ProfileResult)
	for _, r := range existing {
		urlMap[r.URL] = r
	}
	for _, r := range results {
		urlMap[r.URL] = r
	}

	merged := make([]ProfileResult, 0, len(urlMap))
	for _, r := range urlMap {
		merged = append(merged, r)
	}

	return sm.writeSearchResultsJSON(merged)
}

// GetSearchResults retrieves all search results
func (sm *StorageManager) GetSearchResults() ([]ProfileResult, error) {
	if sm.config.Type == "sqlite" {
		return sm.getSearchResultsSQLite()
	}
	return sm.loadSearchResultsJSON()
}

func (sm *StorageManager) getSearchResultsSQLite() ([]ProfileResult, error) {
	query := `SELECT url, name, title, company, location, mutual, premium, timestamp 
	          FROM search_results ORDER BY timestamp DESC`
	rows, err := sm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query search results: %w", err)
	}
	defer rows.Close()

	var results []ProfileResult
	for rows.Next() {
		var result ProfileResult
		if err := rows.Scan(&result.URL, &result.Name, &result.Title, &result.Company,
			&result.Location, &result.Mutual, &result.Premium, &result.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

func (sm *StorageManager) loadSearchResultsJSON() ([]ProfileResult, error) {
	filePath := filepath.Join(sm.config.Path, "search_results.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ProfileResult{}, nil
		}
		return nil, fmt.Errorf("failed to read search results: %w", err)
	}

	var results []ProfileResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search results: %w", err)
	}

	return results, nil
}

func (sm *StorageManager) writeSearchResultsJSON(results []ProfileResult) error {
	filePath := filepath.Join(sm.config.Path, "search_results.json")
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal search results: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write search results: %w", err)
	}

	return nil
}

// Close closes the storage manager
func (sm *StorageManager) Close() error {
	if sm.db != nil {
		return sm.db.Close()
	}
	return nil
}