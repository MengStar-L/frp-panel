package core

import (
	"sync"

	"frppanel/internal/frpman"
)

// ProgressHub broadcasts download/install progress to SSE subscribers and
// retains the most recent event so a client that connects mid-flight sees the
// current state immediately.
type ProgressHub struct {
	mu   sync.Mutex
	last *frpman.Progress
	subs map[chan frpman.Progress]struct{}
}

// NewProgressHub creates an empty hub.
func NewProgressHub() *ProgressHub {
	return &ProgressHub{subs: make(map[chan frpman.Progress]struct{})}
}

// Reset clears the retained event before a new operation begins.
func (h *ProgressHub) Reset() {
	h.mu.Lock()
	h.last = nil
	h.mu.Unlock()
}

// Publish stores the event as the latest and fans it out (non-blocking).
func (h *ProgressHub) Publish(p frpman.Progress) {
	h.mu.Lock()
	cp := p
	h.last = &cp
	for ch := range h.subs {
		select {
		case ch <- p:
		default:
		}
	}
	h.mu.Unlock()
}

// Last returns the most recent event, if any.
func (h *ProgressHub) Last() (frpman.Progress, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.last == nil {
		return frpman.Progress{}, false
	}
	return *h.last, true
}

// Subscribe returns a buffered channel of subsequent events.
func (h *ProgressHub) Subscribe() chan frpman.Progress {
	ch := make(chan frpman.Progress, 64)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes and closes a subscriber.
func (h *ProgressHub) Unsubscribe(ch chan frpman.Progress) {
	h.mu.Lock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
	h.mu.Unlock()
}
