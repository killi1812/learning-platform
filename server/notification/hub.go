package notification

import (
	"sync"

	"github.com/google/uuid"
)

// subscriberBuffer is the per-stream channel size. A full buffer drops the event.
// The DB stays the source of truth, so a dropped event is recovered by refetching the inbox.
const subscriberBuffer = 16

// Hub fans out new notifications to open SSE streams of one process.
// Single-instance only: with several API instances, events reach only streams on the same instance.
type Hub struct {
	mu   sync.Mutex
	subs map[uuid.UUID]map[chan NotificationDto]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[uuid.UUID]map[chan NotificationDto]struct{})}
}

// Subscribe registers a stream for userUuid. The returned cancel func must be called once.
func (h *Hub) Subscribe(userUuid uuid.UUID) (<-chan NotificationDto, func()) {
	ch := make(chan NotificationDto, subscriberBuffer)

	h.mu.Lock()
	if h.subs[userUuid] == nil {
		h.subs[userUuid] = make(map[chan NotificationDto]struct{})
	}
	h.subs[userUuid][ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.subs[userUuid], ch)
		if len(h.subs[userUuid]) == 0 {
			delete(h.subs, userUuid)
		}
	}
	return ch, cancel
}

// Publish sends dto to every open stream of userUuid without blocking.
func (h *Hub) Publish(userUuid uuid.UUID, dto NotificationDto) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[userUuid] {
		select {
		case ch <- dto:
		default:
		}
	}
}
