# Design Document

## Overview

The LinkedIn Automation Framework is a technical proof-of-concept built in Go using the Rod browser automation library. The system demonstrates advanced browser automation techniques, human-like interaction modeling, and anti-detection concepts through a clean, modular architecture. The framework is designed strictly for educational and technical evaluation purposes.

The system follows Rod-native patterns and implements a comprehensive stealth module to simulate human behavior. All components are designed to be maintainable, configurable, and demonstrate best practices in Go development and browser automation.

## Architecture

The system follows a layered architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                     Main Application                        │
├─────────────────────────────────────────────────────────────┤
│  Config │  Logger │  Browser │  Auth │  Search │  Connect   │
│         │         │ Manager  │       │         │            │
├─────────────────────────────────────────────────────────────┤
│              Messaging │  Stealth │  Storage               │
├─────────────────────────────────────────────────────────────┤
│                     Rod Browser Engine                      │
└─────────────────────────────────────────────────────────────┘
```

### Key Architectural Principles

1. **Rod-Native Design**: All browser interactions use Rod's native APIs and patterns
2. **Modular Components**: Each module has a single responsibility and clean interfaces
3. **Context-Aware**: Proper use of Go contexts and Rod's timeout mechanisms
4. **Stealth-First**: Human behavior simulation is integrated throughout the system
5. **Resilient**: Graceful error handling and recovery mechanisms

## Components and Interfaces

### Browser Manager (`internal/browser/`)

The Browser Manager handles Rod browser lifecycle and provides a clean interface for other modules.

```go
type BrowserManager interface {
    Initialize(ctx context.Context) error
    Browser() *rod.Browser
    NewPage() (*rod.Page, error)
    NewIncognitoPage() (*rod.Page, error)
    SaveCookies(path string) error
    LoadCookies(path string) error
    Close() error
}
```

**Key Responsibilities:**
- Initialize Rod browser with proper configuration
- Manage browser flags and fingerprint settings
- Provide page creation with context management
- Handle cookie persistence for session management
- Graceful shutdown and resource cleanup

### Authentication Module (`internal/auth/`)

Handles LinkedIn login processes with human-like behavior simulation.

```go
type Authenticator interface {
    Login(ctx context.Context, page *rod.Page) error
    IsLoggedIn(ctx context.Context, page *rod.Page) (bool, error)
    HandleChallenge(ctx context.Context, page *rod.Page) error
}
```

**Key Responsibilities:**
- Navigate to LinkedIn login page
- Fill credentials using stealth typing simulation
- Detect login success/failure via DOM analysis
- Handle security challenges (2FA, CAPTCHA) detection
- Session validation and maintenance

### Search Module (`internal/search/`)

Implements LinkedIn profile discovery with pagination and deduplication.

```go
type ProfileSearcher interface {
    Search(ctx context.Context, criteria SearchCriteria) ([]ProfileResult, error)
    ExtractProfiles(ctx context.Context, page *rod.Page) ([]ProfileResult, error)
    HandlePagination(ctx context.Context, page *rod.Page) error
}
```

**Key Responsibilities:**
- Accept structured search criteria
- Navigate search result pages
- Extract profile URLs and metadata
- Handle pagination automatically
- Deduplicate results using storage

### Connection Module (`internal/connect/`)

Manages LinkedIn connection requests with rate limiting and personalization.

```go
type ConnectionManager interface {
    SendConnectionRequest(ctx context.Context, profile ProfileResult, note string) error
    DetectConnectButton(ctx context.Context, page *rod.Page) (*rod.Element, error)
    TrackSentRequest(request ConnectionRequest) error
}
```

**Key Responsibilities:**
- Open profile pages using Rod navigation
- Detect and interact with Connect buttons
- Send personalized connection requests
- Enforce configurable rate limits
- Track sent requests for analytics

### Messaging Module (`internal/messaging/`)

Handles follow-up messaging with template support and history tracking.

```go
type MessageSender interface {
    SendMessage(ctx context.Context, connection AcceptedConnection, template MessageTemplate) error
    DetectAcceptedConnections(ctx context.Context) ([]AcceptedConnection, error)
    TrackMessage(message SentMessage) error
}
```

**Key Responsibilities:**
- Detect newly accepted connections
- Support message templates with variables
- Send follow-up messages with human timing
- Track message history and analytics
- Respect messaging frequency limits

### Stealth Module (`internal/stealth/`)

Implements comprehensive human behavior simulation and anti-detection techniques.

```go
type StealthBehavior interface {
    HumanMouseMove(ctx context.Context, page *rod.Page, target *rod.Element) error
    HumanType(ctx context.Context, element *rod.Element, text string) error
    RandomDelay(min, max time.Duration) error
    ScrollNaturally(ctx context.Context, page *rod.Page) error
    ConfigureFingerprint(browser *rod.Browser) error
}
```

**Key Stealth Techniques:**
1. **Human-like mouse movement** with Bézier curves and micro-corrections
2. **Randomized timing** for interactions and page reading
3. **Browser fingerprint configuration** (User-Agent, viewport, webdriver masking)
4. **Typing simulation** with realistic delays and occasional mistakes
5. **Random scrolling behavior** patterns
6. **Mouse hovering and idle movement**
7. **Activity scheduling** respecting business hours
8. **Rate limiting and cooldown periods**

### Storage Module (`internal/storage/`)

Provides persistent storage for application state and analytics.

```go
type Storage interface {
    SaveConnectionRequest(request ConnectionRequest) error
    GetSentRequests() ([]ConnectionRequest, error)
    SaveMessage(message SentMessage) error
    GetMessageHistory() ([]SentMessage, error)
    SaveSearchResults(results []ProfileResult) error
}
```

**Key Responsibilities:**
- Persist connection requests and responses
- Store message history and templates
- Track search results and analytics
- Support resume-after-crash scenarios
- Provide data export capabilities

## Data Models

### Core Data Structures

```go
type SearchCriteria struct {
    Keywords    []string
    Location    string
    Industry    string
    Company     string
    Title       string
    Connections string
    MaxResults  int
}

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

