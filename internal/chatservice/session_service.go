package chatservice

import (
	"context"
	"fmt"
	"sync"

	"github.com/darinhaener/collab/internal/broker"
	"github.com/darinhaener/collab/internal/conversation"
)

// sessionService implements SessionService.
type sessionService struct {
	store    Store
	broker   *broker.Broker
	activeID string
	mu       sync.RWMutex
}

// NewSessionService creates a new session service.
func NewSessionService(store Store, broker *broker.Broker) SessionService {
	return &sessionService{
		store:  store,
		broker: broker,
	}
}

// Create creates a new session.
func (s *sessionService) Create(ctx context.Context, mode conversation.SessionMode, opts CreateSessionOpts) (*conversation.Session, error) {
	var session *conversation.Session

	switch mode {
	case conversation.ModeCollaboration:
		session = conversation.NewCollaborationSession(opts.Title, opts.WorkDir, opts.TaskFile, opts.MaxTurns)
	case conversation.ModeSpecify:
		session = conversation.NewSpecifySession(opts.FeatureID, opts.FeatureName)
	default:
		session = conversation.NewSession(mode, opts.Title)
		if opts.MaxTurns > 0 {
			session.MaxTurns = opts.MaxTurns
		}
	}

	// Add participants
	for _, p := range opts.Participants {
		session.AddParticipant(p)
	}

	// Add metadata
	for k, v := range opts.Metadata {
		session.SetMetadata(k, v)
	}

	// Save to store
	if err := s.store.SaveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	// Publish event
	if s.broker != nil {
		s.broker.Publish(broker.NewSessionCreatedEvent(session))
	}

	return session, nil
}

// Get retrieves a session by ID.
func (s *sessionService) Get(ctx context.Context, id string) (*conversation.Session, error) {
	return s.store.GetSession(ctx, id)
}

// Update saves session state.
func (s *sessionService) Update(ctx context.Context, session *conversation.Session) error {
	if err := s.store.SaveSession(ctx, session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	if s.broker != nil {
		s.broker.Publish(broker.NewSessionUpdatedEvent(session))
	}

	return nil
}

// Delete removes a session.
func (s *sessionService) Delete(ctx context.Context, id string) error {
	// Clear active if this is the active session
	s.mu.Lock()
	if s.activeID == id {
		s.activeID = ""
	}
	s.mu.Unlock()

	return s.store.DeleteSession(ctx, id)
}

// List retrieves sessions matching the filter.
func (s *sessionService) List(ctx context.Context, opts ListSessionOpts) ([]*conversation.Session, error) {
	return s.store.ListSessions(ctx, opts)
}

// Start marks the session as running.
func (s *sessionService) Start(ctx context.Context, id string) error {
	session, err := s.store.GetSession(ctx, id)
	if err != nil {
		return err
	}

	session.Start()

	if err := s.store.SaveSession(ctx, session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	// Set as active
	s.mu.Lock()
	s.activeID = id
	s.mu.Unlock()

	if s.broker != nil {
		s.broker.Publish(broker.NewSessionStartedEvent(session))
	}

	return nil
}

// Pause marks the session as paused.
func (s *sessionService) Pause(ctx context.Context, id string) error {
	session, err := s.store.GetSession(ctx, id)
	if err != nil {
		return err
	}

	session.Pause()

	if err := s.store.SaveSession(ctx, session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	if s.broker != nil {
		s.broker.Publish(broker.NewSessionPausedEvent(session))
	}

	return nil
}

// Resume marks the session as running again.
func (s *sessionService) Resume(ctx context.Context, id string) error {
	session, err := s.store.GetSession(ctx, id)
	if err != nil {
		return err
	}

	session.Resume()

	if err := s.store.SaveSession(ctx, session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	if s.broker != nil {
		s.broker.Publish(broker.NewSessionStartedEvent(session))
	}

	return nil
}

// Complete marks the session as completed.
func (s *sessionService) Complete(ctx context.Context, id string, deliverablePath string) error {
	session, err := s.store.GetSession(ctx, id)
	if err != nil {
		return err
	}

	session.Complete(deliverablePath)

	if err := s.store.SaveSession(ctx, session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	// Clear active
	s.mu.Lock()
	if s.activeID == id {
		s.activeID = ""
	}
	s.mu.Unlock()

	if s.broker != nil {
		s.broker.Publish(broker.NewSessionCompletedEvent(session, deliverablePath))
	}

	return nil
}

// Error marks the session as errored.
func (s *sessionService) Error(ctx context.Context, id string, sessionErr error) error {
	session, err := s.store.GetSession(ctx, id)
	if err != nil {
		return err
	}

	session.Error()

	if err := s.store.SaveSession(ctx, session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	// Clear active
	s.mu.Lock()
	if s.activeID == id {
		s.activeID = ""
	}
	s.mu.Unlock()

	if s.broker != nil {
		s.broker.Publish(broker.NewSessionErrorEvent(session, sessionErr))
	}

	return nil
}

// NextTurn advances to the next turn.
func (s *sessionService) NextTurn(ctx context.Context, id string, nextParticipantID string) error {
	session, err := s.store.GetSession(ctx, id)
	if err != nil {
		return err
	}

	session.NextTurn(nextParticipantID)

	if err := s.store.SaveSession(ctx, session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	if s.broker != nil {
		s.broker.Publish(broker.NewSessionUpdatedEvent(session, "current_turn", "active_participant"))
	}

	return nil
}

// GetActive returns the currently active session.
func (s *sessionService) GetActive(ctx context.Context) (*conversation.Session, error) {
	s.mu.RLock()
	id := s.activeID
	s.mu.RUnlock()

	if id == "" {
		return nil, nil
	}

	return s.store.GetSession(ctx, id)
}
