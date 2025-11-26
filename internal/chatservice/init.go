package chatservice

import (
	"github.com/darinhaener/collab/internal/broker"
)

// Services holds all chat-related services.
type Services struct {
	Broker     *broker.Broker
	Store      Store
	SessionSvc SessionService
	MessageSvc MessageService
}

// NewServices creates and wires up all chat services.
func NewServices() *Services {
	eventBroker := broker.New()
	store := NewMemoryStore()

	return &Services{
		Broker:     eventBroker,
		Store:      store,
		SessionSvc: NewSessionService(store, eventBroker),
		MessageSvc: NewMessageService(store, eventBroker),
	}
}

// Shutdown gracefully shuts down all services.
func (s *Services) Shutdown() {
	if s.Broker != nil {
		s.Broker.Shutdown()
	}
}
