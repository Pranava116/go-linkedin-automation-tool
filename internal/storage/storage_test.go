package storage

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// **Feature: linkedin-automation-framework, Property 35: Connection request tracking**
// **Validates: Requirements 7.1**
func TestConnectionRequestTracking(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random connection request
		request := ConnectionRequest{
			ProfileURL:  rapid.String().Draw(rt, "profile_url"),
			ProfileName: rapid.String().Draw(rt, "profile_name"),
			Note:        rapid.String().Draw(rt, "note"),
			SentAt:      time.Now(),
			Status:      rapid.SampledFrom([]string{"pending", "accepted", "declined"}).Draw(rt, "status"),
		}

		// Test both SQLite and JSON storage
		storageTypes := []string{"sqlite", "json"}
		for _, storageType := range storageTypes {
			// Create temporary storage
			tempDir := t.TempDir()
			config := StorageConfig{
				Type:     storageType,
				Path:     tempDir,
				Database: "test.db",
			}

			storage, err := NewStorageManager(config)
			if err != nil {
				rt.Fatalf("failed to create storage: %v", err)
			}
			defer storage.Close()

			// Save connection request
			if err := storage.SaveConnectionRequest(request); err != nil {
				rt.Fatalf("failed to save connection request: %v", err)
			}

			// Retrieve and verify
			requests, err := storage.GetSentRequests()
			if err != nil {
				rt.Fatalf("failed to get sent requests: %v", err)
			}

			// Verify the request was stored with accurate timestamps and metadata
			found := false
			for _, r := range requests {
				if r.ProfileURL == request.ProfileURL &&
					r.ProfileName == request.ProfileName &&
					r.Note == request.Note &&
					r.Status == request.Status {
					found = true
					// Verify timestamp is preserved (within 1 second tolerance for SQLite)
					if request.SentAt.Sub(r.SentAt).Abs() > time.Second {
						rt.Fatalf("timestamp not preserved: expected %v, got %v", request.SentAt, r.SentAt)
					}
					break
				}
			}

			if !found {
				rt.Fatalf("connection request not found in storage")
			}
		}
	})
}

// **Feature: linkedin-automation-framework, Property 36: Accepted connection recording**
// **Validates: Requirements 7.2**
func TestAcceptedConnectionRecording(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random accepted connection request
		request := ConnectionRequest{
			ProfileURL:  rapid.String().Draw(rt, "profile_url"),
			ProfileName: rapid.String().Draw(rt, "profile_name"),
			Note:        rapid.String().Draw(rt, "note"),
			SentAt:      time.Now(),
			Status:      "accepted", // Specifically test accepted status
		}

		// Test both SQLite and JSON storage
		storageTypes := []string{"sqlite", "json"}
		for _, storageType := range storageTypes {
			tempDir := t.TempDir()
			config := StorageConfig{
				Type:     storageType,
				Path:     tempDir,
				Database: "test.db",
			}

			storage, err := NewStorageManager(config)
			if err != nil {
				rt.Fatalf("failed to create storage: %v", err)
			}
			defer storage.Close()

			// Save accepted connection
			if err := storage.SaveConnectionRequest(request); err != nil {
				rt.Fatalf("failed to save accepted connection: %v", err)
			}

			// Retrieve and verify
			requests, err := storage.GetSentRequests()
			if err != nil {
				rt.Fatalf("failed to get sent requests: %v", err)
			}

			// Verify the accepted connection was properly recorded
			found := false
			for _, r := range requests {
				if r.ProfileURL == request.ProfileURL && r.Status == "accepted" {
					found = true
					if r.ProfileName != request.ProfileName || r.Note != request.Note {
						rt.Fatalf("accepted connection data not properly recorded")
					}
					break
				}
			}

			if !found {
				rt.Fatalf("accepted connection not found in storage")
			}
		}
	})
}

