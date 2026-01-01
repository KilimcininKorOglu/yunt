# Yunt Reverse Proxy Configuration

This guide covers configuring reverse proxies for Yunt, including TLS termination, load balancing, and WebSocket support for the Web UI.

## Table of Contents

1. [Overview](#overview)
2. [Nginx Configuration](#nginx-configuration)
3. [Traefik Configuration](#traefik-configuration)
4. [HAProxy Configuration](#haproxy-configuration)
5. [Caddy Configuration](#caddy-configuration)
6. [TLS Certificates](#tls-certificates)
7. [WebSocket Support](#websocket-support)
8. [Load Balancing](#load-balancing)

## Overview

A reverse proxy provides several benefits for production deployments:

| Feature                | Benefit                                     |
|------------------------|---------------------------------------------|
| TLS Termination        | Handle HTTPS at the proxy layer             |
| Load Balancing         | Distribute traffic across instances         |
| Request Buffering      | Protect backend from slow clients           |
| Caching                | Reduce backend load for static content      |
| Security               | Hide internal architecture                  |
| Compression            | Reduce bandwidth usage                      |

### Architecture

```
Internet
    │
    ▼
┌─────────────────┐
│  Reverse Proxy  │
│  (Nginx/Traefik)│
│   Port 443      │
└────────┬────────┘
         │ HTTP
         ▼
┌─────────────────┐
│      Yunt       │
│   Port 8025     │
└─────────────────┘
```

### Port Mapping

| External Port | Internal Port | Protocol    | Service     |
|---------------|---------------|-------------|-------------|
| 443           | 8025          | HTTPS       | Web UI/API  |
| 25            | 1025          | SMTP        | Mail        |
| 587           | 1025          | SMTP/TLS    | Submission  |
| 993           | 1143          | IMAPS       | Mail client |

## Nginx Configuration

### Basic Setup

Create `/etc/nginx/sites-available/yunt.conf`:

```nginx
# Yunt Mail Server - Nginx Configuration
# 
# This configuration provides:
# - HTTPS termination with Let's Encrypt
# - HTTP to HTTPS redirect
# - WebSocket support for real-time updates
# - Security headers
# - Rate limiting

# Rate limiting zone
limit_req_zone $binary_remote_addr zone=yunt_limit:10m rate=10r/s;

# Upstream definition
upstream yunt_backend {
    server 127.0.0.1:8025;
    keepalive 32;
}

# HTTP to HTTPS redirect
server {
    listen 80;
    listen [::]:80;
    server_name mail.example.com;

    # Allow Let's Encrypt verification
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    # Redirect all other traffic to HTTPS
    location / {
        return 301 https://$host$request_uri;
    }
}

# HTTPS server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name mail.example.com;

    # TLS configuration
    ssl_certificate /etc/letsencrypt/live/mail.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mail.example.com/privkey.pem;
    
    # Modern TLS configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    
    # SSL session caching
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:50m;
    ssl_session_tickets off;
    
    # OCSP stapling
    ssl_stapling on;
    ssl_stapling_verify on;
    resolver 8.8.8.8 8.8.4.4 valid=300s;
    resolver_timeout 5s;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;

    # Logging
    access_log /var/log/nginx/yunt_access.log;
    error_log /var/log/nginx/yunt_error.log;

    # Client settings
    client_max_body_size 50M;
    client_body_timeout 60s;
    client_header_timeout 60s;

    # Proxy settings
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;
    proxy_buffering on;
    proxy_buffer_size 4k;
    proxy_buffers 8 16k;

    # Main location
    location / {
        limit_req zone=yunt_limit burst=20 nodelay;
        proxy_pass http://yunt_backend;
    }

    # API endpoints
    location /api/ {
        limit_req zone=yunt_limit burst=50 nodelay;
        proxy_pass http://yunt_backend;
    }

    # WebSocket support for real-time updates
    location /ws {
        proxy_pass http://yunt_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }

    # Health check endpoint (no rate limiting)
    location /health {
        proxy_pass http://yunt_backend;
        access_log off;
    }

    location /ready {
        proxy_pass http://yunt_backend;
        access_log off;
    }
}
```

Enable the configuration:

```bash
# Create symbolic link
sudo ln -s /etc/nginx/sites-available/yunt.conf /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Reload Nginx
sudo systemctl reload nginx
```

### Nginx with Docker

Create `docker-compose.yml`:

```yaml
services:
  nginx:
    image: nginx:alpine
    container_name: nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/yunt.conf:/etc/nginx/conf.d/default.conf:ro
      - ./certbot/conf:/etc/letsencrypt:ro
      - ./certbot/www:/var/www/certbot:ro
    depends_on:
      - yunt
    restart: unless-stopped

  yunt:
    image: ghcr.io/yunt/yunt:latest
    container_name: yunt
    expose:
      - "8025"
    volumes:
      - yunt-data:/var/lib/yunt
    environment:
      YUNT_DATABASE_DSN: /var/lib/yunt/yunt.db
      YUNT_AUTH_JWTSECRET: ${JWT_SECRET}
      YUNT_ADMIN_PASSWORD: ${ADMIN_PASSWORD}
    restart: unless-stopped

  certbot:
    image: certbot/certbot
    volumes:
      - ./certbot/conf:/etc/letsencrypt
      - ./certbot/www:/var/www/certbot
    entrypoint: "/bin/sh -c 'trap exit TERM; while :; do certbot renew; sleep 12h & wait $${!}; done;'"

volumes:
  yunt-data:
```

## Traefik Configuration

### Traefik with Docker Labels

Create `docker-compose.yml`:

```yaml
services:
  traefik:
    image: traefik:v3.0
    container_name: traefik
    command:
      - "--api.dashboard=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
      - "--certificatesresolvers.letsencrypt.acme.email=admin@example.com"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./letsencrypt:/letsencrypt
    labels:
      # Dashboard
      - "traefik.enable=true"
      - "traefik.http.routers.traefik.rule=Host(`traefik.example.com`)"
      - "traefik.http.routers.traefik.entrypoints=websecure"
      - "traefik.http.routers.traefik.tls.certresolver=letsencrypt"
      - "traefik.http.routers.traefik.service=api@internal"
      - "traefik.http.routers.traefik.middlewares=auth"
      - "traefik.http.middlewares.auth.basicauth.users=admin:$$apr1$$xyz..."
    restart: unless-stopped

  yunt:
    image: ghcr.io/yunt/yunt:latest
    container_name: yunt
    expose:
      - "8025"
    volumes:
      - yunt-data:/var/lib/yunt
    environment:
      YUNT_DATABASE_DSN: /var/lib/yunt/yunt.db
      YUNT_AUTH_JWTSECRET: ${JWT_SECRET}
      YUNT_ADMIN_PASSWORD: ${ADMIN_PASSWORD}
    labels:
      - "traefik.enable=true"
      # HTTP Router
      - "traefik.http.routers.yunt.rule=Host(`mail.example.com`)"
      - "traefik.http.routers.yunt.entrypoints=websecure"
      - "traefik.http.routers.yunt.tls.certresolver=letsencrypt"
      - "traefik.http.routers.yunt.service=yunt"
      # Service
      - "traefik.http.services.yunt.loadbalancer.server.port=8025"
      # Middlewares
      - "traefik.http.routers.yunt.middlewares=yunt-headers,yunt-ratelimit"
      - "traefik.http.middlewares.yunt-headers.headers.stsSeconds=63072000"
      - "traefik.http.middlewares.yunt-headers.headers.stsIncludeSubdomains=true"
      - "traefik.http.middlewares.yunt-headers.headers.frameDeny=true"
      - "traefik.http.middlewares.yunt-headers.headers.contentTypeNosniff=true"
      - "traefik.http.middlewares.yunt-ratelimit.ratelimit.average=100"
      - "traefik.http.middlewares.yunt-ratelimit.ratelimit.burst=50"
      # HTTP to HTTPS redirect
      - "traefik.http.routers.yunt-http.rule=Host(`mail.example.com`)"
      - "traefik.http.routers.yunt-http.entrypoints=web"
      - "traefik.http.routers.yunt-http.middlewares=redirect-to-https"
      - "traefik.http.middlewares.redirect-to-https.redirectscheme.scheme=https"
    restart: unless-stopped

volumes:
  yunt-data:
```

### Traefik Static Configuration

Alternative with static configuration file (`traefik.yml`):

```yaml
# traefik.yml
api:
  dashboard: true

entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
  websecure:
    address: ":443"

providers:
  docker:
    exposedByDefault: false
  file:
    filename: /etc/traefik/dynamic.yml

certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@example.com
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
```

Dynamic configuration (`dynamic.yml`):

```yaml
# dynamic.yml
http:
  routers:
    yunt:
      rule: "Host(`mail.example.com`)"
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt
      service: yunt
      middlewares:
        - security-headers
        - rate-limit

  services:
    yunt:
      loadBalancer:
        servers:
          - url: "http://yunt:8025"

  middlewares:
    security-headers:
      headers:
        stsSeconds: 63072000
        stsIncludeSubdomains: true
        frameDeny: true
        contentTypeNosniff: true
        browserXssFilter: true

    rate-limit:
      rateLimit:
        average: 100
        burst: 50
```

## HAProxy Configuration

Create `/etc/haproxy/haproxy.cfg`:

```haproxy
# Yunt Mail Server - HAProxy Configuration

global
    log /dev/log local0
    log /dev/log local1 notice
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy
    daemon

    # TLS settings
    ssl-default-bind-ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384
    ssl-default-bind-ciphersuites TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256
    ssl-default-bind-options ssl-min-ver TLSv1.2 no-tls-tickets

defaults
    log global
    mode http
    option httplog
    option dontlognull
    timeout connect 5000
    timeout client 50000
    timeout server 50000
    errorfile 400 /etc/haproxy/errors/400.http
    errorfile 403 /etc/haproxy/errors/403.http
    errorfile 408 /etc/haproxy/errors/408.http
    errorfile 500 /etc/haproxy/errors/500.http
    errorfile 502 /etc/haproxy/errors/502.http
    errorfile 503 /etc/haproxy/errors/503.http
    errorfile 504 /etc/haproxy/errors/504.http

# HTTP frontend (redirect to HTTPS)
frontend http_front
    bind *:80
    http-request redirect scheme https unless { ssl_fc }

# HTTPS frontend
frontend https_front
    bind *:443 ssl crt /etc/haproxy/certs/mail.example.com.pem
    
    # Security headers
    http-response set-header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload"
    http-response set-header X-Frame-Options "SAMEORIGIN"
    http-response set-header X-Content-Type-Options "nosniff"
    http-response set-header X-XSS-Protection "1; mode=block"
    
    # Rate limiting (requires stick-tables)
    stick-table type ip size 100k expire 30s store http_req_rate(10s)
    http-request track-sc0 src
    http-request deny deny_status 429 if { sc_http_req_rate(0) gt 100 }
    
    default_backend yunt_backend

# Backend
backend yunt_backend
    balance roundrobin
    option httpchk GET /health
    http-check expect status 200
    
    # Server definition
    server yunt1 127.0.0.1:8025 check inter 5s fall 3 rise 2

# Stats page
listen stats
    bind *:8404
    stats enable
    stats uri /stats
    stats realm HAProxy\ Statistics
    stats auth admin:password
    stats refresh 30s
```

## Caddy Configuration

Create `Caddyfile`:

```caddyfile
# Yunt Mail Server - Caddy Configuration
# Caddy automatically handles TLS certificates via Let's Encrypt

mail.example.com {
    # Reverse proxy to Yunt
    reverse_proxy yunt:8025 {
        # Health checking
        health_uri /health
        health_interval 30s
        health_timeout 5s
        
        # Headers
        header_up Host {host}
        header_up X-Real-IP {remote}
        header_up X-Forwarded-For {remote}
        header_up X-Forwarded-Proto {scheme}
    }

    # Security headers
    header {
        Strict-Transport-Security "max-age=63072000; includeSubDomains; preload"
        X-Frame-Options "SAMEORIGIN"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
        -Server
    }

    # Rate limiting
    rate_limit {
        zone dynamic {
            key {remote_host}
            events 100
            window 1m
        }
    }

    # Logging
    log {
        output file /var/log/caddy/yunt_access.log
        format json
    }

    # Enable compression
    encode gzip zstd
}
```

Docker Compose with Caddy:

```yaml
services:
  caddy:
    image: caddy:2-alpine
    container_name: caddy
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data
      - caddy-config:/config
    depends_on:
      - yunt
    restart: unless-stopped

  yunt:
    image: ghcr.io/yunt/yunt:latest
    container_name: yunt
    expose:
      - "8025"
    volumes:
      - yunt-data:/var/lib/yunt
    environment:
      YUNT_DATABASE_DSN: /var/lib/yunt/yunt.db
    restart: unless-stopped

volumes:
  yunt-data:
  caddy-data:
  caddy-config:
```

## TLS Certificates

### Let's Encrypt with Certbot

**Standalone mode (for initial setup):**

```bash
# Stop any service using port 80
sudo systemctl stop nginx

# Obtain certificate
sudo certbot certonly --standalone \
  -d mail.example.com \
  --email admin@example.com \
  --agree-tos \
  --non-interactive

# Start your reverse proxy
sudo systemctl start nginx
```

**Webroot mode (no downtime):**

```bash
# Create webroot directory
sudo mkdir -p /var/www/certbot

# Obtain certificate
sudo certbot certonly --webroot \
  -w /var/www/certbot \
  -d mail.example.com \
  --email admin@example.com \
  --agree-tos
```

**Automatic renewal:**

```bash
# Test renewal
sudo certbot renew --dry-run

# Add to crontab
echo "0 0 1 * * certbot renew --quiet --deploy-hook 'systemctl reload nginx'" | sudo crontab -
```

### Wildcard Certificates

For wildcard certificates, use DNS validation:

```bash
# Using Cloudflare DNS
sudo certbot certonly \
  --dns-cloudflare \
  --dns-cloudflare-credentials /etc/letsencrypt/cloudflare.ini \
  -d "*.example.com" \
  -d example.com
```

### Self-Signed Certificates (Development Only)

```bash
# Generate self-signed certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/ssl/private/yunt.key \
  -out /etc/ssl/certs/yunt.crt \
  -subj "/CN=mail.example.com"
```

## WebSocket Support

WebSocket connections require special handling for real-time features.

### Nginx WebSocket

```nginx
location /ws {
    proxy_pass http://yunt_backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_read_timeout 86400;
    proxy_send_timeout 86400;
}
```

### Traefik WebSocket

WebSocket is handled automatically by Traefik. For long-lived connections:

```yaml
labels:
  - "traefik.http.services.yunt.loadbalancer.server.scheme=http"
  - "traefik.http.middlewares.yunt-timeout.chain.middlewares=timeout"
  - "traefik.http.middlewares.timeout.headers.customrequestheaders.Connection=keep-alive"
```

## Load Balancing

### Multiple Yunt Instances

For high availability with PostgreSQL or MySQL backend:

**Nginx:**

```nginx
upstream yunt_backend {
    least_conn;
    server yunt1:8025 weight=5;
    server yunt2:8025 weight=5;
    server yunt3:8025 backup;
    keepalive 32;
}
```

**Traefik:**

```yaml
services:
  yunt:
    deploy:
      replicas: 3
    labels:
      - "traefik.http.services.yunt.loadbalancer.server.port=8025"
      - "traefik.http.services.yunt.loadbalancer.healthCheck.path=/health"
      - "traefik.http.services.yunt.loadbalancer.healthCheck.interval=10s"
```

**HAProxy:**

```haproxy
backend yunt_backend
    balance roundrobin
    option httpchk GET /health
    server yunt1 yunt1:8025 check weight 100
    server yunt2 yunt2:8025 check weight 100
    server yunt3 yunt3:8025 check weight 100 backup
```

### Sticky Sessions

If session affinity is required:

**Nginx:**

```nginx
upstream yunt_backend {
    ip_hash;
    server yunt1:8025;
    server yunt2:8025;
}
```

**Traefik:**

```yaml
labels:
  - "traefik.http.services.yunt.loadbalancer.sticky.cookie=true"
  - "traefik.http.services.yunt.loadbalancer.sticky.cookie.name=yunt_session"
```

## Next Steps

- [Deployment Guide](deployment.md) - Initial deployment setup
- [Production Guide](production.md) - Security and optimization
- [Backup and Restore](backup-restore.md) - Data protection
