# LinkedIn Automation Framework

A technical proof-of-concept LinkedIn automation framework built in Go using the Rod browser automation library. This system demonstrates advanced browser automation techniques, human-like interaction modeling, and anti-detection concepts through a clean, modular architecture.

## ⚠️ Legal and Ethical Disclaimer

**This project is strictly for educational and technical evaluation purposes only.**

**The Link to the video is given below:**
**LINK: https://drive.google.com/file/d/1l2XAvb3uzUcdpZMD7iNFmI3Uxl3Jk0Sa/view?usp=sharing**

### Legal Compliance
- This framework is designed to demonstrate browser automation concepts and Go programming patterns
- Users are **solely responsible** for complying with LinkedIn's Terms of Service and all applicable laws
- Automated interaction with LinkedIn **may violate** their Terms of Service
- This software should **never be used** for commercial purposes or at scale
- The authors are **not responsible** for any misuse of this software

### Important Warnings
- **⚠️ ACCOUNT RISK**: Automated scraping or interaction with LinkedIn may result in account suspension or legal action
- **⚠️ TERMS OF SERVICE**: LinkedIn's Terms of Service explicitly prohibit automated data collection and interaction
- **⚠️ RATE LIMITS**: LinkedIn implements sophisticated detection systems that may flag automated behavior
- **⚠️ LEGAL LIABILITY**: Users may face legal consequences for violating platform terms or applicable laws

### Compliance Requirements
- **ALWAYS** review and comply with LinkedIn's robots.txt and Terms of Service before any testing
- **NEVER** use this software on production LinkedIn accounts
- **RESPECT** rate limits and implement appropriate delays
- **OBTAIN** proper authorization before testing on any LinkedIn account
- **CONSIDER** the impact on other users and the platform ecosystem

### Ethical Guidelines
- Use this knowledge responsibly and for legitimate educational purposes only
- Do not use this framework to spam, harass, or negatively impact other LinkedIn users
- Respect privacy and data protection regulations (GDPR, CCPA, etc.)
- Consider the broader implications of automated social media interaction

### Recommended Use Cases
- **Educational**: Learning browser automation and Go programming patterns
- **Research**: Understanding anti-detection techniques and stealth behaviors
- **Development**: Testing browser automation libraries and frameworks
- **Security**: Analyzing detection mechanisms and defensive strategies

**By using this software, you acknowledge that you have read, understood, and agree to comply with all legal and ethical requirements outlined above.**

## Project Structure

```
linkedin-automation-framework/
├── main.go                     # Application entry point
├── go.mod                      # Go module definition
├── internal/                   # Internal packages
│   ├── browser/               # Rod browser management
│   │   ├── browser.go         # Browser manager interface and implementation
│   │   └── browser_test.go    # Property-based tests for browser functionality
│   ├── auth/                  # LinkedIn authentication
│   │   └── auth.go           # Authentication interface and implementation
│   ├── search/                # Profile discovery
│   │   └── search.go         # Search interface and implementation
│   ├── connect/               # Connection requests
│   │   └── connect.go        # Connection manager interface and implementation
│   ├── messaging/             # Follow-up messaging
│   │   └── messaging.go      # Messaging interface and implementation
│   ├── stealth/               # Human behavior simulation
│   │   └── stealth.go        # Stealth behavior interface and implementation
│   ├── storage/               # Data persistence
│   │   └── storage.go        # Storage interface and implementation
│   ├── logger/                # Structured logging
│   │   └── logger.go         # Logger interface and implementation
│   └── config/                # Configuration management
│       └── config.go         # Configuration structures and interface
└── README.md                  # This file
```

## Key Features

### Rod-Native Architecture
- Built using Rod's native APIs and patterns
- Proper context management and timeout handling
- Clean separation between browser automation and business logic

### Modular Design
- Each module has a single responsibility
- Clean interfaces for easy testing and maintenance
- Dependency injection for flexible configuration

### Stealth Capabilities
- Human-like mouse movement with Bézier curves
- Randomized timing and interaction patterns
- Browser fingerprint configuration
- Activity scheduling and rate limiting

### Comprehensive Testing
- Property-based testing using pgregory.net/rapid
- Unit tests for specific functionality
- Both approaches ensure correctness and reliability

## Dependencies

