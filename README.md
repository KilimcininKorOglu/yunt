# Yunt - Development Mail Server

Yunt is a lightweight, powerful mail server written in Go, designed for developers and test environments. The name comes from the Gokturk Turkish word for "horse" -- just as mounted couriers carried letters, Yunt safely delivers your emails.

Point your application's SMTP settings to `localhost:1025` and every outgoing email is captured by Yunt instead of reaching real users. Browse captured emails through the Web UI at `localhost:8025`, compose and send emails internally between users, connect with any IMAP client like Thunderbird on `localhost:1143`, or access everything programmatically via the REST API.

## Features

| Feature            | Description                                                         |
|--------------------|---------------------------------------------------------------------|
| SMTP Server        | RFC 5321 compliant mail capture with relay support                  |
| IMAP Server        | RFC 3501 compliant client access (Thunderbird, Apple Mail, etc.)    |
| JMAP Server        | RFC 8620/8621/9610 compliant modern JSON API for mail and contacts  |
| Web UI             | MSN Hotmail 2006 nostalgic interface with compose, drafts, contacts |
| Internal Delivery  | Send mail between users without external relay                      |
| External Relay     | Forward mail to external SMTP servers when configured               |
| REST API           | Full-featured API for integration and automation                    |
| Draft System       | Save, edit, attach files, and send drafts                           |
| Email Signatures   | Per-user signatures auto-appended to outgoing mail                  |
| Multi-user         | Role-based access (admin, user, viewer) with isolated mailboxes     |
| Multi-database     | SQLite, PostgreSQL, MySQL, MongoDB                                  |
| Real-time Updates  | Server-Sent Events for instant notifications                        |
| Prometheus Metrics | /metrics endpoint with Grafana dashboard                            |
| Attachment Storage | Pluggable storage backends (database, filesystem, S3)               |
| Circuit Breaker    | Automatic database failure recovery                                 |

## Quick Start

### Using Docker (Recommended)

```bash
docker run -d \
  -p 1025:1025 \
  -p 1143:1143 \
  -p 8025:8025 \
  -v yunt-data:/var/lib/yunt \
  ghcr.io/kilimcininkorglu/yunt:latest
```

Open `http://localhost:8025` in your browser. Default credentials: `admin` / `admin123`.

### Using Docker Compose

```bash
# SQLite (default)
docker compose up -d

# PostgreSQL
docker compose -f docker-compose.yml -f docker-compose.postgres.yml up -d

# MySQL
docker compose -f docker-compose.yml -f docker-compose.mysql.yml up -d

# MongoDB
docker compose -f docker-compose.yml -f docker-compose.mongodb.yml up -d
```

### Building from Source

Requires Go 1.25+ and Node.js 22+ (for Web UI).

```bash
# Build everything (Web UI + Go binary)
make build-full

# Start the server
./bin/yunt serve

# Or with custom config
./bin/yunt serve --config configs/yunt.example.yaml
```

## Default Ports

| Service | Port | Protocol |
|---------|------|----------|
| SMTP    | 1025 | TCP      |
| IMAP    | 1143 | TCP      |
| Web UI  | 8025 | HTTP     |
| JMAP    | 8025 | HTTP     |

## Web UI

The Web UI uses a nostalgic MSN Hotmail 2006 design with full email functionality:

- **Inbox** -- View, search, filter, sort, star, and manage received emails
- **Compose** -- Send emails to internal users (and external via relay), with rich text editor, file attachments, and auto-save drafts
- **Message View** -- Read emails with HTML rendering, download attachments, reply, forward
- **Contacts** -- Auto-collected sender addresses from received emails
- **Calendar** -- Monthly calendar view
- **Settings** -- Profile, email signature, notification preferences, mailbox management, webhook configuration
- **User Management** -- Admin panel for creating and managing users (admin only)

## Mail Delivery

### Internal Delivery

Yunt supports internal mail delivery without any external relay configuration. Messages sent to addresses on local domains are delivered directly to the recipient's mailbox:

```
admin@localhost  -->  test@localhost     (delivered internally)
admin@localhost  -->  user@localhost     (delivered internally)
```

Local domains include `localhost` by default. Additional domains can be configured:

```yaml
server:
  domain: mail.example.com
  localDomains:
    - localhost
    - mail.example.com
    - dev.example.com
```

### External Relay

For sending to external addresses, configure an SMTP relay:

```yaml
smtp:
  allowRelay: true
  relayHost: smtp.gmail.com
  relayPort: 587
  relayUsername: your-email@gmail.com
  relayPassword: your-app-password
```

Mixed recipients are supported -- internal addresses are delivered directly, external addresses go through the relay.

## REST API

The API is available at `http://localhost:8025/api/v1/`. Authentication uses JWT tokens.

### Key Endpoints

| Method | Endpoint                          | Description                |
|--------|-----------------------------------|----------------------------|
| POST   | `/api/v1/auth/login`              | Authenticate and get token |
| GET    | `/api/v1/messages`                | List messages              |
| GET    | `/api/v1/messages/:id`            | Get message details        |
| POST   | `/api/v1/messages/send`           | Send a message             |
| POST   | `/api/v1/messages/draft`          | Save a draft               |
| PUT    | `/api/v1/messages/draft/:id`      | Update a draft             |
| POST   | `/api/v1/messages/draft/:id/send` | Send a draft               |
| GET    | `/api/v1/mailboxes`               | List mailboxes             |
| GET    | `/api/v1/stats`                   | Server statistics          |
| GET    | `/api/v1/users`                   | List users (admin)         |
| GET    | `/api/v1/events/stream`           | SSE event stream           |

