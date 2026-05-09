# Feature 6: Web UI Development

**Feature ID:** F006  
**Priority:** P1 - CRITICAL  
**Target Version:** v0.5.0  
**Estimated Duration:** 2-3 weeks  
**Status:** NOT_STARTED

## Overview

This feature implements a modern, responsive web-based user interface for Yunt using Svelte 5 and SvelteKit. The Web UI provides an intuitive way to browse emails, manage mailboxes, configure webhooks, and administer users without needing an email client. It includes a dashboard with statistics, inbox view with message list, message detail view with HTML rendering, user management for admins, and settings pages.

The Web UI is embedded into the Go binary using `go:embed`, eliminating the need for separate deployment. It communicates with the backend via the REST API and provides a seamless development experience for testing emails.

## Goals

- Build a modern, responsive web interface with Svelte 5
- Implement authentication flow (login, logout, token refresh)
- Create dashboard with email statistics and visualizations
- Build inbox view with message list, filtering, and search
- Implement message detail view with HTML/text rendering
- Create user management interface for administrators
- Build settings pages for configuration and webhooks
- Embed built Web UI into Go binary with go:embed
- Support real-time updates via WebSocket or polling

## Success Criteria

- [ ] All tasks completed
- [ ] All tests passing
- [ ] Web UI accessible from browser
- [ ] Login and authentication work correctly
- [ ] All features accessible via UI
- [ ] Responsive design works on mobile and desktop
- [ ] HTML emails render safely
- [ ] Real-time updates work
- [ ] Web UI embedded in Go binary

## Tasks

### T040: Initialize SvelteKit Project and Development Environment

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1 day

#### Description

Set up the SvelteKit project structure with TypeScript, configure build tools, install dependencies, and establish the development workflow. Configure Tailwind CSS for styling and set up the integration with the Go backend.

#### Technical Details

- Initialize SvelteKit project with TypeScript template
- Configure Vite for development and production builds
- Install and configure Tailwind CSS
- Install Lucide icons for UI elements
- Set up TypeScript configuration
- Configure API proxy for development (avoid CORS)
- Set up ESLint and Prettier for code quality
- Configure build output directory for go:embed
- Create basic layout structure

#### Files to Touch

- `web/package.json` (new)
- `web/svelte.config.js` (new)
- `web/vite.config.js` (new)
- `web/tsconfig.json` (new)
- `web/tailwind.config.js` (new)
- `web/src/app.html` (new)
- `web/src/routes/+layout.svelte` (new)
- `web/.eslintrc.cjs` (new)
- `web/.prettierrc` (new)

#### Dependencies

- T030 (API server must exist for development)

#### Success Criteria

- [ ] `npm run dev` starts development server
- [ ] `npm run build` creates production build
- [ ] TypeScript compilation works
- [ ] Tailwind CSS styles apply correctly
- [ ] Linting and formatting work
- [ ] Development proxy connects to API
- [ ] Build output is in correct directory

---

### T041: Create API Client Library

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Build a TypeScript API client library that wraps all REST API endpoints. Include authentication handling, token refresh, error handling, and type-safe request/response interfaces. Use fetch API with proper error handling.

#### Technical Details

- Create API client class with base configuration
- Implement authentication methods (login, logout, refresh)
- Implement user endpoints (list, create, update, delete)
- Implement mailbox endpoints (list, create, update, delete, stats)
- Implement message endpoints (list, get, delete, update flags, move)
- Implement attachment endpoints (list, download)
- Implement search endpoints
- Implement webhook endpoints
- Implement system endpoints (health, stats)
- Handle JWT token storage (localStorage)
- Implement automatic token refresh
- Handle network errors gracefully
- Type all requests and responses with TypeScript interfaces

#### Files to Touch

- `web/src/lib/api/client.ts` (new)
- `web/src/lib/api/auth.ts` (new)
- `web/src/lib/api/users.ts` (new)
- `web/src/lib/api/mailboxes.ts` (new)
- `web/src/lib/api/messages.ts` (new)
- `web/src/lib/api/attachments.ts` (new)
- `web/src/lib/api/search.ts` (new)
- `web/src/lib/api/webhooks.ts` (new)
- `web/src/lib/api/system.ts` (new)
- `web/src/lib/api/types.ts` (new)

