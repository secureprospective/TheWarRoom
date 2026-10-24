package mfl

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

// A discovered base URL off MFL is refused and the cached host is kept, so the next keyed
// request cannot carry the API key to it.
func TestDiscoverHost_RefusesForeignBaseURL(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"league":{"baseURL":"https://www47.myfantasyleague.com.evil.example/2026/home/14432"}}`))
	}))
	defer srv.Close()
	c, _ := newTestClient(t, srv)
	if err := c.DiscoverHost(context.Background(), "2026", "14432"); !errors.Is(err, ErrDisallowedHost) {
		t.Fatalf("DiscoverHost: want ErrDisallowedHost, got %v", err)
	}
	if c.host != "api" {
		t.Fatalf("host changed to %q", c.host)
	}
}

// A request to a host off MFL never reaches the transport, keyed or not.
func TestDo_RefusesForeignHost(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"evil.example", "evil.example:443", "myfantasyleague.com.evil.example", "a.b.myfantasyleague.com"} {
		rt := &keyedTransport{}
		c, err := New(host, 10000, WithTransport(rt), WithKeySource(
			func(context.Context) (mflkey.Key, error) { return "stored-key", nil }))
		if err != nil {
			t.Fatal(err)
		}
		c.hostFor, c.hostAt = "2026/14432", time.Now()
		for _, keyed := range []bool{false, true} {
			req := Request{Type: "rosters", Year: "2026", Params: map[string]string{"L": "14432"}, Keyed: keyed}
			if _, err := c.Do(context.Background(), req); !errors.Is(err, ErrDisallowedHost) {
				t.Errorf("%s keyed=%v: want ErrDisallowedHost, got %v", host, keyed, err)
			}
		}
		if len(rt.seen) != 0 {
			t.Errorf("%s: %d requests reached the transport", host, len(rt.seen))
		}
	}
}

// Redirects are followed within MFL and refused off it.
func TestDo_RedirectsStayOnMFL(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("TYPE") {
		case "rosters":
			http.Redirect(w, r, "https://www12.myfantasyleague.com/2026/export?TYPE=moved", http.StatusFound)
		case "players":
			http.Redirect(w, r, "https://evil.example/2026/export?TYPE=moved", http.StatusFound)
		default:
			_, _ = w.Write([]byte(`{"ok":1}`))
		}
	}))
	defer srv.Close()
	c, rt := newTestClient(t, srv)
	resp, err := c.Do(context.Background(), Request{Type: "rosters", Year: "2026"})
	if err != nil || string(resp.Body) != `{"ok":1}` {
		t.Fatalf("in-MFL redirect: %q %v", resp.Body, err)
	}
	if hosts := rt.builtHosts(); len(hosts) != 2 || hosts[1] != "www12.myfantasyleague.com" {
		t.Fatalf("hosts = %v", hosts)
	}
	if _, err := c.Do(context.Background(), Request{Type: "players", Year: "2026"}); !errors.Is(err, ErrDisallowedHost) {
		t.Fatalf("foreign redirect: want ErrDisallowedHost, got %v", err)
	}
	if hosts := rt.builtHosts(); len(hosts) != 3 {
		t.Fatalf("foreign redirect reached the transport: %v", hosts)
	}
}

// A body over maxBodyBytes is refused; one at the limit is read whole.
func TestDo_CapsResponseBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := maxBodyBytes
		if r.URL.Query().Get("TYPE") == "players" {
			n++
		}
		_, _ = w.Write([]byte(strings.Repeat("x", n)))
	}))
	defer srv.Close()
	c, _ := newTestClient(t, srv)
	if resp, err := c.Do(context.Background(), Request{Type: "rosters", Year: "2026"}); err != nil || len(resp.Body) != maxBodyBytes {
		t.Fatalf("at the limit: %d bytes, %v", len(resp.Body), err)
	}
	if _, err := c.Do(context.Background(), Request{Type: "players", Year: "2026"}); !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("over the limit: want ErrBodyTooLarge, got %v", err)
	}
}

type countingReader struct{ n, limit int }

func (r *countingReader) Read(p []byte) (int, error) {
	if r.n >= r.limit {
		return 0, io.EOF
	}
	k := min(len(p), r.limit-r.n)
	r.n += k
	return k, nil
}

// An oversized body is not read past the cap, so a huge or endless one costs bounded memory.
func TestReadBody_StopsAtTheCap(t *testing.T) {
	t.Parallel()
	r := &countingReader{limit: 4 * maxBodyBytes}
	if _, err := readBody(r); !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("want ErrBodyTooLarge, got %v", err)
	}
	if r.n != maxBodyBytes+1 {
		t.Fatalf("read %d bytes, want %d", r.n, maxBodyBytes+1)
	}
}

func TestDiscoverHostRejectsURLTricks(t *testing.T) {
	for _, base := range []string{
		"http://www47.myfantasyleague.com", "https://www47.myfantasyleague.com:443",
		"https://sentinel-private-key@www47.myfantasyleague.com", "https://www47.myfantasyleague.com.evil.test",
		"https://www47.myfantasyleague.com./", "https://a.b.myfantasyleague.com", "https://[sentinel-private-key",
	} {
		t.Run(base, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprintf(w, `{"league":{"baseURL":%q}}`, base)
			}))
			defer srv.Close()
			c, _ := newTestClient(t, srv)
			err := c.DiscoverHost(t.Context(), "2026", "1")
			if !errors.Is(err, ErrDisallowedHost) || c.host != "api" || strings.Contains(err.Error(), "sentinel-private-key") {
				t.Fatalf("discovery changed host or leaked input: host=%s error=%v", c.host, err)
			}
		})
	}
}

func TestDoCapsDecompressedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, _ = io.Copy(gz, io.LimitReader(&countingReader{limit: maxBodyBytes + 1}, int64(maxBodyBytes+1)))
		_ = gz.Close()
	}))
	defer srv.Close()
	c, _ := newTestClient(t, srv)
	response, err := c.Do(t.Context(), Request{Type: "players", Year: "2026"})
	if !errors.Is(err, ErrBodyTooLarge) || len(response.Body) != 0 {
		t.Fatalf("decompressed oversize accepted: bytes=%d err=%v", len(response.Body), err)
	}
}

func TestKeyedRedirectRefusesDestinationWithoutLeakingSecret(t *testing.T) {
	const sentinel = "sentinel-private-key"
	for _, destination := range []string{
		"https://evil.test/" + sentinel, "http://www47.myfantasyleague.com/" + sentinel,
		"https://www47.myfantasyleague.com:444/" + sentinel, "https://" + sentinel + "@www47.myfantasyleague.com/",
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("APIKEY") != sentinel {
				t.Error("key missing on initial request")
			}
			http.Redirect(w, r, destination, http.StatusFound)
		}))
		c, rt := newTestClient(t, srv)
		c.keySource = func(context.Context) (mflkey.Key, error) { return sentinel, nil }
		c.host, c.hostFor, c.hostAt = "www47", "2026/1", time.Now()
		response, err := c.Do(t.Context(), Request{Type: "rosters", Year: "2026", Params: map[string]string{"L": "1"}, Keyed: true})
		srv.Close()
		if err != nil || response.StatusCode != http.StatusFound || strings.Contains(string(response.Body), sentinel) || len(rt.builtHosts()) != 1 {
			t.Fatalf("redirect reached destination or leaked key: hosts=%v err=%v", rt.builtHosts(), err)
		}
	}
}
