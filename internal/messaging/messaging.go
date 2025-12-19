package messaging

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

// MessageSender interface for LinkedIn messaging functionality
type MessageSender interface {
	SendMessage(ctx context.Context, page *rod.Page, connection AcceptedConnection, template MessageTemplate) error
	DetectAcceptedConnections(ctx context.Context, page *rod.Page) ([]AcceptedConnection, error)
	TrackMessage(message SentMessage) error
	SubstituteVariables(template MessageTemplate, variables map[string]string) (string, error)
	NavigateToMessaging(ctx context.Context, page *rod.Page) error
	FindConversation(ctx context.Context, page *rod.Page, connectionName string) (*rod.Element, error)
}

// AcceptedConnection represents a newly accepted LinkedIn connection
type AcceptedConnection struct {
	ProfileURL  string
	Name        string
	Title       string
	Company     string
	AcceptedAt  time.Time
	MessageSent bool
}

// MessageTemplate represents a message template with variables
type MessageTemplate struct {
	Name        string
	Subject     string
	Body        string
	Variables   map[string]string
}

// SentMessage represents a sent message record
type SentMessage struct {
	RecipientURL string
	RecipientName string
	Template     string
	Content      string
	SentAt       time.Time
	Response     string
}

// MessagingManager implements MessageSender interface
type MessagingManager struct {
	storage     StorageInterface
	rateLimiter RateLimiterInterface
	stealth     StealthInterface
}

// StorageInterface defines storage operations needed by messaging
type StorageInterface interface {
	SaveMessage(message SentMessage) error
	GetMessageHistory() ([]SentMessage, error)
	GetSentRequests() ([]ConnectionRequest, error)
}

// ConnectionRequest represents a connection request (from storage)
type ConnectionRequest struct {
	ProfileURL  string
	ProfileName string
	Note        string
	SentAt      time.Time
	Status      string // pending, accepted, declined
}

// RateLimiterInterface defines rate limiting operations for messaging
type RateLimiterInterface interface {
	CanSendMessage() bool
	RecordMessage()
	GetLastMessageTime() time.Time
	GetMessageCount(window time.Duration) int
}

// StealthInterface defines stealth operations needed by messaging
type StealthInterface interface {
	HumanMouseMove(ctx context.Context, page *rod.Page, target *rod.Element) error
	HumanType(ctx context.Context, element *rod.Element, text string) error
	RandomDelay(min, max time.Duration) error
}

// NewMessagingManager creates a new messaging manager
func NewMessagingManager(storage StorageInterface, rateLimiter RateLimiterInterface, stealth StealthInterface) *MessagingManager {
	return &MessagingManager{
		storage:     storage,
		rateLimiter: rateLimiter,
		stealth:     stealth,
	}
}