#### Dependencies

- T040 (SvelteKit project)
- T031-T038 (API endpoints)

#### Success Criteria

- [ ] All API endpoints have client methods
- [ ] Authentication flow works end-to-end
- [ ] Token refresh happens automatically
- [ ] Network errors handled gracefully
- [ ] TypeScript types match API contracts
- [ ] Client methods return typed responses
- [ ] 401 responses trigger re-authentication

---

### T042: Implement Authentication Flow and Login Page

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1.5 days

#### Description

Create the login page with email/password form, implement authentication state management with Svelte stores, and create route guards to protect authenticated pages. Handle token storage and automatic redirect on authentication.

#### Technical Details

- Create login page with form
- Implement form validation (email, password)
- Connect to API client login method
- Store authentication state in Svelte store
- Store JWT tokens in localStorage
- Create auth guard for protected routes
- Implement automatic redirect to login
- Create logout functionality
- Handle "remember me" option
- Display authentication errors clearly
- Implement loading states

#### Files to Touch

- `web/src/routes/login/+page.svelte` (new)
- `web/src/lib/stores/auth.ts` (new)
- `web/src/lib/guards/auth.ts` (new)
- `web/src/routes/+layout.svelte` (update)

#### Dependencies

- T041 (API client)

#### Success Criteria

- [ ] Login form validates input
- [ ] Successful login redirects to dashboard
- [ ] Failed login shows error message
- [ ] Authentication state persists across refreshes
- [ ] Protected routes redirect to login
- [ ] Logout clears tokens and redirects
- [ ] Loading states display during login

---

### T043: Build Dashboard Page with Statistics

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Create the main dashboard page displaying email statistics, recent messages, and activity visualizations. Include stat cards for total/unread/today/week message counts, storage usage visualization, message timeline chart, and recent messages list.

#### Technical Details

- Create dashboard layout with stat cards
- Fetch statistics from API
- Display total messages, unread, today, this week
- Create storage usage progress bar
- Implement messages-per-hour chart (simple bar chart)
- Display recent messages list (last 10)
- Make message items clickable to detail view
- Implement auto-refresh for statistics
- Handle loading and error states
- Make responsive for mobile

#### Files to Touch

- `web/src/routes/+page.svelte` (new)
- `web/src/lib/components/StatCard.svelte` (new)
- `web/src/lib/components/MessageList.svelte` (new)
- `web/src/lib/components/StorageUsage.svelte` (new)
- `web/src/lib/components/ActivityChart.svelte` (new)

#### Dependencies

- T042 (authentication)
- T041 (API client for stats)

#### Success Criteria

- [ ] Dashboard loads and displays statistics
- [ ] Stat cards show accurate counts
- [ ] Recent messages listed correctly
- [ ] Storage usage visualized clearly
- [ ] Auto-refresh updates statistics
- [ ] Responsive layout works on mobile
- [ ] Loading states displayed while fetching

---

### T044: Build Inbox View with Message List

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2.5 days

#### Description

Create the inbox view with a filterable, sortable, paginated message list. Include search bar, filter controls (unread, starred), bulk actions (delete, mark read), and mailbox navigation sidebar.

#### Technical Details

- Create inbox layout with sidebar and message list
- Display mailbox list in sidebar (INBOX, Sent, Drafts, etc.)
- Show unread count badges on mailboxes
- Implement message list with table/card layout
- Display sender, subject, preview, timestamp
- Indicate read/unread with visual style
- Show star icon for starred messages
- Show attachment icon when present
- Implement checkbox for bulk selection
- Create bulk action toolbar (delete, mark read, move)
- Implement search bar with live search
- Implement filters (unread, starred, has attachments)
- Implement sorting (date, sender, subject)
- Implement pagination controls
- Make messages clickable to detail view
- Highlight selected mailbox

#### Files to Touch

- `web/src/routes/inbox/+page.svelte` (new)
- `web/src/lib/components/Sidebar.svelte` (new)
- `web/src/lib/components/MessageTable.svelte` (new)
- `web/src/lib/components/SearchBar.svelte` (new)
- `web/src/lib/components/FilterControls.svelte` (new)
- `web/src/lib/components/BulkActions.svelte` (new)
- `web/src/lib/stores/messages.ts` (new)

