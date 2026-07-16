package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const stampedVersion = "v1.2.3"

func TestVersionEndpoint_ReturnsStampedVersion(t *testing.T) {
	gw := newTestGateway(newAuthRepo())
	gw.verifier = stubVerifier{}
	gw.version = stampedVersion

	mux := setupHTTPServer(gw, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/version", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body %q: %v", w.Body.String(), err)
	}
	if body.Version != stampedVersion {
		t.Errorf("version = %q, want v1.2.3", body.Version)
	}
}

func TestCheckClientVersion_MismatchRejected(t *testing.T) {
	gw := newTestGateway(newAuthRepo())
	gw.version = stampedVersion

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws?token=good&v=v0.9.0", nil)
	if checkClientVersion(gw, w, req) {
		t.Fatal("expected mismatched client version to be rejected")
	}
	if w.Code != http.StatusUpgradeRequired {
		t.Errorf("status = %d, want 426", w.Code)
	}
}

func TestCheckClientVersion_MissingParamRejected(t *testing.T) {
	gw := newTestGateway(newAuthRepo())
	gw.version = stampedVersion

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws?token=good", nil)
	if checkClientVersion(gw, w, req) {
		t.Fatal("expected missing client version to be rejected")
	}
	if w.Code != http.StatusUpgradeRequired {
		t.Errorf("status = %d, want 426", w.Code)
	}
}

func TestCheckClientVersion_MatchAllowed(t *testing.T) {
	gw := newTestGateway(newAuthRepo())
	gw.version = stampedVersion

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws?token=good&v=v1.2.3", nil)
	if !checkClientVersion(gw, w, req) {
		t.Fatalf("expected matching client version to pass, status=%d", w.Code)
	}
}

func TestCheckClientVersion_DevModeSkipsEnforcement(t *testing.T) {
	gw := newTestGateway(newAuthRepo())
	gw.version = stampedVersion
	gw.devMode = true

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws?token=good&v=v0.9.0", nil)
	if !checkClientVersion(gw, w, req) {
		t.Fatalf("expected dev mode to skip version enforcement, status=%d", w.Code)
	}
}

func TestCheckClientVersion_UnstampedGatewaySkipsEnforcement(t *testing.T) {
	gw := newTestGateway(newAuthRepo())
	gw.version = ""

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws?token=good", nil)
	if !checkClientVersion(gw, w, req) {
		t.Fatalf("expected unstamped gateway to skip version enforcement, status=%d", w.Code)
	}
}
