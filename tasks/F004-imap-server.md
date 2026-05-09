# Feature 4: IMAP Server Implementation

**Feature ID:** F004  
**Priority:** P1 - CRITICAL  
**Target Version:** v0.3.0  
**Estimated Duration:** 2-3 weeks  
**Status:** NOT_STARTED

## Overview

This feature implements a full IMAP4rev1 server that allows email clients like Thunderbird, Apple Mail, and Outlook to connect and access messages stored in Yunt. The server supports all essential IMAP operations including authentication, mailbox listing and selection, message fetching, flag management, search capabilities, and real-time notifications via IDLE. The implementation uses the `emersion/go-imap/v2` library.

IMAP support is a key differentiator for Yunt compared to alternatives like Mailhog and Mailpit. It enables developers to use their preferred email clients to browse test emails, making the development workflow more natural and efficient.

## Goals

- Implement RFC 3501 compliant IMAP4rev1 server
- Support LOGIN and PLAIN authentication
- Enable mailbox operations (LIST, CREATE, DELETE, RENAME)
- Implement message operations (FETCH, STORE, COPY, EXPUNGE)
- Support SEARCH command for message filtering
- Implement IDLE for real-time push notifications
- Support message flags (\Seen, \Flagged, \Deleted, etc.)
- Enable STARTTLS for encrypted connections

## Success Criteria

- [ ] All tasks completed
- [ ] All tests passing
- [ ] IMAP server accepts client connections
- [ ] Email clients can authenticate successfully
- [ ] Mailboxes are listed correctly
- [ ] Messages display properly in clients
- [ ] Flags update correctly
- [ ] SEARCH finds messages accurately
- [ ] IDLE pushes updates in real-time

## Tasks

### T021: Set Up IMAP Server Foundation

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1 day

#### Description

Create the basic IMAP server setup using `emersion/go-imap/v2` library. Implement server initialization, configuration loading, connection handling, and graceful shutdown. Establish the foundation for IMAP backend implementation.

#### Technical Details

- Use `emersion/go-imap/v2` for server implementation
- Load IMAP configuration from config system
- Implement server lifecycle (start, stop, graceful shutdown)
- Configure timeouts (read, write, idle)
- Support concurrent client connections
- Implement connection logging
- Handle server errors gracefully
- Coordinate with SMTP server for same port binding issues

#### Files to Touch

- `internal/imap/server.go` (new)
- `internal/imap/config.go` (new)

#### Dependencies

- T002 (configuration system)
- T003 (logging)
- T006 (go-imap dependency)

#### Success Criteria

- [ ] IMAP server starts on configured port
- [ ] Server responds to CAPABILITY command
- [ ] Server shuts down gracefully
- [ ] Configuration values are applied correctly
- [ ] Timeouts work as configured
- [ ] Connection logs are generated
- [ ] Multiple clients can connect simultaneously

---

### T022: Implement IMAP Backend and User Authentication

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Implement the IMAP backend interface that handles user authentication and provides access to user-specific data. Support LOGIN and PLAIN authentication methods. Create user session management.

#### Technical Details

- Implement `imap.Backend` interface from go-imap
- Implement LOGIN authentication (username/password)
- Implement PLAIN SASL authentication
- Validate credentials against user repository
- Create authenticated user sessions
- Return user-specific mailbox access
- Track session state per connection
- Handle authentication failures with proper IMAP responses
- Support logout and session cleanup

#### Files to Touch

- `internal/imap/backend.go` (new)
- `internal/imap/auth.go` (new)
- `internal/imap/session.go` (new)

#### Dependencies

- T021 (IMAP server foundation)
- T009 (user repository)

#### Success Criteria

- [ ] LOGIN command authenticates successfully
- [ ] PLAIN authentication works correctly
- [ ] Invalid credentials are rejected
- [ ] User sessions are created after authentication
- [ ] CAPABILITY advertises supported auth methods
- [ ] Session state is maintained correctly
- [ ] Logout cleans up session resources

---

### T023: Implement Mailbox Operations

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Implement IMAP mailbox operations including LIST, SELECT, CREATE, DELETE, RENAME, and STATUS. Provide access to both system mailboxes (INBOX, Sent, Trash) and user-created custom mailboxes.

#### Technical Details

- Implement LIST command to enumerate mailboxes
- Implement SELECT command to open a mailbox
- Implement EXAMINE command (read-only SELECT)
- Implement CREATE command for new mailboxes
- Implement DELETE command for mailbox removal
- Implement RENAME command for mailbox renaming
- Implement STATUS command for mailbox statistics
- Support mailbox hierarchy and naming
- Prevent deletion of system mailboxes
- Update mailbox statistics (message count, unseen count)

#### Files to Touch

- `internal/imap/mailbox.go` (new)
- `internal/imap/mailbox_list.go` (new)
- `internal/imap/mailbox_ops.go` (new)

#### Dependencies

