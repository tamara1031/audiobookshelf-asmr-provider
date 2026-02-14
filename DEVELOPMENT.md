# Development Guide

This document outlines the steps to set up your development environment and contribute to the `audiobookshelf-asmr-provider`.

## Prerequisites

- **Go**: Version 1.25 or later.
- **Docker**: For containerized builds and testing.
- **Git**: Version control.

## Local Setup

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/tamara1031/audiobookshelf-asmr-provider.git
    cd audiobookshelf-asmr-provider
    ```

2.  **Install dependencies**:
    ```bash
    go mod download
    ```

3.  **Run the server**:
    ```bash
    go run cmd/server/main.go
    ```
    The server will start on port 8080 by default.

    To enable debug logging:
    ```bash
    LOG_LEVEL=DEBUG go run cmd/server/main.go
    ```

## Testing

We use the standard Go testing framework.

### Running Unit Tests

To run all unit tests in the project:

```bash
go test ./...
```

### Running Integration Tests

Integration tests are located in `test/integration`.

```bash
go test ./test/...
```

### Linting

To run the linter:

```bash
make lint
```

### Formatting

To format your code:

```bash
make fmt
```


## Project Structure

Refer to [ARCHITECTURE.md](ARCHITECTURE.md) for a detailed breakdown of the codebase organization.

## Creating a New Provider

The project is designed to be easily extensible. All providers are registered in a central factory.

### Steps

1.  **Create a new package**:
    Create a new directory under `internal/domain/provider/` (e.g., `internal/domain/provider/myprovider/`).

2.  **Implement the `service.Provider` interface**:
    The interface is defined in `internal/service/types.go`:

    ```go
    type Provider interface {
        ID() string
        Search(ctx context.Context, query string) ([]AbsBookMetadata, error)
        CacheTTL() time.Duration
    }
    ```

    At minimum, create a constructor that returns `service.Provider`:

    ```go
    // internal/domain/provider/myprovider/scraper.go
    package myprovider

    import (
        "context"
        "time"

        "audiobookshelf-asmr-provider/internal/service"
    )

    type myFetcher struct{}

    func NewProvider() service.Provider {
        return &myFetcher{}
    }

    func (f *myFetcher) ID() string                { return "myprovider" }
    func (f *myFetcher) CacheTTL() time.Duration   { return 24 * time.Hour }
    func (f *myFetcher) Search(ctx context.Context, query string) ([]service.AbsBookMetadata, error) {
        // your scraping / API logic here
        return []service.AbsBookMetadata{}, nil
    }
    ```

3.  **Register in the provider registry**:
    Open `internal/domain/provider/registry.go` and add your provider to the `NewAll()` function:

    ```go
        "audiobookshelf-asmr-provider/internal/domain/provider/all"
        "audiobookshelf-asmr-provider/internal/domain/provider/dlsite"
        "audiobookshelf-asmr-provider/internal/domain/provider/myprovider"
        "audiobookshelf-asmr-provider/internal/domain/provider/void"
        "audiobookshelf-asmr-provider/internal/service"
    )

    func NewAll() []service.Provider {
        dlsiteProvider := dlsite.NewDLsiteFetcher()
        allProvider := all.NewProvider(dlsiteProvider)
        voidProvider := void.NewProvider()

        return []service.Provider{
            dlsiteProvider,
            allProvider,
            voidProvider,
            myprovider.NewProvider(), // ← Add your new provider here
        }
    }
    ```

    `main.go` calls `provider.NewAll()` and injects them into the service automatically.

4.  **Test**:
    Add unit tests alongside your scraper (e.g., `myprovider/ scraper_test.go`).
    Once registered, the provider endpoint is available at `/api/myprovider/search`.

## Contributing

1.  **Fork the repository**.
2.  **Create a feature branch** on your fork: `git checkout -b feature/my-feature`.
3.  **Commit your changes**: `git commit -am 'Add new feature'`.
4.  **Push to your fork**: `git push origin feature/my-feature`.
5.  **Submit a Pull Request** from your fork to the `master` branch of the upstream repository.

Please ensure your code passes all tests and lint checks before submitting.
