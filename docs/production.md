# Yunt Production Guide

This guide covers best practices for running Yunt in production environments, including security hardening, performance tuning, and operational procedures.

## Table of Contents

1. [Pre-Deployment Checklist](#pre-deployment-checklist)
2. [Security Hardening](#security-hardening)
3. [Environment Configuration](#environment-configuration)
4. [TLS/SSL Configuration](#tlsssl-configuration)
5. [Performance Tuning](#performance-tuning)
6. [Monitoring and Alerting](#monitoring-and-alerting)
7. [Logging Configuration](#logging-configuration)
8. [Troubleshooting](#troubleshooting)

## Pre-Deployment Checklist

Before deploying to production, ensure the following:

| Item                          | Status | Notes                                    |
|-------------------------------|--------|------------------------------------------|
| JWT secret configured         | [ ]    | Generate with `openssl rand -hex 32`     |
| Admin password set            | [ ]    | Use strong, unique password              |
| Database credentials secured  | [ ]    | Use secrets management                   |
| TLS certificates obtained     | [ ]    | Use Let's Encrypt or commercial CA       |
| Reverse proxy configured      | [ ]    | Nginx, Traefik, or similar               |
| Backup strategy defined       | [ ]    | Automated backups with verification      |
| Monitoring configured         | [ ]    | Health checks and alerting               |
| Firewall rules set            | [ ]    | Allow only necessary ports               |
| Log aggregation configured    | [ ]    | Central logging for analysis             |
| Resource limits defined       | [ ]    | CPU and memory limits                    |

## Security Hardening

### Authentication Configuration

Generate and configure secure authentication settings:

```bash
# Generate JWT secret (minimum 32 bytes)
openssl rand -hex 32

# Generate secure admin password
openssl rand -base64 24
```

Configure in `yunt.yaml`:

```yaml
auth:
  # JWT signing key - NEVER commit to version control
  jwtSecret: "${YUNT_AUTH_JWTSECRET}"
  
  # Token expiration (shorter is more secure)
  jwtExpiration: 1h
  refreshExpiration: 24h
  
  # BCrypt cost (12+ recommended for production)
  bcryptCost: 12
  
  # Lockout settings
  maxLoginAttempts: 5
  lockoutDuration: 30m
```

### Secrets Management

Never store secrets in configuration files. Use environment variables or secrets management:

**Docker Secrets:**

```yaml
services:
  yunt:
    secrets:
      - jwt_secret
      - db_password
    environment:
      YUNT_AUTH_JWTSECRET_FILE: /run/secrets/jwt_secret
      YUNT_DATABASE_PASSWORD_FILE: /run/secrets/db_password

secrets:
  jwt_secret:
    external: true
  db_password:
    external: true
```

**HashiCorp Vault:**

```bash
# Store secrets
vault kv put secret/yunt \
  jwt_secret="$(openssl rand -hex 32)" \
  db_password="secure-password"

# Retrieve in application
export YUNT_AUTH_JWTSECRET=$(vault kv get -field=jwt_secret secret/yunt)
```

### Network Security

**Restrict Network Access:**

```yaml
# docker-compose.yml
services:
  yunt:
    networks:
      - frontend  # Exposed to reverse proxy
      - backend   # Database access only
    
  postgres:
    networks:
      - backend   # Not exposed externally

networks:
  frontend:
  backend:
    internal: true
```

**Kubernetes Network Policy:**

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: yunt-policy
spec:
  podSelector:
    matchLabels:
      app: yunt
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress
      ports:
        - port: 8025
    - from:
        - podSelector: {}
      ports:
        - port: 1025
        - port: 1143
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: postgres
      ports:
        - port: 5432
```

### Container Security

Run containers with minimal privileges:

```yaml
services:
  yunt:
    user: "1000:1000"
    read_only: true
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    tmpfs:
      - /tmp
```

## Environment Configuration

### Production Environment Variables

Create a comprehensive `.env` file:

```bash
# =============================================================================
# Yunt Production Environment Configuration
# =============================================================================

# Server
YUNT_SERVER_NAME=mail.example.com
YUNT_SERVER_DOMAIN=example.com

# Database (PostgreSQL recommended for production)
YUNT_DATABASE_DRIVER=postgres
YUNT_DATABASE_HOST=postgres
YUNT_DATABASE_PORT=5432
YUNT_DATABASE_NAME=yunt
YUNT_DATABASE_USERNAME=yunt
YUNT_DATABASE_PASSWORD=     # Set via secrets management
YUNT_DATABASE_SSLMODE=require
YUNT_DATABASE_MAXOPENCONNS=25
YUNT_DATABASE_MAXIDLECONNS=10

# Authentication
YUNT_AUTH_JWTSECRET=        # Set via secrets management
YUNT_AUTH_JWTEXPIRATION=1h
YUNT_AUTH_BCRYPTCOST=12

# Admin (set secure password)
YUNT_ADMIN_USERNAME=admin
YUNT_ADMIN_PASSWORD=        # Set via secrets management
YUNT_ADMIN_EMAIL=admin@example.com

# SMTP
YUNT_SMTP_ENABLED=true
YUNT_SMTP_HOST=0.0.0.0
YUNT_SMTP_PORT=1025
YUNT_SMTP_AUTHREQUIRED=true
YUNT_SMTP_ALLOWRELAY=false

# IMAP
YUNT_IMAP_ENABLED=true
YUNT_IMAP_HOST=0.0.0.0
YUNT_IMAP_PORT=1143

# API
YUNT_API_ENABLED=true
YUNT_API_HOST=0.0.0.0
YUNT_API_PORT=8025
YUNT_API_RATELIMIT=100
YUNT_API_CORSALLOWEDORIGINS=https://mail.example.com

# Logging
YUNT_LOGGING_LEVEL=info
YUNT_LOGGING_FORMAT=json
YUNT_LOGGING_OUTPUT=stdout
```

### Resource Limits

Configure appropriate resource limits:

**Docker Compose:**

```yaml
services:
  yunt:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 256M
```

**Kubernetes:**

```yaml
resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: 2000m
    memory: 1Gi
```

## TLS/SSL Configuration

### Obtaining Certificates

**Using Let's Encrypt with Certbot:**

```bash
# Install certbot
sudo apt install certbot

# Obtain certificate (standalone mode)
sudo certbot certonly --standalone \
  -d mail.example.com \
  --email admin@example.com \
  --agree-tos

# Certificates will be in /etc/letsencrypt/live/mail.example.com/
```

**Using acme.sh:**

```bash
# Install acme.sh
curl https://get.acme.sh | sh

# Issue certificate with DNS validation
acme.sh --issue --dns dns_cf \
  -d mail.example.com \
  --key-file /etc/yunt/tls/key.pem \
  --fullchain-file /etc/yunt/tls/cert.pem
```

### Direct TLS Configuration

Configure TLS directly in Yunt (without reverse proxy):

```yaml
smtp:
  tls:
    enabled: true
    certFile: /etc/yunt/tls/cert.pem
    keyFile: /etc/yunt/tls/key.pem
    startTLS: true

imap:
  tls:
    enabled: true
    certFile: /etc/yunt/tls/cert.pem
    keyFile: /etc/yunt/tls/key.pem
    startTLS: true

api:
  tls:
    enabled: true
    certFile: /etc/yunt/tls/cert.pem
    keyFile: /etc/yunt/tls/key.pem
```

**Docker with TLS:**

```yaml
services:
  yunt:
    volumes:
      - /etc/letsencrypt/live/mail.example.com:/etc/yunt/tls:ro
    environment:
      YUNT_SMTP_TLS_ENABLED: "true"
      YUNT_SMTP_TLS_CERTFILE: /etc/yunt/tls/fullchain.pem
      YUNT_SMTP_TLS_KEYFILE: /etc/yunt/tls/privkey.pem
      YUNT_IMAP_TLS_ENABLED: "true"
      YUNT_IMAP_TLS_CERTFILE: /etc/yunt/tls/fullchain.pem
      YUNT_IMAP_TLS_KEYFILE: /etc/yunt/tls/privkey.pem
      YUNT_API_TLS_ENABLED: "true"
      YUNT_API_TLS_CERTFILE: /etc/yunt/tls/fullchain.pem
      YUNT_API_TLS_KEYFILE: /etc/yunt/tls/privkey.pem
```

### Certificate Renewal

Set up automatic certificate renewal:

```bash
# Create renewal script
cat > /etc/cron.daily/yunt-cert-renew << 'EOF'
#!/bin/bash
certbot renew --quiet --deploy-hook "docker-compose -f /opt/yunt/docker-compose.yml restart yunt"
EOF

chmod +x /etc/cron.daily/yunt-cert-renew
```

## Performance Tuning

### Database Optimization

**PostgreSQL:**

```yaml
database:
  driver: postgres
  maxOpenConns: 50
  maxIdleConns: 25
  connMaxLifetime: 30m
  connMaxIdleTime: 5m
```

PostgreSQL server settings (`postgresql.conf`):

```ini
# Memory
shared_buffers = 256MB
effective_cache_size = 768MB
work_mem = 16MB
maintenance_work_mem = 128MB

# Connections
max_connections = 100

# Performance
random_page_cost = 1.1
effective_io_concurrency = 200

# Logging
log_min_duration_statement = 500
```

**MySQL:**

```yaml
database:
  driver: mysql
  maxOpenConns: 50
  maxIdleConns: 25
```

MySQL server settings (`my.cnf`):

```ini
[mysqld]
innodb_buffer_pool_size = 256M
innodb_log_file_size = 64M
innodb_flush_log_at_trx_commit = 2
max_connections = 100
```

### Connection Pooling

For high-traffic deployments, use external connection pooling:

**PgBouncer for PostgreSQL:**

```ini
[databases]
yunt = host=postgres port=5432 dbname=yunt

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
auth_type = md5
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
```

### Caching

Enable HTTP caching with your reverse proxy:

```nginx
location /api/v1/messages {
    proxy_pass http://yunt:8025;
    proxy_cache api_cache;
    proxy_cache_valid 200 1m;
    proxy_cache_key $request_uri$http_authorization;
}
```

## Monitoring and Alerting

### Health Endpoints

Yunt provides health check endpoints:

| Endpoint       | Description                              |
|----------------|------------------------------------------|
| `/health`      | Basic liveness check                     |
| `/ready`       | Readiness check (all services)           |
| `/api/v1/health` | Detailed health with component status  |

### Prometheus Metrics

If metrics are enabled, configure Prometheus scraping:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'yunt'
    static_configs:
      - targets: ['yunt:8025']
    metrics_path: /metrics
```

### Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: yunt
    rules:
      - alert: YuntDown
        expr: up{job="yunt"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Yunt mail server is down"

      - alert: YuntHighLatency
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{job="yunt"}[5m])) > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High API latency detected"

      - alert: YuntHighErrorRate
        expr: rate(http_requests_total{job="yunt",status=~"5.."}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
```

### External Monitoring

Configure external health checks:

```bash
# Uptime check script
#!/bin/bash
if ! curl -sf http://localhost:8025/ready > /dev/null; then
    echo "CRITICAL: Yunt is not ready" >&2
    exit 2
fi
echo "OK: Yunt is healthy"
exit 0
```

## Logging Configuration

### JSON Logging (Recommended)

```yaml
logging:
  level: info
  format: json
  output: stdout
  includeCaller: false
```

### Log Aggregation

**Docker with Fluentd:**

```yaml
services:
  yunt:
    logging:
      driver: fluentd
      options:
        fluentd-address: localhost:24224
        tag: yunt.{{.Name}}
```

**Docker with Loki:**

```yaml
services:
  yunt:
    logging:
      driver: loki
      options:
        loki-url: http://loki:3100/loki/api/v1/push
        loki-batch-size: "400"
```

### Log Rotation

For file-based logging:

```yaml
logging:
  output: file
  filePath: /var/log/yunt/yunt.log
  maxSize: 100      # MB
  maxBackups: 5
  maxAge: 30        # days
  compress: true
```

Or use logrotate:

```
/var/log/yunt/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 640 yunt yunt
    postrotate
        systemctl reload yunt
    endscript
}
```

## Troubleshooting

### Common Issues

**Container fails to start:**

```bash
# Check logs
docker logs yunt

# Common causes:
# - Missing environment variables
# - Database connection failure
# - Port already in use
```

**Database connection errors:**

```bash
# Test database connectivity
docker exec yunt curl -sf http://localhost:8025/health | jq .database

# PostgreSQL diagnostics
docker exec postgres pg_isready -U yunt -d yunt
```

**TLS certificate issues:**

```bash
# Verify certificate
openssl x509 -in /etc/yunt/tls/cert.pem -text -noout

# Test TLS connection
openssl s_client -connect mail.example.com:465 -starttls smtp
```

**Performance issues:**

```bash
# Check resource usage
docker stats yunt

# Database slow queries
docker exec postgres psql -U yunt -c "SELECT * FROM pg_stat_activity WHERE state = 'active';"
```

### Debug Mode

Enable debug logging temporarily:

```bash
# Docker
docker exec -it yunt sh -c 'export YUNT_LOGGING_LEVEL=debug && kill -HUP 1'

# Systemd
systemctl edit yunt --runtime
# Add: Environment=YUNT_LOGGING_LEVEL=debug
systemctl restart yunt
```

### Support Diagnostics

Gather diagnostics for support:

```bash
#!/bin/bash
# collect-diagnostics.sh

mkdir -p /tmp/yunt-diag
cd /tmp/yunt-diag

# System info
uname -a > system.txt
docker version >> system.txt 2>&1

# Container info
docker inspect yunt > container.json 2>&1
docker logs --tail 1000 yunt > logs.txt 2>&1

# Health check
curl -s http://localhost:8025/health > health.json 2>&1
curl -s http://localhost:8025/ready > ready.json 2>&1

# Create archive
tar czf yunt-diagnostics-$(date +%Y%m%d).tar.gz *
echo "Diagnostics saved to: /tmp/yunt-diag/yunt-diagnostics-$(date +%Y%m%d).tar.gz"
```

## Next Steps

- [Reverse Proxy Setup](reverse-proxy.md) - Configure Nginx or Traefik
- [Backup and Restore](backup-restore.md) - Data protection procedures
- [Performance Guide](performance.md) - Database optimization
