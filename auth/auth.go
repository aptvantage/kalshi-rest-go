// Package auth provides Kalshi RSA-PSS request signing for HTTP clients.
//
// Kalshi authentication requires signing each request with an RSA private key.
// The signature covers: {timestampMs}{HTTP_METHOD}{path} (no query string).
// Headers added: KALSHI-ACCESS-KEY, KALSHI-ACCESS-TIMESTAMP, KALSHI-ACCESS-SIGNATURE.
package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Signer holds the Kalshi API key ID and RSA private key.
type Signer struct {
	keyID      string
	privateKey *rsa.PrivateKey
}

// NewSignerFromFile loads a PEM-encoded RSA private key from a file.
func NewSignerFromFile(keyID, keyPath string) (*Signer, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	return NewSignerFromPEM(keyID, data)
}

// NewSignerFromPEM parses a PEM-encoded RSA private key.
func NewSignerFromPEM(keyID string, pemData []byte) (*Signer, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in key data")
	}

	var privateKey *rsa.PrivateKey
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS1 private key: %w", err)
		}
		privateKey = key
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS8 private key: %w", err)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA (got %T)", key)
		}
		privateKey = rsaKey
	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}

	return &Signer{keyID: keyID, privateKey: privateKey}, nil
}

// Sign computes the Kalshi RSA-PSS signature for the given method and path.
// Returns the millisecond timestamp string and base64-encoded signature.
func (s *Signer) Sign(method, path string) (timestampMs string, signature string, err error) {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	// Path only — strip query string as Kalshi spec requires
	if idx := strings.IndexByte(path, '?'); idx >= 0 {
		path = path[:idx]
	}
	msg := ts + strings.ToUpper(method) + path
	hash := sha256.Sum256([]byte(msg))

	opts := &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	}
	sig, err := rsa.SignPSS(rand.Reader, s.privateKey, crypto.SHA256, hash[:], opts)
	if err != nil {
		return "", "", fmt.Errorf("sign: %w", err)
	}
	return ts, base64.StdEncoding.EncodeToString(sig), nil
}

// Transport is an http.RoundTripper that injects Kalshi auth headers on every request.
type Transport struct {
	signer *Signer
	base   http.RoundTripper
}

// NewTransport wraps base (or http.DefaultTransport if nil) with Kalshi auth signing.
func NewTransport(signer *Signer, base http.RoundTripper) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &Transport{signer: signer, base: base}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ts, sig, err := t.signer.Sign(req.Method, req.URL.RequestURI())
	if err != nil {
		return nil, fmt.Errorf("kalshi auth: %w", err)
	}
	// Clone request to avoid mutating the original
	r := req.Clone(req.Context())
	r.Header.Set("KALSHI-ACCESS-KEY", t.signer.keyID)
	r.Header.Set("KALSHI-ACCESS-TIMESTAMP", ts)
	r.Header.Set("KALSHI-ACCESS-SIGNATURE", sig)
	return t.base.RoundTrip(r)
}

// NewClient returns an *http.Client that automatically signs every request.
func NewClient(signer *Signer) *http.Client {
	return &http.Client{Transport: NewTransport(signer, nil)}
}
