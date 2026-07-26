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
	keys                   map[string]*rsa.PublicKey
	fetched                time.Time
}

func New(issuer, audience, mode string) *Verifier {
	return &Verifier{Issuer: strings.TrimRight(issuer, "/") + "/", Audience: audience, Mode: mode, Client: &http.Client{Timeout: 5 * time.Second}, keys: map[string]*rsa.PublicKey{}}
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
	fresh := time.Since(v.fetched) < time.Hour
	v.mu.RUnlock()
	if k != nil && fresh {
		return k, nil
	}
	if err := v.refresh(ctx); err != nil {
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
func (v *Verifier) refresh(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, v.Issuer+".well-known/jwks.json", nil)
	resp, e := v.Client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("JWKS fetch: %s", resp.Status)
	}
	var set struct {
		Keys []struct{ Kty, Kid, N, E string }
	}
	if e = json.NewDecoder(resp.Body).Decode(&set); e != nil {
		return e
	}
	keys := map[string]*rsa.PublicKey{}
	for _, j := range set.Keys {
		if j.Kty != "RSA" {
			continue
		}
		nb, e1 := base64.RawURLEncoding.DecodeString(j.N)
		eb, e2 := base64.RawURLEncoding.DecodeString(j.E)
		if e1 != nil || e2 != nil || len(eb) > 4 {
			continue
		}
		var padded [4]byte
		copy(padded[4-len(eb):], eb)
		keys[j.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: int(binary.BigEndian.Uint32(padded[:]))}
	}
	v.mu.Lock()
	v.keys = keys
	v.fetched = time.Now()
	v.mu.Unlock()
	return nil
}
