package identity

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Claims struct {
	Subject  string `json:"sub"`
	Email    string `json:"email"`
	Issuer   string `json:"iss"`
	Audience any    `json:"aud"`
	Expires  int64  `json:"exp"`
	Scope    string `json:"scope"`
}
type Verifier struct {
	Issuer, Audience, Mode string
	Client                 *http.Client
	mu                     sync.RWMutex
	refreshMu              sync.Mutex
	keys                   map[string]*rsa.PublicKey
	fetched                time.Time
	lastForcedRefresh      time.Time
	now                    func() time.Time
}

const (
	maxJWKSResponseBytes = 1 << 20
	forcedRefreshThrottle = 30 * time.Second
)

func New(issuer, audience, mode string) *Verifier {
	return &Verifier{Issuer: strings.TrimRight(issuer, "/") + "/", Audience: audience, Mode: mode, Client: &http.Client{Timeout: 5 * time.Second}, keys: map[string]*rsa.PublicKey{}, now: time.Now}
}
func (v *Verifier) Verify(ctx context.Context, token string) (Claims, error) {
	if v.Mode == "dev" && strings.HasPrefix(token, "dev:") {
		parts := strings.SplitN(token, "::", 2)
		subject := strings.TrimPrefix(parts[0], "dev:")
		if subject == "" {
			return Claims{}, errors.New("empty dev subject")
		}
		email := subject + "@local.test"
		if len(parts) == 2 && parts[1] != "" {
			email = parts[1]
		}
		return Claims{Subject: subject, Email: email, Issuer: "development", Audience: v.Audience, Expires: time.Now().Add(time.Hour).Unix()}, nil
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("malformed JWT")
	}
	decode := func(x string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(x) }
	hraw, e := decode(parts[0])
	if e != nil {
		return Claims{}, e
	}
	praw, e := decode(parts[1])
	if e != nil {
		return Claims{}, e
	}
	sig, e := decode(parts[2])
	if e != nil {
		return Claims{}, e
	}
	var h struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if json.Unmarshal(hraw, &h) != nil || h.Algorithm != "RS256" || h.KeyID == "" {
		return Claims{}, errors.New("JWT must use RS256 and a kid")
	}
	var c Claims
	if json.Unmarshal(praw, &c) != nil {
		return c, errors.New("invalid claims")
	}
	if c.Issuer != v.Issuer || !audienceContains(c.Audience, v.Audience) || c.Expires <= time.Now().Unix() || c.Subject == "" {
		return c, errors.New("JWT claims rejected")
	}
	key, e := v.key(ctx, h.KeyID)
	if e != nil {
		return c, e
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if e = rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); e != nil {
		return c, errors.New("JWT signature rejected")
	}
	return c, nil
}
func audienceContains(v any, want string) bool {
	switch x := v.(type) {
	case string:
		return x == want
	case []any:
		for _, a := range x {
			if s, ok := a.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}
func (v *Verifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	k := v.keys[kid]
	fresh := v.now().Sub(v.fetched) < time.Hour
	observedFetch := v.fetched
	v.mu.RUnlock()
	if k != nil && fresh {
		return k, nil
	}
	if err := v.refresh(ctx, observedFetch, k == nil && !observedFetch.IsZero()); err != nil {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	k = v.keys[kid]
	if k == nil {
		return nil, errors.New("unknown JWT key")
	}
	return k, nil
}
func (v *Verifier) refresh(ctx context.Context, observedFetch time.Time, force bool) error {
	v.refreshMu.Lock()
	defer v.refreshMu.Unlock()

	// Another request may have refreshed while this request waited for the
	// single refresh lock. Avoid stampeding the identity provider.
	v.mu.RLock()
	fresh := v.now().Sub(v.fetched) < time.Hour
	refreshedByAnotherRequest := !v.fetched.Equal(observedFetch)
	lastForcedRefresh := v.lastForcedRefresh
	v.mu.RUnlock()
	if refreshedByAnotherRequest || (fresh && !force) {
		return nil
	}
	if force {
		now := v.now()
		if !lastForcedRefresh.IsZero() && now.Sub(lastForcedRefresh) < forcedRefreshThrottle {
			return nil
		}
		// Throttle forced refreshes even when the provider is unavailable so
		// attacker-controlled random key IDs cannot amplify outbound traffic.
		v.mu.Lock()
		v.lastForcedRefresh = now
		v.mu.Unlock()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.Issuer+".well-known/jwks.json", nil)
	if err != nil {
		return errors.New("invalid identity provider URL")
	}
	resp, e := v.Client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("JWKS fetch failed with status %d", resp.StatusCode)
	}
	var set struct {
		Keys []struct{ Kty, Kid, N, E string }
	}
	limited := io.LimitReader(resp.Body, maxJWKSResponseBytes+1)
	body, e := io.ReadAll(limited)
	if e != nil {
		return errors.New("unable to read JWKS response")
	}
	if len(body) > maxJWKSResponseBytes {
		return errors.New("JWKS response is too large")
	}
	if e = json.Unmarshal(body, &set); e != nil {
		return errors.New("invalid JWKS response")
	}
	keys := map[string]*rsa.PublicKey{}
	for _, j := range set.Keys {
		if j.Kty != "RSA" {
			continue
		}
		if j.Kid == "" {
			continue
		}
		nb, e1 := base64.RawURLEncoding.DecodeString(j.N)
		eb, e2 := base64.RawURLEncoding.DecodeString(j.E)
		if e1 != nil || e2 != nil || len(nb) < 256 || len(eb) == 0 || len(eb) > 4 {
			continue
		}
		var padded [4]byte
		copy(padded[4-len(eb):], eb)
		exponent := int(binary.BigEndian.Uint32(padded[:]))
		if exponent < 3 || exponent%2 == 0 {
			continue
		}
		keys[j.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: exponent}
	}
	if len(keys) == 0 {
		return errors.New("JWKS response contained no usable RSA keys")
	}
	v.mu.Lock()
	v.keys = keys
	v.fetched = v.now()
	v.mu.Unlock()
	return nil
}
