package sse

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[chan []byte]struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[chan []byte]struct{}),
	}
}

// Subscribe joins a company room and returns an event channel and cleanup function
func (h *Hub) Subscribe(companyID string) (chan []byte, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan []byte, 32)
	if _, ok := h.rooms[companyID]; !ok {
		h.rooms[companyID] = make(map[chan []byte]struct{})
	}
	h.rooms[companyID][ch] = struct{}{}

	cleanup := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if clients, ok := h.rooms[companyID]; ok {
			delete(clients, ch)
			close(ch)
			if len(clients) == 0 {
				delete(h.rooms, companyID)
			}
		}
	}

	return ch, cleanup
}

// Broadcast sends an SSE formatted event to all subscribers in a company room
func (h *Hub) Broadcast(companyID string, eventType string, payload any) {
	h.mu.RLock()
	clients, ok := h.rooms[companyID]
	if !ok || len(clients) == 0 {
		h.mu.RUnlock()
		return
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		h.mu.RUnlock()
		return
	}

	// Format SSE frame: "event: <type>\ndata: <json>\n\n"
	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, payloadJSON))

	// Fan out non-blocking
	for ch := range clients {
		select {
		case ch <- msg:
		default:
			// Client buffer full; skip to maintain zero latency
		}
	}
	h.mu.RUnlock()
}
