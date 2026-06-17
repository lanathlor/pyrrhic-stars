package settings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"codex-online/server/internal/auth"
	"codex-online/server/internal/persistence"
)

// fakeRepo implements just the settings methods of persistence.Repository.
type fakeRepo struct {
	persistence.Repository
	data map[string]string
}

func newFakeRepo() *fakeRepo { return &fakeRepo{data: map[string]string{}} }

func (r *fakeRepo) GetUserSettings(userID string) (*persistence.UserSettings, error) {
	d, ok := r.data[userID]
	if !ok {
		return nil, nil
	}
	return &persistence.UserSettings{UserID: userID, Data: d}, nil
}

func (r *fakeRepo) UpsertUserSettings(userID, data string) error {
	r.data[userID] = data
	return nil
}

// fakeVerifier resolves a single known-good token.
type fakeVerifier struct {
	token string
	id    string
}

func (v fakeVerifier) Whoami(_ context.Context, token string) (*auth.Identity, error) {
	if token == v.token {
		return &auth.Identity{ID: v.id}, nil
	}
	return nil, auth.ErrUnauthenticated
}

const testUUID = "11111111-2222-4333-8444-555555555555"

func newTestHandler(devMode bool) (*Handler, *fakeRepo) {
	repo := newFakeRepo()
	v := fakeVerifier{token: "good", id: testUUID}
	return NewHandler(v, NewService(repo), devMode), repo
}

func serve(h *Handler, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	h.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func authedReq(method, body string) *http.Request {
	req := httptest.NewRequest(method, "/api/v1/settings", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	return req
}

func TestGetBeforePutReturnsEmptyObject(t *testing.T) {
	h, _ := newTestHandler(false)
	w := serve(h, authedReq("GET", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := strings.TrimSpace(w.Body.String()); got != "{}" {
		t.Errorf("body = %q, want {}", got)
	}
}

func TestPutThenGetRoundTrips(t *testing.T) {
	h, _ := newTestHandler(false)
	doc := `{"graphics":{"vsync":true},"keybinds":{"jump":32}}`

	w := serve(h, authedReq("PUT", doc))
	if w.Code != http.StatusNoContent {
		t.Fatalf("PUT status = %d, want 204", w.Code)
	}

	w = serve(h, authedReq("GET", ""))
	if got := strings.TrimSpace(w.Body.String()); got != doc {
		t.Errorf("GET body = %q, want %q", got, doc)
	}
}

func TestPutInvalidJSONRejected(t *testing.T) {
	h, _ := newTestHandler(false)
	w := serve(h, authedReq("PUT", "not json"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestMissingTokenRejected(t *testing.T) {
	h, _ := newTestHandler(false)
	req := httptest.NewRequest("GET", "/api/v1/settings", nil)
	w := serve(h, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestBadTokenRejected(t *testing.T) {
	h, _ := newTestHandler(false)
	req := httptest.NewRequest("GET", "/api/v1/settings", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w := serve(h, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestDevUUIDBypass(t *testing.T) {
	h, repo := newTestHandler(true)
	req := httptest.NewRequest("PUT", "/api/v1/settings?uuid="+testUUID, strings.NewReader(`{"a":1}`))
	w := serve(h, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("dev PUT status = %d, want 204", w.Code)
	}
	if repo.data[testUUID] != `{"a":1}` {
		t.Errorf("stored = %q, want dev doc", repo.data[testUUID])
	}
}

func TestDevUUIDRejectedWhenNotDev(t *testing.T) {
	h, _ := newTestHandler(false)
	req := httptest.NewRequest("GET", "/api/v1/settings?uuid="+testUUID, nil)
	w := serve(h, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (uuid bypass disabled outside dev)", w.Code)
	}
}
