package serverhttp_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp/landing"
)

func (f *hostedFixture) do(t *testing.T, method, path, origin string) (int, http.Header, []byte) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), method, f.server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	response, err := f.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, response.Header.Clone(), body
}

// The landing page and its assets are a fixed allowlist with their own CSP and caching; nothing
// else in the embedded site is reachable, and every other route keeps refusing a query.
func TestLandingPageServesOnlyItsAllowlist(t *testing.T) {
	f := newHostedFixture(t, true)
	const week = "public, max-age=604800"
	for _, tc := range []struct {
		method, path, origin string
		status               int
		contentType, cache   string
	}{
		{"GET", "/", "", http.StatusOK, "text/html; charset=utf-8", "no-store"},
		{"GET", "/?fbclid=x", "", http.StatusOK, "text/html; charset=utf-8", "no-store"},
		{"GET", "/", f.server.URL, http.StatusOK, "text/html; charset=utf-8", "no-store"},
		{"HEAD", "/", "", http.StatusOK, "text/html; charset=utf-8", "no-store"},
		{"GET", "/site.css", "", http.StatusOK, "text/css; charset=utf-8", "public, max-age=300"},
		{"GET", "/site.js", "", http.StatusOK, "text/javascript; charset=utf-8", "public, max-age=300"},
		{"GET", "/steel.js", "", http.StatusOK, "text/javascript; charset=utf-8", "public, max-age=300"},
		{"GET", "/og.png", "", http.StatusOK, "image/png", week},
		{"GET", "/fonts/inter-latin-400-normal.woff2", "", http.StatusOK, "font/woff2", week},
		{"POST", "/", "", http.StatusMethodNotAllowed, "", ""},
		{"GET", "/", "https://evil.test", http.StatusBadRequest, "", ""},
		{"GET", "/index.html", "", http.StatusNotFound, "", ""},
		{"GET", "/fonts/OFL.txt", "", http.StatusNotFound, "", ""},
		{"GET", "/fonts/../landing.go", "", http.StatusNotFound, "", ""},
		{"GET", "/nope.css", "", http.StatusNotFound, "", ""},
		{"GET", serverhttp.LoginPath + "?x=1", "", http.StatusBadRequest, "", ""},
	} {
		t.Run(tc.method+" "+tc.path+" "+tc.origin, func(t *testing.T) {
			status, header, body := f.do(t, tc.method, tc.path, tc.origin)
			if status != tc.status {
				t.Fatalf("status=%d want=%d body=%s", status, tc.status, body)
			}
			if tc.status == http.StatusMethodNotAllowed && header.Get("Allow") != "GET, HEAD" {
				t.Fatalf("allow=%q", header.Get("Allow"))
			}
			if tc.status != http.StatusOK {
				return
			}
			if header.Get("Content-Type") != tc.contentType || header.Get("Cache-Control") != tc.cache ||
				header.Get("Content-Security-Policy") != landing.CSP || header.Get("X-Content-Type-Options") != "nosniff" ||
				header.Get("Referrer-Policy") != "no-referrer" || header.Get("Content-Length") == "" {
				t.Fatalf("headers: %v", header)
			}
			switch {
			case tc.method == "HEAD" && len(body) != 0:
				t.Fatalf("HEAD sent %d body bytes", len(body))
			case tc.method == "GET" && tc.path == "/" && !strings.Contains(string(body), "<title>The War Room"):
				t.Fatal("index has no title")
			}
		})
	}
	status, header, _ := f.get(t, serverhttp.JoinPagePath)
	if csp := header.Get("Content-Security-Policy"); status != http.StatusOK || csp == landing.CSP || !strings.Contains(csp, "'nonce-") {
		t.Fatalf("join page status=%d csp=%q", status, csp)
	}
}