type ConnectionRequest struct {
    ProfileURL  string
    ProfileName string
    Note        string
    SentAt      time.Time
    Status      string // pending, accepted, declined
}

type MessageTemplate struct {
    Name        string
    Subject     string
    Body        string
    Variables   map[string]string
}

type SentMessage struct {
    RecipientURL string
    Template     string
    Content      string
    SentAt       time.Time
    Response     string
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

<function_calls>
<invoke name="prework">
<parameter name="featureName">linkedin-automation-framework
### Brow
ser Management Properties

Property 1: Browser initialization consistency
*For any* system configuration, browser initialization using launcher.New() should succeed consistently across multiple attempts
**Validates: Requirements 1.1**

Property 2: Mode configuration support
*For any* browser configuration (headless or non-headless), the Browser_Manager should successfully initialize and operate in the specified mode
**Validates: Requirements 1.2**

Property 3: Browser flag application
*For any* set of browser flags, when applied during configuration, the browser should reflect those flags in its runtime behavior
**Validates: Requirements 1.3**

Property 4: Resource cleanup on shutdown
*For any* initialized browser instance, calling shutdown should properly release all associated resources without memory leaks
**Validates: Requirements 1.4**

Property 5: Page creation with context management
*For any* page creation request, the Browser_Manager should create pages using browser.MustPage() with proper context handling
**Validates: Requirements 1.5**

### Stealth Behavior Properties

Property 6: Human-like mouse movement patterns
*For any* mouse movement operation, the generated path should follow Bézier curves with overshoot and micro-corrections that match human movement characteristics
**Validates: Requirements 2.1**

Property 7: Randomized interaction timing
*For any* sequence of interactions, the timing delays should be properly randomized within configured ranges and show statistical variation
**Validates: Requirements 2.2**

Property 8: Fingerprint configuration application
*For any* fingerprint configuration, the browser should have User-Agent, viewport size, and navigator.webdriver properties set as specified
**Validates: Requirements 2.3**

Property 9: Human typing simulation
*For any* text input operation, the typing should include realistic delays between keystrokes and occasional correction patterns
**Validates: Requirements 2.4**

Property 10: Random scrolling behavior
*For any* scrolling operation, the scroll patterns should show appropriate variation in speed, distance, and timing
**Validates: Requirements 2.5**

Property 11: Idle behavior simulation
*For any* idle period, the system should perform mouse hovering and small movements that simulate natural user behavior
**Validates: Requirements 2.6**

Property 12: Activity scheduling and rate limiting
*For any* scheduled activity, the system should respect business hours and enforce configured rate limits
**Validates: Requirements 2.7**

Property 13: Cooldown period enforcement
*For any* completed operation, the system should enforce appropriate cooldown delays before allowing the next operation
**Validates: Requirements 2.8**

### Authentication Properties

Property 14: Credential loading from environment
*For any* environment configuration, the Authentication_Module should successfully read and validate credentials from environment variables
**Validates: Requirements 3.1**

Property 15: Rod navigation for login
*For any* login attempt, navigation to the login page should use proper Rod navigation methods
**Validates: Requirements 3.2**

Property 16: Stealth typing integration
*For any* credential filling operation, the Authentication_Module should use typing simulation from the Stealth_Module
**Validates: Requirements 3.3**

Property 17: Login state detection
*For any* login attempt, the system should correctly detect success or failure through DOM state analysis
**Validates: Requirements 3.4**

Property 18: Security challenge detection
*For any* login process, the system should detect security challenges (captcha, 2FA) without attempting to bypass them
**Validates: Requirements 3.5**

Property 19: Session persistence round-trip
*For any* established session, saving cookies and then loading them should preserve the session state
**Validates: Requirements 3.6**

### Search and Discovery Properties

Property 20: Search criteria acceptance
*For any* structured search parameters, the Search_Module should accept and process them correctly
**Validates: Requirements 4.1**

Property 21: Rod-based page navigation
*For any* search result navigation, the system should use proper Rod page management methods
**Validates: Requirements 4.2**

Property 22: Profile URL extraction
*For any* search results page, the system should correctly extract profile URLs regardless of page structure variations
**Validates: Requirements 4.3**

Property 23: Pagination handling
*For any* multi-page search results, the system should automatically process all pages without missing results
**Validates: Requirements 4.4**

Property 24: Result deduplication
*For any* search results containing duplicates, the system should properly remove duplicates using the Storage_Module
**Validates: Requirements 4.5**

### Connection Management Properties

Property 25: Profile page navigation
*For any* profile URL, the Connection_Module should successfully open the page using Rod navigation methods
**Validates: Requirements 5.1**

Property 26: Connect button detection
*For any* profile page structure, the system should reliably detect Connect buttons using Rod selectors
**Validates: Requirements 5.2**

Property 27: Connection request sending
*For any* connection request, the system should send it with proper note handling (including optional personalized notes)
**Validates: Requirements 5.3**

Property 28: Rate limit enforcement
*For any* sequence of connection requests, the system should enforce configurable rate limits and prevent exceeding them
**Validates: Requirements 5.4**

Property 29: Request data persistence
*For any* sent connection request, the data should be properly stored and retrievable from the Storage_Module
**Validates: Requirements 5.5**

### Messaging Properties

Property 30: Accepted connection detection
*For any* set of connections, the Messaging_Module should correctly identify newly accepted connections
**Validates: Requirements 6.1**

Property 31: Template variable substitution
*For any* message template with variables, the system should properly substitute all variables with their corresponding values
**Validates: Requirements 6.2**

Property 32: Message sending to correct recipients
*For any* follow-up message, the system should send it to the correct accepted connection recipient
**Validates: Requirements 6.3**

Property 33: Message history persistence
*For any* sent message, the history should be properly stored and retrievable with complete recipient information
**Validates: Requirements 6.4**

Property 34: Messaging rate limit compliance
*For any* sequence of messages, the system should respect configured messaging frequency limits
**Validates: Requirements 6.5**

### Storage and Persistence Properties

Property 35: Connection request tracking
*For any* sent connection request, the Storage_Module should store it with accurate timestamps and metadata
**Validates: Requirements 7.1**

Property 36: Accepted connection recording
*For any* accepted connection, the Storage_Module should properly record the connection data
**Validates: Requirements 7.2**

Property 37: Message history storage
*For any* sent message, the Storage_Module should store it with complete recipient information and metadata
**Validates: Requirements 7.3**

Property 38: Crash recovery capability
*For any* system interruption, the Storage_Module should enable the system to resume from the stored state
**Validates: Requirements 7.4**

Property 39: Storage format round-trip
*For any* data stored using SQLite or JSON, retrieving the data should produce equivalent objects to what was stored
**Validates: Requirements 7.5**

### Logging and Error Handling Properties

Property 40: Structured logging levels
*For any* system event, the Logger_Module should generate logs at appropriate levels (debug, info, warn, error) with proper structure
**Validates: Requirements 8.1**

Property 41: Contextual log information
*For any* logged action, the log should include contextual information (module, action, profile data) as specified
**Validates: Requirements 8.2**

Property 42: Graceful error handling
*For any* error condition, the system should handle it gracefully without causing system crashes
**Validates: Requirements 8.3**

Property 43: Exponential backoff retry
*For any* failed operation, the retry logic should follow exponential backoff patterns
**Validates: Requirements 8.4**

Property 44: Rod timeout and context usage
*For any* timeout operation, the system should use Rod's native timeout and context management mechanisms
**Validates: Requirements 8.5**

### Configuration Management Properties

Property 45: YAML configuration loading
*For any* valid YAML configuration file, the Configuration_Module should successfully load all settings
**Validates: Requirements 9.1**

Property 46: Environment variable override
*For any* configuration setting, environment variables should properly override corresponding YAML settings
**Validates: Requirements 9.2**

Property 47: Configuration validation with defaults
*For any* invalid or missing configuration, the system should apply sensible defaults and continue operation
**Validates: Requirements 9.3**

Property 48: Stealth parameter configuration
*For any* stealth behavior setting, the Configuration_Module should support configurable timing and behavior parameters
**Validates: Requirements 9.4**

Property 49: Rate limit parameter configuration
*For any* rate limiting setting, the Configuration_Module should allow customizable rate limiting parameters
**Validates: Requirements 9.5**

## Error Handling

The system implements comprehensive error handling strategies:

### Rod-Specific Error Handling
- Use `rod.Try()` for safe element operations
- Implement proper timeout handling with contexts
- Handle page navigation failures gracefully
- Manage browser crashes and recovery

### Module-Level Error Handling
- Each module implements its own error recovery strategies
- Exponential backoff for transient failures
- Circuit breaker patterns for external dependencies
- Graceful degradation when services are unavailable

### Logging and Monitoring
- Structured logging with contextual information
- Error categorization (transient, permanent, configuration)
- Performance metrics and timing information
- Debug information for troubleshooting

## Testing Strategy

The system employs a dual testing approach combining unit tests and property-based tests to ensure comprehensive coverage and correctness validation.

### Unit Testing Approach

Unit tests verify specific examples, edge cases, and integration points:

- **Component Integration**: Test interactions between modules (e.g., Auth module using Stealth module)
- **Edge Cases**: Test boundary conditions, empty inputs, and error scenarios
- **Rod Integration**: Verify proper use of Rod APIs and patterns
- **Configuration Validation**: Test various configuration scenarios

Unit tests focus on concrete scenarios and specific behaviors that demonstrate correct functionality.

### Property-Based Testing Approach

Property-based tests verify universal properties that should hold across all inputs using **Rapid** (Go property-based testing library):

- **Configuration**: Each property-based test runs a minimum of 100 iterations
- **Tagging**: Each test includes a comment referencing the design document property
- **Format**: Tests use the format `**Feature: linkedin-automation-framework, Property {number}: {property_text}**`
- **Coverage**: Each correctness property is implemented by a single property-based test

Property-based tests generate random inputs to verify that system properties hold universally, catching edge cases that might be missed by unit tests.

### Testing Framework Requirements

- **Property-Based Testing Library**: Rapid (github.com/flyingmutant/rapid)
- **Unit Testing**: Go's built-in testing package with testify for assertions
- **Test Organization**: Tests co-located with source code using `_test.go` suffix
- **Coverage**: Both unit and property tests are required for comprehensive validation
- **Integration**: Tests verify Rod integration and browser automation correctness

The combination of unit tests and property-based tests provides both concrete validation of specific behaviors and mathematical verification of universal system properties.