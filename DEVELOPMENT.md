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

The project is designed to be easily extensible. All providers are auto-discovered through a central registry, so `cmd/server/main.go` never needs to be modified.

### Steps

1.  **Create a new package**:
    Create a new directory under `internal/adapter/provider/` (e.g., `internal/adapter/provider/myprovider/`).

2.  **Implement the `domain.Provider` interface**:
    The interface is defined in `internal/domain/domain.go`:

    ```go
    type Provider interface {
        ID() string
        Search(ctx context.Context, query string) ([]AbsBookMetadata, error)
        CacheTTL() time.Duration
    }
    ```

    At minimum, create a constructor that returns `domain.Provider`:

    ```go
    // internal/adapter/provider/myprovider/scraper.go
    package myprovider

    import (
        "context"
        "time"

        "audiobookshelf-asmr-provider/internal/domain"
    )

    type myFetcher struct{}

    func NewMyFetcher() domain.Provider {
        return &myFetcher{}
    }

    func (f *myFetcher) ID() string                { return "myprovider" }
    func (f *myFetcher) CacheTTL() time.Duration   { return 24 * time.Hour }
    func (f *myFetcher) Search(ctx context.Context, query string) ([]domain.AbsBookMetadata, error) {
        // your scraping / API logic here
        return nil, nil
    }
    ```

3.  **Register in the provider registry**:
    Open `internal/adapter/provider/registry.go` and add your provider to the `NewAll()` function:

    ```go
    import (
        "audiobookshelf-asmr-provider/internal/adapter/provider/dlsite"
        "audiobookshelf-asmr-provider/internal/adapter/provider/myprovider"
        "audiobookshelf-asmr-provider/internal/domain"
    )

    func NewAll() []domain.Provider {
        return []domain.Provider{
            dlsite.NewDLsiteFetcher(),
            myprovider.NewMyFetcher(), // ← add here
        }
    }
    ```

    `main.go` calls `provider.NewAll()` automatically, so no other wiring is needed.

4.  **Test**:
    Add unit tests alongside your scraper (e.g., `myprovider/scraper_test.go`).
    Once registered, the provider endpoint is available at `/api/myprovider/search`.

## Contributing

1.  **Fork the repository**.
2.  **Create a feature branch**: `git checkout -b feature/my-feature`.
3.  **Commit your changes**: `git commit -am 'Add new feature'`.
4.  **Push to the branch**: `git push origin feature/my-feature`.
5.  **Submit a pull request**.

Please ensure your code passes all tests and lint checks before submitting.
