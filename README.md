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

### Prerequisites

- Go 1.22 or higher
- Make (optional, for build commands)

### Build

```bash
# Build the binary
make build

# Or using Go directly
go build -o bin/yunt ./cmd/yunt
```

### Run

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
