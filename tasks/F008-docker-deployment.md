# Feature 8: Docker & Deployment

**Feature ID:** F008  
**Priority:** P2 - HIGH  
**Target Version:** v1.0.0  
**Estimated Duration:** 1-2 weeks  
**Status:** NOT_STARTED

## Overview

This feature provides comprehensive Docker support and deployment configurations for Yunt, making it easy to run in containerized environments. It includes a multi-stage Dockerfile for building optimized images, Docker Compose configurations for various deployment scenarios (standalone, with PostgreSQL, with MySQL, with MongoDB), health checks, and documentation for production deployments.

The Docker support enables developers to run Yunt with a single command, supports various database backends through Docker Compose profiles, and provides production-ready configurations with proper security, logging, and persistence.

## Goals

- Create optimized multi-stage Dockerfile
- Build Docker Compose configurations for all database backends
- Implement health checks for container orchestration
- Create production-ready deployment examples
- Document Docker deployment procedures
- Support both development and production configurations
- Provide Kubernetes deployment manifests (optional)
- Create release automation for multi-platform images

## Success Criteria

- [ ] All tasks completed
- [ ] All tests passing
- [ ] Docker image builds successfully
- [ ] Docker Compose launches all configurations
- [ ] Health checks work correctly
- [ ] Images are optimized (< 50MB)
- [ ] Multi-platform builds work (amd64, arm64)
- [ ] Documentation is complete
- [ ] Production deployment tested

## Tasks

### T059: Create Multi-stage Dockerfile

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1 day

#### Description

Create an optimized multi-stage Dockerfile that builds both the Web UI and Go binary, resulting in a minimal runtime image. Use Alpine Linux for small image size, include only necessary runtime dependencies, and follow Docker best practices.

#### Technical Details

- Use multi-stage build (builder + runtime)
- Builder stage: golang:1.22-alpine
- Install Node.js and npm in builder for Web UI
- Build Web UI first (npm ci && npm run build)
- Build Go binary with CGO_ENABLED=1 for SQLite
- Use static linking where possible
- Runtime stage: alpine:3.19
- Install only runtime dependencies (ca-certificates, tzdata)
- Copy binary and embedded Web UI
- Run as non-root user (yunt)
- Set up volume mounts for data persistence
- Expose ports 1025 (SMTP), 1143 (IMAP), 8025 (HTTP)
- Include HEALTHCHECK instruction
- Set proper labels (version, description, etc.)

#### Files to Touch

- `Dockerfile` (new)
- `.dockerignore` (new)

#### Dependencies

- T001 (project structure)
- T049 (Web UI embedding)

#### Success Criteria

- [ ] Docker image builds successfully
- [ ] Image size < 50MB
- [ ] Web UI embedded correctly
- [ ] Binary runs in container
- [ ] All services start correctly
- [ ] Health check passes
- [ ] Runs as non-root user
- [ ] Volumes work for data persistence

---

### T060: Create Docker Compose for Standalone Mode

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 0.5 days

#### Description

Create a Docker Compose configuration for running Yunt in standalone mode with SQLite database. This is the simplest deployment option for development and testing.

#### Technical Details

- Create docker-compose.yml
- Define yunt service
- Use build context or image
- Map all ports (1025, 1143, 8025)
- Create named volume for data persistence
- Set environment variables for configuration
- Configure restart policy (unless-stopped)
- Set up logging configuration
- Create default network
- Include example .env file
- Document usage in comments

#### Files to Touch

- `docker-compose.yml` (new)
- `.env.example` (new)

#### Dependencies

- T059 (Dockerfile)

#### Success Criteria

- [ ] `docker-compose up -d` starts Yunt
- [ ] All ports accessible from host
- [ ] Data persists across restarts
- [ ] Configuration via .env works
- [ ] Logs visible with `docker-compose logs`
- [ ] `docker-compose down` stops cleanly

---

### T061: Create Docker Compose with PostgreSQL

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 0.5 days

#### Description

Create Docker Compose configuration with PostgreSQL database using Docker Compose profiles. This provides a production-like setup for development and testing.

#### Technical Details

- Add postgres service to docker-compose.yml
- Use official postgres:16-alpine image
- Create named volume for postgres data
- Set PostgreSQL environment variables
- Configure Yunt to use PostgreSQL
- Set up service dependencies (yunt depends_on postgres)
- Use health check for postgres readiness
- Create postgres profile for activation
- Document profile usage
- Include initialization scripts if needed

#### Files to Touch

- `docker-compose.yml` (update)
- `configs/docker/postgres.env` (new)

#### Dependencies

- T060 (base Docker Compose)
- T050-T051 (PostgreSQL repository)

#### Success Criteria

- [ ] `docker-compose --profile postgres up -d` starts both services
- [ ] Yunt connects to PostgreSQL
- [ ] Data persists in postgres volume
- [ ] PostgreSQL initialization works
- [ ] Migrations run automatically
- [ ] Health checks work

---

### T062: Create Docker Compose with MySQL

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 0.5 days

#### Description