// DetectAcceptedConnections detects newly accepted connections
func (mm *MessagingManager) DetectAcceptedConnections(ctx context.Context, page *rod.Page) ([]AcceptedConnection, error) {
	if page == nil {
		return nil, fmt.Errorf("page cannot be nil")
	}

	// Navigate to the connections page
	err := page.Navigate("https://www.linkedin.com/mynetwork/invite-connect/connections/")
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to connections page: %w", err)
	}

	err = page.WaitLoad()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for connections page to load: %w", err)
	}

	// Add delay for page to fully render
	if mm.stealth != nil {
		err = mm.stealth.RandomDelay(2*time.Second, 4*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to add page load delay: %w", err)
		}
	}

	// Get sent connection requests from storage to compare
	sentRequests, err := mm.storage.GetSentRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to get sent requests: %w", err)
	}

	// Create a map of sent requests for quick lookup
	sentRequestsMap := make(map[string]ConnectionRequest)
	for _, req := range sentRequests {
		sentRequestsMap[req.ProfileURL] = req
	}

	// Find connection elements on the page
	connectionSelectors := []string{
		".mn-connection-card",
		".connection-card",
		"[data-test-id='connection-card']",
		".mn-connections__card",
	}

	var connections []AcceptedConnection
	var connectionElements []*rod.Element

	// Try different selectors to find connection cards
	for _, selector := range connectionSelectors {
		elements, err := page.Elements(selector)
		if err == nil && len(elements) > 0 {
			connectionElements = elements
			break
		}
	}

	if len(connectionElements) == 0 {
		// Try a more general approach
		elements, err := page.Elements("li")
		if err != nil {
			return nil, fmt.Errorf("failed to find any connection elements: %w", err)
		}
		
		// Filter for elements that look like connection cards
		for _, element := range elements {
			text, err := element.Text()
			if err != nil {
				continue
			}
			if strings.Contains(strings.ToLower(text), "connect") || 
			   strings.Contains(strings.ToLower(text), "connection") {
				connectionElements = append(connectionElements, element)
			}
		}
	}

	// Process each connection element
	for _, element := range connectionElements {
		if err := ctx.Err(); err != nil {
			return connections, err
		}

		connection, err := mm.parseConnectionElement(element)
		if err != nil {
			continue // Skip elements we can't parse
		}

		// Check if this was a connection we sent a request to
		if sentReq, exists := sentRequestsMap[connection.ProfileURL]; exists {
			// This is an accepted connection from our sent requests
			connection.AcceptedAt = time.Now()
			connections = append(connections, connection)
			
			// Update the status in storage (this would require extending the storage interface)
			// For now, we'll just track it as accepted
			sentReq.Status = "accepted"
		}
	}

	return connections, nil
}

// parseConnectionElement extracts connection information from a DOM element
func (mm *MessagingManager) parseConnectionElement(element *rod.Element) (AcceptedConnection, error) {
	var connection AcceptedConnection

	// Try to extract name
	nameSelectors := []string{
		".mn-connection-card__name",
		".connection-card__name",
		"[data-test-id='connection-name']",
		"h3",
		".name",
	}

	for _, selector := range nameSelectors {
		nameElement, err := element.Element(selector)
		if err == nil && nameElement != nil {
			name, err := nameElement.Text()
			if err == nil && name != "" {
				connection.Name = strings.TrimSpace(name)
				break
			}
		}
	}

	// Try to extract title
	titleSelectors := []string{
		".mn-connection-card__occupation",
		".connection-card__title",
		"[data-test-id='connection-title']",
		".occupation",
		".title",
	}

	for _, selector := range titleSelectors {
		titleElement, err := element.Element(selector)
		if err == nil && titleElement != nil {
			title, err := titleElement.Text()
			if err == nil && title != "" {
				connection.Title = strings.TrimSpace(title)
				break
			}
		}
	}

	// Try to extract profile URL
	linkSelectors := []string{
		"a[href*='/in/']",
		".mn-connection-card__link",
		".connection-card__link",
	}

	for _, selector := range linkSelectors {
		linkElement, err := element.Element(selector)
		if err == nil && linkElement != nil {
			href, err := linkElement.Attribute("href")
			if err == nil && href != nil && strings.Contains(*href, "/in/") {
				connection.ProfileURL = *href
				break
			}
		}
	}

	// Validate that we have at least a name
	if connection.Name == "" {
		return connection, fmt.Errorf("could not extract connection name")
	}

	return connection, nil
}

// SubstituteVariables replaces template variables with actual values
func (mm *MessagingManager) SubstituteVariables(template MessageTemplate, variables map[string]string) (string, error) {
	content := template.Body
	
	// Merge template variables with provided variables
	allVariables := make(map[string]string)
	
	// Start with template's default variables
	for k, v := range template.Variables {
		allVariables[k] = v
	}
	
	// Override with provided variables
	for k, v := range variables {
		allVariables[k] = v
	}
	
	// Replace variables in the format {{variable_name}}
	for key, value := range allVariables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		content = strings.ReplaceAll(content, placeholder, value)
	}
	
	// Check for any unreplaced variables
	if strings.Contains(content, "{{") && strings.Contains(content, "}}") {
		return content, fmt.Errorf("template contains unreplaced variables")
	}
	
	return content, nil
}

