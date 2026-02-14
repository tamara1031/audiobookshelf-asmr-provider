# Audiobookshelf ASMR Provider

A specialized metadata provider for [Audiobookshelf](https://www.audiobookshelf.org/) that fetches ASMR content metadata from DLsite and other sources.

## Features

- **DLsite Integration**: Fetches comprehensive metadata (title, circle, voice actors, tags, description) from DLsite. Supports RJ codes and keyword search.
- **Audiobookshelf Compatible**: Exposes endpoints tailored for Audiobookshelf's custom metadata provider interface.
- **Docker Support**: Ready-to-use Docker image for easy deployment.
- **Microservice Architecture**: Designed to run alongside Audiobookshelf as a standalone service.
- **In-Memory Caching**: built-in caching (1-hour TTL, max 10k items) to reduce load on upstream providers.
- **Configurable Logging**: Adjust logging verbosity via environment variables for debugging or production monitoring.

## Installation

### Using Docker (Recommended)

1.  **Pull the image**:
    ```bash
    docker pull ghcr.io/tamara1031/audiobookshelf-asmr-provider:latest
    ```

2.  **Run the container**:
    ```bash
    docker run -d \
      -p 8080:8080 \
      --name abs-asmr-provider \
      ghcr.io/tamara1031/audiobookshelf-asmr-provider:latest
    ```

### Manual Installation

1.  **Prerequisites**: Go 1.25 or later.
2.  **Clone the repository**:
    ```bash
    git clone https://github.com/tamara1031/audiobookshelf-asmr-provider.git
    cd audiobookshelf-asmr-provider
    ```
3.  **Build and Run**:
    ```bash
    go run cmd/server/main.go
    ```

## Configuration

The application is configured via environment variables:

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | The port the server listens on. | `8080` |
| `LOG_LEVEL` | Logging verbosity (`DEBUG`, `INFO`, `WARN`, `ERROR`). | `INFO` |

## Usage

### API Endpoints

- **`GET /health`**: Health check endpoint. Returns `200 OK`.
- **`GET /api/search?q={query}`**: Search across all configured providers. Supports `q` or `query` parameter.
- **`GET /api/{provider}/search?q={query}`**: Search a specific provider (e.g., `/api/dlsite/search`).

### Audiobookshelf Configuration

1.  In Audiobookshelf, go to **Settings** > **Metadata Providers**.
2.  Add a new custom provider.
3.  Set the URL to your provider instance (e.g., `http://localhost:8080`).
4.  Save and test.

## Contributing

See [DEVELOPMENT.md](DEVELOPMENT.md) for instructions on how to add new metadata providers.

## License

MIT
