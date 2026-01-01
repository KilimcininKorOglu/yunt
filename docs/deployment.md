# Yunt Deployment Guide

This guide covers deploying Yunt mail server in various environments, from development to production.

## Table of Contents

1. [Deployment Overview](#deployment-overview)
2. [Docker Deployment](#docker-deployment)
3. [Docker Compose](#docker-compose)
4. [Kubernetes Deployment](#kubernetes-deployment)
5. [Binary Deployment](#binary-deployment)
6. [Database Selection](#database-selection)
7. [Port Configuration](#port-configuration)

## Deployment Overview

Yunt supports multiple deployment methods:

| Method           | Best For                          | Complexity |
|------------------|-----------------------------------|------------|
| Docker           | Quick setup, single server        | Low        |
| Docker Compose   | Development, small teams          | Low        |
| Kubernetes       | Production, scalability           | Medium     |
| Binary           | Custom environments, edge cases   | Medium     |

### Architecture Overview

```
                    ┌─────────────────────────────────────────┐
                    │            Reverse Proxy                │
                    │     (Nginx/Traefik/HAProxy)             │
                    │         TLS Termination                 │
                    └──────────────┬──────────────────────────┘
                                   │
          ┌────────────────────────┼────────────────────────┐
          │                        │                        │
    ┌─────▼─────┐           ┌──────▼──────┐          ┌──────▼──────┐
    │   SMTP    │           │   Web UI    │          │    IMAP     │
    │  :1025    │           │    :8025    │          │   :1143     │
    └─────┬─────┘           └──────┬──────┘          └──────┬──────┘
          │                        │                        │
          └────────────────────────┼────────────────────────┘
                                   │
                          ┌────────▼────────┐
                          │    Database     │
                          │ (SQLite/PG/MySQL)│
                          └─────────────────┘
```

## Docker Deployment

### Quick Start

```bash
# Pull the latest image
docker pull ghcr.io/yunt/yunt:latest

# Run with default settings (SQLite)
docker run -d \
  --name yunt \
  -p 1025:1025 \
  -p 1143:1143 \
  -p 8025:8025 \
  ghcr.io/yunt/yunt:latest
```

### With Persistent Storage

```bash
# Create data volume
docker volume create yunt-data

# Run with persistence
docker run -d \
  --name yunt \
  -p 1025:1025 \
  -p 1143:1143 \
  -p 8025:8025 \
  -v yunt-data:/var/lib/yunt \
  -e YUNT_AUTH_JWTSECRET="$(openssl rand -hex 32)" \
  -e YUNT_ADMIN_PASSWORD="secure-password-here" \
  --restart unless-stopped \
  ghcr.io/yunt/yunt:latest
```

### With Custom Configuration

```bash
# Create configuration directory
mkdir -p /etc/yunt

# Copy example configuration
cp yunt.example.yaml /etc/yunt/yunt.yaml

# Edit configuration
vim /etc/yunt/yunt.yaml

# Run with config file
docker run -d \
  --name yunt \
  -p 1025:1025 \
  -p 1143:1143 \
  -p 8025:8025 \
  -v yunt-data:/var/lib/yunt \
  -v /etc/yunt:/etc/yunt:ro \
  --restart unless-stopped \
  ghcr.io/yunt/yunt:latest \
  serve --config /etc/yunt/yunt.yaml
```

### Docker Image Tags

| Tag            | Description                                    |
|----------------|------------------------------------------------|
| `latest`       | Latest stable release                          |
| `v1.2.3`       | Specific version                               |
| `1.2`          | Major.minor (tracks latest patch)              |
| `sha-abc1234`  | Specific commit SHA                            |

### Container Health Checks

The Docker image includes built-in health checks:

```bash
# Check container health
docker inspect --format='{{.State.Health.Status}}' yunt

# View health check logs
docker inspect --format='{{json .State.Health}}' yunt | jq
```

## Docker Compose

### Basic Setup (SQLite)

Create `docker-compose.yml`:

```yaml
services:
  yunt:
    image: ghcr.io/yunt/yunt:latest
    container_name: yunt
    ports:
      - "1025:1025"  # SMTP
      - "1143:1143"  # IMAP
      - "8025:8025"  # Web UI
    volumes:
      - yunt-data:/var/lib/yunt
    environment:
      YUNT_DATABASE_DRIVER: sqlite
      YUNT_DATABASE_DSN: /var/lib/yunt/yunt.db
      YUNT_AUTH_JWTSECRET: "your-secret-key-here"
      YUNT_ADMIN_PASSWORD: "secure-password"
      YUNT_LOGGING_LEVEL: info
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8025/ready"]
      interval: 30s
      timeout: 5s
      retries: 3

volumes:
  yunt-data:
```

Start the service:

```bash
docker-compose up -d
```

### With PostgreSQL

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: yunt-postgres
    environment:
      POSTGRES_USER: yunt
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-yunt_secret}
      POSTGRES_DB: yunt
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U yunt -d yunt"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  yunt:
    image: ghcr.io/yunt/yunt:latest
    container_name: yunt
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "1025:1025"
      - "1143:1143"
      - "8025:8025"
    volumes:
      - yunt-data:/var/lib/yunt
    environment:
      YUNT_DATABASE_DRIVER: postgres
      YUNT_DATABASE_HOST: postgres
      YUNT_DATABASE_PORT: "5432"
      YUNT_DATABASE_NAME: yunt
      YUNT_DATABASE_USERNAME: yunt
      YUNT_DATABASE_PASSWORD: ${POSTGRES_PASSWORD:-yunt_secret}
      YUNT_DATABASE_AUTOMIGRATE: "true"
      YUNT_AUTH_JWTSECRET: ${JWT_SECRET}
      YUNT_ADMIN_PASSWORD: ${ADMIN_PASSWORD}
    restart: unless-stopped

volumes:
  postgres-data:
  yunt-data:
```

### With MySQL

```yaml
services:
  mysql:
    image: mysql:8.0
    container_name: yunt-mysql
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-root_secret}
      MYSQL_DATABASE: yunt
      MYSQL_USER: yunt
      MYSQL_PASSWORD: ${MYSQL_PASSWORD:-yunt_secret}
    command:
      - --character-set-server=utf8mb4
      - --collation-server=utf8mb4_unicode_ci
    volumes:
      - mysql-data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  yunt:
    image: ghcr.io/yunt/yunt:latest
    container_name: yunt
    depends_on:
      mysql:
        condition: service_healthy
    ports:
      - "1025:1025"
      - "1143:1143"
      - "8025:8025"
    volumes:
      - yunt-data:/var/lib/yunt
    environment:
      YUNT_DATABASE_DRIVER: mysql
      YUNT_DATABASE_HOST: mysql
      YUNT_DATABASE_PORT: "3306"
      YUNT_DATABASE_NAME: yunt
      YUNT_DATABASE_USERNAME: yunt
      YUNT_DATABASE_PASSWORD: ${MYSQL_PASSWORD:-yunt_secret}
      YUNT_DATABASE_AUTOMIGRATE: "true"
      YUNT_AUTH_JWTSECRET: ${JWT_SECRET}
      YUNT_ADMIN_PASSWORD: ${ADMIN_PASSWORD}
    restart: unless-stopped

volumes:
  mysql-data:
  yunt-data:
```

### Using Environment File

Create `.env`:

```bash
# Database
POSTGRES_PASSWORD=secure-db-password
MYSQL_PASSWORD=secure-db-password
MYSQL_ROOT_PASSWORD=secure-root-password

# Yunt
JWT_SECRET=your-jwt-secret-key-here-min-32-chars
ADMIN_PASSWORD=your-admin-password
```

Start with environment file:

```bash
docker-compose --env-file .env up -d
```

## Kubernetes Deployment

Yunt provides Kubernetes manifests in the `deployments/kubernetes/` directory.

### Quick Start

```bash
# Create namespace
kubectl create namespace yunt

# Apply all manifests
kubectl apply -f deployments/kubernetes/ -n yunt

# Verify deployment
kubectl get pods -n yunt
```

### Required Steps

1. **Configure Secrets**: Edit `secret.yaml` with secure values
2. **Configure Ingress**: Update `ingress.yaml` with your domain
3. **Configure Storage**: Adjust `pvc.yaml` for your storage class

For detailed Kubernetes deployment instructions, see `deployments/kubernetes/README.md`.

## Binary Deployment

### Download Binary

```bash
# Download latest release
curl -LO https://github.com/yunt/yunt/releases/latest/download/yunt-linux-amd64
chmod +x yunt-linux-amd64
mv yunt-linux-amd64 /usr/local/bin/yunt
```

### Build from Source

```bash
# Prerequisites: Go 1.22+, Make
git clone https://github.com/yunt/yunt.git
cd yunt
make build
```

### Create System User

```bash
# Create dedicated user
sudo useradd -r -s /sbin/nologin -d /var/lib/yunt yunt

# Create directories
sudo mkdir -p /var/lib/yunt /etc/yunt /var/log/yunt
sudo chown -R yunt:yunt /var/lib/yunt /var/log/yunt
```

### Systemd Service

Create `/etc/systemd/system/yunt.service`:

```ini
[Unit]
Description=Yunt Mail Server
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=yunt
Group=yunt
ExecStart=/usr/local/bin/yunt serve --config /etc/yunt/yunt.yaml
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/yunt /var/log/yunt
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable yunt
sudo systemctl start yunt
```

## Database Selection

### Choosing a Database

| Database   | Recommended For                           | Max Volume        |
|------------|-------------------------------------------|-------------------|
| SQLite     | Development, small deployments            | <10K messages/day |
| PostgreSQL | Production, high concurrency              | 100K+ messages/day|
| MySQL      | Production, read-heavy workloads          | 100K+ messages/day|
| MongoDB    | Flexible schemas, document storage        | 100K+ messages/day|

### SQLite Configuration

Best for single-server deployments:

```yaml
database:
  driver: sqlite
  dsn: /var/lib/yunt/yunt.db
  maxOpenConns: 1
  maxIdleConns: 1
```

### PostgreSQL Configuration

Recommended for production:

```yaml
database:
  driver: postgres
  host: localhost
  port: 5432
  name: yunt
  username: yunt
  password: secure-password
  sslMode: require
  maxOpenConns: 25
  maxIdleConns: 10
```

### MySQL Configuration

```yaml
database:
  driver: mysql
  host: localhost
  port: 3306
  name: yunt
  username: yunt
  password: secure-password
  maxOpenConns: 25
  maxIdleConns: 10
```

## Port Configuration

### Default Ports

| Service | Default Port | Standard Port | Description        |
|---------|--------------|---------------|--------------------|
| SMTP    | 1025         | 25, 587       | Mail submission    |
| IMAP    | 1143         | 143, 993      | Mail retrieval     |
| Web UI  | 8025         | 80, 443       | Admin interface    |

### Using Standard Ports

Standard ports (25, 143, etc.) require root privileges. Use a reverse proxy or capabilities:

```bash
# Using capabilities (recommended)
sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/yunt

# Or configure via environment
YUNT_SMTP_PORT=25
YUNT_IMAP_PORT=143
YUNT_API_PORT=443
```

### Firewall Configuration

```bash
# UFW
sudo ufw allow 1025/tcp  # SMTP
sudo ufw allow 1143/tcp  # IMAP
sudo ufw allow 8025/tcp  # Web UI

# firewalld
sudo firewall-cmd --permanent --add-port=1025/tcp
sudo firewall-cmd --permanent --add-port=1143/tcp
sudo firewall-cmd --permanent --add-port=8025/tcp
sudo firewall-cmd --reload
```

## Next Steps

- [Production Configuration](production.md) - Security hardening and optimization
- [Reverse Proxy Setup](reverse-proxy.md) - TLS termination and routing
- [Backup and Restore](backup-restore.md) - Data protection procedures