// NavigateToMessaging navigates to LinkedIn messaging interface
func (mm *MessagingManager) NavigateToMessaging(ctx context.Context, page *rod.Page) error {
	if page == nil {
		return fmt.Errorf("page cannot be nil")
	}

	// Navigate to messaging page
	err := page.Navigate("https://www.linkedin.com/messaging/")
	if err != nil {
		return fmt.Errorf("failed to navigate to messaging page: %w", err)
	}

	err = page.WaitLoad()
	if err != nil {
		return fmt.Errorf("failed to wait for messaging page to load: %w", err)
	}

	// Add delay for page to fully render
	if mm.stealth != nil {
		err = mm.stealth.RandomDelay(2*time.Second, 4*time.Second)
		if err != nil {
			return fmt.Errorf("failed to add messaging page load delay: %w", err)
		}
	}

	return nil
}

// FindConversation finds a conversation with a specific connection
func (mm *MessagingManager) FindConversation(ctx context.Context, page *rod.Page, connectionName string) (*rod.Element, error) {
	if page == nil {
		return nil, fmt.Errorf("page cannot be nil")
	}

	if connectionName == "" {
		return nil, fmt.Errorf("connection name cannot be empty")
	}

	// Try different selectors for conversation list items
	conversationSelectors := []string{
		".msg-conversation-listitem",
		".conversation-item",
		"[data-test-id='conversation-item']",
		".msg-conversations-container li",
	}

	var conversationElements []*rod.Element

	// Find conversation elements
	for _, selector := range conversationSelectors {
		elements, err := page.Elements(selector)
		if err == nil && len(elements) > 0 {
			conversationElements = elements
			break
		}
	}

	if len(conversationElements) == 0 {
		return nil, fmt.Errorf("no conversation elements found")
	}

	// Search for conversation with matching name
	for _, element := range conversationElements {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		text, err := element.Text()
		if err != nil {
			continue
		}

		// Check if this conversation contains the connection name
		if strings.Contains(strings.ToLower(text), strings.ToLower(connectionName)) {
			return element, nil
		}
	}

	return nil, fmt.Errorf("conversation with %s not found", connectionName)
}