// **Feature: linkedin-automation-framework, Property 37: Message history storage**
// **Validates: Requirements 7.3**
func TestMessageHistoryStorage(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random message
		message := SentMessage{
			RecipientURL: rapid.String().Draw(rt, "recipient_url"),
			Template:     rapid.String().Draw(rt, "template"),
			Content:      rapid.String().Draw(rt, "content"),
			SentAt:       time.Now(),
			Response:     rapid.String().Draw(rt, "response"),
		}

		// Test both SQLite and JSON storage
		storageTypes := []string{"sqlite", "json"}
		for _, storageType := range storageTypes {
			tempDir := t.TempDir()
			config := StorageConfig{
				Type:     storageType,
				Path:     tempDir,
				Database: "test.db",
			}

			storage, err := NewStorageManager(config)
			if err != nil {
				rt.Fatalf("failed to create storage: %v", err)
			}
			defer storage.Close()

			// Save message
			if err := storage.SaveMessage(message); err != nil {
				rt.Fatalf("failed to save message: %v", err)
			}

			// Retrieve and verify
			messages, err := storage.GetMessageHistory()
			if err != nil {
				rt.Fatalf("failed to get message history: %v", err)
			}

			// Verify message was stored with complete recipient information and metadata
			found := false
			for _, m := range messages {
				if m.RecipientURL == message.RecipientURL &&
					m.Template == message.Template &&
					m.Content == message.Content &&
					m.Response == message.Response {
					found = true
					// Verify timestamp is preserved
					if message.SentAt.Sub(m.SentAt).Abs() > time.Second {
						rt.Fatalf("timestamp not preserved: expected %v, got %v", message.SentAt, m.SentAt)
					}
					break
				}
			}

			if !found {
				rt.Fatalf("message not found in storage")
			}
		}
	})
}

// **Feature: linkedin-automation-framework, Property 38: Crash recovery capability**
// **Validates: Requirements 7.4**
func TestCrashRecoveryCapability(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random data
		request := ConnectionRequest{
			ProfileURL:  rapid.String().Draw(rt, "profile_url"),
			ProfileName: rapid.String().Draw(rt, "profile_name"),
			Note:        rapid.String().Draw(rt, "note"),
			SentAt:      time.Now(),
			Status:      rapid.SampledFrom([]string{"pending", "accepted", "declined"}).Draw(rt, "status"),
		}

		message := SentMessage{
			RecipientURL: rapid.String().Draw(rt, "recipient_url"),
			Template:     rapid.String().Draw(rt, "template"),
			Content:      rapid.String().Draw(rt, "content"),
			SentAt:       time.Now(),
			Response:     rapid.String().Draw(rt, "response"),
		}

		// Test both SQLite and JSON storage
		storageTypes := []string{"sqlite", "json"}
		for _, storageType := range storageTypes {
			tempDir := t.TempDir()
			config := StorageConfig{
				Type:     storageType,
				Path:     tempDir,
				Database: "test.db",
			}

			// First storage instance - save data
			storage1, err := NewStorageManager(config)
			if err != nil {
				rt.Fatalf("failed to create storage: %v", err)
			}

			if err := storage1.SaveConnectionRequest(request); err != nil {
				rt.Fatalf("failed to save connection request: %v", err)
			}

			if err := storage1.SaveMessage(message); err != nil {
				rt.Fatalf("failed to save message: %v", err)
			}

			// Simulate crash by closing storage
			storage1.Close()

			// Second storage instance - resume after crash
			storage2, err := NewStorageManager(config)
			if err != nil {
				rt.Fatalf("failed to create storage after crash: %v", err)
			}
			defer storage2.Close()

			// Verify data is still accessible (crash recovery)
			requests, err := storage2.GetSentRequests()
			if err != nil {
				rt.Fatalf("failed to get sent requests after crash: %v", err)
			}

			messages, err := storage2.GetMessageHistory()
			if err != nil {
				rt.Fatalf("failed to get message history after crash: %v", err)
			}

			// Verify the data survived the crash
			requestFound := false
			for _, r := range requests {
				if r.ProfileURL == request.ProfileURL {
					requestFound = true
					break
				}
			}

			messageFound := false
			for _, m := range messages {
				if m.RecipientURL == message.RecipientURL {
					messageFound = true
					break
				}
			}

			if !requestFound {
				rt.Fatalf("connection request not recovered after crash")
			}

			if !messageFound {
				rt.Fatalf("message not recovered after crash")
			}
		}
	})
}

