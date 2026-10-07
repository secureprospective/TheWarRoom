package pendingtrades

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

type fixtureTransport struct {
	body   string
	status int
}

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	body, status := f.body, f.status
	if r.URL.Query().Get("TYPE") == "league" {
		body, status = `{"league":{"baseURL":"https://www47.myfantasyleague.com"}}`, 200
	}
	return &http.Response{
		StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{},
	}, nil
}

func TestVerifySyntheticShapes(t *testing.T) {
	// Synthetic documented XML-to-JSON shapes, not a capture of a real authenticated response.
	cases := []struct {
		name, body      string
		status          int
		valid, rejected bool
	}{
		{"empty", `{"pendingTrades":{}}`, 200, true, false},
		{"trade", `{"pendingTrades":{"trade":{"franchise":"0025"}}}`, 200, true, false},
		{"error", `{"error":{"$t":"not authorized"}}`, 200, false, true},
		{"empty error", `{"error":{"$t":""},"pendingTrades":{}}`, 200, false, true},
		{"absent", `{}`, 200, false, true},
		{"null", `{"pendingTrades":null}`, 200, false, true},
		{"array", `{"pendingTrades":[]}`, 200, false, true},
		{"malformed", `{`, 200, false, false},
		{"redirect", ``, 302, false, true},
		{"non200 error", `{"error":{"$t":"not authorized"}}`, 403, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := mfl.New("api", 10000, mfl.WithTransport(fixtureTransport{tc.body, tc.status}),
				mfl.WithKeySource(func(context.Context) (mflkey.Key, error) { return "synthetic-key", nil }))
			if err != nil {
				t.Fatal(err)
			}
			got, err := Verify(context.Background(), c, "2026", "league")
			if (err == nil) != tc.valid {
				t.Fatalf("verify: %v", err)
			}
			if errors.Is(err, ErrRejected) != tc.rejected {
				t.Fatalf("rejected: %v", err)
			}
			if got.FranchiseProven || got.Franchise != "" {
				t.Fatal("guessed franchise")
			}
			if tc.valid && got.Detail == "" {
				t.Fatal("missing explanation")
			}
		})
	}
}
