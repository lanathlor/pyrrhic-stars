// Package auth verifies Ory Kratos session tokens for the gateway handshake.
//
// The Godot client cannot set custom WebSocket headers, so it logs in against
// Kratos over HTTP (the native "API" flow), receives a session token, and
// passes it as the ?token= query parameter on the WebSocket URL. The gateway
// resolves that token to a Kratos identity via this package.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ErrUnauthenticated is returned when a token is missing, invalid, or expired.
var ErrUnauthenticated = errors.New("auth: unauthenticated")

// Identity is the subset of a Kratos identity the game cares about.
type Identity struct {
	ID       string // Kratos identity UUID, used as the game User.ID
	Email    string
	Username string
}

// SessionVerifier resolves a session token to an authenticated identity.
// It is an interface so the gateway can be tested with a stub and so an
// alternative auth backend could be swapped in.
type SessionVerifier interface {
	Whoami(ctx context.Context, token string) (*Identity, error)
}

// KratosVerifier calls the Kratos public API /sessions/whoami endpoint.
type KratosVerifier struct {
	publicURL string
	client    *http.Client
}

// NewKratosVerifier builds a verifier pointed at the Kratos public base URL
// (e.g. http://localhost:4433).
func NewKratosVerifier(publicURL string) *KratosVerifier {
	return &KratosVerifier{
		publicURL: strings.TrimRight(publicURL, "/"),
		client:    &http.Client{Timeout: 5 * time.Second},
	}
}

// whoamiResponse mirrors the relevant fields of the Kratos /sessions/whoami body.
type whoamiResponse struct {
	Active   bool           `json:"active"`
	Identity kratosIdentity `json:"identity"`
}

type kratosIdentity struct {
	ID     string       `json:"id"`
	Traits kratosTraits `json:"traits"`
}

type kratosTraits struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

// Whoami resolves a session token to an identity. Returns ErrUnauthenticated
// for an empty token or any non-2xx Kratos response (expired/invalid session).
func (v *KratosVerifier) Whoami(ctx context.Context, token string) (*Identity, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrUnauthenticated
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.publicURL+"/sessions/whoami", nil)
	if err != nil {
		return nil, fmt.Errorf("auth: build request: %w", err)
	}
	req.Header.Set("X-Session-Token", token)
	req.Header.Set("Accept", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth: kratos request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrUnauthenticated
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth: kratos whoami status %d", resp.StatusCode)
	}

	var body whoamiResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("auth: decode whoami: %w", err)
	}
	if !body.Active || body.Identity.ID == "" {
		return nil, ErrUnauthenticated
	}

	return &Identity{
		ID:       body.Identity.ID,
		Email:    body.Identity.Traits.Email,
		Username: body.Identity.Traits.Username,
	}, nil
}