- T022 (IMAP backend and authentication)
- T009 (mailbox repository)

#### Success Criteria

- [ ] LIST returns all user mailboxes
- [ ] SELECT opens mailbox and returns stats
- [ ] CREATE creates new custom mailboxes
- [ ] DELETE removes custom mailboxes only
- [ ] RENAME changes mailbox names correctly
- [ ] STATUS returns accurate statistics
- [ ] System mailboxes cannot be deleted
- [ ] Mailbox hierarchy works correctly

---

### T024: Implement Message Fetching (FETCH Command)

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 3 days

#### Description

Implement the IMAP FETCH command to retrieve messages and message parts. Support fetching headers, bodies, flags, envelope information, and attachments. Handle various FETCH data items (RFC822, BODY, BODYSTRUCTURE, etc.).

#### Technical Details

- Implement FETCH command handler
- Support sequence numbers and UID addressing
- Implement FETCH data items: FLAGS, ENVELOPE, INTERNALDATE
- Implement RFC822, RFC822.SIZE, RFC822.HEADER
- Implement BODY, BODYSTRUCTURE for MIME structure
- Implement BODY[HEADER], BODY[TEXT] for message parts
- Support partial fetches (BODY[]<start.length>)
- Parse stored raw message for MIME structure
- Return attachment parts correctly
- Optimize for client prefetch patterns

#### Files to Touch

- `internal/imap/fetch.go` (new)
- `internal/imap/message.go` (new)
- `internal/imap/bodystructure.go` (new)

#### Dependencies

- T023 (mailbox operations)
- T009 (message repository)
- T015 (MIME parser for reconstructing structure)

#### Success Criteria

- [ ] FETCH returns message headers correctly
- [ ] FETCH returns message bodies correctly
- [ ] FETCH FLAGS returns current flags
- [ ] BODYSTRUCTURE represents MIME correctly
- [ ] Partial fetches work for large messages
- [ ] Attachments are accessible via FETCH
- [ ] Performance is acceptable for large mailboxes
- [ ] Email clients display messages correctly

---

### T025: Implement Flag Management (STORE Command)

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1 day

#### Description

Implement the STORE command to modify message flags. Support standard IMAP flags (\Seen, \Answered, \Flagged, \Deleted, \Draft) and synchronize with database. Implement flag change notifications to other connected clients.

#### Technical Details

- Implement STORE command handler
- Support +FLAGS, -FLAGS, FLAGS operations
- Handle \Seen flag for read/unread status
- Handle \Flagged flag for starred messages
- Handle \Deleted flag for deletion marking
- Handle \Answered and \Draft flags
- Update message repository with flag changes
- Send untagged FETCH responses for flag updates
- Support silent flag changes (FLAGS.SILENT)
- Broadcast flag changes to IDLE connections

#### Files to Touch

- `internal/imap/store.go` (new)
- `internal/imap/flags.go` (new)

#### Dependencies

- T024 (message fetching)
- T009 (message repository for updates)

#### Success Criteria

- [ ] STORE sets flags correctly
- [ ] STORE removes flags correctly
- [ ] Flag changes persist to database
- [ ] Other clients see flag updates
- [ ] \Seen flag marks messages as read
- [ ] \Flagged flag stars messages
- [ ] Silent operations don't trigger responses
- [ ] Unseen count updates with \Seen changes

---

### T026: Implement SEARCH Command

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 2 days

#### Description

Implement the IMAP SEARCH command to find messages based on various criteria. Support searching by flags, dates, sender, subject, and message content. Optimize search performance with database queries.

#### Technical Details

- Implement SEARCH command handler
- Support flag-based search (SEEN, UNSEEN, FLAGGED, etc.)
- Support date-based search (BEFORE, SINCE, ON)
- Support address search (FROM, TO, CC, BCC)
- Support subject search (SUBJECT)
- Support body search (BODY, TEXT)
- Support message ID search (HEADER Message-ID)
- Support logical operators (AND, OR, NOT)
- Translate IMAP search to SQL/repository queries
- Return sequence numbers or UIDs based on request

#### Files to Touch

- `internal/imap/search.go` (new)
- `internal/imap/search_parser.go` (new)
- `internal/repository/search.go` (update)

#### Dependencies

- T024 (message fetching)
- T009 (message repository with search support)

#### Success Criteria

- [ ] SEARCH ALL returns all messages
- [ ] SEARCH UNSEEN returns unread messages
- [ ] SEARCH FROM finds by sender
- [ ] SEARCH SUBJECT finds by subject
- [ ] SEARCH BEFORE filters by date
- [ ] Complex queries with AND/OR work
- [ ] Search performance is acceptable
- [ ] Results are accurate

---

### T027: Implement COPY and EXPUNGE Commands

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Implement COPY/MOVE commands to copy or move messages between mailboxes. Implement EXPUNGE command to permanently delete messages marked with \Deleted flag. Maintain message integrity during these operations.

