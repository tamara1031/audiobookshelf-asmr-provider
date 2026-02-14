# Architecture

This document describes the high-level architecture of the `audiobookshelf-asmr-provider`.

## Overview

The project follows a Clean Architecture / Hexagonal Architecture approach, separating the business logic from external dependencies (HTTP, Scraping).

## Directory Structure

```text
├── cmd/
│   └── server/            # Application entry point
├── internal/
│   ├── handler/           # HTTP delivery layer (formerly internal/api)
│   ├── service/           # Application service layer
│   │   ├── types.go       # Core abstractions (Provider/Cache) and domain models
│   │   └── metadata.go    # Service orchestration logic
│   ├── domain/            # Domain implementations
│   │   ├── cache/         # Concrete caching implementations
│   │   └── provider/      # Concrete metadata providers
│   │       ├── all/       # Aggregation provider
│   │       ├── dlsite/    # DLsite scraper
│   │       ├── void/      # Fallback provider
│   │       └── registry.go # Provider registration logic
│   └── config/            # Configuration
└── test/                  # Integration tests
```

## Key Components

### Service Layer (`internal/service`)

Defines the core business logic and models.
- **`types.go`**: Contains the `Provider` and `Cache` interfaces, and the `AbsBookMetadata` model. This is the "source of truth" for the application's domain.
- **`Service`**: Orchestrates searches across providers. It implements the logic for single-provider and aggregated searches.

### Domain Layer (`internal/domain`)

Contains concrete implementations of domain interfaces.
- **`provider/`**: Houses all metadata providers.
  - **`registry.go`**: A central point to register available providers.
- **`cache/`**: Concrete cache implementation (MemoryCache).

### Handler Layer (`internal/handler`)

Handles the delivery mechanism (HTTP).
- **`handler.go`**: HTTP handlers that translate Audiobookshelf requests into service calls.
- **Go 1.22+ Routing**: Uses descriptive patterns like `"GET /api/{provider}/search"` to automatically extract parameters.

## Design Decisions

- **Dependency Inversion Principle (DIP)**: High-level service modules do not depend on low-level provider modules. Both depend on abstractions.
- **Interface-Based Design**: The service relies on interfaces (`Provider`), allowing new providers to be added without modifying core logic.
- **Parallel Execution**: Aggregated searches are executed in parallel using goroutines to minimize response time.
