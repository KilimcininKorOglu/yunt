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

	// TODO: Implement mailbox selection
	return nil, &imap.Error{
		Type: imap.StatusResponseTypeNo,
		Code: imap.ResponseCodeNonExistent,
		Text: "Mailbox does not exist",
	}
}

// Create creates a new mailbox.
func (s *Session) Create(mailbox string, options *imap.CreateOptions) error {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("CREATE command")

	// TODO: Implement mailbox creation
	return &imap.Error{
		Type: imap.StatusResponseTypeNo,
		Text: "CREATE not yet implemented",
	}
}

// Delete deletes a mailbox.
func (s *Session) Delete(mailbox string) error {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("DELETE command")

	// TODO: Implement mailbox deletion
	return &imap.Error{
		Type: imap.StatusResponseTypeNo,
		Text: "DELETE not yet implemented",
	}
}

// Rename renames a mailbox.
func (s *Session) Rename(mailbox, newName string, options *imap.RenameOptions) error {
	s.logger.Debug().
		Str("old_name", mailbox).
		Str("new_name", newName).
		Msg("RENAME command")

	// TODO: Implement mailbox renaming
	return &imap.Error{
		Type: imap.StatusResponseTypeNo,
		Text: "RENAME not yet implemented",
	}
}

// Subscribe subscribes to a mailbox.
func (s *Session) Subscribe(mailbox string) error {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("SUBSCRIBE command")

	// TODO: Implement subscription
	return &imap.Error{
		Type: imap.StatusResponseTypeNo,
		Text: "SUBSCRIBE not yet implemented",
	}
}

// Unsubscribe unsubscribes from a mailbox.
func (s *Session) Unsubscribe(mailbox string) error {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("UNSUBSCRIBE command")

	// TODO: Implement unsubscription
	return &imap.Error{
		Type: imap.StatusResponseTypeNo,
		Text: "UNSUBSCRIBE not yet implemented",
	}
}

// List lists mailboxes matching the given criteria.
func (s *Session) List(w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) error {
	s.logger.Debug().
		Str("ref", ref).
		Strs("patterns", patterns).
		Msg("LIST command")

	// TODO: Implement mailbox listing
	// For now, return empty list (no mailboxes)
	return nil
}

// Status returns the status of a mailbox.
func (s *Session) Status(mailbox string, options *imap.StatusOptions) (*imap.StatusData, error) {
	s.logger.Debug().
		Str("mailbox", mailbox).
		Msg("STATUS command")

	// TODO: Implement status
	return nil, &imap.Error{
		Type: imap.StatusResponseTypeNo,
		Code: imap.ResponseCodeNonExistent,
		Text: "Mailbox does not exist",
	}
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

// Idle waits for mailbox updates.
func (s *Session) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) error {
	s.logger.Debug().Msg("IDLE command started")
	<-stop
	s.logger.Debug().Msg("IDLE command ended")
	return nil
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

	// TODO: Implement expunge
	return nil
}

// Search searches for messages matching the given criteria.
func (s *Session) Search(kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error) {
	s.logger.Debug().Msg("SEARCH command")

	// TODO: Implement search
	return &imap.SearchData{}, nil
}

// Fetch retrieves message data.
func (s *Session) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error {
	s.logger.Debug().Msg("FETCH command")

	// TODO: Implement fetch
	return nil
}

// Store alters message flags.
func (s *Session) Store(w *imapserver.FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error {
	s.logger.Debug().Msg("STORE command")

	// TODO: Implement store
	return nil
}

// Copy copies messages to another mailbox.
func (s *Session) Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error) {
	s.logger.Debug().
		Str("dest", dest).
		Msg("COPY command")

	// TODO: Implement copy
	return nil, &imap.Error{
		Type: imap.StatusResponseTypeNo,
		Code: imap.ResponseCodeTryCreate,
		Text: "Destination mailbox does not exist",
	}
}

// Ensure Session implements the imapserver.Session interface.
var _ imapserver.Session = (*Session)(nil)
