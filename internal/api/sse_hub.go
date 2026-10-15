package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type SSEEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type clientChan chan []byte

type SSEHub struct {
	clients map[clientChan]bool
	mu      sync.RWMutex
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[clientChan]bool),
	}
}

func (h *SSEHub) Subscribe() clientChan {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(clientChan, 64)
	h.clients[ch] = true
	return ch
}

func (h *SSEHub) Unsubscribe(ch clientChan) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
}

func (h *SSEHub) Broadcast(eventType string, payload any) {
	evt := SSEEvent{
		Type:    eventType,
		Payload: payload,
	}
	bytes, err := json.Marshal(evt)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.clients {
		select {
		case ch <- bytes:
		default:
			// client buffer full, drop to prevent blocking
		}
	}
}

func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := h.Subscribe()
	defer h.Unsubscribe(ch)

	// Send initial ping
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}
