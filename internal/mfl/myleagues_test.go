package mfl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

type loginTransport struct {
	target string
	base   http.RoundTripper
}

func (r loginTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != "api.myfantasyleague.com" {
		panic("wrong login host")
	}
	clone := req.Clone(req.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = r.target
	return r.base.RoundTrip(clone)
}

func TestMyFranchise(t *testing.T) {
	const secret = "sentinel-private-key"
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{"object", `{"leagues":{"league":{"league_id":"14432","franchise_id":"0025"}}}`, "0025", 200},
		{"array", `{"leagues":{"league":[{"league_id":"1","franchise_id":"0001"},{"league_id":"14432","franchise_id":"0025"}]}}`, "0025", 200},
		{"missing", `{"leagues":{"league":{"league_id":"1","franchise_id":"0001"}}}`, "", 200},
		{"http", secret, "", 403},
		{"echo", `{"leagues":{"league":{"league_id":"14432","franchise_id":"` + secret + `"}}}`, "", 200},
		{"malformed", secret, "", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("APIKEY") != secret || r.URL.Query().Get("TYPE") != "myleagues" {
					t.Error("missing keyed login request")
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer srv.Close()
			c, err := New("www47", 10000, WithTransport(loginTransport{strings.TrimPrefix(srv.URL, "http://"), srv.Client().Transport}), WithKeySource(func(context.Context) (mflkey.Key, error) { return secret, nil }))
			if err != nil {
				t.Fatal(err)
			}
			got, err := c.MyFranchise(context.Background(), "2026", "14432")
			if got != tc.want || (err != nil) != (tc.want == "") {
				t.Fatalf("got %q, %v", got, err)
			}
			if err != nil && strings.Contains(err.Error(), secret) {
				t.Fatal("key leaked")
			}
		})
	}
}