// SendMessage sends a follow-up message to an accepted connection
func (mm *MessagingManager) SendMessage(ctx context.Context, page *rod.Page, connection AcceptedConnection, template MessageTemplate) error {
	// Check rate limiting first
	if mm.rateLimiter != nil && !mm.rateLimiter.CanSendMessage() {
		return fmt.Errorf("rate limit exceeded, cannot send message")
	}

	if page == nil {
		return fmt.Errorf("page cannot be nil")
	}

	// Navigate to messaging interface
	err := mm.NavigateToMessaging(ctx, page)
	if err != nil {
		return fmt.Errorf("failed to navigate to messaging: %w", err)
	}

	// Find the conversation with this connection
	conversation, err := mm.FindConversation(ctx, page, connection.Name)
	if err != nil {
		return fmt.Errorf("failed to find conversation with %s: %w", connection.Name, err)
	}

	// Click on the conversation to open it
	if mm.stealth != nil {
		err = mm.stealth.HumanMouseMove(ctx, page, conversation)
		if err != nil {
			return fmt.Errorf("failed to move mouse to conversation: %w", err)
		}

		err = mm.stealth.RandomDelay(500*time.Millisecond, 1500*time.Millisecond)
		if err != nil {
			return fmt.Errorf("failed to add pre-click delay: %w", err)
		}
	}

	err = conversation.Click("left", 1)
	if err != nil {
		return fmt.Errorf("failed to click conversation: %w", err)
	}

	// Wait for conversation to load
	if mm.stealth != nil {
		err = mm.stealth.RandomDelay(2*time.Second, 4*time.Second)
		if err != nil {
			return fmt.Errorf("failed to add conversation load delay: %w", err)
		}
	}

	// Prepare message content with variable substitution
	variables := map[string]string{
		"name":    connection.Name,
		"title":   connection.Title,
		"company": connection.Company,
	}

	messageContent, err := mm.SubstituteVariables(template, variables)
	if err != nil {
		return fmt.Errorf("failed to substitute template variables: %w", err)
	}

	// Find the message input field
	messageInput, err := mm.findMessageInput(page)
	if err != nil {
		return fmt.Errorf("failed to find message input field: %w", err)
	}

	// Type the message using stealth behavior
	if mm.stealth != nil {
		err = mm.stealth.HumanType(ctx, messageInput, messageContent)
		if err != nil {
			return fmt.Errorf("failed to type message: %w", err)
		}
	} else {
		err = messageInput.Input(messageContent)
		if err != nil {
			return fmt.Errorf("failed to input message: %w", err)
		}
	}

	// Find and click the send button
	sendButton, err := mm.findSendButton(page)
	if err != nil {
		return fmt.Errorf("failed to find send button: %w", err)
	}

	if mm.stealth != nil {
		err = mm.stealth.HumanMouseMove(ctx, page, sendButton)
		if err != nil {
			return fmt.Errorf("failed to move mouse to send button: %w", err)
		}

		err = mm.stealth.RandomDelay(500*time.Millisecond, 1000*time.Millisecond)
		if err != nil {
			return fmt.Errorf("failed to add pre-send delay: %w", err)
		}
	}

	err = sendButton.Click("left", 1)
	if err != nil {
		return fmt.Errorf("failed to click send button: %w", err)
	}

	// Track the sent message
	sentMessage := SentMessage{
		RecipientURL:  connection.ProfileURL,
		RecipientName: connection.Name,
		Template:      template.Name,
		Content:       messageContent,
		SentAt:        time.Now(),
		Response:      "",
	}

	err = mm.TrackMessage(sentMessage)
	if err != nil {
		return fmt.Errorf("failed to track sent message: %w", err)
	}

	// Record with rate limiter
	if mm.rateLimiter != nil {
		mm.rateLimiter.RecordMessage()
	}

	return nil
}

// findMessageInput finds the message input field
func (mm *MessagingManager) findMessageInput(page *rod.Page) (*rod.Element, error) {
	inputSelectors := []string{
		".msg-form__contenteditable",
		"[data-test-id='message-input']",
		".msg-form__msg-content-container div[contenteditable='true']",
		"div[contenteditable='true'][role='textbox']",
		".compose-publisher__editor div[contenteditable='true']",
	}

	for _, selector := range inputSelectors {
		element, err := page.Element(selector)
		if err == nil && element != nil {
			visible, err := element.Visible()
			if err == nil && visible {
				return element, nil
			}
		}
	}

	return nil, fmt.Errorf("message input field not found")
}

// findSendButton finds the send button
func (mm *MessagingManager) findSendButton(page *rod.Page) (*rod.Element, error) {
	sendSelectors := []string{
		".msg-form__send-button",
		"[data-test-id='send-button']",
		"button[type='submit'][aria-label*='Send']",
		".msg-form__send-btn",
		"button:has-text('Send')",
	}

	for _, selector := range sendSelectors {
		element, err := page.Element(selector)
		if err == nil && element != nil {
			visible, err := element.Visible()
			if err == nil && visible {
				return element, nil
			}
		}
	}

	return nil, fmt.Errorf("send button not found")
}

// TrackMessage persists message history and tracking data
func (mm *MessagingManager) TrackMessage(message SentMessage) error {
	if mm.storage == nil {
		return fmt.Errorf("storage interface not configured")
	}

	err := mm.storage.SaveMessage(message)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}