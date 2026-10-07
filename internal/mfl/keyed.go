package mfl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

// WithKeySource configures credential retrieval without holding a key in caller requests.
func WithKeySource(src func(context.Context) (mflkey.Key, error)) Option {
	return func(c *Client) { c.keySource = src }
}

// WithKeySource returns a copy sharing discovery, transport and rate limiting, not credentials.
func (c *Client) WithKeySource(src func(context.Context) (mflkey.Key, error)) *Client {
	copyClient := *c
	copyClient.keySource = src
	return &copyClient
}

func (c *Client) doKeyed(ctx context.Context, req Request) (out Response, err error) {
	if c.keySource == nil {
		return out, errors.New("mfl: keyed request has no key source")
	}
	key, err := c.keySource(ctx)
	secret := key.Reveal()
	cleanURL := ""
	defer func() { err = scrubError(err, cleanURL, secret) }()
	if err != nil {
		return out, fmt.Errorf("mfl: retrieve key: %w", err)
	}
	if secret == "" || req.Params["L"] == "" || req.Type == "league" {
		return out, errors.New("mfl: keyed export requires a key and league")
	}
	c.mu.RLock()
	host, scope, at := c.host, c.hostFor, c.hostAt
	c.mu.RUnlock()
	if scope != req.Year+"/"+req.Params["L"] || host == "api" || host == "" || time.Since(at) >= hostTTL {
		return out, errors.New("mfl: keyed export requires current discovered league host")
	}
	cleanURL = c.buildURL(host, req.Year, req.Type, req.Params)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, cleanURL, nil)
	if err != nil {
		return out, fmt.Errorf("mfl: create keyed request: %w", err)
	}
	q := httpReq.URL.Query()
	q.Set("APIKEY", secret)
	httpReq.URL.RawQuery = q.Encode()
	resp, err := c.executeWithRetry(ctx, httpReq, c.keyedHTTP)
	if err != nil {
		return out, fmt.Errorf("mfl: execute keyed request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return out, fmt.Errorf("mfl: read keyed response: %w", err)
	}
	// Parsers and bindings must not echo credentials even when MFL echoes them in an error body.
	body = []byte(redact(string(body), secret))
	return Response{StatusCode: resp.StatusCode, Body: body}, nil
}

type scrubbedError struct {
	message string
	cause   error
}

func (e scrubbedError) Error() string { return e.message }
func (e scrubbedError) Unwrap() error { return e.cause }

func scrubError(err error, cleanURL, secret string) error {
	if err == nil {
		return nil
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		err = &url.Error{Op: ue.Op, URL: cleanURL, Err: scrubError(ue.Err, cleanURL, secret)}
	}
	return scrubbedError{message: redact(err.Error(), secret), cause: err}
}

// redact removes the key as written and as it travels in a query string.
func redact(text, secret string) string {
	if secret == "" {
		return text
	}
	text = strings.ReplaceAll(text, secret, "[redacted]")
	return strings.ReplaceAll(text, url.QueryEscape(secret), "[redacted]")
}
