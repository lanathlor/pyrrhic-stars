package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKratosVerifier_Whoami_Active(t *testing.T) {
	const wantToken = "valid-session-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/whoami" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("X-Session-Token"); got != wantToken {
			t.Errorf("token header = %q, want %q", got, wantToken)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"active": true,
			"identity": {
				"id": "0a1b2c3d-0000-1111-2222-333344445555",
				"traits": {"email": "a@b.com", "username": "Tester"}
			}
		}`))
	}))
	defer srv.Close()

	v := NewKratosVerifier(srv.URL)
	id, err := v.Whoami(context.Background(), wantToken)
	if err != nil {
		t.Fatalf("Whoami: %v", err)
	}
	if id.ID != "0a1b2c3d-0000-1111-2222-333344445555" {
		t.Errorf("ID = %q", id.ID)
	}
	if id.Email != "a@b.com" {
		t.Errorf("Email = %q", id.Email)
	}
	if id.Username != "Tester" {
		t.Errorf("Username = %q", id.Username)
	}
}

func TestKratosVerifier_Whoami_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	v := NewKratosVerifier(srv.URL)
	if _, err := v.Whoami(context.Background(), "bad-token"); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("err = %v, want ErrUnauthenticated", err)
	}
}

func TestKratosVerifier_Whoami_EmptyToken(t *testing.T) {
	v := NewKratosVerifier("http://unused.invalid")
	if _, err := v.Whoami(context.Background(), ""); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("err = %v, want ErrUnauthenticated", err)
	}
}

func TestKratosVerifier_Whoami_InactiveSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"active": false, "identity": {"id": "x"}}`))
	}))
	defer srv.Close()

	v := NewKratosVerifier(srv.URL)
	if _, err := v.Whoami(context.Background(), "tok"); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("err = %v, want ErrUnauthenticated", err)
	}
}
