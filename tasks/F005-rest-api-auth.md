# Feature 5: REST API & Authentication

**Feature ID:** F005  
**Priority:** P1 - CRITICAL  
**Target Version:** v0.4.0  
**Estimated Duration:** 2-3 weeks  
**Status:** NOT_STARTED

## Overview

This feature implements a comprehensive REST API that provides programmatic access to all Yunt functionality and serves as the backend for the Web UI. The API includes JWT-based authentication, role-based authorization (admin/user), and complete endpoints for managing users, mailboxes, messages, attachments, webhooks, and system settings. The implementation uses Echo framework for routing and middleware.

The REST API enables integration with external tools, automation of email testing workflows, and provides the foundation for the Web UI. Security is paramount, with proper authentication, authorization, CORS support, and rate limiting.

## Goals

- Implement comprehensive REST API covering all features
- Build JWT-based authentication with refresh tokens
- Implement role-based authorization (admin, user)
- Create endpoints for users, mailboxes, messages, attachments, webhooks
- Support pagination, filtering, and search
- Implement CORS for web browser access
- Add rate limiting and security controls
- Provide OpenAPI/Swagger documentation

## Success Criteria

- [ ] All tasks completed
- [ ] All tests passing
- [ ] API server accepts HTTP requests
- [ ] JWT authentication works correctly
- [ ] All CRUD operations functional
- [ ] Pagination and filtering work
- [ ] CORS allows Web UI access
- [ ] API documentation is complete
- [ ] Performance meets targets (< 50ms p95)

## Tasks

### T030: Set Up API Server and HTTP Router

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1 day

#### Description

Create the HTTP server setup using Echo framework. Implement server initialization, routing structure, graceful shutdown, and basic middleware (logging, recovery, CORS). Establish the foundation for API endpoints.

#### Technical Details

- Use `labstack/echo/v4` for HTTP server and routing
- Load API configuration from config system
- Implement server lifecycle (start, stop, graceful shutdown)
- Configure timeouts (read, write)
- Set up route groups for versioning (/api/v1)
- Implement recovery middleware for panic handling
- Implement request logging middleware
- Implement CORS middleware with configuration
- Create health check endpoint
- Return JSON responses with consistent format

#### Files to Touch

- `internal/api/server.go` (new)
- `internal/api/router.go` (new)
- `internal/api/response.go` (new)
- `internal/api/middleware/recovery.go` (new)
- `internal/api/middleware/logger.go` (new)
- `internal/api/middleware/cors.go` (new)

#### Dependencies

- T002 (configuration system)
- T003 (logging)
- T006 (Echo dependency)

#### Success Criteria

- [ ] API server starts on configured port
- [ ] Health check endpoint returns 200 OK
- [ ] CORS headers are set correctly
- [ ] Request logging captures all requests
- [ ] Panic recovery prevents server crashes
- [ ] Graceful shutdown works
- [ ] JSON responses follow consistent format

---

### T031: Implement JWT Authentication System

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Build a complete JWT-based authentication system with access tokens and refresh tokens. Implement login, logout, token refresh, and token validation middleware. Use secure token signing and validation.

#### Technical Details

- Use `golang-jwt/jwt/v5` for JWT handling
- Implement access tokens (short-lived, 24h)
- Implement refresh tokens (long-lived, 7 days)
- Store JWT secret in configuration
- Include user ID and role in token claims
- Implement token signing with HS256
- Implement token validation and parsing
- Create auth middleware for protected routes
- Handle token expiration gracefully
- Implement logout by invalidating tokens
- Store refresh tokens in database (optional)

#### Files to Touch

- `internal/service/auth.go` (new)
- `internal/api/middleware/auth.go` (new)
- `internal/api/handlers/auth.go` (new)
- `internal/domain/token.go` (new)

#### Dependencies

- T030 (API server foundation)
- T009 (user repository)
- T002 (JWT secret configuration)

#### Success Criteria

