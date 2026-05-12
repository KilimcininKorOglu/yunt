# Docker Deployment

## Quick Start

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

## Building the Image

```bash
# Build with default settings
docker build -t yunt .

# Build with version info
docker build \
  --build-arg VERSION=1.0.0 \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t yunt .
```

## Configuration

Environment variables override config file values. Use `YUNT_` prefix with underscore-delimited nesting.

| Variable                    | Default              | Description                |
|-----------------------------|----------------------|----------------------------|
| `YUNT_DATABASE_DSN`        | `/var/lib/yunt/yunt.db` | Database connection string |
| `YUNT_DATABASE_DRIVER`     | `sqlite`             | Database driver            |
| `YUNT_SMTP_PORT`           | `1025`               | SMTP listen port           |
| `YUNT_IMAP_PORT`           | `1143`               | IMAP listen port           |
| `YUNT_API_PORT`            | `8025`               | API/Web UI listen port     |
| `YUNT_LOGGING_OUTPUT`      | `stdout`             | Log output (stdout/file)   |
| `YUNT_LOGGING_FORMAT`      | `json`               | Log format (json/text)     |
| `YUNT_AUTH_JWT_SECRET`     | (generated)          | JWT signing secret         |

## Exposed Ports

| Port   | Protocol | Description    |
|--------|----------|----------------|
| `1025` | TCP      | SMTP server    |
| `1143` | TCP      | IMAP server    |
| `8025` | TCP      | API and Web UI |

## Volumes

| Path              | Description              |
|-------------------|--------------------------|
| `/var/lib/yunt`   | SQLite database and data |
| `/etc/yunt`       | Configuration files      |

## Health Check

The container includes a built-in health check that polls `http://localhost:8025/ready` every 30 seconds.

```bash
# Check container health
docker inspect --format='{{.State.Health.Status}}' yunt
```

## Multi-platform

The Docker image supports `linux/amd64` and `linux/arm64`. GitHub Actions builds multi-platform images automatically on push to `ghcr.io`.

## Compose File Structure

Each database backend has its own override file:

| File                         | Description                                |
|------------------------------|--------------------------------------------|
| `docker-compose.yml`        | Base config: Yunt with SQLite (standalone) |
| `docker-compose.mysql.yml`  | Adds MySQL container, overrides DB config  |
| `docker-compose.postgres.yml` | Adds PostgreSQL container, overrides DB config |
| `docker-compose.mongodb.yml`  | Adds MongoDB container, overrides DB config  |

Override files extend the base `yunt` service with `depends_on`, database-specific environment variables, and a separate DB container. Data is stored in `docker-data/` bind mounts.
