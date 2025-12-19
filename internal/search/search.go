package search

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
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

// Validate validates search criteria and applies defaults
func (sc *SearchCriteria) Validate() error {
	if len(sc.Keywords) == 0 && sc.Location == "" && sc.Industry == "" && 
	   sc.Company == "" && sc.Title == "" && sc.Connections == "" {
		return fmt.Errorf("at least one search criterion must be provided")
	}
	
	if sc.MaxResults <= 0 {
		sc.MaxResults = 100 // Default max results
	}
	
	return nil
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

// Search performs LinkedIn profile search with given criteria
func (sm *SearchManager) Search(ctx context.Context, criteria SearchCriteria) ([]ProfileResult, error) {
	if err := criteria.Validate(); err != nil {
		return nil, fmt.Errorf("invalid search criteria: %w", err)
	}

	// This would normally navigate to LinkedIn search page and perform the search
	// For now, we'll return a mock implementation that demonstrates the structure
	results := []ProfileResult{
		{
			URL:       "https://linkedin.com/in/example1",
			Name:      "John Doe",
			Title:     "Software Engineer",
			Company:   "Tech Corp",
			Location:  "San Francisco, CA",
			Mutual:    5,
			Premium:   false,
			Timestamp: time.Now(),
		},
	}

	// Deduplicate results using storage
	deduplicatedResults, err := sm.deduplicateResults(results)
	if err != nil {
		return nil, fmt.Errorf("failed to deduplicate results: %w", err)
	}

	// Save results to storage
	if err := sm.storage.SaveSearchResults(deduplicatedResults); err != nil {
		return nil, fmt.Errorf("failed to save search results: %w", err)
	}

	return deduplicatedResults, nil
}

// ExtractProfiles extracts profile information from a search results page
func (sm *SearchManager) ExtractProfiles(ctx context.Context, page *rod.Page) ([]ProfileResult, error) {
	if page == nil {
		return nil, fmt.Errorf("page cannot be nil")
	}

	var results []ProfileResult
	
	// Wait for search results to load
	err := page.WaitLoad()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Extract profile links - LinkedIn uses various selectors for profile links
	profileSelectors := []string{
		"a[href*='/in/']",
		".search-result__person a",
		".entity-result__title-text a",
		".app-aware-link[href*='/in/']",
	}

	var profileElements []*rod.Element
	for _, selector := range profileSelectors {
		elements, err := page.Elements(selector)
		if err == nil && len(elements) > 0 {
			profileElements = elements
			break
		}
	}

	for _, element := range profileElements {
		profile, err := sm.extractProfileFromElement(element)
		if err != nil {
			continue // Skip invalid profiles
		}
		results = append(results, profile)
	}

	return results, nil
}

// extractProfileFromElement extracts profile data from a DOM element
func (sm *SearchManager) extractProfileFromElement(element *rod.Element) (ProfileResult, error) {
	profile := ProfileResult{
		Timestamp: time.Now(),
	}

	// Extract profile URL
	href, err := element.Attribute("href")
	if err != nil || href == nil {
		return profile, fmt.Errorf("no href attribute found")
	}
	
	profileURL := *href
	if !strings.Contains(profileURL, "/in/") {
		return profile, fmt.Errorf("invalid profile URL: %s", profileURL)
	}
	
	// Clean and validate URL
	if strings.HasPrefix(profileURL, "/") {
		profileURL = "https://linkedin.com" + profileURL
	}
	profile.URL = profileURL

	// Extract name from the link text or nearby elements
	name, err := element.Text()
	if err == nil && strings.TrimSpace(name) != "" {
		profile.Name = strings.TrimSpace(name)
	}

	// Try to extract additional information from parent elements
	parent, err := element.Parent()
	if err == nil && parent != nil {
		// Look for title information
		titleSelectors := []string{
			".entity-result__primary-subtitle",
			".search-result__snippets",
			".subline-level-1",
		}
		
		for _, selector := range titleSelectors {
			titleElement, err := parent.Element(selector)
			if err == nil {
				title, err := titleElement.Text()
				if err == nil && strings.TrimSpace(title) != "" {
					profile.Title = strings.TrimSpace(title)
					break
				}
			}
		}

		// Look for company information
		companySelectors := []string{
			".entity-result__secondary-subtitle",
			".search-result__snippets .t-14",
			".subline-level-2",
		}
		
		for _, selector := range companySelectors {
			companyElement, err := parent.Element(selector)
			if err == nil {
				company, err := companyElement.Text()
				if err == nil && strings.TrimSpace(company) != "" {
					profile.Company = strings.TrimSpace(company)
					break
				}
			}
		}
	}

	return profile, nil
}

// HandlePagination handles automatic pagination through search results
func (sm *SearchManager) HandlePagination(ctx context.Context, page *rod.Page) error {
	if page == nil {
		return fmt.Errorf("page cannot be nil")
	}

	// Look for pagination elements
	paginationSelectors := []string{
		"button[aria-label='Next']",
		".artdeco-pagination__button--next",
		"a[aria-label='Next']",
		".pv-s-profile-actions--next",
	}

	var nextButton *rod.Element
	for _, selector := range paginationSelectors {
		element, err := page.Element(selector)
		if err == nil {
			nextButton = element
			break
		}
	}

	if nextButton == nil {
		return fmt.Errorf("no next button found - end of results")
	}

	// Check if the next button is disabled
	disabled, err := nextButton.Attribute("disabled")
	if err == nil && disabled != nil {
		return fmt.Errorf("next button is disabled - end of results")
	}

	// Check if the button has aria-disabled
	ariaDisabled, err := nextButton.Attribute("aria-disabled")
	if err == nil && ariaDisabled != nil && *ariaDisabled == "true" {
		return fmt.Errorf("next button is aria-disabled - end of results")
	}

	// Click the next button
	nextButton.MustClick()

	// Wait for the new page to load
	err = page.WaitLoad()
	if err != nil {
		return fmt.Errorf("failed to wait for next page load: %w", err)
	}

	return nil
}

// deduplicateResults removes duplicate profiles based on URL
func (sm *SearchManager) deduplicateResults(newResults []ProfileResult) ([]ProfileResult, error) {
	// Get existing results from storage
	existingResults, err := sm.storage.GetSearchResults()
	if err != nil {
		// If we can't get existing results, just deduplicate within new results
		return sm.deduplicateWithinResults(newResults), nil
	}

	// Create a map of existing URLs
	existingURLs := make(map[string]bool)
	for _, result := range existingResults {
		existingURLs[result.URL] = true
	}

	// First deduplicate within new results, then filter against existing
	deduplicatedNew := sm.deduplicateWithinResults(newResults)
	
	// Filter out duplicates from existing results
	var deduplicatedResults []ProfileResult
	for _, result := range deduplicatedNew {
		if !existingURLs[result.URL] {
			deduplicatedResults = append(deduplicatedResults, result)
		}
	}

	return deduplicatedResults, nil
}

// deduplicateWithinResults removes duplicates within a single slice of results
func (sm *SearchManager) deduplicateWithinResults(results []ProfileResult) []ProfileResult {
	seenURLs := make(map[string]bool)
	var deduplicatedResults []ProfileResult
	
	for _, result := range results {
		if !seenURLs[result.URL] {
			seenURLs[result.URL] = true
			deduplicatedResults = append(deduplicatedResults, result)
		}
	}
	
	return deduplicatedResults
}

// IsValidLinkedInProfileURL validates if a URL is a valid LinkedIn profile URL
func IsValidLinkedInProfileURL(profileURL string) bool {
	if profileURL == "" {
		return false
	}

	// Parse the URL
	parsedURL, err := url.Parse(profileURL)
	if err != nil {
		return false
	}

	// Check if it's a LinkedIn domain
	if !strings.Contains(parsedURL.Host, "linkedin.com") {
		return false
	}

	// Check if it contains /in/ path
	if !strings.Contains(parsedURL.Path, "/in/") {
		return false
	}

	// Use regex to validate the profile path format
	profileRegex := regexp.MustCompile(`^/in/[a-zA-Z0-9\-]+/?$`)
	return profileRegex.MatchString(parsedURL.Path)
}

// ExtractMutualConnections extracts mutual connection count from text
func ExtractMutualConnections(text string) int {
	if text == "" {
		return 0
	}

	// Look for patterns like "5 mutual connections", "1 mutual connection"
	mutualRegex := regexp.MustCompile(`(\d+)\s+mutual\s+connections?`)
	matches := mutualRegex.FindStringSubmatch(strings.ToLower(text))
	
	if len(matches) >= 2 {
		count, err := strconv.Atoi(matches[1])
		if err == nil {
			return count
		}
	}

	return 0
}