#### Dependencies

- T042 (authentication)
- T041 (API client for messages)

#### Success Criteria

- [ ] Message list displays correctly
- [ ] Mailbox navigation works
- [ ] Unread counts accurate
- [ ] Bulk selection and actions work
- [ ] Search filters messages
- [ ] Filters apply correctly
- [ ] Sorting works on all columns
- [ ] Pagination navigates pages
- [ ] Responsive layout works

---

### T045: Build Message Detail View

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Create the message detail view showing full message content with HTML rendering, attachment list with download capability, and message actions (delete, move, star, mark unread). Safely render HTML emails to prevent XSS.

#### Technical Details

- Create message detail page layout
- Fetch full message by ID
- Display message headers (from, to, cc, subject, date)
- Implement tab view for HTML/Text/Headers/Raw
- Safely render HTML content in sandboxed iframe
- Display plain text with proper formatting
- Show raw headers in monospace
- Display raw EML with download button
- List attachments with icons and sizes
- Implement attachment download
- Create action toolbar (back, delete, move, star)
- Mark message as read when opened
- Handle inline images in HTML
- Support keyboard navigation (prev/next message)
- Implement loading and error states

#### Files to Touch

- `web/src/routes/message/[id]/+page.svelte` (new)
- `web/src/lib/components/MessageHeader.svelte` (new)
- `web/src/lib/components/MessageBody.svelte` (new)
- `web/src/lib/components/AttachmentList.svelte` (new)
- `web/src/lib/components/MessageActions.svelte` (new)

#### Dependencies

- T041 (API client for messages and attachments)
- T044 (inbox for navigation context)

#### Success Criteria

- [ ] Message content displays correctly
- [ ] HTML tab renders emails safely
- [ ] Text tab shows plain text
- [ ] Headers and raw tabs show correct data
- [ ] Attachments download correctly
- [ ] Actions (delete, star, move) work
- [ ] Message marked read on open
- [ ] Inline images display
- [ ] No XSS vulnerabilities

---

### T046: Build User Management Interface (Admin)

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1.5 days

#### Description

Create the user management interface for administrators including user list, create user form, edit user form, and delete user confirmation. Only accessible to admin users.

#### Technical Details

- Create users page with user list table
- Display username, email, role, status, last login
- Implement create user button and modal/form
- Create user form with validation
- Implement edit user functionality
- Implement delete user with confirmation dialog
- Show admin-only route guard
- Handle role selection (admin/user)
- Implement user activation toggle
- Display user statistics (message count)
- Implement search/filter for users

#### Files to Touch

- `web/src/routes/users/+page.svelte` (new)
- `web/src/lib/components/UserTable.svelte` (new)
- `web/src/lib/components/UserForm.svelte` (new)
- `web/src/lib/components/ConfirmDialog.svelte` (new)

#### Dependencies

- T042 (authentication with role check)
- T041 (API client for users)

#### Success Criteria

- [ ] User list displays all users
- [ ] Create user form works
- [ ] Edit user form works
- [ ] Delete confirmation prevents accidents
- [ ] Admin guard blocks non-admin users
- [ ] Form validation catches errors
- [ ] Role changes work correctly

---

### T047: Build Settings and Webhook Configuration Pages

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 2 days

#### Description

Create settings pages for user preferences, webhook configuration, and system settings. Include webhook creation form, webhook list with test/delete actions, and general settings.

#### Technical Details

- Create settings page with tabbed interface
- Create webhooks tab with list and form
- Display webhook name, URL, events, status
- Implement create webhook form
- Implement edit webhook functionality
- Implement delete webhook with confirmation
- Implement test webhook button
- Display webhook delivery logs
- Create general settings tab (future)
- Implement mailbox management in settings
- Allow custom mailbox creation

#### Files to Touch

- `web/src/routes/settings/+page.svelte` (new)
- `web/src/lib/components/WebhookList.svelte` (new)
- `web/src/lib/components/WebhookForm.svelte` (new)
- `web/src/lib/components/MailboxSettings.svelte` (new)

#### Dependencies

- T042 (authentication)
- T041 (API client for webhooks)