Create Docker Compose configuration with MySQL database using Docker Compose profiles. Provide an alternative to PostgreSQL for users with MySQL preference or existing infrastructure.

#### Technical Details

- Add mysql service to docker-compose.yml
- Use official mysql:8.0 image
- Create named volume for mysql data
- Set MySQL environment variables (MYSQL_ROOT_PASSWORD, MYSQL_DATABASE, etc.)
- Configure Yunt to use MySQL
- Set up service dependencies
- Use health check for mysql readiness
- Create mysql profile for activation
- Set character set to utf8mb4
- Document profile usage

#### Files to Touch

- `docker-compose.yml` (update)
- `configs/docker/mysql.env` (new)

#### Dependencies

- T060 (base Docker Compose)
- T052-T053 (MySQL repository)

#### Success Criteria

- [ ] `docker-compose --profile mysql up -d` starts both services
- [ ] Yunt connects to MySQL
- [ ] Data persists in mysql volume
- [ ] UTF8MB4 encoding configured
- [ ] Migrations run automatically
- [ ] Health checks work

---

### T063: Create Docker Compose with MongoDB

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 0.5 days

#### Description

Create Docker Compose configuration with MongoDB database using Docker Compose profiles. Provide document database option for users who prefer MongoDB.

#### Technical Details

- Add mongodb service to docker-compose.yml
- Use official mongo:7 image
- Create named volume for mongodb data
- Set MongoDB environment variables
- Configure Yunt to use MongoDB
- Set up service dependencies
- Use health check for mongodb readiness
- Create mongodb profile for activation
- Initialize database and collections
- Document profile usage

#### Files to Touch

- `docker-compose.yml` (update)
- `configs/docker/mongodb.env` (new)

#### Dependencies

- T060 (base Docker Compose)
- T054-T055 (MongoDB repository)

#### Success Criteria

- [ ] `docker-compose --profile mongodb up -d` starts both services
- [ ] Yunt connects to MongoDB
- [ ] Data persists in mongodb volume
- [ ] Indexes created automatically
- [ ] Initialization works correctly
- [ ] Health checks work

---

### T064: Implement Container Health Checks

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 0.5 days

#### Description

Implement comprehensive health checks for Docker containers and Kubernetes liveness/readiness probes. Health checks should verify that all services (SMTP, IMAP, API) are running and database connectivity is working.

#### Technical Details

- Enhance /api/v1/health endpoint
- Check SMTP server status
- Check IMAP server status
- Check API server status
- Check database connectivity
- Return detailed health status
- Return appropriate HTTP status codes (200 OK, 503 Service Unavailable)
- Implement liveness probe (simple check)
- Implement readiness probe (full check)
- Add health check to Dockerfile
- Configure health check in Docker Compose
- Document health check endpoints

#### Files to Touch

- `internal/api/handlers/health.go` (update)
- `Dockerfile` (update)
- `docker-compose.yml` (update)

#### Dependencies

- T038 (system endpoints)
- T059 (Dockerfile)

#### Success Criteria

- [ ] Health endpoint returns detailed status
- [ ] Failed checks return 503 status
- [ ] Docker health check works
- [ ] Container marked unhealthy on failure
- [ ] All services verified in health check
- [ ] Database connectivity verified

---

### T065: Create Production Deployment Documentation

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Create comprehensive documentation for deploying Yunt in production environments. Cover Docker deployment, reverse proxy configuration (Nginx, Traefik), TLS certificates, backup procedures, and operational best practices.

#### Technical Details

