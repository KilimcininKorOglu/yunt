package imap

import (
	"context"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/rs/zerolog"
)

// SessionState represents the current state of an IMAP session.
type SessionState int

const (
	// SessionStateNotAuthenticated indicates the session has not yet authenticated.
	SessionStateNotAuthenticated SessionState = iota
	// SessionStateAuthenticated indicates the session is authenticated but no mailbox is selected.
	SessionStateAuthenticated
	// SessionStateSelected indicates the session has a mailbox selected.
	SessionStateSelected
	// SessionStateLogout indicates the session is logging out.
	SessionStateLogout
)

// String returns a string representation of the session state.
func (s SessionState) String() string {
	switch s {
	case SessionStateNotAuthenticated:
		return "not_authenticated"
	case SessionStateAuthenticated:
		return "authenticated"
	case SessionStateSelected:
		return "selected"
	case SessionStateLogout:
		return "logout"
	default:
		return "unknown"
	}
}

// Session represents an IMAP session for a connected client.
// It implements the imapserver.Session interface.
type Session struct {
	server     *Server
	conn       *imapserver.Conn
	logger     zerolog.Logger
	remoteAddr string
	createdAt  time.Time
	username   string
	sessionID  string // Unique session identifier for IDLE tracking

	// Authentication and session state
	state       SessionState
	userSession *UserSession
}

// Close is called when the session is closed.
func (s *Session) Close() error {
	s.state = SessionStateLogout
	duration := time.Since(s.createdAt)

	// Clean up user session if authenticated
	if s.userSession != nil && s.server.backend != nil {
		s.server.backend.Logout(s.userSession.ID)
	}

	s.logger.Info().
		Dur("duration", duration).
		Str("username", s.username).
		Str("state", s.state.String()).
		Msg("Session closed")
	s.server.onSessionClose(s.remoteAddr)
	return nil
}

// Not authenticated state commands

// Login authenticates the user with username and password.
// Returns ErrAuthFailed if authentication fails.
func (s *Session) Login(username, password string) error {
	s.logger.Debug().
		Str("username", username).
		Msg("Login attempt")

	// Check if backend is available
	if s.server.backend == nil {
		s.logger.Warn().
			Str("username", username).
			Msg("Login rejected - backend not configured")
		return imapserver.ErrAuthFailed
	}

	// Create a context with timeout for the authentication
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt authentication
	userSession, err := s.server.backend.Login(ctx, username, password)
	if err != nil {
		// Log authentication failure with reason
		if authErr, ok := err.(*AuthenticationError); ok {
			s.logger.Warn().
				Str("username", username).
				Str("reason", authErr.Reason).
				Msg("Login rejected")
		} else {
			s.logger.Error().
				Str("username", username).
				Err(err).
				Msg("Login failed due to internal error")
		}
		return imapserver.ErrAuthFailed
	}

	// Authentication successful
	s.username = username
	s.userSession = userSession
	s.state = SessionStateAuthenticated

	s.logger.Info().
		Str("username", username).
		Str("userID", userSession.User.ID.String()).
		Str("sessionID", userSession.ID).
		Msg("Login successful")

	return nil
}

// IsAuthenticated returns true if the session is authenticated.
func (s *Session) IsAuthenticated() bool {
	return s.state >= SessionStateAuthenticated && s.userSession != nil
}

// GetUserSession returns the authenticated user session, or nil if not authenticated.
func (s *Session) GetUserSession() *UserSession {
	return s.userSession
}

// GetState returns the current session state.
func (s *Session) GetState() SessionState {
	return s.state
}

// Authenticated state commands

