package settings

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"codex-online/server/internal/auth"
	"codex-online/server/internal/validation"
)

// maxBodyBytes bounds how much we read from a PUT request body.
const maxBodyBytes = maxDataBytes + 1

// Handler serves the user settings REST API. Unlike the WebSocket handshake,
// HTTP requests can carry headers, so the Kratos token is read from the
// Authorization header rather than a query parameter.
type Handler struct {
	verifier auth.SessionVerifier
	svc      *Service
	devMode  bool
}

// NewHandler builds a settings handler. devMode enables the ?uuid= bypass used
// for local iteration and the MCP harness when Kratos is not running.
func NewHandler(verifier auth.SessionVerifier, svc *Service, devMode bool) *Handler {
	return &Handler{verifier: verifier, svc: svc, devMode: devMode}
}

// Register mounts the settings routes on the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/settings", h.get)
	mux.HandleFunc("PUT /api/v1/settings", h.put)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUser(w, r)
	if !ok {
		return
	}
	data, err := h.svc.Get(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, data)
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUser(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	if err := h.svc.Save(userID, string(body)); err != nil {
		switch {
		case errors.Is(err, ErrTooLarge):
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		case errors.Is(err, ErrInvalidJSON):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// resolveUser authenticates the request and returns the user UUID. On failure it
// writes a 401 and returns ok=false.
func (h *Handler) resolveUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	if token := bearerToken(r); token != "" {
		id, err := h.verifier.Whoami(r.Context(), token)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return "", false
		}
		return id.ID, true
	}
	// Dev bypass: trust a client-supplied UUID only when dev mode is enabled.
	if h.devMode {
		if uuid := r.URL.Query().Get("uuid"); validation.IsValidUUID(uuid) {
			return uuid, true
		}
	}
	http.Error(w, "missing session token", http.StatusUnauthorized)
	return "", false
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
