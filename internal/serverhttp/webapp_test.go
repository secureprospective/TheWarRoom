package serverhttp

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestPublicHostIsABareExactDNSName(t *testing.T) {
	for _, host := range []string{"league.example.org", "twr-257.secureprospective.com"} {
		if err := ValidatePublicHost(host); err != nil {
			t.Errorf("valid host %q: %v", host, err)
		}
	}
	for _, host := range []string{"", "*.example.org", "https://league.example.org", "league.example.org:443", "192.0.2.1", "league.example.org.", "LEAGUE.example.org", "one"} {
		if err := ValidatePublicHost(host); err == nil {
			t.Errorf("invalid public host accepted: %q", host)
		}
	}
}

func TestBuiltBrowserBundleIsEmbeddedWhenStaged(t *testing.T) {
	index, err := webassets.ReadFile("webassets/index.html")
	if errors.Is(err, fs.ErrNotExist) {
		t.Skip("web bundle is staged by make verify and the server release build")
	}
	if err != nil {
		t.Fatalf("read embedded browser index: %v", err)
	}
	if strings.TrimSpace(string(index)) == "" {
		t.Fatal("embedded browser index is empty")
	}
	response := httptest.NewRecorder()
	webAppHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/app/", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "<") {
		t.Fatalf("embedded browser app: status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestBrowserAppRoutesUseSafeHeadersAndAssetTypes(t *testing.T) {
	assets := fstest.MapFS{
		"index.html":           {Data: []byte(`<script src="/assets/main.js"></script><link href='/manifest.webmanifest'><script>register("/sw.js")</script>`)},
		"assets/main.js":       {Data: []byte("main")},
		"manifest.webmanifest": {Data: []byte(`{"name":"TWR"}`)},
		"sw.js":                {Data: []byte("self.addEventListener('fetch', () => {})")},
	}
	handler := webAssetsHandler(assets)
	for _, route := range []string{"/app/", "/app/league/franchise"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, route, nil))
		if response.Code != http.StatusOK || response.Header().Get("Content-Security-Policy") != browserAppCSP ||
			response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Referrer-Policy") != "same-origin" {
			t.Fatalf("%s: status=%d headers=%v", route, response.Code, response.Header())
		}
		if !strings.Contains(response.Body.String(), `/app/assets/main.js`) || !strings.Contains(response.Body.String(), `/app/manifest.webmanifest`) ||
			!strings.Contains(response.Body.String(), `/app/sw.js`) {
			t.Fatalf("%s: app base paths were not scoped: %s", route, response.Body.String())
		}
	}
	manifest := httptest.NewRecorder()
	handler.ServeHTTP(manifest, httptest.NewRequest(http.MethodGet, "/app/manifest.webmanifest", nil))
	if manifest.Code != http.StatusOK || manifest.Header().Get("Content-Type") != "application/manifest+json" {
		t.Fatalf("manifest: status=%d type=%q", manifest.Code, manifest.Header().Get("Content-Type"))
	}
	worker := httptest.NewRecorder()
	handler.ServeHTTP(worker, httptest.NewRequest(http.MethodGet, "/app/sw.js", nil))
	if worker.Code != http.StatusOK || worker.Header().Get("Content-Type") != "text/javascript; charset=utf-8" || worker.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("service worker: status=%d headers=%v", worker.Code, worker.Header())
	}
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/app/assets/missing.js", nil))
	if missing.Code != http.StatusNotFound || missing.Header().Get("Clear-Site-Data") != `"cache", "storage"` {
		t.Fatalf("missing build file: status=%d clear=%q (an old page must drop its app copy)",
			missing.Code, missing.Header().Get("Clear-Site-Data"))
	}
	method := httptest.NewRecorder()
	handler.ServeHTTP(method, httptest.NewRequest(http.MethodPost, "/app/", nil))
	if method.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST app asset status=%d", method.Code)
	}
}

func TestBrowserAppCSPMatchesTheAppBuild(t *testing.T) {
	for _, want := range []string{"script-src 'self';", "img-src 'self' data:;", "style-src 'self' 'unsafe-inline';",
		"object-src 'none';", "frame-ancestors 'none'"} {
		if !strings.Contains(browserAppCSP, want) {
			t.Fatalf("browser app policy lacks %q: %s", want, browserAppCSP)
		}
	}
}