- [ ] Login endpoint returns valid JWT tokens
- [ ] Token validation middleware works correctly
- [ ] Expired tokens are rejected
- [ ] Invalid tokens are rejected
- [ ] Refresh token flow works
- [ ] Logout invalidates tokens
- [ ] Protected routes require authentication
- [ ] Token claims include correct user info

---

### T032: Implement User Management Endpoints

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Create REST endpoints for user management including listing users, creating users, updating users, and deleting users. Implement role-based authorization where admin-only operations are protected.

#### Technical Details

- Implement GET /api/v1/users (admin only)
- Implement POST /api/v1/users (admin only)
- Implement GET /api/v1/users/:id (admin or self)
- Implement PUT /api/v1/users/:id (admin or self)
- Implement DELETE /api/v1/users/:id (admin only)
- Implement GET /api/v1/auth/me (current user)
- Support pagination for user list
- Validate user input (email format, password strength)
- Hash passwords with bcrypt
- Prevent username/email duplicates
- Don't return password hashes in responses

#### Files to Touch

- `internal/api/handlers/users.go` (new)
- `internal/service/user.go` (new)
- `internal/api/middleware/rbac.go` (new)

#### Dependencies

- T031 (JWT authentication)
- T009 (user repository)

#### Success Criteria

- [ ] Admin can list all users
- [ ] Admin can create new users
- [ ] Users can view their own profile
- [ ] Users can update their own profile
- [ ] Admin can delete users
- [ ] Non-admin cannot access admin endpoints
- [ ] Input validation catches invalid data
- [ ] Password hashes never returned

---

### T033: Implement Mailbox Management Endpoints

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1.5 days

#### Description

Create REST endpoints for mailbox operations including listing mailboxes, creating custom mailboxes, renaming mailboxes, deleting mailboxes, and retrieving mailbox statistics. Users can only access their own mailboxes.

#### Technical Details