#### Technical Details

- Implement COPY command to duplicate messages
- Implement MOVE command (if supported)
- Implement EXPUNGE command to delete marked messages
- Update message repository for mailbox changes
- Maintain message UIDs correctly
- Update mailbox statistics after operations
- Send untagged EXPUNGE responses
- Support UID COPY and UID EXPUNGE
- Handle failures gracefully (partial copy)

#### Files to Touch

- `internal/imap/copy.go` (new)
- `internal/imap/expunge.go` (new)

#### Dependencies

- T024 (message fetching)
- T025 (flag management)
- T009 (message repository)

#### Success Criteria

- [ ] COPY duplicates messages to target mailbox
- [ ] EXPUNGE removes deleted messages
- [ ] Mailbox statistics update correctly
- [ ] Message UIDs are maintained
- [ ] Other clients see EXPUNGE notifications
- [ ] Partial failures don't corrupt state
- [ ] MOVE command works (if implemented)

---

### T028: Implement IDLE for Real-time Notifications

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 2 days

#### Description

Implement the IMAP IDLE extension (RFC 2177) to provide real-time push notifications to email clients. Notify clients when new messages arrive, flags change, or messages are deleted. Handle multiple IDLE connections efficiently.

#### Technical Details

- Implement IDLE command handler
- Advertise IDLE capability in CAPABILITY response
- Support long-lived IDLE connections (up to 30 minutes)
- Create notification channel for mailbox changes
- Send untagged EXISTS when messages arrive
- Send untagged FETCH when flags change
- Send untagged EXPUNGE when messages deleted
- Handle DONE to exit IDLE mode
- Support multiple clients IDLE on same mailbox
- Implement efficient notification routing

#### Files to Touch

- `internal/imap/idle.go` (new)
- `internal/imap/notify.go` (new)
- `internal/service/notify.go` (new)

#### Dependencies

- T024 (message fetching)
- T025 (flag management)
- T016 (message service for notifications)

#### Success Criteria

- [ ] IDLE command accepted
- [ ] New message notifications sent
- [ ] Flag change notifications sent
- [ ] EXPUNGE notifications sent
- [ ] DONE exits IDLE correctly
- [ ] Multiple IDLE clients work simultaneously
- [ ] Idle timeout handled correctly
- [ ] Performance acceptable with many IDLE connections

---

### T029: Add STARTTLS Support for IMAP

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Add TLS encryption support via STARTTLS command for IMAP connections. Load TLS certificates from configuration and enable encrypted IMAP sessions.

#### Technical Details

- Load TLS certificate and key from config
- Implement STARTTLS command support
- Advertise STARTTLS in CAPABILITY response
- Upgrade connection to TLS when requested
- Support optional vs required TLS modes
- Handle TLS handshake errors gracefully
- Log TLS connection upgrades
- Disable LOGIN before STARTTLS (if required TLS)

#### Files to Touch

- `internal/imap/tls.go` (new)
- `internal/imap/server.go` (update)

#### Dependencies

- T021 (IMAP server foundation)
- T002 (TLS configuration)

#### Success Criteria

- [ ] STARTTLS advertised in CAPABILITY
- [ ] TLS handshake succeeds
- [ ] Encrypted connections work correctly
- [ ] TLS can be disabled via configuration
- [ ] Certificate errors logged clearly
- [ ] LOGIN disabled before STARTTLS (if configured)

---

## Performance Targets

- Connection handling: > 25 concurrent IMAP clients
- FETCH performance: < 50ms for typical message
- LIST performance: < 20ms for 50 mailboxes
- SEARCH performance: < 100ms for 10k messages
- IDLE connections: > 100 simultaneous
- Memory per connection: < 5MB

## Risk Assessment

| Risk                              | Probability | Impact | Mitigation                                   |
|-----------------------------------|-------------|--------|----------------------------------------------|
| IDLE scalability issues           | Medium      | Medium | Efficient notification routing, connection limits |
| FETCH performance with large msgs | Medium      | Medium | Streaming, caching, partial fetch support    |
| IMAP RFC compliance gaps          | Low         | Medium | Use well-tested library, test with multiple clients |
| TLS certificate configuration     | Low         | Low    | Clear documentation, validation on startup   |
| Client compatibility issues       | Medium      | Medium | Test with popular clients (Thunderbird, Apple Mail) |

## Notes

- IMAP IDLE is critical for real-time updates in email clients
- BODYSTRUCTURE generation can be expensive for complex MIME messages
- Consider implementing CONDSTORE extension for efficient sync (future)
- IMAP COMPRESS extension could reduce bandwidth (future)
- Test with multiple email clients: Thunderbird, Apple Mail, Outlook
- IMAP logs should include client name/version for debugging
- Consider implementing IMAP QUOTA extension (future)
- UID persistence is important across restarts
- Some clients use non-standard IMAP extensions, handle gracefully
