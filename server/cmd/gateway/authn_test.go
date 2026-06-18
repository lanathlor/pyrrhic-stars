package main

import (
	"context"
	"net/http/httptest"
	"testing"

	"codex-online/server/internal/auth"
	"codex-online/server/internal/persistence"
)

// authRepo records upserts and returns a stored user, so authenticateRequest
// can resolve a username after auth.
type authRepo struct {
	stubRepo
	users map[string]string // id -> username
}

func newAuthRepo() *authRepo { return &authRepo{users: map[string]string{}} }

func (r *authRepo) UpsertUser(id, username string) error {
	if _, ok := r.users[id]; !ok {
		r.users[id] = username
	}
	return nil
}

func (r *authRepo) UpsertUserSyncName(id, username string) error {
	r.users[id] = username
	return nil
}

func (r *authRepo) GetUser(id string) (*persistence.User, error) {
	name, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	return &persistence.User{ID: id, Username: name}, nil
}

func (r *authRepo) GetCharacters(string) ([]*persistence.Character, error) { return nil, nil }

// stubVerifier resolves a single known-good token.
type stubVerifier struct {
	goodToken string
	identity  *auth.Identity
}

func (v stubVerifier) Whoami(_ context.Context, token string) (*auth.Identity, error) {
	if token == v.goodToken {
		return v.identity, nil
	}
	return nil, auth.ErrUnauthenticated
}

const (
	kratosUUID = "11111111-2222-3333-4444-555555555555"
	validToken = "good"
)

func TestAuthenticateRequest_ValidToken(t *testing.T) {
	gw := newTestGateway(newAuthRepo())
	gw.verifier = stubVerifier{
		goodToken: validToken,
		identity:  &auth.Identity{ID: kratosUUID, Email: "a@b.com", Username: "Kratonaut"},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws?token=good", nil)
	uuid, username, _, ok := authenticateRequest(gw, w, req)
	if !ok {
		t.Fatalf("auth failed, status=%d", w.Code)
	}
	if uuid != kratosUUID {
		t.Errorf("uuid = %q, want kratos identity id", uuid)
	}
	if username != "Kratonaut" {
		t.Errorf("username = %q, want synced from kratos", username)
	}
}

func TestAuthenticateRequest_BadToken(t *testing.T) {
	gw := newTestGateway(newAuthRepo())
	gw.verifier = stubVerifier{goodToken: validToken}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws?token=wrong", nil)
	if _, _, _, ok := authenticateRequest(gw, w, req); ok {
		t.Fatal("expected auth to fail for bad token")
	}
	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthenticateRequest_NoTokenRejectedWhenNotDev(t *testing.T) {
	gw := newTestGateway(newAuthRepo())
	gw.verifier = stubVerifier{goodToken: validToken}
	gw.devMode = false

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws?uuid="+kratosUUID, nil)
	if _, _, _, ok := authenticateRequest(gw, w, req); ok {
		t.Fatal("expected uuid path to be rejected outside dev mode")
	}
	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthenticateRequest_DevBypass(t *testing.T) {
	gw := newTestGateway(newAuthRepo())
	gw.verifier = stubVerifier{goodToken: validToken}
	gw.devMode = true

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws?uuid="+kratosUUID+"&username=DevGuy", nil)
	uuid, username, _, ok := authenticateRequest(gw, w, req)
	if !ok {
		t.Fatalf("dev bypass failed, status=%d", w.Code)
	}
	if uuid != kratosUUID {
		t.Errorf("uuid = %q, want passthrough uuid", uuid)
	}
	if username != "DevGuy" {
		t.Errorf("username = %q, want DevGuy", username)
	}
}

func TestAuthenticateRequest_DevBypassRequiresValidUUID(t *testing.T) {
	gw := newTestGateway(newAuthRepo())
	gw.verifier = stubVerifier{goodToken: validToken}
	gw.devMode = true

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws?uuid=not-a-uuid", nil)
	if _, _, _, ok := authenticateRequest(gw, w, req); ok {
		t.Fatal("expected rejection for malformed dev uuid")
	}
}