// Select selects a mailbox for the session.
func (s *Session) Select(mailbox string, options *imap.SelectOptions) (*imap.SelectData, error) {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("SELECT command")

	if !s.IsAuthenticated() {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the mailbox operator
	op := NewMailboxOperator(s.server.backend.Repository(), s.userSession.User.ID)

	// Select the mailbox
	domainMailbox, selectData, err := op.Select(ctx, mailbox, options)
	if err != nil {
		s.logger.Warn().
			Str("mailbox", mailbox).
			Err(err).
			Msg("SELECT failed")
		return nil, err
	}

	// Update session state
	readOnly := options != nil && options.ReadOnly
	s.userSession.SelectMailbox(domainMailbox, readOnly)
	s.state = SessionStateSelected

	s.logger.Info().
		Str("mailbox", mailbox).
		Int64("messages", domainMailbox.MessageCount).
		Int64("unseen", domainMailbox.UnreadCount).
		Bool("readOnly", readOnly).
		Msg("Mailbox selected")

	return selectData, nil
}

// Create creates a new mailbox.
func (s *Session) Create(mailbox string, options *imap.CreateOptions) error {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("CREATE command")

	if !s.IsAuthenticated() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the mailbox operator
	op := NewMailboxOperator(s.server.backend.Repository(), s.userSession.User.ID)

	// Create the mailbox
	if err := op.Create(ctx, mailbox, options); err != nil {
		s.logger.Warn().
			Str("mailbox", mailbox).
			Err(err).
			Msg("CREATE failed")
		return err
	}

	s.logger.Info().
		Str("mailbox", mailbox).
		Msg("Mailbox created")

	return nil
}

// Delete deletes a mailbox.
func (s *Session) Delete(mailbox string) error {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("DELETE command")

	if !s.IsAuthenticated() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the mailbox operator
	op := NewMailboxOperator(s.server.backend.Repository(), s.userSession.User.ID)

	// Delete the mailbox
	if err := op.Delete(ctx, mailbox); err != nil {
		s.logger.Warn().
			Str("mailbox", mailbox).
			Err(err).
			Msg("DELETE failed")
		return err
	}

	s.logger.Info().
		Str("mailbox", mailbox).
		Msg("Mailbox deleted")

	return nil
}

// Rename renames a mailbox.
func (s *Session) Rename(mailbox, newName string, options *imap.RenameOptions) error {
	s.logger.Debug().
		Str("old_name", mailbox).
		Str("new_name", newName).
		Msg("RENAME command")

	if !s.IsAuthenticated() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the mailbox operator
	op := NewMailboxOperator(s.server.backend.Repository(), s.userSession.User.ID)

	// Rename the mailbox
	if err := op.Rename(ctx, mailbox, newName, options); err != nil {
		s.logger.Warn().
			Str("old_name", mailbox).
			Str("new_name", newName).
			Err(err).
			Msg("RENAME failed")
		return err
	}

	s.logger.Info().
		Str("old_name", mailbox).
		Str("new_name", newName).
		Msg("Mailbox renamed")

	return nil
}

// Subscribe subscribes to a mailbox.
func (s *Session) Subscribe(mailbox string) error {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("SUBSCRIBE command")

	if !s.IsAuthenticated() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the mailbox operator
	op := NewMailboxOperator(s.server.backend.Repository(), s.userSession.User.ID)

	// Subscribe to the mailbox
	if err := op.Subscribe(ctx, mailbox); err != nil {
		s.logger.Warn().
			Str("mailbox", mailbox).
			Err(err).
			Msg("SUBSCRIBE failed")
		return err
	}

	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("Subscribed to mailbox")

	return nil
}

// Unsubscribe unsubscribes from a mailbox.
func (s *Session) Unsubscribe(mailbox string) error {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("UNSUBSCRIBE command")

	if !s.IsAuthenticated() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the mailbox operator
	op := NewMailboxOperator(s.server.backend.Repository(), s.userSession.User.ID)

	// Unsubscribe from the mailbox
	if err := op.Unsubscribe(ctx, mailbox); err != nil {
		s.logger.Warn().
			Str("mailbox", mailbox).
			Err(err).
			Msg("UNSUBSCRIBE failed")
		return err
	}

	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("Unsubscribed from mailbox")

	return nil
}

// List lists mailboxes matching the given criteria.
func (s *Session) List(w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) error {
	s.logger.Debug().
		Str("ref", ref).
		Strs("patterns", patterns).
		Msg("LIST command")

	if !s.IsAuthenticated() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the mailbox lister
	lister := NewMailboxLister(s.server.backend.Repository(), s.userSession.User.ID)

	// List mailboxes
	if err := lister.List(ctx, w, ref, patterns, options); err != nil {
		s.logger.Warn().
			Str("ref", ref).
			Strs("patterns", patterns).
			Err(err).
			Msg("LIST failed")
		return err
	}

	return nil
}

// Status returns the status of a mailbox.
func (s *Session) Status(mailbox string, options *imap.StatusOptions) (*imap.StatusData, error) {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("STATUS command")

	if !s.IsAuthenticated() {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the mailbox operator
	op := NewMailboxOperator(s.server.backend.Repository(), s.userSession.User.ID)

	// Get status
	statusData, err := op.Status(ctx, mailbox, options)
	if err != nil {
		s.logger.Warn().
			Str("mailbox", mailbox).
			Err(err).
			Msg("STATUS failed")
		return nil, err
	}

	return statusData, nil
}

// Append appends a message to a mailbox.
func (s *Session) Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error) {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("APPEND command")

	// TODO: Implement append
	return nil, &imap.Error{
		Type: imap.StatusResponseTypeNo,
		Code: imap.ResponseCodeTryCreate,
		Text: "Mailbox does not exist",
	}
}

// Poll checks for mailbox updates (used for unilateral updates).
func (s *Session) Poll(w *imapserver.UpdateWriter, allowExpunge bool) error {
	// No updates to send in this basic implementation
	return nil
}

// Idle waits for mailbox updates using the IMAP IDLE extension (RFC 2177).
// It allows the client to receive real-time notifications about mailbox changes.
func (s *Session) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) error {
	s.logger.Debug().Msg("IDLE command started")

	// Check if a mailbox is selected
	if s.userSession == nil || s.userSession.SelectedMailbox == nil {
		s.logger.Debug().Msg("IDLE ended - no mailbox selected")
		<-stop
		return nil
	}

	// Check if IDLE manager is configured
	idleManager := s.server.IdleManager()
	if idleManager == nil {
		// Fallback to simple wait without notifications
		s.logger.Debug().Msg("IDLE manager not configured, using fallback mode")
		<-stop
		s.logger.Debug().Msg("IDLE command ended")
		return nil
	}

	// Create a context for the IDLE operation
	ctx := context.Background()

	// Get the idle handler and process
	handler := idleManager.CreateHandler(s.sessionID)
	defer idleManager.RemoveHandler(s.sessionID)

	err := handler.HandleIdle(
		ctx,
		w,
		stop,
		s.sessionID,
		s.userSession.SelectedMailbox.ID,
		s.userSession.User.ID,
	)

	s.logger.Debug().Msg("IDLE command ended")
	return err
}

