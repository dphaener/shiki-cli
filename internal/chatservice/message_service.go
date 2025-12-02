package chatservice

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/dphaener/shiki-cli/internal/broker"
	"github.com/dphaener/shiki-cli/internal/conversation"
)

// messageService implements MessageService.
type messageService struct {
	store  Store
	broker *broker.Broker

	// Stream tracking
	streamingMessages map[string]*conversation.Message
	streamDeltaIndex  map[string]*int32
	mu                sync.RWMutex
}

// NewMessageService creates a new message service.
func NewMessageService(store Store, broker *broker.Broker) MessageService {
	return &messageService{
		store:             store,
		broker:            broker,
		streamingMessages: make(map[string]*conversation.Message),
		streamDeltaIndex:  make(map[string]*int32),
	}
}

// Create creates a new message.
func (s *messageService) Create(ctx context.Context, sessionID string, msg *conversation.Message) error {
	msg.SessionID = sessionID

	if err := s.store.SaveMessage(ctx, msg); err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	if s.broker != nil {
		s.broker.Publish(broker.NewMessageCreatedEvent(msg))
	}

	return nil
}

// Get retrieves a message by ID.
func (s *messageService) Get(ctx context.Context, id string) (*conversation.Message, error) {
	// Check streaming messages first
	s.mu.RLock()
	if msg, ok := s.streamingMessages[id]; ok {
		s.mu.RUnlock()
		return msg, nil
	}
	s.mu.RUnlock()

	return s.store.GetMessage(ctx, id)
}

// Update saves message state.
func (s *messageService) Update(ctx context.Context, msg *conversation.Message) error {
	if err := s.store.SaveMessage(ctx, msg); err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	if s.broker != nil {
		s.broker.Publish(broker.NewMessageUpdatedEvent(msg))
	}

	return nil
}

// Delete removes a message.
func (s *messageService) Delete(ctx context.Context, id string) error {
	// Remove from streaming if present
	s.mu.Lock()
	delete(s.streamingMessages, id)
	delete(s.streamDeltaIndex, id)
	s.mu.Unlock()

	return s.store.DeleteMessage(ctx, id)
}

// List retrieves messages for a session.
func (s *messageService) List(ctx context.Context, sessionID string, opts ListMessagesOpts) ([]*conversation.Message, error) {
	return s.store.ListMessages(ctx, sessionID, opts)
}

// StartStream creates a new streaming message.
func (s *messageService) StartStream(ctx context.Context, sessionID, participantID string, turn int) (*conversation.Message, error) {
	msg := conversation.NewMessage(sessionID, participantID, conversation.RoleAssistant, turn)
	msg.StartStreaming()

	// Track as streaming
	s.mu.Lock()
	s.streamingMessages[msg.ID] = msg
	idx := int32(0)
	s.streamDeltaIndex[msg.ID] = &idx
	s.mu.Unlock()

	// Save initial message
	if err := s.store.SaveMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("save message: %w", err)
	}

	if s.broker != nil {
		s.broker.Publish(broker.NewStreamStartedEvent(sessionID, msg.ID, participantID, turn))
		s.broker.Publish(broker.NewMessageCreatedEvent(msg))
	}

	return msg, nil
}

// AppendDelta appends text to a streaming message.
func (s *messageService) AppendDelta(ctx context.Context, messageID string, delta string) error {
	s.mu.RLock()
	msg, ok := s.streamingMessages[messageID]
	idxPtr := s.streamDeltaIndex[messageID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("message %s is not streaming", messageID)
	}

	msg.AppendStreamDelta(delta)

	// Get and increment index atomically
	idx := int(atomic.AddInt32(idxPtr, 1) - 1)

	if s.broker != nil {
		s.broker.Publish(broker.NewStreamDeltaEvent(msg.SessionID, messageID, msg.ParticipantID, delta, idx))
	}

	return nil
}

