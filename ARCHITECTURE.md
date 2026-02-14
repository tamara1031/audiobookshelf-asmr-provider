# Architecture

This document describes the high-level architecture of the `audiobookshelf-asmr-provider`.

## Overview

The project follows a Clean Architecture / Hexagonal Architecture approach, separating the business logic from external dependencies (HTTP, Scraping).

## Directory Structure

├── cmd/
│   └── server/        # Application entry point
├── internal/
│   ├── handler/       # HTTP handlers (formerly internal/api)
│   ├── domain/        # Domain logic and providers
│   │   ├── provider/  # External data providers (DLsite)
│   │   └── cache/     # In-memory cache implementation
│   ├── service/       # Application service layer (formerly internal/metadata)
│   │   └── metadata.go # Service orchestration & domain models
│   └── config/        # Configuration
└── test/              # Integration tests
```

## Key Components

### Service Layer (`internal/service`)

Defines the core business logic and models.
- **`metadata.go`**: Contains the `Service` struct, `Provider` and `Cache` interfaces, and the `AbsBookMetadata` model.
- **`Service`**: Orchestrates parallel searches across providers.

### Domain Layer (`internal/domain`)

Contains concrete implementations of domain interfaces that aren't core service logic.
- **`provider/`**: Concrete metadata providers (e.g., DLsite scraper).
- **`cache/`**: Concrete cache implementation (MemoryCache).

### Handler Layer (`internal/handler`)

Handles the delivery mechanism (HTTP).
- **`handler.go`**: HTTP handlers that translate Audiobookshelf requests into service calls.

**Caching Strategy**:
- In-memory `map[string]cacheEntry` in `internal/domain/cache`.
- Cache entries respect each provider's `CacheTTL()`.
- A background goroutine in the cache implementation handles cleanup.

### Server (`cmd/server`)

Wires everything together. It initializes the adapters, injects them into the service, sets up the HTTP router, and starts the server.

## Design Decisions

- **Dependency Injection**: Dependencies (like the DLsite fetcher) are injected into the service, making testing easier.
- **Interface-Based Design**: The service relies on interfaces (`Provider`), allowing new providers to be added without modifying core logic.