- Document production deployment steps
- Provide Nginx reverse proxy configuration
- Provide Traefik reverse proxy configuration
- Document TLS certificate setup (Let's Encrypt)
- Document backup and restore procedures
- Document monitoring and logging
- Document security hardening
- Provide production configuration examples
- Document scaling considerations
- Document upgrade procedures
- Include troubleshooting guide

#### Files to Touch

- `docs/deployment.md` (new)
- `docs/production.md` (new)
- `docs/reverse-proxy.md` (new)
- `docs/backup-restore.md` (new)
- `examples/nginx/yunt.conf` (new)
- `examples/traefik/docker-compose.yml` (new)

#### Dependencies

- T059-T063 (Docker configurations)

#### Success Criteria

- [ ] Deployment documentation is complete
- [ ] Reverse proxy examples work
- [ ] TLS configuration documented
- [ ] Backup procedures documented
- [ ] Security hardening covered
- [ ] Troubleshooting guide helpful
- [ ] Production config examples provided

---

### T066: Create Kubernetes Deployment Manifests

**Status:** COMPLETED
**Priority:** P3  
**Estimated Effort:** 1.5 days

#### Description

Create Kubernetes deployment manifests for running Yunt in Kubernetes clusters. Include deployments, services, config maps, secrets, persistent volume claims, and ingress configurations.

#### Technical Details

- Create Deployment for Yunt
- Create Service for SMTP (type: LoadBalancer or NodePort)
- Create Service for IMAP (type: LoadBalancer or NodePort)
- Create Service for HTTP (type: ClusterIP)
- Create ConfigMap for configuration
- Create Secret for sensitive data (JWT secret, database passwords)
- Create PersistentVolumeClaim for data storage
- Create Ingress for HTTP access
- Configure liveness and readiness probes
- Set resource limits and requests
- Create separate manifests for each database
- Support Helm chart (optional)

#### Files to Touch

- `deployments/kubernetes/deployment.yaml` (new)
- `deployments/kubernetes/service-smtp.yaml` (new)
- `deployments/kubernetes/service-imap.yaml` (new)
- `deployments/kubernetes/service-http.yaml` (new)
- `deployments/kubernetes/configmap.yaml` (new)
- `deployments/kubernetes/secret.yaml` (new)
- `deployments/kubernetes/pvc.yaml` (new)
- `deployments/kubernetes/ingress.yaml` (new)
- `deployments/kubernetes/README.md` (new)

#### Dependencies

- T059 (Docker image)
- T064 (health checks)

#### Success Criteria

- [ ] Manifests deploy to Kubernetes successfully
- [ ] All services accessible
- [ ] ConfigMap configuration works
- [ ] Secrets mounted correctly
- [ ] Persistent storage works
- [ ] Ingress routes HTTP traffic
- [ ] Probes keep containers healthy
- [ ] Resource limits prevent runaway usage

---

### T067: Set Up Multi-platform Docker Builds

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Configure multi-platform Docker image builds for AMD64 and ARM64 architectures. Set up automated builds in CI/CD pipeline and publish to Docker Hub or GitHub Container Registry.

#### Technical Details

- Use Docker buildx for multi-platform builds
- Build for linux/amd64 and linux/arm64
- Configure GitHub Actions for automated builds
- Build on push to main branch
- Build on git tags (releases)
- Tag images with version number
- Tag images with 'latest' for main branch
- Push to Docker Hub and/or GitHub Container Registry
- Sign images for security (optional)
- Create build cache for faster builds
- Document build process

#### Files to Touch

- `.github/workflows/docker.yml` (new)
- `scripts/docker-build.sh` (new)
- `README.md` (update with Docker Hub link)

#### Dependencies

- T059 (Dockerfile)

#### Success Criteria

- [ ] Multi-platform builds work locally
- [ ] GitHub Actions builds automatically
- [ ] Images pushed to registry
- [ ] Both architectures available
- [ ] Version tags correct
- [ ] Latest tag updates on main
- [ ] Build cache improves speed
- [ ] Documentation complete

---

### T068: Create Release Automation

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Create automated release process that builds binaries for multiple platforms, creates GitHub releases, uploads artifacts, and publishes Docker images. Use GitHub Actions or similar CI/CD.

#### Technical Details

- Create release workflow triggered on git tags
- Build binaries for multiple platforms:
  - linux/amd64, linux/arm64
  - darwin/amd64, darwin/arm64
  - windows/amd64
- Build Web UI and embed in binaries
- Create release archives (tar.gz, zip)
- Generate checksums (SHA256)
- Create GitHub release
- Upload release artifacts
- Generate release notes from commits
- Trigger Docker image build
- Update documentation with new version

#### Files to Touch

- `.github/workflows/release.yml` (new)
- `scripts/release.sh` (update)
- `CHANGELOG.md` (new)

#### Dependencies

- T005 (build scripts)
- T067 (Docker builds)

#### Success Criteria

- [ ] Release workflow triggers on tags
- [ ] Binaries built for all platforms
- [ ] Archives created correctly
- [ ] Checksums generated
- [ ] GitHub release created
- [ ] Artifacts uploaded
- [ ] Docker images published
- [ ] Release notes generated

---

## Performance Targets

- Docker image size: < 50MB
- Container startup time: < 5 seconds
- Build time (multi-stage): < 5 minutes
- Multi-platform build: < 10 minutes
- Health check response: < 100ms
- Resource usage (idle): < 50MB RAM, < 5% CPU

## Risk Assessment

| Risk                            | Probability | Impact | Mitigation                                    |
|---------------------------------|-------------|--------|-----------------------------------------------|
| Large Docker image size         | Low         | Low    | Multi-stage builds, Alpine base               |
| Cross-platform build failures   | Medium      | Medium | Test on multiple architectures                |
| Database connection issues      | Medium      | Medium | Health checks, proper initialization order    |
| Volume permission issues        | Low         | Medium | Proper user configuration, documentation      |
| CI/CD pipeline failures         | Low         | High   | Comprehensive testing, retry mechanisms       |

## Notes

- Use Docker Compose profiles for flexible configurations
- Health checks critical for production reliability
- Multi-platform builds support ARM devices (Raspberry Pi, Apple Silicon)
- Consider using distroless images for even smaller size (future)
- Document resource requirements for each configuration
- Provide migration guide for upgrading between versions
- Consider implementing database backup sidecar containers
- Kubernetes manifests should follow best practices
- Release automation saves time and reduces errors
- Consider implementing automated security scanning of images
- Document environment variables clearly for configuration
- Provide production-ready docker-compose.yml with comments
- Consider implementing container monitoring (Prometheus metrics)
