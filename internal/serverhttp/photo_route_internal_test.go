package serverhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The web app reads player photos from /headshot/<mfl id> on the app host, so img-src stays 'self'.
func TestHostedPhotoRoute(t *testing.T) {
	stub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		if r.URL.Path != "/headshot/16162" {
			t.Errorf("cache got %q", r.URL.Path)
		}
		_, _ = w.Write([]byte("png"))
	})
	cases := []struct {
		name   string
		h      Hosted
		host   string
		status int
	}{
		{"served on app host", Hosted{Photos: stub, PublicHost: "league.example.test"}, "league.example.test", http.StatusOK},
		{"other host refused", Hosted{Photos: stub, PublicHost: "league.example.test"}, "evil.example.test", http.StatusNotFound},
		{"no cache is 404", Hosted{PublicHost: "league.example.test"}, "league.example.test", http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "https://"+c.host+"/v1/headshot/16162", nil)
			r.Host = c.host
			w := httptest.NewRecorder()
			c.h.photo(w, r)
			if w.Code != c.status {
				t.Fatalf("status %d, want %d", w.Code, c.status)
			}
			if c.status == http.StatusOK && w.Body.String() != "png" {
				t.Fatalf("body %q", w.Body.String())
			}
		})
	}
}
