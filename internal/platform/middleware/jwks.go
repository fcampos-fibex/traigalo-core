package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

const (
	jwksRefreshCooldown = time.Minute
	jwksFetchTimeout    = 5 * time.Second
)

var errUnknownKey = errors.New("unknown signing key")

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type KeySet struct {
	url         string
	client      *http.Client
	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	lastFetched time.Time
}

func NewKeySet(url string) *KeySet {
	return &KeySet{
		url:    url,
		client: &http.Client{Timeout: jwksFetchTimeout},
		keys:   map[string]*rsa.PublicKey{},
	}
}

func (k *KeySet) Key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	if key := k.cached(kid); key != nil {
		return key, nil
	}
	if err := k.refresh(ctx); err != nil {
		return nil, err
	}
	if key := k.cached(kid); key != nil {
		return key, nil
	}
	return nil, errUnknownKey
}

func (k *KeySet) cached(kid string) *rsa.PublicKey {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.keys[kid]
}

func (k *KeySet) refresh(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if time.Since(k.lastFetched) < jwksRefreshCooldown && len(k.keys) > 0 {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.url, nil)
	if err != nil {
		return fmt.Errorf("build jwks request: %w", err)
	}
	res, err := k.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch jwks: status %d", res.StatusCode)
	}
	var doc jwksDocument
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, entry := range doc.Keys {
		if entry.Kty != "RSA" || (entry.Use != "" && entry.Use != "sig") {
			continue
		}
		key, err := parseRSAKey(entry)
		if err != nil {
			continue
		}
		keys[entry.Kid] = key
	}
	k.keys = keys
	k.lastFetched = time.Now()
	return nil
}

func parseRSAKey(entry jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(entry.N)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(entry.E)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}
