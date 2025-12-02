package chatservice

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/dphaener/shiki-cli/internal/conversation"
)

// ErrNotFound is returned when an entity is not found.
var ErrNotFound = errors.New("not found")

// MemoryStore provides an in-memory implementation of Store.
type MemoryStore struct {
	sessions map[string]*conversation.Session
	messages map[string]*conversation.Message
	mu       sync.RWMutex
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*conversation.Session),
		messages: make(map[string]*conversation.Message),
	}
}

// SaveSession persists a session.
func (s *MemoryStore) SaveSession(ctx context.Context, session *conversation.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

// GetSession retrieves a session by ID.
func (s *MemoryStore) GetSession(ctx context.Context, id string) (*conversation.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return session, nil
}

// ListSessions retrieves sessions matching the filter.
func (s *MemoryStore) ListSessions(ctx context.Context, opts ListSessionOpts) ([]*conversation.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*conversation.Session
	for _, session := range s.sessions {
		// Apply filters
		if opts.Mode != nil && session.Mode != *opts.Mode {
			continue
		}
		if opts.Status != nil && session.Status != *opts.Status {
			continue
		}
		result = append(result, session)
	}

	// Sort by created time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	// Apply pagination
	if opts.Offset > 0 {
		if opts.Offset >= len(result) {
			return []*conversation.Session{}, nil
		}
		result = result[opts.Offset:]
	}
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, nil
}

// DeleteSession removes a session.
func (s *MemoryStore) DeleteSession(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

// SaveMessage persists a message.
func (s *MemoryStore) SaveMessage(ctx context.Context, msg *conversation.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[msg.ID] = msg
	return nil
}

// GetMessage retrieves a message by ID.
func (s *MemoryStore) GetMessage(ctx context.Context, id string) (*conversation.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msg, ok := s.messages[id]
	if !ok {
		return nil, ErrNotFound
	}
	return msg, nil
}

// ListMessages retrieves messages for a session.
func (s *MemoryStore) ListMessages(ctx context.Context, sessionID string, opts ListMessagesOpts) ([]*conversation.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*conversation.Message
	for _, msg := range s.messages {
		if msg.SessionID != sessionID {
			continue
		}
		// Apply filters
		if opts.ParticipantID != "" && msg.ParticipantID != opts.ParticipantID {
			continue
		}
		if opts.SinceTurn > 0 && msg.Turn < opts.SinceTurn {
			continue
		}
		result = append(result, msg)
	}

	// Sort by created time
	if opts.Order == "desc" {
		sort.Slice(result, func(i, j int) bool {
			return result[i].CreatedAt.After(result[j].CreatedAt)
		})
	} else {
		sort.Slice(result, func(i, j int) bool {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		})
	}

	// Apply limit
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, nil
}

// DeleteMessage removes a message.
func (s *MemoryStore) DeleteMessage(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.messages, id)
	return nil
}

// DeleteSessionMessages removes all messages for a session.
func (s *MemoryStore) DeleteSessionMessages(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, msg := range s.messages {
		if msg.SessionID == sessionID {
			delete(s.messages, id)
		}
	}
	return nil
}

// Clear removes all data from the store.
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = make(map[string]*conversation.Session)
	s.messages = make(map[string]*conversation.Message)
}

// SessionCount returns the number of sessions in the store.
func (s *MemoryStore) SessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// MessageCount returns the number of messages in the store.
func (s *MemoryStore) MessageCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.messages)
}
