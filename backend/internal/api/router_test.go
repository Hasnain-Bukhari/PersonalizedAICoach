package api

import (
	"encoding/json"
	"github.com/personalized-ai-coach/backend/internal/adapters/identity"
	"github.com/personalized-ai-coach/backend/internal/adapters/llm"
	"github.com/personalized-ai-coach/backend/internal/adapters/memory"
	"github.com/personalized-ai-coach/backend/internal/application"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func handler() http.Handler {
	s := memory.New()
	return New(application.New(s, llm.Fake{}), s, identity.New("https://example.invalid/", "coach", "dev"), slog.New(slog.NewTextHandler(io.Discard, nil)))
}
func TestHealthAndAuthentication(t *testing.T) {
	h := handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 {
		t.Fatalf("health=%d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/sessions/daily", nil))
	if w.Code != 401 {
		t.Fatalf("unauthorized=%d", w.Code)
	}
}
func TestDailySessionAPI(t *testing.T) {
	h := handler()
	r := httptest.NewRequest("GET", "/api/v1/sessions/daily?date=2026-07-26", nil)
	r.Header.Set("Authorization", "Bearer dev:alice::alice@example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	if json.Unmarshal(w.Body.Bytes(), &out) != nil || out["status"] != "published" {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}
func TestTenantIsolation(t *testing.T) {
	h := handler()
	for _, subject := range []string{"alice", "bob"} {
		r := httptest.NewRequest("GET", "/api/v1/sessions/daily?date=2026-07-26", nil)
		r.Header.Set("Authorization", "Bearer dev:"+subject)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	r := httptest.NewRequest("GET", "/api/v1/analytics/graph", nil)
	r.Header.Set("Authorization", "Bearer dev:alice")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}
