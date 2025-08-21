# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `bamboo-base-go`, a Go library that provides foundational components for bamboo services. It's designed as a reusable base library for building web APIs using the Gin framework with standardized error handling, logging, configuration management, and response formatting.

## Architecture

### Core Components

- **Initialization System** (`init/`): Centralized registration system that bootstraps the application
  - `register.go` contains the main `Register()` function that initializes all components
  - Handles configuration loading, logger setup, Gin engine initialization, and system context setup

- **Error Handling** (`error/`): Comprehensive error management system
  - `ErrorInterface` defines standard error contracts
  - `Error` struct implements structured error with code, message, and data
  - `ErrorCode` provides predefined error constants

- **Response System**: Standardized API response structure
  - `BaseResponse` in `base_response.go` defines the common response format
  - `result/` package handles response formatting
  - `middleware/response.go` provides unified response middleware

- **Configuration** (`models/config.go`): YAML-based configuration system
  - `AwakenConfig` structure supports database, NoSQL (Redis), and debug settings
  - Structured as nested YAML configuration

- **Utilities** (`utility/`): Common helper functions and context utilities
  - `ctxutil/` provides context-related utilities for database, logging, and common operations
  - Generic utility functions like `Ptr()` for pointer handling

### Key Dependencies

- **Gin Framework**: Web framework for HTTP routing and middleware
- **Zap Logger**: Structured logging with multiple output formats
- **GORM**: ORM for database operations
- **Validator**: Request validation using go-playground/validator
- **UUID**: Google UUID for unique identifier generation

## Development Commands

### Building and Testing
```bash
# Run tests
go test ./...

# Run specific test
go test ./test -v

# Build the module
go build

# Format code
go fmt ./...

# Vet code for common issues
go vet ./...

# Run tests with coverage
go test -cover ./...
```

### Module Management
```bash
# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify

# Download dependencies
go mod download
```

## Project Structure

```
bamboo-base/
├── base_response.go          # Standard API response structure
├── config/                   # Logger configuration (core, encoder)
├── constants/               # Application constants (context keys, headers, logger names)
├── error/                   # Error handling system (interface, codes, constructors)
├── go.mod                   # Module definition and dependencies
├── handler/                 # HTTP handlers (currently empty)
├── helper/                  # Helper utilities (panic recovery)
├── init/                    # Initialization and registration system
├── middleware/              # Gin middleware (response handling)
├── models/                  # Data models (configuration structures)
├── result/                  # Response result formatting
├── test/                    # Test files
├── utility/                 # Common utilities and context helpers
└── validator/               # Custom validation logic and messages
```

## Usage Patterns

### Initializing the Application
```go
reg := xInit.Register()
// reg.Serve is the *gin.Engine
// reg.Config is the *xModels.AwakenConfig  
// reg.Logger is the *zap.Logger
```

### Error Handling
Create structured errors using the error package and let the response middleware handle formatting automatically.

### Configuration
The system expects YAML configuration with `awaken`, `database`, and `nosql` sections.

## Testing

The project uses Go's standard testing framework. Test files are located in the `test/` directory. The existing test (`util_test.go`) demonstrates testing utility functions with proper assertion patterns.