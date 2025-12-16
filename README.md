# LinkedIn Automation Framework

A technical proof-of-concept LinkedIn automation framework built in Go using the Rod browser automation library. This system demonstrates advanced browser automation techniques, human-like interaction modeling, and anti-detection concepts through a clean, modular architecture.

## ⚠️ Legal and Ethical Disclaimer

**This project is strictly for educational and technical evaluation purposes only.**

- This framework is designed to demonstrate browser automation concepts and Go programming patterns
- Users are responsible for complying with LinkedIn's Terms of Service and applicable laws
- Automated interaction with LinkedIn may violate their Terms of Service
- This software should not be used for commercial purposes or at scale
- The authors are not responsible for any misuse of this software

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

1. **Install Go 1.21 or later**

2. **Clone and setup the project:**
   ```bash
   git clone <repository-url>
   cd linkedin-automation-framework
   go mod tidy
   ```

3. **Run tests:**
   ```bash
   go test ./...
   ```

4. **Build the application:**
   ```bash
   go build -o linkedin-automation-framework
   ```

## Configuration

The application uses YAML configuration files with environment variable overrides. See the config package for available options including:

- Browser settings (headless mode, viewport, flags)
- Stealth behavior parameters (timing, delays, patterns)
- Rate limiting configuration
- Storage options (SQLite or JSON)
- Logging configuration

## Development

This project follows Go best practices:

- Clean architecture with clear module boundaries
- Interface-driven design for testability
- Property-based testing for correctness validation
- Comprehensive error handling and logging
- Context-aware operations with proper timeouts

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