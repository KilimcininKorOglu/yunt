# Feature 3: SMTP Server Implementation

**Feature ID:** F003  
**Priority:** P1 - CRITICAL  
**Target Version:** v0.2.0  
**Estimated Duration:** 2-3 weeks  
**Status:** NOT_STARTED

## Overview

This feature implements a fully functional SMTP server capable of receiving emails from applications, parsing MIME messages with attachments, and storing them in the database. The server supports authentication (PLAIN, LOGIN), TLS encryption via STARTTLS, and optional relay functionality to forward emails to external SMTP servers. The implementation uses the `emersion/go-smtp` library and handles multi-recipient messages, size limits, and rate limiting.

The SMTP server is the core ingestion point for emails in Yunt, making it critical for the entire system. It must be robust, performant, and RFC-compliant while remaining simple to configure for development use cases.

## Goals

- Implement RFC 5321 compliant SMTP server
- Support SMTP AUTH (PLAIN, LOGIN methods)
- Parse MIME messages with attachments and HTML content
- Store received emails in database via repository layer
- Implement optional SMTP relay to external servers
- Support STARTTLS for encrypted connections
- Handle multiple recipients per message
- Enforce message size and rate limits

## Success Criteria

- [ ] All tasks completed
- [ ] All tests passing
- [ ] SMTP server accepts connections on configured port
- [ ] Messages are parsed and stored correctly
- [ ] Attachments are extracted and saved
- [ ] Authentication works for configured users
- [ ] Relay functionality forwards to external SMTP
- [ ] Performance meets targets (100+ msg/sec)

## Tasks

### T013: Set Up SMTP Server Foundation

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1 day

#### Description

Create the basic SMTP server setup using `emersion/go-smtp` library. Implement server initialization, configuration loading, graceful shutdown, and connection handling. Establish the foundation for SMTP backend implementation.

#### Technical Details

- Use `emersion/go-smtp` for server implementation
- Load SMTP configuration from config system
- Implement server lifecycle (start, stop, graceful shutdown)
- Configure timeouts (read, write)
- Set maximum message size
- Set maximum recipients per message
- Implement basic connection logging
- Handle server errors gracefully

#### Files to Touch

- `internal/smtp/server.go` (new)
- `internal/smtp/config.go` (new)

#### Dependencies

- T002 (configuration system)
- T003 (logging)
- T006 (go-smtp dependency)

#### Success Criteria

- [ ] SMTP server starts on configured port
- [ ] Server responds to EHLO/HELO commands
- [ ] Server shuts down gracefully
- [ ] Configuration values are applied correctly
- [ ] Timeouts work as configured
- [ ] Connection logs are generated

---

### T014: Implement SMTP Backend and Session Handler

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Implement the SMTP backend interface that handles the SMTP protocol flow: MAIL FROM, RCPT TO, and DATA commands. Create session handler to manage per-connection state and validate recipients against the database.

#### Technical Details

- Implement `smtp.Backend` interface from go-smtp
- Implement `smtp.Session` interface for connection state
- Validate MAIL FROM addresses
- Validate RCPT TO addresses against database users/mailboxes
- Handle multiple recipients per message
- Collect message data from DATA command
- Enforce maximum message size
- Implement proper SMTP response codes
- Track session state (sender, recipients, data)

#### Files to Touch

- `internal/smtp/backend.go` (new)
- `internal/smtp/session.go` (new)

#### Dependencies

- T013 (SMTP server foundation)
- T009 (repository for user/mailbox lookup)

#### Success Criteria

- [ ] Backend accepts valid MAIL FROM
- [ ] Backend validates RCPT TO against database
- [ ] Multiple recipients are handled correctly
- [ ] Message data is captured completely
- [ ] Size limits are enforced
- [ ] SMTP responses follow RFC standards
- [ ] Invalid recipients are rejected appropriately

---

### T015: Implement MIME Message Parser

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 3 days

#### Description

