package ingestion_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

type seasonTransport struct {
	t                       *testing.T
	export, week, key, body string
	status                  int
	calls                   int
}

func (s *seasonTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	s.calls++
	q := r.URL.Query()
	body, status := s.body, s.status
	if q.Get("TYPE") == "league" {
		if q.Get("APIKEY") != "" {
			s.t.Fatal("discovery must be unkeyed")
		}
		body, status = `{"league":{"baseURL":"https://www47.myfantasyleague.com"}}`, 200
	} else if r.URL.Host != "www47.myfantasyleague.com" || q.Get("L") != "synthetic-league" ||
		q.Get("TYPE") != s.export || q.Get("W") != s.week || q.Get("APIKEY") != s.key {
		s.t.Fatalf("wrong request for %s (key presence %t)", s.export, q.Get("APIKEY") != "")
	}
	return &http.Response{
		StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)),
	}, nil
}

func TestSeasonFetchRequests(t *testing.T) {
	cases := []struct {
		export, week, key, fixture string
		fetch                      func(context.Context, *mfl.Client) error
	}{
		{"transactions", "", "", "transactions/testdata/transactions.json",
			func(ctx context.Context, c *mfl.Client) error {
				_, err := transactions.Fetch(ctx, c, "2026", "synthetic-league")
				return err
			}},
		{"liveScoring", "", "", "livescoring/testdata/liveScoring.json",
			func(ctx context.Context, c *mfl.Client) error {
				_, err := livescoring.Fetch(ctx, c, "2026", "synthetic-league", 0)
				return err
			}},
		{"liveScoring", "5", "", "livescoring/testdata/liveScoring-w5.json",
			func(ctx context.Context, c *mfl.Client) error {
				_, err := livescoring.Fetch(ctx, c, "2026", "synthetic-league", 5)
				return err
			}},
		{"pendingTrades", "", "synthetic-key", "",
			func(ctx context.Context, c *mfl.Client) error {
				_, err := pendingtrades.Fetch(ctx, c, "2026", "synthetic-league")
				return err
			}},
	}
	for _, tc := range cases {
		t.Run(tc.export+tc.week, func(t *testing.T) {
			body := []byte(`{"pendingTrades":{}}`) // Synthetic authenticated shape.
			if tc.fixture != "" {
				var err error
				body, err = os.ReadFile(tc.fixture)
				if err != nil {
					t.Fatal(err)
				}
			}
			for _, status := range []int{200, 503} {
				tr := &seasonTransport{
					t: t, export: tc.export, week: tc.week, key: tc.key, body: string(body), status: status,
				}
				c, err := mfl.New("api", 10000, mfl.WithTransport(tr),
					mfl.WithKeySource(func(context.Context) (mflkey.Key, error) { return "synthetic-key", nil }))
				if err != nil {
					t.Fatal(err)
				}
				err = tc.fetch(context.Background(), c)
				if (err == nil) != (status == 200) || tr.calls != 2 {
					t.Fatalf("status %d calls %d error %v", status, tr.calls, err)
				}
			}
		})
	}
}

func TestNegativeLineupWeekDoesNotFetch(t *testing.T) {
	_, err := livescoring.Fetch(context.Background(), nil, "2026", "synthetic-league", -1)
	if err == nil {
		t.Fatal("negative week accepted")
	}
}