- **Rod**: High-level browser automation library
- **Rapid**: Property-based testing framework
- **YAML**: Configuration file parsing
- **SQLite**: Data persistence (optional)

## Getting Started

### Prerequisites

- **Go 1.21 or later** - [Download Go](https://golang.org/dl/)
- **Chrome/Chromium browser** - Rod will download if not present
- **Git** - For cloning the repository

### Installation

1. **Clone and setup the project:**
   ```bash
   git clone <repository-url>
   cd linkedin-automation-framework
   go mod tidy
   ```

2. **Setup environment configuration:**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Create data directory:**
   ```bash
   mkdir -p data
   ```

4. **Run tests to verify setup:**
   ```bash
   go test ./...
   ```

5. **Build the application:**
   ```bash
   go build -o linkedin-automation-framework
   ```

### Quick Demo

**⚠️ Educational Use Only - Do Not Use on Real LinkedIn Accounts**

1. **Test browser initialization:**
   ```bash
   ./linkedin-automation-framework --mode=test --action=browser-test
   ```

2. **Run stealth behavior demo:**
   ```bash
   ./linkedin-automation-framework --mode=demo --action=stealth-demo
   ```

3. **Test configuration loading:**
   ```bash
   ./linkedin-automation-framework --mode=test --action=config-test
   ```
4. **Test manual-login:**
   ```bash
   ./linkedin-automation-framework --mode=manual-login --headless=false
   ```

### Configuration Setup

The application requires proper configuration before use:

1. **Copy the example environment file:**
   ```bash
   cp .env.example .env
   ```

2. **Edit `.env` with your settings** (see Configuration section below)

3. **Verify configuration:**
   ```bash
   ./linkedin-automation-framework --validate-config
   ```

## Configuration

The application supports both YAML configuration files and environment variable overrides. Environment variables take precedence over YAML settings.

### Configuration Files

Create `config.yaml` in the project root:

```yaml
browser:
  headless: true
  user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  viewport:
    width: 1920
    height: 1080
  flags:
    - "--no-sandbox"
    - "--disable-blink-features=AutomationControlled"

stealth:
  delays:
    min: "1s"
    max: "5s"
  typing:
    min_delay: "50ms"
    max_delay: "200ms"
  respect_business_hours: true
  cooldown_period: "30m"

rate_limits:
  connections_per_hour: 10
  messages_per_hour: 5
  searches_per_hour: 20
  cooldown_between: "5m"

storage:
  type: "sqlite"
  path: "./data"
  database: "linkedin_automation.db"

logging:
  level: "info"
  format: "json"
  output: "stdout"
```

### Environment Variables

All configuration can be overridden with environment variables. See `.env.example` for complete list.

**Critical Settings:**
- `LINKEDIN_USERNAME` - Your LinkedIn email (required for auth testing)
- `LINKEDIN_PASSWORD` - Your LinkedIn password (required for auth testing)
- `BROWSER_HEADLESS` - Run browser in headless mode (true/false)
- `STORAGE_TYPE` - Storage backend (sqlite/json)

### Configuration Validation

The system validates all configuration on startup and applies sensible defaults for missing values.

## Rod Architecture and Implementation Patterns

This project demonstrates advanced Rod browser automation patterns and best practices. Rod is a high-level driver directly based on the DevTools Protocol, providing a clean and idiomatic Go API for browser automation.

### Why Rod?

Rod was chosen for this project because:
- **Native Go**: Pure Go implementation with no external dependencies
- **DevTools Protocol**: Direct access to Chrome DevTools Protocol
- **Performance**: Efficient and fast browser automation
- **Type Safety**: Strong typing and compile-time safety
- **Context Support**: First-class support for Go contexts and cancellation
- **Clean API**: Intuitive and idiomatic Go interface

### Rod Integration Patterns

#### 1. Browser Lifecycle Management

Rod provides clean patterns for managing browser instances:

```go
// Proper browser initialization with launcher
launcher := launcher.New().
    Headless(headless).
    Set("disable-blink-features", "AutomationControlled")

browser := rod.New().ControlURL(launcher.MustLaunch()).MustConnect()
defer browser.MustClose()

// Context-aware page creation
page := browser.Context(ctx).MustPage(url)
defer page.MustClose()
```

**Key Points:**
- Use `launcher.New()` for browser configuration
- Always defer `MustClose()` for proper cleanup
- Pass contexts for cancellation and timeout support
- Configure browser flags to reduce detection fingerprints

#### 2. Element Interaction Patterns

Safe and robust element interactions:

```go
// Safe element operations with error handling
element, err := page.Timeout(5*time.Second).Element(selector)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        return ErrTimeout
    }
    return fmt.Errorf("element not found: %w", err)
}

// Human-like interactions through stealth module
if err := stealth.HumanClick(ctx, page, element); err != nil {
    return fmt.Errorf("click failed: %w", err)
}

// Type text with human-like behavior
if err := stealth.HumanType(ctx, element, "text to type"); err != nil {
    return fmt.Errorf("typing failed: %w", err)
}
```

**Key Points:**
- Always set appropriate timeouts for element operations
- Use error wrapping for better debugging
- Integrate stealth behaviors for human-like interactions
- Handle common errors (timeout, element not found, etc.)

#### 3. Context and Timeout Management

Proper context propagation throughout the application:

```go
// Create context with timeout for operation
ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
defer cancel()

// Rod timeout chaining - each operation can have its own timeout
page = page.Context(ctx).Timeout(10*time.Second)

// Element operations inherit the page timeout
element, err := page.Element(selector)

// Override timeout for specific operations
element, err = page.Timeout(2*time.Second).Element(fastSelector)
```

**Key Points:**
- Use contexts for cancellation and deadline propagation
- Set page-level timeouts for default behavior
- Override timeouts for specific operations when needed
- Always defer cancel() to prevent context leaks

#### 4. Error Handling Patterns

Rod-specific error handling strategies:

```go
// Rod-specific error handling
if err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        return ErrTimeout
    case errors.Is(err, context.Canceled):
        return ErrCanceled
    case rod.IsError(err, rod.ErrElementNotFound):
        return ErrElementMissing
    case rod.IsError(err, rod.ErrNavigation):
        return ErrNavigationFailed
    default:
        return fmt.Errorf("unexpected error: %w", err)
    }
}

// Retry logic with exponential backoff
err := retry.Do(ctx, func() error {
    return performRodOperation(page)
}, retry.WithMaxAttempts(3), retry.WithBackoff(retry.Exponential))
```

**Key Points:**
- Use `errors.Is()` for standard Go errors
- Use `rod.IsError()` for Rod-specific errors
- Implement retry logic for transient failures
- Wrap errors with context for debugging

#### 5. Page Navigation Patterns

Robust navigation with proper waiting:

```go
// Navigate and wait for page load
err := page.Navigate(url)
if err != nil {
    return fmt.Errorf("navigation failed: %w", err)
}

// Wait for network idle
page.MustWaitIdle()

// Wait for specific element to appear
element, err := page.Timeout(10*time.Second).Element(selector)
if err != nil {
    return fmt.Errorf("element not ready: %w", err)
}

// Wait for custom condition
err = page.Wait(func() bool {
    // Custom condition logic
    return page.MustHas(selector)
})
```

**Key Points:**
- Always wait for page load after navigation
- Use `WaitIdle()` for network activity to complete
- Wait for specific elements before interaction
- Implement custom wait conditions when needed

#### 6. Cookie and Session Management

Persistent sessions across browser restarts:

```go
// Save cookies for session persistence
cookies := page.MustCookies()
data, err := json.Marshal(cookies)
if err != nil {
    return fmt.Errorf("cookie serialization failed: %w", err)
}
err = os.WriteFile(cookiePath, data, 0600)

// Load cookies to restore session
data, err := os.ReadFile(cookiePath)
if err != nil {
    return fmt.Errorf("cookie load failed: %w", err)
}
var cookies []*proto.NetworkCookie
err = json.Unmarshal(data, &cookies)
if err != nil {
    return fmt.Errorf("cookie deserialization failed: %w", err)
}
err = page.SetCookies(cookies)
```

**Key Points:**
- Serialize cookies as JSON for persistence
- Store cookies securely with appropriate file permissions
- Load cookies before navigating to authenticated pages
- Validate cookie expiration before use

### Rod Best Practices Demonstrated

1. **Resource Management**: Proper cleanup of browsers and pages using defer
2. **Context Propagation**: Consistent context usage throughout the call chain
3. **Timeout Handling**: Appropriate timeouts for different operations
4. **Error Recovery**: Graceful handling of Rod-specific errors with retry logic
5. **Performance**: Efficient element selection and interaction patterns
6. **Stealth Integration**: Human-like behaviors integrated with Rod operations
7. **Session Management**: Cookie persistence for authenticated sessions
8. **Concurrent Safety**: Proper synchronization for concurrent page operations

### Browser Automation Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                        │
│  • CLI Interface  • Configuration  • Orchestration          │
├─────────────────────────────────────────────────────────────┤
│  Auth │ Search │ Connect │ Messaging │ Stealth │ Storage    │
│  • Login Logic  • Profile Discovery  • Human Behavior       │
├─────────────────────────────────────────────────────────────┤
│                   Browser Manager                           │
│  • Lifecycle Management  • Page Creation  • Session Mgmt   │
│  • Cookie Persistence   • Context Handling                 │
├─────────────────────────────────────────────────────────────┤
│                      Rod Library                            │
│  • DevTools Protocol   • Element Selection  • Interaction  │
│  • Network Control     • JavaScript Execution              │
├─────────────────────────────────────────────────────────────┤
│                   Chrome/Chromium                           │
│  • Rendering Engine    • JavaScript Runtime                │
└─────────────────────────────────────────────────────────────┘
```

### Advanced Rod Techniques

#### JavaScript Execution

Execute custom JavaScript in the page context:

```go
// Evaluate JavaScript and get result
result, err := page.Eval(`() => {
    return document.querySelectorAll('.profile-card').length;
}`)
if err != nil {
    return fmt.Errorf("eval failed: %w", err)
}
count := result.Value.Int()

// Modify page behavior
err = page.Eval(`() => {
    // Remove webdriver detection
    Object.defineProperty(navigator, 'webdriver', {
        get: () => undefined
    });
}`)
```

#### Network Interception

Monitor and modify network requests:

```go
// Intercept network requests
router := page.HijackRequests()
defer router.Stop()

router.MustAdd("*.js", func(ctx *rod.Hijack) {
    // Modify or block JavaScript files
    if strings.Contains(ctx.Request.URL().String(), "tracking") {
        ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
        return
    }
    ctx.ContinueRequest(&proto.FetchContinueRequest{})
})

go router.Run()
```

#### Screenshot and PDF Generation

Capture page content:

```go
// Take screenshot
screenshot, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
    Format: proto.PageCaptureScreenshotFormatPng,
})
if err != nil {
    return fmt.Errorf("screenshot failed: %w", err)
}
err = os.WriteFile("screenshot.png", screenshot, 0644)