Create a MIME message parser that extracts headers, plain text body, HTML body, and attachments from raw email data. Handle multipart messages, quoted-printable encoding, base64 encoding, and inline images.

#### Technical Details

- Use `emersion/go-message` and `jhillyerd/enmime` for parsing
- Extract RFC 5322 headers (From, To, CC, BCC, Subject, Message-ID, etc.)
- Parse multipart/alternative for text and HTML bodies
- Extract attachments with correct filenames and content types
- Handle inline attachments (Content-ID for images)
- Decode quoted-printable and base64 content
- Extract and parse email addresses (name + address)
- Handle malformed messages gracefully
- Calculate message size accurately

#### Files to Touch

- `internal/parser/mime.go` (new)
- `internal/parser/address.go` (new)
- `internal/parser/attachment.go` (new)
- `internal/parser/mime_test.go` (new)

#### Dependencies

- T006 (MIME parsing dependencies)
- T007 (domain models)

#### Success Criteria

- [ ] Plain text emails parsed correctly
- [ ] HTML emails with text alternative parsed correctly
- [ ] Attachments extracted with correct content
- [ ] Inline images identified properly
- [ ] Email addresses parsed with names
- [ ] Encoded content decoded correctly
- [ ] Malformed messages don't crash parser
- [ ] Unit tests cover common email formats

---

### T016: Implement Message Storage Service

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Create a message service that takes parsed MIME messages and stores them in the database. Handle message creation, attachment storage, mailbox routing, and duplicate detection. Coordinate between message and attachment repositories.

#### Technical Details

- Accept parsed message from MIME parser
- Determine target mailbox for each recipient (INBOX by default)
- Create Message domain object with all fields
- Store raw message bytes for later retrieval
- Create Attachment domain objects
- Use transactions for atomic storage
- Handle duplicate Message-ID detection
- Update mailbox statistics (message count, unread count)
- Implement error recovery for partial failures

#### Files to Touch

- `internal/service/message.go` (new)
- `internal/service/message_test.go` (new)

#### Dependencies

- T015 (MIME parser)
- T009 (repository for storage)
- T007 (domain models)

#### Success Criteria

- [ ] Messages stored with all fields populated
- [ ] Attachments linked to correct messages
- [ ] Raw message preserved for EML export
- [ ] Mailbox statistics updated correctly
- [ ] Duplicate messages handled appropriately
- [ ] Storage operations are transactional
- [ ] Unit tests verify correct storage

---

### T017: Add SMTP Authentication Support

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Implement SMTP authentication using PLAIN and LOGIN mechanisms. Validate credentials against the user repository. Support optional authentication (enabled via configuration).

#### Technical Details

- Implement PLAIN authentication (RFC 4616)
- Implement LOGIN authentication
- Validate username/password against user repository
- Use bcrypt for password comparison
- Support optional authentication mode
- Log authentication attempts
- Rate limit authentication failures
- Return proper SMTP auth response codes

#### Files to Touch

- `internal/smtp/auth.go` (new)
- `internal/smtp/backend.go` (update)

#### Dependencies

- T014 (SMTP backend)
- T009 (user repository)

#### Success Criteria

- [ ] PLAIN authentication works with valid credentials
- [ ] LOGIN authentication works with valid credentials
- [ ] Invalid credentials are rejected
- [ ] Authentication can be disabled via config
- [ ] Failed attempts are logged
- [ ] Authentication doesn't leak timing information

---

### T018: Implement STARTTLS Support

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Add TLS encryption support via STARTTLS command. Load TLS certificates from configuration and enable encrypted SMTP connections.

#### Technical Details

- Load TLS certificate and key from config
- Implement STARTTLS command support
- Advertise STARTTLS in EHLO response
- Upgrade connection to TLS when requested
- Support optional vs required TLS modes
- Handle TLS handshake errors gracefully
- Log TLS connection upgrades

#### Files to Touch

