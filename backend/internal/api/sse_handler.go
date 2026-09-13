package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sushi-clocks/backend/internal/auth"
	"github.com/sushi-clocks/backend/internal/domain"
	"github.com/sushi-clocks/backend/internal/sse"
)

type SSEHandler struct {
	hub *sse.Hub
}

func NewSSEHandler(hub *sse.Hub) *SSEHandler {
	return &SSEHandler{hub: hub}
}

func (h *SSEHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		RespondError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	claims := auth.GetClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	companyID := claims.CompanyID
	if claims.SystemRole == domain.RoleSuperAdmin {
		// Super Admin can subscribe to any company if specified in query
		if qCompanyID := r.URL.Query().Get("company_id"); qCompanyID != "" {
			companyID = qCompanyID
		}
	}

	if companyID == "" {
		RespondError(w, http.StatusBadRequest, "company_id required")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Disable server write deadline so long-lived SSE stream isn't terminated by server.WriteTimeout
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	ch, cleanup := h.hub.Subscribe(companyID)
	defer cleanup()

	// Initial connect handshake event
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\",\"company_id\":\"%s\"}\n\n", companyID)
	flusher.Flush()

	// 15-second keep-alive heartbeat ticker to prevent proxy or browser idle timeouts
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Send SSE comment to keep TCP connection alive
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