// AddPart adds a part to a message.
func (s *messageService) AddPart(ctx context.Context, messageID string, part conversation.Part) error {
	// Check streaming messages first
	s.mu.RLock()
	msg, isStreaming := s.streamingMessages[messageID]
	s.mu.RUnlock()

	if isStreaming {
		msg.AddPart(part)
	} else {
		var err error
		msg, err = s.store.GetMessage(ctx, messageID)
		if err != nil {
			return err
		}
		msg.AddPart(part)
		if err := s.store.SaveMessage(ctx, msg); err != nil {
			return fmt.Errorf("save message: %w", err)
		}
	}

	if s.broker != nil {
		s.broker.Publish(broker.NewPartAddedEvent(msg.SessionID, messageID, msg.ParticipantID, part))

		// If it's a tool call, also publish tool started event
		if tc, ok := part.(conversation.ToolCallPart); ok {
			s.broker.Publish(broker.NewToolStartedEvent(
				msg.SessionID,
				messageID,
				msg.ParticipantID,
				tc.ID,
				tc.ToolName,
				tc.Input,
				msg.Turn,
			))
		}
	}

	return nil
}

// FinishStream finalizes a streaming message.
func (s *messageService) FinishStream(ctx context.Context, messageID string, reason conversation.FinishReason, inputTokens, outputTokens int) error {
	s.mu.Lock()
	msg, ok := s.streamingMessages[messageID]
	if ok {
		delete(s.streamingMessages, messageID)
		delete(s.streamDeltaIndex, messageID)
	}
	s.mu.Unlock()

	if !ok {
		var err error
		msg, err = s.store.GetMessage(ctx, messageID)
		if err != nil {
			return err
		}
	}

	msg.FinishStreaming()

	// Save to store
	if err := s.store.SaveMessage(ctx, msg); err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	if s.broker != nil {
		s.broker.Publish(broker.NewStreamFinishedEvent(msg.SessionID, messageID, msg.ParticipantID, reason, inputTokens, outputTokens))
		s.broker.Publish(broker.NewMessageUpdatedEvent(msg))
	}

	return nil
}

// UpdateToolCallStatus updates the status of a tool call within a message.
func (s *messageService) UpdateToolCallStatus(ctx context.Context, messageID, toolCallID string, status conversation.ToolCallStatus) error {
	// Check streaming messages first
	s.mu.RLock()
	msg, isStreaming := s.streamingMessages[messageID]
	s.mu.RUnlock()

	if !isStreaming {
		var err error
		msg, err = s.store.GetMessage(ctx, messageID)
		if err != nil {
			return err
		}
	}

	if !msg.UpdateToolCallStatus(toolCallID, status) {
		return fmt.Errorf("tool call %s not found in message %s", toolCallID, messageID)
	}

	if !isStreaming {
		if err := s.store.SaveMessage(ctx, msg); err != nil {
			return fmt.Errorf("save message: %w", err)
		}
	}

	if s.broker != nil {
		s.broker.Publish(broker.NewMessageUpdatedEvent(msg))
	}

	return nil
}

// GetByParticipant retrieves messages for a specific participant.
func (s *messageService) GetByParticipant(ctx context.Context, sessionID, participantID string) ([]*conversation.Message, error) {
	return s.store.ListMessages(ctx, sessionID, ListMessagesOpts{
		ParticipantID: participantID,
		Order:         "asc",
	})
}

// GetLatest retrieves the most recent message for a session.
func (s *messageService) GetLatest(ctx context.Context, sessionID string) (*conversation.Message, error) {
	msgs, err := s.store.ListMessages(ctx, sessionID, ListMessagesOpts{
		Limit: 1,
		Order: "desc",
	})
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, nil
	}
	return msgs[0], nil
}

// GetStreamingMessages returns all currently streaming messages.
func (s *messageService) GetStreamingMessages() []*conversation.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*conversation.Message, 0, len(s.streamingMessages))
	for _, msg := range s.streamingMessages {
		result = append(result, msg)
	}
	return result
}
