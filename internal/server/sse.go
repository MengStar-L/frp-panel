package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// sseBegin sets event-stream headers and returns the flusher.
func sseBegin(w http.ResponseWriter) (http.Flusher, bool) {
	fl, ok := w.(http.Flusher)
	if !ok {
		return nil, false
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no") // disable proxy buffering
	w.WriteHeader(http.StatusOK)
	fl.Flush()
	return fl, true
}

// writeSSE emits one named event. Multi-line data is split across data: lines
// per the SSE spec.
func writeSSE(w io.Writer, event, data string) {
	fmt.Fprintf(w, "event: %s\n", event)
	for _, line := range strings.Split(data, "\n") {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	io.WriteString(w, "\n")
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// handleLogStream streams the frp process log: backlog first, then live lines.
func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	fl, ok := sseBegin(w)
	if !ok {
		writeError(w, http.StatusInternalServerError, "服务器不支持流式响应")
		return
	}
	hub := s.app.Logs()
	for _, line := range hub.History() {
		writeSSE(w, "log", line)
	}
	fl.Flush()

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()

	for {
		select {
		case line, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(w, "log", line)
			fl.Flush()
		case <-ping.C:
			io.WriteString(w, ": ping\n\n")
			fl.Flush()
		case <-r.Context().Done():
			return
		case <-s.baseCtx.Done():
			return
		}
	}
}

// handleProgressStream streams install/update progress. It replays the most
// recent event so a late subscriber renders the current state immediately.
func (s *Server) handleProgressStream(w http.ResponseWriter, r *http.Request) {
	fl, ok := sseBegin(w)
	if !ok {
		writeError(w, http.StatusInternalServerError, "服务器不支持流式响应")
		return
	}
	hub := s.app.Progress()
	if last, ok := hub.Last(); ok {
		writeSSE(w, "progress", mustJSON(last))
		fl.Flush()
	}

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()

	for {
		select {
		case p, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(w, "progress", mustJSON(p))
			fl.Flush()
		case <-ping.C:
			io.WriteString(w, ": ping\n\n")
			fl.Flush()
		case <-r.Context().Done():
			return
		case <-s.baseCtx.Done():
			return
		}
	}
}