// Generate PDF
pdf, err := page.PDF(&proto.PagePrintToPDF{
    PrintBackground: true,
})
if err != nil {
    return fmt.Errorf("pdf generation failed: %w", err)
}
```

### Common Pitfalls and Solutions

1. **Forgetting to wait for elements**: Always use timeouts and wait for elements before interaction
2. **Not handling context cancellation**: Properly propagate contexts and handle cancellation
3. **Memory leaks**: Always defer Close() on browsers and pages
4. **Race conditions**: Use proper synchronization for concurrent operations
5. **Detection fingerprints**: Configure browser flags and implement stealth behaviors

### Performance Optimization

- **Reuse browser instances** across multiple operations
- **Use incognito pages** for isolated sessions
- **Disable unnecessary features** (images, CSS) when not needed
- **Implement connection pooling** for concurrent operations
- **Cache selectors** for frequently accessed elements

## Development

This project follows Go best practices:

- Clean architecture with clear module boundaries
- Interface-driven design for testability
- Property-based testing for correctness validation
- Comprehensive error handling and logging
- Context-aware operations with proper timeouts
- Rod-native patterns and idioms

## Testing Strategy

The project uses a dual testing approach:

1. **Property-Based Tests**: Verify universal properties across random inputs
2. **Unit Tests**: Test specific examples and edge cases

Both approaches complement each other to ensure comprehensive coverage and correctness.

## Contributing

This is an educational project. Contributions should focus on:

- Improving code quality and architecture
- Adding comprehensive tests
- Enhancing documentation
- Demonstrating advanced Go patterns

## License

This project is for educational purposes only. Please ensure compliance with all applicable terms of service and laws.
