package archive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ErrNotHeld means no held copy answers a replayed read. Replay never falls back to the network.
var ErrNotHeld = errors.New("archive: no held copy of this read")

// Replay is an http.RoundTripper that answers GET reads from held copies only. It opens no
// connection, so a read through it can never reach, submit to or spend a quota at the site.
type Replay struct {
	// Lookup returns the newest held body whose HeldKey equals key, or found=false.
	Lookup func(ctx context.Context, key string) (body []byte, found bool, err error)
}

// RoundTrip answers req from the archive, or fails with ErrNotHeld.
func (r Replay) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		_ = req.Body.Close()
	}
	if req.Method != http.MethodGet || r.Lookup == nil {
		return nil, fmt.Errorf("archive: replay answers reads only")
	}
	key, err := HeldKey(originURL(req))
	if err != nil {
		return nil, err
	}
	body, found, err := r.Lookup(req.Context(), key)
	if err != nil {
		return nil, fmt.Errorf("archive: replay %s: %w", key, err)
	}
	if !found {
		return nil, fmt.Errorf("%w: %s", ErrNotHeld, key)
	}
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1,
		Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)), Request: req}, nil
}

// HeldKey is a read's identity regardless of which site host answered it or whether a credential
// was sent: the path plus the sorted query with credential-like values removed.
func HeldKey(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("archive: held key: %w", err)
	}
	q := u.Query()
	for k := range q {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "key") || strings.Contains(lk, "token") || strings.Contains(lk, "secret") ||
			strings.Contains(lk, "password") {
			q.Del(k)
		}
	}
	return u.Path + "?" + q.Encode(), nil
}
