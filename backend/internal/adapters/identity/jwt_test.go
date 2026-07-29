package identity

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentKeyMissesShareJWKSRefresh(t *testing.T) {
	var requests atomic.Int32
	modulus := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xff}, 256))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"keys":[{"Kty":"RSA","Kid":"current","N":%q,"E":"AQAB"}]}`, modulus)
	}))
	defer server.Close()

	verifier := New(server.URL, "coach", "auth0")
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := verifier.key(context.Background(), "current")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("JWKS requests=%d, want 1", got)
	}
}

func TestJWKSResponseSizeIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), maxJWKSResponseBytes+1))
	}))
	defer server.Close()

	verifier := New(server.URL, "coach", "auth0")
	if _, err := verifier.key(context.Background(), "current"); err == nil || err.Error() != "JWKS response is too large" {
		t.Fatalf("error=%v", err)
	}
}

func TestUnknownKeyForcesOneRefreshForRotation(t *testing.T) {
	var requests atomic.Int32
	modulus := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xfe}, 256))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestNumber := requests.Add(1)
		kid := "old"
		if requestNumber > 1 {
			kid = "rotated"
		}
		fmt.Fprintf(w, `{"keys":[{"Kty":"RSA","Kid":%q,"N":%q,"E":"AQAB"}]}`, kid, modulus)
	}))
	defer server.Close()

	verifier := New(server.URL, "coach", "auth0")
	if _, err := verifier.key(context.Background(), "old"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := verifier.key(context.Background(), "rotated")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("JWKS requests=%d, want initial fetch plus one rotation refresh", got)
	}
}

func TestSequentialUnknownKeysAreThrottled(t *testing.T) {
	var requests atomic.Int32
	modulus := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xfd}, 256))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		fmt.Fprintf(w, `{"keys":[{"Kty":"RSA","Kid":"current","N":%q,"E":"AQAB"}]}`, modulus)
	}))
	defer server.Close()

	verifier := New(server.URL, "coach", "auth0")
	if _, err := verifier.key(context.Background(), "current"); err != nil {
		t.Fatal(err)
	}
	for _, kid := range []string{"random-1", "random-2", "random-3", "random-4"} {
		if _, err := verifier.key(context.Background(), kid); err == nil {
			t.Fatalf("expected %q to remain unknown", kid)
		}
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("JWKS requests=%d, want initial fetch plus one throttled forced refresh", got)
	}
}

func TestRotationRefreshesAfterUnknownKeyThrottle(t *testing.T) {
	var requests atomic.Int32
	modulus := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xfc}, 256))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestNumber := requests.Add(1)
		kid := "old"
		if requestNumber >= 3 {
			kid = "rotated"
		}
		fmt.Fprintf(w, `{"keys":[{"Kty":"RSA","Kid":%q,"N":%q,"E":"AQAB"}]}`, kid, modulus)
	}))
	defer server.Close()

	now := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	verifier := New(server.URL, "coach", "auth0")
	verifier.now = func() time.Time { return now }
	if _, err := verifier.key(context.Background(), "old"); err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.key(context.Background(), "attacker-controlled"); err == nil {
		t.Fatal("expected attacker key to remain unknown")
	}
	if _, err := verifier.key(context.Background(), "rotated"); err == nil {
		t.Fatal("rotation should wait for the forced-refresh throttle")
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("requests before throttle expiry=%d, want 2", got)
	}

	now = now.Add(forcedRefreshThrottle + time.Second)
	if _, err := verifier.key(context.Background(), "rotated"); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 3 {
		t.Fatalf("requests after throttle expiry=%d, want 3", got)
	}
}