- `internal/smtp/tls.go` (new)
- `internal/smtp/server.go` (update)

#### Dependencies

- T013 (SMTP server foundation)
- T002 (TLS configuration)

#### Success Criteria

- [ ] STARTTLS advertised in EHLO when enabled
- [ ] TLS handshake succeeds with valid certificates
- [ ] Encrypted connections work correctly
- [ ] TLS can be disabled via configuration
- [ ] Certificate errors are logged clearly
- [ ] Connection security state is tracked

---

### T019: Implement SMTP Relay Functionality

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 2 days

#### Description

Add optional SMTP relay capability to forward received emails to an external SMTP server. Support authentication with external server and domain-based filtering for relay.

#### Technical Details

- Use `emersion/go-smtp` client for outbound connections
- Load relay configuration (host, port, credentials)
- Implement relay decision logic (allow list check)
- Forward messages to external SMTP server
- Handle relay authentication (PLAIN, LOGIN)
- Support TLS for relay connections
- Log successful and failed relay attempts
- Implement retry logic for temporary failures
- Fall back to local storage if relay fails

#### Files to Touch

- `internal/service/relay.go` (new)
- `internal/smtp/session.go` (update)

#### Dependencies

- T014 (SMTP session)
- T016 (message storage)
- T002 (relay configuration)

#### Success Criteria

- [ ] Relay forwards to external SMTP when enabled
- [ ] Allow list filtering works correctly
- [ ] Authentication with external server succeeds
- [ ] TLS connections to relay server work
- [ ] Failed relay attempts are logged
- [ ] Messages stored locally even if relay fails
- [ ] Relay can be disabled via configuration

---

### T020: Add Rate Limiting and Security Controls

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Implement rate limiting based on IP address and authenticated user. Add security controls to prevent abuse, including connection limits, message size validation, and recipient limits.

#### Technical Details

- Implement IP-based rate limiting (messages per minute)
- Implement user-based rate limiting (messages per hour)
- Track connection counts per IP
- Enforce maximum recipients per message
- Enforce maximum message size
- Implement backoff for rate limit violations
- Log rate limit hits
- Return appropriate SMTP error codes for violations

#### Files to Touch

- `internal/smtp/ratelimit.go` (new)
- `internal/smtp/session.go` (update)

#### Dependencies

- T014 (SMTP session)

#### Success Criteria

- [ ] Rate limits prevent spam-like behavior
- [ ] Connection limits prevent DoS
- [ ] Size limits are enforced
- [ ] Recipient limits work correctly
- [ ] Rate limit violations are logged
- [ ] SMTP error responses are appropriate
- [ ] Legitimate traffic is not blocked

---

## Performance Targets

- Message throughput: > 100 messages/second
- Connection handling: > 50 concurrent connections
- MIME parsing: < 50ms for typical message
- Message storage: < 100ms including attachments
- Memory usage: < 10MB per connection
- Relay latency: < 500ms to external server

## Risk Assessment

| Risk                           | Probability | Impact | Mitigation                                      |
|--------------------------------|-------------|--------|-------------------------------------------------|
| Large attachment memory usage  | Medium      | Medium | Stream attachments, limit sizes                 |
| MIME parsing vulnerabilities   | Low         | High   | Use well-tested libraries, sanitize input       |
| Relay authentication failures  | Medium      | Low    | Clear error messages, retry logic               |
| Rate limiting bypass           | Low         | Medium | Multiple rate limit strategies (IP + user)      |
| TLS certificate issues         | Low         | Medium | Validation on startup, clear error messages     |

## Notes

- SMTP server should handle malformed messages gracefully
- Consider implementing SPF/DKIM verification for relay (future)
- Webhook notifications for received messages (future feature)
- Message deduplication based on Message-ID prevents duplicates
- Raw message storage enables full EML export for debugging
- SMTP logs should include all connection details for troubleshooting
- Performance testing should include various message sizes
- Consider implementing SMTP pipelining for performance (future)
