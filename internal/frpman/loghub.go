package frpman

import (
	"bytes"
	"regexp"
	"sync"
)

// ansiRe matches ANSI/VT100 escape sequences (e.g. color codes) that frp emits
// to a TTY, so logs render cleanly in the browser.
var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*[A-Za-z]")

// LogHub keeps a bounded ring buffer of recent log lines and fans new lines out
// to live subscribers (SSE connections). Sends to subscribers are non-blocking:
// a slow consumer drops lines rather than stalling the writer.
type LogHub struct {
	mu   sync.Mutex
	ring []string
	max  int
	subs map[chan string]struct{}
}

// NewLogHub returns a hub retaining up to max recent lines.
func NewLogHub(max int) *LogHub {
	if max <= 0 {
		max = 1000
	}
	return &LogHub{max: max, subs: make(map[chan string]struct{})}
}

// Append stores a line and broadcasts it to subscribers.
func (h *LogHub) Append(line string) {
	h.mu.Lock()
	h.ring = append(h.ring, line)
	if len(h.ring) > h.max {
		h.ring = h.ring[len(h.ring)-h.max:]
	}
	for ch := range h.subs {
		select {
		case ch <- line:
		default: // subscriber is behind; drop
		}
	}
	h.mu.Unlock()
}

// History returns a copy of the retained lines.
func (h *LogHub) History() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, len(h.ring))
	copy(out, h.ring)
	return out
}

// Clear empties the ring buffer.
func (h *LogHub) Clear() {
	h.mu.Lock()
	h.ring = nil
	h.mu.Unlock()
}

// Subscribe returns a buffered channel that receives subsequent lines.
func (h *LogHub) Subscribe() chan string {
	ch := make(chan string, 256)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes and closes a subscriber channel.
func (h *LogHub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// lineWriter is an io.Writer that splits incoming bytes into lines and appends
// each completed line to a LogHub. Partial lines are buffered until the
// terminating newline arrives.
type lineWriter struct {
	hub *LogHub
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(p)
	for {
		data := w.buf.Bytes()
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			break
		}
		line := string(bytes.TrimRight(data[:idx], "\r"))
		w.hub.Append(ansiRe.ReplaceAllString(line, ""))
		w.buf.Next(idx + 1)
	}
	// Guard against an unbounded line with no newline.
	if w.buf.Len() > 1<<20 {
		w.hub.Append(w.buf.String())
		w.buf.Reset()
	}
	return len(p), nil
}
