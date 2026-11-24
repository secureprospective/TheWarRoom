// Package mflread reads one MFL league's public exports for the hosted mirror (internal/mflmirror).
// It only ever sends GET requests for public exports: no MFL login, no key, and never an import
// or submit type.
package mflread

import (
	"errors"
	"net/http"
	"strings"
)

// ErrNotReadOnly refuses any request that could change something on MFL.
var ErrNotReadOnly = errors.New("mflread: only public read-only GET exports are allowed")

// ReadOnly wraps base so that every request must be a GET for a public export. A request with
// any other method, an import or submit path or TYPE, or an APIKEY is refused before it is sent.
func ReadOnly(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return readOnly{base: base}
}

type readOnly struct{ base http.RoundTripper }

func (t readOnly) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := checkReadOnly(req); err != nil {
		return nil, err
	}
	return t.base.RoundTrip(req) //nolint:wrapcheck // transport errors pass through unchanged.
}

func checkReadOnly(req *http.Request) error {
	if req.Method != http.MethodGet || req.Body != nil && req.Body != http.NoBody {
		return ErrNotReadOnly
	}
	if req.URL == nil {
		return ErrNotReadOnly
	}
	path := strings.ToLower(req.URL.Path)
	if strings.Contains(path, "import") || strings.Contains(path, "submit") || strings.Contains(path, "login") {
		return ErrNotReadOnly
	}
	for key, values := range req.URL.Query() {
		lower := strings.ToLower(key)
		if lower == "apikey" || lower == "password" {
			return ErrNotReadOnly
		}
		if lower != "type" {
			continue
		}
		for _, v := range values {
			v = strings.ToLower(v)
			if strings.HasPrefix(v, "import") || strings.Contains(v, "submit") {
				return ErrNotReadOnly
			}
		}
	}
	return nil
}
