package llm

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/personalized-ai-coach/backend/internal/ports"
)

func TestGatewayRejectsMalformedAndEmptySuccess(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"malformed", `{`, "model returned invalid JSON"},
		{"no choices", `{"choices":[]}`, "model returned no choices"},
		{"empty content", `{"choices":[{"message":{"content":"  "}}]}`, "model returned empty content"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			_, err := New(server.URL, "", nil).Complete(context.Background(), ports.LLMRequest{})
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
		})
	}
}

func TestGatewayBoundsResponseAndRequestTime(t *testing.T) {
	t.Run("response size", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(bytes.Repeat([]byte("x"), maxModelResponseBytes+1))
		}))
		defer server.Close()
		_, err := New(server.URL, "", nil).Complete(context.Background(), ports.LLMRequest{})
		if err == nil || !strings.Contains(err.Error(), "too large") {
			t.Fatalf("error=%v", err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(100 * time.Millisecond)
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"late"}}]}`))
		}))
		defer server.Close()
		gateway := New(server.URL, "", nil)
		gateway.Client.Timeout = 10 * time.Millisecond
		_, err := gateway.Complete(context.Background(), ports.LLMRequest{})
		if err == nil {
			t.Fatal("expected timeout")
		}
	})
}
