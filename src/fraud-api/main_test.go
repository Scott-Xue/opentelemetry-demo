package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScreenHandlerReturnsDecision(t *testing.T) {
	body, _ := json.Marshal(baseRequest())
	req := httptest.NewRequest(http.MethodPost, "/v1/screen", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newServer(behaviour{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var resp ScreenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if resp.Decision == "" || resp.ProviderRef == "" {
		t.Fatalf("incomplete response: %+v", resp)
	}
}

func TestScreenHandlerRejectsNonPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/screen", nil)
	rec := httptest.NewRecorder()
	newServer(behaviour{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestScreenHandlerRejectsMalformedJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/screen", bytes.NewReader([]byte("{nope")))
	rec := httptest.NewRecorder()
	newServer(behaviour{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestScreenHandlerRequiresOrderID(t *testing.T) {
	r := baseRequest()
	r.OrderID = ""
	body, _ := json.Marshal(r)
	req := httptest.NewRequest(http.MethodPost, "/v1/screen", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	newServer(behaviour{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing order_id, got %d", rec.Code)
	}
}

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	newServer(behaviour{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestErrorRateOneAlwaysFails(t *testing.T) {
	body, _ := json.Marshal(baseRequest())
	req := httptest.NewRequest(http.MethodPost, "/v1/screen", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	newServer(behaviour{errorRate: 1.0}).ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 with errorRate 1.0, got %d", rec.Code)
	}
}

func TestHealthzIgnoresErrorRate(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	newServer(behaviour{errorRate: 1.0}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz must stay up so Kubernetes does not restart the pod during a fault injection; got %d", rec.Code)
	}
}