// Selected state commands

// Unselect closes the currently selected mailbox.
func (s *Session) Unselect() error {
	s.logger.Debug().Msg("UNSELECT command")
	return nil
}

// Expunge permanently removes all messages with the \Deleted flag.
func (s *Session) Expunge(w *imapserver.ExpungeWriter, uids *imap.UIDSet) error {
	s.logger.Debug().Msg("EXPUNGE command")

	if !s.IsAuthenticated() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Check if a mailbox is selected
	if s.userSession == nil || s.userSession.SelectedMailbox == nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Check if mailbox is read-only
	if s.userSession.IsReadOnly {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Mailbox is read-only",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create the expunge handler
	handler := NewExpungeHandler(
		s.server.backend.Repository(),
		s.userSession.User.ID,
		s.userSession.SelectedMailbox,
	)

	// Execute the expunge
	if err := handler.Expunge(ctx, w, uids); err != nil {
		s.logger.Warn().
			Err(err).
			Msg("EXPUNGE failed")
		return err
	}

	s.logger.Debug().Msg("EXPUNGE completed successfully")

	return nil
}

// Search searches for messages matching the given criteria.
func (s *Session) Search(kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error) {
	s.logger.Debug().Msg("SEARCH command")

	if !s.IsAuthenticated() {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Check if a mailbox is selected
	if s.userSession == nil || s.userSession.SelectedMailbox == nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create the search handler
	handler := NewSearchHandler(
		s.server.backend.Repository(),
		s.userSession.User.ID,
		s.userSession.SelectedMailbox,
	)

	// Execute the search
	result, err := handler.Search(ctx, kind, criteria, options)
	if err != nil {
		s.logger.Warn().
			Err(err).
			Msg("SEARCH failed")
		return nil, err
	}

	// Log search results
	if result != nil && result.All != nil {
		switch ns := result.All.(type) {
		case imap.SeqSet:
			s.logger.Debug().
				Str("results", ns.String()).
				Msg("SEARCH completed")
		case imap.UIDSet:
			s.logger.Debug().
				Str("results", ns.String()).
				Msg("UID SEARCH completed")
		}
	} else {
		s.logger.Debug().Msg("SEARCH completed with no results")
	}

	return result, nil
}

// Fetch retrieves message data.
func (s *Session) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error {
	s.logger.Debug().Msg("FETCH command")

	if !s.IsAuthenticated() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Check if a mailbox is selected
	if s.userSession == nil || s.userSession.SelectedMailbox == nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create the fetch handler
	handler := NewFetchHandler(
		s.server.backend.Repository(),
		s.userSession.User.ID,
		s.userSession.SelectedMailbox,
	)

	// Execute the fetch
	if err := handler.Fetch(ctx, w, numSet, options); err != nil {
		s.logger.Warn().
			Err(err).
			Msg("FETCH failed")
		return err
	}

	return nil
}

// Store alters message flags.
func (s *Session) Store(w *imapserver.FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error {
	s.logger.Debug().
		Bool("silent", flags.Silent).
		Int("op", int(flags.Op)).
		Msg("STORE command")

	if !s.IsAuthenticated() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Check if a mailbox is selected
	if s.userSession == nil || s.userSession.SelectedMailbox == nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Check if mailbox is read-only
	if s.userSession.IsReadOnly {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Mailbox is read-only",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create the store handler
	handler := NewStoreHandler(
		s.server.backend.Repository(),
		s.userSession.User.ID,
		s.userSession.SelectedMailbox,
	)

	// Execute the store operation
	if err := handler.Store(ctx, w, numSet, flags, options); err != nil {
		s.logger.Warn().
			Err(err).
			Msg("STORE failed")
		return err
	}

	s.logger.Debug().Msg("STORE completed successfully")

	return nil
}

// Copy copies messages to another mailbox.
func (s *Session) Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error) {
	s.logger.Debug().
		Str("dest", dest).
		Msg("COPY command")

	if !s.IsAuthenticated() {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Check if a mailbox is selected
	if s.userSession == nil || s.userSession.SelectedMailbox == nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create the copy handler
	handler := NewCopyHandler(
		s.server.backend.Repository(),
		s.userSession.User.ID,
		s.userSession.SelectedMailbox,
	)

	// Execute the copy
	copyData, err := handler.Copy(ctx, numSet, dest)
	if err != nil {
		s.logger.Warn().
			Str("dest", dest).
			Err(err).
			Msg("COPY failed")
		return nil, err
	}

	uids, _ := copyData.DestUIDs.Nums()
	s.logger.Info().
		Str("dest", dest).
		Int("copied", len(uids)).
		Msg("COPY completed successfully")

	return copyData, nil
}

// Move moves messages to another mailbox (RFC 6851).
func (s *Session) Move(w *imapserver.MoveWriter, numSet imap.NumSet, dest string) error {
	s.logger.Debug().
		Str("dest", dest).
		Msg("MOVE command")

	if !s.IsAuthenticated() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Not authenticated",
		}
	}

	// Check if a mailbox is selected
	if s.userSession == nil || s.userSession.SelectedMailbox == nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "No mailbox selected",
		}
	}

	// Check if mailbox is read-only
	if s.userSession.IsReadOnly {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Mailbox is read-only",
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create the copy handler
	handler := NewCopyHandler(
		s.server.backend.Repository(),
		s.userSession.User.ID,
		s.userSession.SelectedMailbox,
	)

	// Execute the move
	copyData, expungedSeqNums, err := handler.Move(ctx, numSet, dest)
	if err != nil {
		s.logger.Warn().
			Str("dest", dest).
			Err(err).
			Msg("MOVE failed")
		return err
	}

	// Write the COPYUID response
	if err := w.WriteCopyData(copyData); err != nil {
		return err
	}

	// Write EXPUNGE responses for moved messages
	// Messages must be expunged in descending sequence number order
	sortUint32Desc(expungedSeqNums)
	for _, seqNum := range expungedSeqNums {
		if err := w.WriteExpunge(seqNum); err != nil {
			return err
		}
	}

	s.logger.Info().
		Str("dest", dest).
		Int("moved", len(expungedSeqNums)).
		Msg("MOVE completed successfully")

	return nil
}

// Ensure Session implements the imapserver.Session interface.
var _ imapserver.Session = (*Session)(nil)

// Ensure Session implements the imapserver.SessionMove interface for MOVE support.
var _ imapserver.SessionMove = (*Session)(nil)
