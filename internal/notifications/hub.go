package notifications

import (
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[chan NotificationResponse]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[uuid.UUID]map[chan NotificationResponse]struct{})}
}

func (h *Hub) Subscribe(userID uuid.UUID) (<-chan NotificationResponse, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	events := make(chan NotificationResponse, 32)
	if h.subscribers[userID] == nil {
		h.subscribers[userID] = make(map[chan NotificationResponse]struct{})
	}
	h.subscribers[userID][events] = struct{}{}

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		if _, ok := h.subscribers[userID][events]; !ok {
			return
		}
		delete(h.subscribers[userID], events)
		close(events)
		if len(h.subscribers[userID]) == 0 {
			delete(h.subscribers, userID)
		}
	}

	return events, unsubscribe
}

func (h *Hub) Publish(userID uuid.UUID, notification NotificationResponse) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for events := range h.subscribers[userID] {
		select {
		case events <- notification:
		default:
		}
	}
}
