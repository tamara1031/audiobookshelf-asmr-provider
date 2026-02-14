# Architecture

This document describes the high-level architecture of the `audiobookshelf-asmr-provider`.

## Overview

The project follows a Clean Architecture / Hexagonal Architecture approach, separating the business logic from external dependencies (HTTP, Scraping).

## Directory Structure

```
.
├── cmd/
│   └── server/        # Application entry point
├── internal/
│   ├── adapter/       # Infrastructure layer (Implementations)
│   │   ├── http/      # HTTP handlers
│   │   └── provider/  # External data providers (DLsite)
│   ├── domain/        # Domain layer (Interfaces & Models)
│   │   └── domain.go  # Provider interface + ABS metadata structs
│   └── service/       # Application logic
│       ├── metadata.go # Service orchestration
│       └── cache.go    # In-memory cache
└── test/              # Integration tests
```

## Key Components

### Domain Layer (`internal/domain`)

Defines the core business rules and entities in a single package. It has no external dependencies.
- **`Provider`**: Interface that all metadata providers must implement (`ID`, `Search`, `CacheTTL`).
- **`AbsBookMetadata`** / **`AbsMetadataResponse`**: Structs matching the Audiobookshelf custom provider API contract.

### Adapter Layer (`internal/adapter`)

Implements the interfaces defined in the domain layer.
- **`http`**: Handles incoming HTTP requests from Audiobookshelf, parses parameters, and invokes the service layer.
- **`provider/registry.go`**: Central registry that exposes `NewAll()` to instantiate every available provider. New providers only need to be added here.
- **`provider/dlsite`**: Fetches HTML from DLsite and parses it into domain models using `goquery`.

### Service Layer (`internal/service`)

Orchestrates the flow of data between adapters. It receives requests from the HTTP handler, calls the appropriate provider, and returns the result.

- **`metadata.go`**: Business logic — routing queries to providers and aggregating results.
- **`cache.go`**: Standalone `Cache` struct with `Get`, `Put`, `EvictExpired`, and `Len` methods.

**Caching Strategy**:
- In-memory `map[string]cacheEntry` protected by a `sync.RWMutex`.
- Cache entries respect each provider's `CacheTTL()`, defaulting to 1 hour.
- A background goroutine runs hourly to clean up expired entries.
- A hard limit of 10,000 items prevents memory exhaustion.

### Server (`cmd/server`)

Wires everything together. It initializes the adapters, injects them into the service, sets up the HTTP router, and starts the server.

## Design Decisions

- **Dependency Injection**: Dependencies (like the DLsite fetcher) are injected into the service, making testing easier.
- **Interface-Based Design**: The service relies on interfaces (`Provider`), allowing new providers to be added without modifying core logic.