- Implement GET /api/v1/mailboxes (user's mailboxes)
- Implement POST /api/v1/mailboxes (create custom mailbox)
- Implement GET /api/v1/mailboxes/:id (mailbox details)
- Implement PUT /api/v1/mailboxes/:id (rename mailbox)
- Implement DELETE /api/v1/mailboxes/:id (custom mailboxes only)
- Implement GET /api/v1/mailboxes/:id/stats (message counts)
- Validate mailbox ownership
- Prevent deletion of system mailboxes
- Validate mailbox names
- Return mailbox statistics (total, unread, starred)

#### Files to Touch

- `internal/api/handlers/mailboxes.go` (new)
- `internal/service/mailbox.go` (new)

#### Dependencies

- T031 (authentication)
- T009 (mailbox repository)

#### Success Criteria

- [ ] Users can list their mailboxes
- [ ] Users can create custom mailboxes
- [ ] Users can rename mailboxes
- [ ] Users can delete custom mailboxes
- [ ] System mailboxes cannot be deleted
- [ ] Users cannot access other users' mailboxes
- [ ] Statistics are accurate
- [ ] Input validation works

---

### T034: Implement Message Management Endpoints

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 3 days

#### Description

Create comprehensive REST endpoints for message operations including listing messages with filters, retrieving message details, deleting messages, updating message flags, moving messages, and searching messages. Support various output formats (JSON, HTML, plain text, raw EML).

#### Technical Details

- Implement GET /api/v1/messages (list with filters)
- Implement GET /api/v1/messages/:id (message details)
- Implement DELETE /api/v1/messages/:id (delete message)
- Implement DELETE /api/v1/messages (bulk delete)
- Implement GET /api/v1/messages/:id/raw (EML format)
- Implement GET /api/v1/messages/:id/html (HTML body)
- Implement GET /api/v1/messages/:id/text (plain text)
- Implement PUT /api/v1/messages/:id/read (mark read)
- Implement PUT /api/v1/messages/:id/unread (mark unread)
- Implement PUT /api/v1/messages/:id/star (toggle star)
- Implement PUT /api/v1/messages/:id/move (move to mailbox)
- Support pagination (page, per_page)
- Support filtering (mailbox_id, unread, starred, from, to, subject, date range)
- Support sorting (received_at, subject, from)
- Validate message ownership

#### Files to Touch

- `internal/api/handlers/messages.go` (new)
- `internal/service/message.go` (update)

#### Dependencies

- T031 (authentication)
- T009 (message repository)
- T033 (mailbox validation)

#### Success Criteria

- [ ] Message list returns paginated results
- [ ] Filtering by all parameters works
- [ ] Sorting works correctly
- [ ] Message details include all fields
- [ ] HTML/text/raw formats returned correctly
- [ ] Flag operations update database
- [ ] Move operation changes mailbox
- [ ] Bulk delete removes multiple messages
- [ ] Users cannot access other users' messages

---

### T035: Implement Attachment Endpoints

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Create REST endpoints for attachment operations including listing message attachments and downloading individual attachments. Support proper content types and content disposition headers for browser downloads.

#### Technical Details

- Implement GET /api/v1/messages/:id/attachments (list)
- Implement GET /api/v1/messages/:id/attachments/:aid (download)
- Return correct Content-Type headers
- Return Content-Disposition for downloads
- Stream large attachments efficiently
- Support inline attachments
- Validate attachment ownership via message
- Return 404 for non-existent attachments

#### Files to Touch

- `internal/api/handlers/attachments.go` (new)

#### Dependencies

- T034 (message endpoints for validation)
- T009 (attachment repository)

#### Success Criteria

- [ ] Attachment list returns all attachments
- [ ] Download returns correct file content
- [ ] Content-Type header is correct
- [ ] Browser downloads file correctly
- [ ] Large files stream efficiently
- [ ] Users cannot access other users' attachments
- [ ] Missing attachments return 404

---

### T036: Implement Search Endpoints

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1.5 days

#### Description

Create search endpoints for finding messages with simple and advanced search criteria. Support full-text search on subject and body, as well as structured search by sender, recipient, date range, and flags.

#### Technical Details

- Implement GET /api/v1/search (simple search by query string)
- Implement POST /api/v1/search/advanced (advanced search)
- Support search parameters: term, from, to, subject, body, has_attachments, date range
- Support combining search criteria with AND logic
- Use repository search functionality
- Return paginated results
- Validate search input
- Optimize search queries for performance

#### Files to Touch

- `internal/api/handlers/search.go` (new)

#### Dependencies

- T034 (message endpoints)
- T009 (message repository with search)

#### Success Criteria

- [ ] Simple search finds messages by text
- [ ] Advanced search supports all criteria
- [ ] Search results are accurate
- [ ] Pagination works with search
- [ ] Performance is acceptable
- [ ] Invalid search parameters rejected
- [ ] Users only search their own messages

---

### T037: Implement Webhook Management Endpoints

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 2 days

#### Description

Create REST endpoints for webhook configuration including creating webhooks, listing webhooks, updating webhooks, deleting webhooks, and testing webhooks. Implement webhook dispatch service that triggers HTTP callbacks on message events.

#### Technical Details

- Implement GET /api/v1/webhooks (list webhooks)
- Implement POST /api/v1/webhooks (create webhook)
- Implement PUT /api/v1/webhooks/:id (update webhook)
- Implement DELETE /api/v1/webhooks/:id (delete webhook)
- Implement POST /api/v1/webhooks/:id/test (test webhook)
- Validate webhook URLs
- Support webhook events: message.received, message.deleted, user.created
- Implement webhook dispatch service
- Sign webhook payloads with HMAC
- Handle webhook delivery failures
- Log webhook delivery attempts
- Support custom headers in webhooks

#### Files to Touch

- `internal/api/handlers/webhooks.go` (new)
- `internal/service/webhook.go` (new)
- `internal/domain/webhook.go` (update)

#### Dependencies

- T031 (authentication)
- T009 (webhook repository)

#### Success Criteria

- [ ] Webhooks can be created and configured
- [ ] Webhook list returns all webhooks
- [ ] Webhooks can be updated and deleted
- [ ] Test endpoint triggers webhook
- [ ] Webhooks fire on message.received events
- [ ] Payload is signed correctly
- [ ] Failed deliveries are logged
- [ ] Custom headers are sent

---

### T038: Implement System and Admin Endpoints

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Create system management and administrative endpoints including health checks, system statistics, database cleanup, and configuration viewing. These endpoints provide operational visibility and maintenance capabilities.

#### Technical Details

- Implement GET /api/v1/health (health check)
- Implement GET /api/v1/stats (system statistics)
- Implement DELETE /api/v1/system/messages (delete all messages, admin only)
- Implement GET /api/v1/system/info (version, uptime, config)
- Return database connection status in health check
- Calculate statistics: total users, total messages, storage usage
- Support force flag for dangerous operations
- Sanitize configuration in info response (hide secrets)

#### Files to Touch

- `internal/api/handlers/system.go` (new)
- `internal/api/handlers/health.go` (new)

#### Dependencies

- T031 (authentication for admin endpoints)
- T009 (repository for statistics)

#### Success Criteria

- [ ] Health endpoint returns server status
- [ ] Stats endpoint returns accurate counts
- [ ] Delete all messages works (with confirmation)
- [ ] System info returns version and config
- [ ] Admin-only endpoints protected
- [ ] Secrets not exposed in responses

---

### T039: Add Rate Limiting and Security Middleware

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Implement rate limiting middleware to prevent API abuse. Add security headers, request size limits, and IP-based rate limiting. Implement different rate limits for authenticated vs unauthenticated requests.

#### Technical Details

- Implement rate limiting middleware
- Use IP address for unauthenticated rate limits
- Use user ID for authenticated rate limits
- Configure different limits per endpoint group
- Add security headers (X-Content-Type-Options, X-Frame-Options, etc.)
- Implement request body size limits
- Return 429 Too Many Requests on rate limit
- Add Retry-After header
- Log rate limit violations
- Support rate limit configuration

#### Files to Touch

- `internal/api/middleware/ratelimit.go` (new)
- `internal/api/middleware/security.go` (new)

#### Dependencies

- T030 (API server)
- T031 (authentication for user-based limits)

#### Success Criteria

- [ ] Rate limits prevent excessive requests
- [ ] Different limits for auth vs unauth work
- [ ] 429 status returned on limit exceeded
- [ ] Retry-After header set correctly
- [ ] Security headers added to responses
- [ ] Request size limits enforced
- [ ] Legitimate users not blocked

---

## Performance Targets

- API response time (p95): < 50ms
- Health check response: < 5ms
- Throughput: > 1000 requests/second
- Authentication overhead: < 5ms per request
- Search query performance: < 100ms
- Memory per request: < 1MB

## Risk Assessment

| Risk                          | Probability | Impact | Mitigation                                    |
|-------------------------------|-------------|--------|-----------------------------------------------|
| JWT security vulnerabilities  | Low         | High   | Use well-tested library, secure secret storage |
| Rate limiting bypass          | Medium      | Medium | Multiple rate limit strategies, monitoring     |
| API breaking changes          | Low         | Medium | API versioning, comprehensive testing          |
| Search performance issues     | Medium      | Medium | Database indexing, query optimization          |
| CORS misconfiguration         | Low         | Low    | Test with Web UI, clear configuration docs     |

## Notes

- API versioning (/api/v1) allows future changes without breaking clients
- JWT secret must be strong and kept secure
- Consider implementing API key authentication for integrations (future)
- OpenAPI/Swagger documentation improves developer experience
- Rate limiting should be configurable per deployment
- Webhook retry logic should implement exponential backoff (future)
- Consider implementing GraphQL API as alternative (future)
- API responses should include request IDs for debugging
- Implement proper HTTP status codes (200, 201, 400, 401, 403, 404, 429, 500)
- JSON response format should be consistent across all endpoints
