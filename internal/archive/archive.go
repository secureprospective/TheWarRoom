// Package archive records every HTTP response the app receives, exactly as received, before any
// parser sees it. Transport wraps an http.RoundTripper. As the caller reads a response body, its
// bytes are hashed and gzipped; once the body is read to the end, the Sink stores them. A body
// the caller abandons is logged without its bytes, and a request that fails is logged with its
// error, so the archive shows outages as well as data.
package archive

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Fetch is one request's record. SHA256, Size and Gzip are set only when the whole body was
// read.
type Fetch struct {
	URL       string // as first requested, before redirects, with secret query values removed
	Status    int    // 0 when no response arrived
	SHA256    string // of the body as received
	Size      int64
	Gzip      []byte
	Err       string
	FetchedAt time.Time
}

// Sink stores fetch records.
type Sink interface {
	Record(ctx context.Context, f Fetch) error
}

// Transport is an http.RoundTripper that records every request to Sink. A record that cannot be
// stored fails the read, so the archive never silently stops.
type Transport struct {
	Base http.RoundTripper // nil means http.DefaultTransport
	Sink Sink
}

// RoundTrip performs the request and wraps the response body in the recorder.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	f := Fetch{URL: originURL(req), FetchedAt: time.Now()}
	ctx := context.WithoutCancel(req.Context())
	resp, err := base.RoundTrip(req)
	if err != nil {
		f.Err = redactSecret(err.Error(), req.URL.Query().Get("APIKEY"))
		if rerr := t.Sink.Record(ctx, f); rerr != nil {
			return nil, errors.Join(err, rerr)
		}
		return nil, err //nolint:wrapcheck // a transport returns its base transport's error as is
	}
	f.Status = resp.StatusCode
	gz, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed) // only an invalid level errors
	rec := &recorder{body: resp.Body, sink: t.Sink, ctx: ctx, fetch: f, hash: sha256.New(), gz: gz}
	rec.secret = req.URL.Query().Get("APIKEY")
	gz.Reset(&rec.buf)
	resp.Body = rec
	return resp, nil
}

// recorder tees a response body into a hash and a gzip buffer, and records once: at the end
// of the body, or at Close if the caller stopped early.
type recorder struct {
	body   io.ReadCloser
	sink   Sink
	ctx    context.Context //nolint:containedctx // the record outlives the request's own context
	fetch  Fetch
	hash   hash.Hash
	buf    bytes.Buffer
	gz     *gzip.Writer
	size   int64
	done   bool
	secret string
	tail   []byte
	leaked bool
}

func (r *recorder) Read(p []byte) (int, error) {
	n, err := r.body.Read(p)
	if n > 0 && !r.done {
		if r.secret != "" && !r.leaked {
			r.tail = append(r.tail, p[:n]...)
			joined := r.tail
			escaped := url.QueryEscape(r.secret) // never shorter than the key itself
			r.leaked = bytes.Contains(joined, []byte(r.secret)) || bytes.Contains(joined, []byte(escaped))
			keep := len(escaped) - 1
			if len(joined) > keep {
				joined = joined[len(joined)-keep:]
			}
			r.tail = append([]byte(nil), joined...)
		}
		r.size += int64(n)
		r.hash.Write(p[:n])
		if _, gerr := r.gz.Write(p[:n]); gerr != nil {
			return n, fmt.Errorf("archive: compress %s: %w", r.fetch.URL, gerr)
		}
	}
	if errors.Is(err, io.EOF) && !r.done {
		if ferr := r.finish(true); ferr != nil {
			return n, ferr
		}
	}
	return n, err //nolint:wrapcheck // io.EOF must reach the caller unwrapped
}

func (r *recorder) Close() error {
	var ferr error
	if !r.done {
		ferr = r.finish(false)
	}
	return errors.Join(r.body.Close(), ferr)
}

// finish records the fetch, with its body when the body was read to the end.
func (r *recorder) finish(complete bool) error {
	r.done = true
	f := r.fetch
	switch {
	case r.leaked:
		f.Err = "body omitted: credential echoed in response"
	case complete:
		if err := r.gz.Close(); err != nil {
			return fmt.Errorf("archive: compress %s: %w", f.URL, err)
		}
		f.SHA256, f.Size, f.Gzip = hex.EncodeToString(r.hash.Sum(nil)), r.size, r.buf.Bytes()
	default:
		f.Err = "body not read to the end"
	}
	if err := r.sink.Record(r.ctx, f); err != nil {
		return fmt.Errorf("archive: record %s: %w", f.URL, err)
	}
	return nil
}

// originURL is the URL the caller asked for: during a redirect, req is the follow-up request and
// req.Response points back to the hop that caused it. Query values that look like credentials are
// removed, so the archive never holds a secret.
func originURL(req *http.Request) string {
	for req.Response != nil && req.Response.Request != nil {
		req = req.Response.Request
	}
	u := *req.URL
	q := u.Query()
	for k := range q {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "key") || strings.Contains(lk, "token") || strings.Contains(lk, "secret") ||
			strings.Contains(lk, "password") {
			q.Set(k, "REDACTED")
		}
	}
	u.RawQuery = q.Encode()
	u.User = nil
	return u.String()
}

// Gunzip returns an archived body as it was received.
func Gunzip(body []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("archive: open body: %w", err)
	}
	out, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("archive: read body: %w", err)
	}
	return out, nil
}

func redactSecret(message, secret string) string {
	if secret == "" {
		return message
	}
	message = strings.ReplaceAll(message, secret, "[redacted]")
	return strings.ReplaceAll(message, url.QueryEscape(secret), "[redacted]")
}
