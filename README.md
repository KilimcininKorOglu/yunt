# Yunt - Development Mail Server

Yunt is a lightweight, powerful mail server written in Go, designed for developers and test environments. The name comes from the Gokturk Turkish word for "horse" - just as mounted couriers carried letters, Yunt safely delivers your emails.

## Features

| Feature               | Description                                    |
|-----------------------|------------------------------------------------|
| SMTP Server           | Mail capture and relay support                 |
| IMAP Server           | Mail client support (Thunderbird, etc.)        |
| Web UI                | Modern admin panel                             |
| REST API              | Full-featured API for integration              |
| Multi-user Support    | Team collaboration with isolated mailboxes     |
| Multi-database        | SQLite, PostgreSQL, MySQL, MongoDB             |

## Quick Start

### Using Docker (Recommended)

The fastest way to get started is using Docker. Multi-platform images are available for both AMD64 and ARM64 architectures.

```bash
# Pull the latest image
docker pull ghcr.io/yunt/yunt:latest

# Run with default settings
docker run -d \
  -p 1025:1025 \
  -p 1143:1143 \
  -p 8025:8025 \
  ghcr.io/yunt/yunt:latest

# Run with persistent data
docker run -d \
  -p 1025:1025 \
  -p 1143:1143 \
  -p 8025:8025 \
  -v yunt-data:/var/lib/yunt \
  -v ./yunt.yaml:/etc/yunt/yunt.yaml:ro \
  ghcr.io/yunt/yunt:latest
```

### Using Docker Compose

```yaml
services:
  yunt:
    image: ghcr.io/yunt/yunt:latest
    ports:
      - "1025:1025"  # SMTP
      - "1143:1143"  # IMAP
      - "8025:8025"  # Web UI
    volumes:
      - yunt-data:/var/lib/yunt
    environment:
      - YUNT_DATABASE_DSN=/var/lib/yunt/yunt.db
    restart: unless-stopped

volumes:
  yunt-data:
```

### Building from Source

#### Prerequisites

- Go 1.22 or higher
- Make (optional, for build commands)

#### Build

```bash
# Build the binary
make build

# Or using Go directly
go build -o bin/yunt ./cmd/yunt
```

#### Run

```bash
# Start the server with default configuration
./bin/yunt serve

# Start with a custom configuration file
./bin/yunt serve --config /path/to/yunt.yaml
```

## Default Ports

| Service | Port  |
|---------|-------|
| SMTP    | 1025  |
| IMAP    | 1143  |
| Web UI  | 8025  |

## Configuration

Yunt can be configured via YAML file or environment variables. See `configs/yunt.example.yaml` for all available options.

Environment variables use the `YUNT_` prefix. For example:
- `YUNT_SMTP_PORT=2025`
- `YUNT_DATABASE_DRIVER=postgres`

## Project Structure

```
yunt/
├── cmd/yunt/           # Application entry point
├── internal/
│   ├── config/         # Configuration management
│   ├── domain/         # Domain models
│   ├── repository/     # Data access layer
│   ├── service/        # Business logic
│   ├── smtp/           # SMTP server
│   ├── imap/           # IMAP server
│   ├── api/            # REST API and Web UI
│   └── parser/         # MIME parser
├── configs/            # Configuration examples
├── scripts/            # Build and utility scripts
├── go.mod              # Go module definition
└── Makefile            # Build automation
```

## Docker

### Available Images

Multi-platform Docker images are automatically built and published to GitHub Container Registry (ghcr.io) for both AMD64 and ARM64 architectures.

| Tag            | Description                                    |
|----------------|------------------------------------------------|
| `latest`       | Latest stable release from main branch         |
| `v1.2.3`       | Specific version release                       |
| `1.2`          | Major.minor version (tracks latest patch)      |
| `sha-abc1234`  | Specific commit SHA                            |

### Building Multi-Platform Images Locally

```bash
# Build for local testing (single platform)
./scripts/docker-build.sh -l

# Build for specific platform
./scripts/docker-build.sh -p linux/arm64 -l

# Build and push to registry
./scripts/docker-build.sh -t v1.0.0 -P

# Build with registry cache
./scripts/docker-build.sh -t latest -P -c
```

### Environment Variables

| Variable               | Default                    | Description              |
|------------------------|----------------------------|--------------------------|
| `YUNT_DATABASE_DSN`    | `/var/lib/yunt/yunt.db`    | Database connection      |
| `YUNT_LOGGING_OUTPUT`  | `stdout`                   | Log output destination   |
| `YUNT_LOGGING_FORMAT`  | `json`                     | Log format (json/text)   |
| `YUNT_SMTP_PORT`       | `1025`                     | SMTP server port         |
| `YUNT_IMAP_PORT`       | `1143`                     | IMAP server port         |
| `YUNT_API_PORT`        | `8025`                     | Web UI/API port          |

### Volumes

| Path               | Description                             |
|--------------------|-----------------------------------------|
| `/var/lib/yunt`    | Data directory (database, attachments)  |
| `/etc/yunt`        | Configuration files                     |

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Clean build artifacts
make clean
```

## License

MIT License - see LICENSE file for details.