// **Feature: linkedin-automation-framework, Property 39: Storage format round-trip**
// **Validates: Requirements 7.5**
func TestStorageFormatRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random data structures
		request := ConnectionRequest{
			ProfileURL:  rapid.String().Draw(rt, "profile_url"),
			ProfileName: rapid.String().Draw(rt, "profile_name"),
			Note:        rapid.String().Draw(rt, "note"),
			SentAt:      time.Now().Truncate(time.Second), // Truncate for comparison
			Status:      rapid.SampledFrom([]string{"pending", "accepted", "declined"}).Draw(rt, "status"),
		}

		message := SentMessage{
			RecipientURL: rapid.String().Draw(rt, "recipient_url"),
			Template:     rapid.String().Draw(rt, "template"),
			Content:      rapid.String().Draw(rt, "content"),
			SentAt:       time.Now().Truncate(time.Second),
			Response:     rapid.String().Draw(rt, "response"),
		}

		result := ProfileResult{
			URL:       rapid.String().Draw(rt, "url"),
			Name:      rapid.String().Draw(rt, "name"),
			Title:     rapid.String().Draw(rt, "title"),
			Company:   rapid.String().Draw(rt, "company"),
			Location:  rapid.String().Draw(rt, "location"),
			Mutual:    rapid.IntRange(0, 500).Draw(rt, "mutual"),
			Premium:   rapid.Bool().Draw(rt, "premium"),
			Timestamp: time.Now().Truncate(time.Second),
		}

		// Test both SQLite and JSON storage
		storageTypes := []string{"sqlite", "json"}
		for _, storageType := range storageTypes {
			tempDir := t.TempDir()
			config := StorageConfig{
				Type:     storageType,
				Path:     tempDir,
				Database: "test.db",
			}

			storage, err := NewStorageManager(config)
			if err != nil {
				rt.Fatalf("failed to create storage: %v", err)
			}
			defer storage.Close()

			// Save data
			if err := storage.SaveConnectionRequest(request); err != nil {
				rt.Fatalf("failed to save connection request: %v", err)
			}

			if err := storage.SaveMessage(message); err != nil {
				rt.Fatalf("failed to save message: %v", err)
			}

			if err := storage.SaveSearchResults([]ProfileResult{result}); err != nil {
				rt.Fatalf("failed to save search results: %v", err)
			}

			// Retrieve data
			requests, err := storage.GetSentRequests()
			if err != nil {
				rt.Fatalf("failed to get sent requests: %v", err)
			}

			messages, err := storage.GetMessageHistory()
			if err != nil {
				rt.Fatalf("failed to get message history: %v", err)
			}

			results, err := storage.GetSearchResults()
			if err != nil {
				rt.Fatalf("failed to get search results: %v", err)
			}

			// Verify round-trip produces equivalent objects
			requestFound := false
			for _, r := range requests {
				if r.ProfileURL == request.ProfileURL &&
					r.ProfileName == request.ProfileName &&
					r.Note == request.Note &&
					r.Status == request.Status &&
					r.SentAt.Equal(request.SentAt) {
					requestFound = true
					break
				}
			}

			messageFound := false
			for _, m := range messages {
				if m.RecipientURL == message.RecipientURL &&
					m.Template == message.Template &&
					m.Content == message.Content &&
					m.Response == message.Response &&
					m.SentAt.Equal(message.SentAt) {
					messageFound = true
					break
				}
			}

			resultFound := false
			for _, res := range results {
				if res.URL == result.URL &&
					res.Name == result.Name &&
					res.Title == result.Title &&
					res.Company == result.Company &&
					res.Location == result.Location &&
					res.Mutual == result.Mutual &&
					res.Premium == result.Premium &&
					res.Timestamp.Equal(result.Timestamp) {
					resultFound = true
					break
				}
			}

			if !requestFound {
				rt.Fatalf("connection request round-trip failed")
			}

			if !messageFound {
				rt.Fatalf("message round-trip failed")
			}

			if !resultFound {
				rt.Fatalf("search result round-trip failed")
			}
		}
	})
}