#### Success Criteria

- [ ] Settings page displays tabs
- [ ] Webhook list shows all webhooks
- [ ] Create webhook form works
- [ ] Edit webhook updates correctly
- [ ] Delete removes webhook
- [ ] Test webhook triggers delivery
- [ ] Mailbox creation works
- [ ] Form validation works

---

### T048: Implement Real-time Updates and Notifications

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1.5 days

#### Description

Implement real-time updates for new messages using polling or WebSocket. Display toast notifications when new messages arrive. Update message counts and inbox list automatically.

#### Technical Details

- Implement polling mechanism (every 10-30 seconds)
- Check for new messages via API
- Update message list when new messages detected
- Update unread counts in sidebar
- Display toast notification for new messages
- Play sound on new message (optional)
- Implement WebSocket alternative (future)
- Allow notification configuration in settings
- Handle visibility API to pause when tab inactive
- Implement efficient polling with etag/if-modified

#### Files to Touch

- `web/src/lib/services/polling.ts` (new)
- `web/src/lib/components/Toast.svelte` (new)
- `web/src/lib/stores/notifications.ts` (new)

#### Dependencies

- T044 (inbox view)
- T041 (API client)

#### Success Criteria

- [ ] Polling checks for new messages
- [ ] New message notifications display
- [ ] Message counts update automatically
- [ ] Inbox list refreshes with new messages
- [ ] Polling stops when tab inactive
- [ ] Performance impact is minimal
- [ ] Users can disable notifications

---

### T049: Embed Web UI in Go Binary

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 0.5 days

#### Description

Configure the Web UI build process to output files for go:embed. Create Go package that embeds the built Web UI and serves it via the API server. Set up proper routing to serve index.html for client-side routing.

#### Technical Details

- Configure SvelteKit adapter-static for static build
- Set build output to `webui/dist`
- Create `webui/embed.go` with go:embed directive
- Embed all static files (HTML, JS, CSS, images)
- Serve embedded files via Echo static middleware
- Handle SPA routing (serve index.html for all routes)
- Set proper cache headers for static assets
- Support serving API and UI on same port
- Update build scripts to build UI before Go binary

#### Files to Touch

- `web/svelte.config.js` (update)
- `webui/embed.go` (new)
- `internal/api/server.go` (update)
- `Makefile` (update)
- `scripts/build.sh` (update)

#### Dependencies

- T040 (SvelteKit project)
- T030 (API server)

#### Success Criteria

- [ ] `npm run build` creates static output
- [ ] go:embed includes all necessary files
- [ ] Web UI accessible at <http://localhost:8025/>
- [ ] API accessible at <http://localhost:8025/api/v1/>
- [ ] Client-side routing works (refresh works)
- [ ] Static assets have proper cache headers
- [ ] Single binary serves both API and UI

---

## Performance Targets

- Initial page load: < 2 seconds
- Route transition: < 200ms
- API call overhead: < 50ms
- Bundle size: < 500KB (gzipped)
- Time to interactive: < 3 seconds
- Message list render: < 100ms for 50 messages

## Risk Assessment

| Risk                          | Probability | Impact | Mitigation                                    |
|-------------------------------|-------------|--------|-----------------------------------------------|
| XSS vulnerabilities in HTML   | Medium      | High   | Sandboxed iframe, CSP headers                 |
| Bundle size too large         | Low         | Medium | Code splitting, tree shaking, lazy loading    |
| Real-time updates overhead    | Medium      | Low    | Efficient polling, pause when inactive        |
| Browser compatibility issues  | Low         | Low    | Test on major browsers, use modern build      |
| go:embed file size limits     | Low         | Medium | Optimize assets, use compression              |

## Notes

- Use Content Security Policy headers to prevent XSS
- Sanitize HTML emails before rendering (or use iframe sandbox)
- Consider dark mode support for better user experience (future)
- Implement keyboard shortcuts for power users (future)
- Consider progressive web app (PWA) support (future)
- Use lazy loading for heavy components
- Optimize images and assets for web
- Test on mobile devices for responsive design
- Consider implementing email composition (future)
- Web UI should work without JavaScript for critical content
- Use semantic HTML for accessibility
- Add loading skeletons for better perceived performance