### Sending a Message

```bash
# Login
TOKEN=$(curl -s -X POST http://localhost:8025/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.data.tokens.accessToken')

# Send to internal recipient
curl -X POST http://localhost:8025/api/v1/messages/send \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "fromMailboxId": "MAILBOX_ID",
    "to": ["test@localhost"],
    "subject": "Hello",
    "textBody": "This is a test message."
  }'
```

## Configuration

Yunt is configured via YAML file or environment variables with the `YUNT_` prefix.

### Environment Variables

| Variable                     | Default     | Description                   |
|------------------------------|-------------|-------------------------------|
| `YUNT_SERVER_DOMAIN`         | `localhost` | Primary mail domain           |
| `YUNT_DATABASE_DRIVER`       | `sqlite`    | sqlite/postgres/mysql/mongodb |
| `YUNT_DATABASE_DSN`          | `yunt.db`   | Database connection string    |
| `YUNT_SMTP_PORT`             | `1025`      | SMTP server port              |
| `YUNT_IMAP_PORT`             | `1143`      | IMAP server port              |
| `YUNT_API_PORT`              | `8025`      | Web UI and API port           |
| `YUNT_SMTP_ALLOW_RELAY`      | `false`     | Enable external relay         |
| `YUNT_API_ENABLE_RATE_LIMIT` | `false`     | Enable API rate limiting      |

See `configs/yunt.example.yaml` for all available options.

## IMAP Client Setup

Connect any IMAP client (Thunderbird, Apple Mail, Outlook) with these settings:

| Setting  | Value     |
|----------|-----------|
| Server   | localhost |
| Port     | 1143      |
| Security | None      |
| Username | admin     |
| Password | admin123  |

## JMAP

Yunt supports the JMAP protocol (JSON Meta Application Protocol) as a modern alternative to IMAP. JMAP clients can connect to the same HTTP port as the Web UI.

### Endpoints

| Endpoint                                       | Method | Description                |
|------------------------------------------------|--------|----------------------------|
| `/.well-known/jmap`                            | GET    | Session discovery          |
| `/jmap/api`                                    | POST   | API method calls           |
| `/jmap/upload/{accountId}/`                    | POST   | Binary blob upload         |
| `/jmap/download/{accountId}/{blobId}/{name}`   | GET    | Binary blob download       |
| `/jmap/eventsource`                            | GET    | Server-Sent Events push    |

### Supported Capabilities

| Capability                              | RFC   | Description           |
|-----------------------------------------|-------|-----------------------|
| `urn:ietf:params:jmap:core`             | 8620  | Core protocol         |
| `urn:ietf:params:jmap:mail`             | 8621  | Email, Mailbox, Thread|
| `urn:ietf:params:jmap:submission`       | 8621  | Email submission      |
| `urn:ietf:params:jmap:vacationresponse` | 8621  | Vacation auto-reply   |
| `urn:ietf:params:jmap:contacts`         | 9610  | Contacts, AddressBook |

### Methods (28 total)

Core/echo, Mailbox/get/changes/query, Thread/get/changes, Email/get/query/changes, Identity/get/changes/set, EmailSubmission/get/changes/query/set, VacationResponse/get/set, PushSubscription/get/set, AddressBook/get/changes/set, ContactCard/get/changes/query/set.

## Docker

### Images

Multi-platform Docker images (AMD64/ARM64) are published to GitHub Container Registry.

| Tag           | Description           |
|---------------|-----------------------|
| `latest`      | Latest stable release |
| `v1.2.3`      | Specific version      |
| `sha-abc1234` | Specific commit       |

### Volumes

| Path            | Description                            |
|-----------------|----------------------------------------|
| `/var/lib/yunt` | Data directory (database, attachments) |

## Development

```bash
make test           # Run all tests
make lint           # Run linter
make fmt            # Format code
make build-full     # Build Web UI + Go binary
make web-dev        # Web UI dev server (port 3000)
make web-check      # Type checking
make clean          # Clean build artifacts
```

### Running a Single Test

```bash
go test -v -run TestFunctionName ./internal/package/...
```

## Architecture

```
cmd/yunt/           CLI entry point (Cobra: serve, migrate, user, health, version)
internal/
  config/           Configuration (Viper: YAML + env vars)
  domain/           Pure domain models
  repository/       Data access (4 database drivers)
  service/          Business logic
  api/              REST API (Echo v4) + Web UI
  smtp/             SMTP server (go-smtp)
  imap/             IMAP server (go-imap/v2)
  parser/           MIME parser and message builder
  storage/          Attachment storage backends
web/                SvelteKit 2 + Svelte 5 (Web UI source)
webui/              Embedded static files (go:embed)
configs/            Configuration examples + Grafana dashboard
```

## License

MIT License - see LICENSE file for details.
