package marketfeeds

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

func sample(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/" + name + ".json") //nolint:gosec // fixture name supplied only by table cases and canned requests
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func TestRealSamples(t *testing.T) {
	cases := []struct {
		kind, file       string
		players, samples int
	}{
		{Projections, Projections, 957, 0}, {Ranks, Ranks, 1321, 0},
		{ADP, ADP, 370, 6734}, {AAV, AAV, 358, 111},
		{ADP, "adp-2025", 362, 7185}, {AAV, "aav-2025", 702, 2103},
		{ADP, "adp-2026", 34, 5}, {AAV, "aav-2026", 358, 111},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			feed, err := Parse(tc.kind, sample(t, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			if tc.players > 0 && len(feed.Players) != tc.players {
				t.Fatalf("players: %d", len(feed.Players))
			}
			if tc.samples >= 0 && feed.Samples != tc.samples {
				t.Fatalf("samples: %d", feed.Samples)
			}
			if tc.kind == Projections && (feed.Week != 6 || feed.Players[0].Value != 32.21) {
				t.Fatal(feed)
			}
			if tc.kind == Ranks && (feed.Players[1].Rank != 2 || feed.Players[1].Last != 3 || feed.Players[1].Change != 1) {
				t.Fatal(feed.Players[1])
			}
			if tc.kind == ADP && tc.file == ADP && (feed.Players[0].Value != 1.85 || feed.Players[0].Low != 1 || feed.Players[0].High != 34) {
				t.Fatal(feed.Players[0])
			}
			if tc.kind == AAV && tc.file == AAV && (feed.Players[0].Value != 30.10 || feed.Players[0].Low != 3.44 || feed.Players[0].High != 113.64) {
				t.Fatal(feed.Players[0])
			}
		})
	}
}
func TestRejectMalformedAndAcceptEmpty(t *testing.T) {
	for _, kind := range []string{Projections, Ranks, ADP, AAV} {
		for _, body := range []string{`{}`, `{"error":"offline"}`, `null`, `broken`} {
			if _, err := Parse(kind, []byte(body)); err == nil {
				t.Fatalf("accepted %s %s", kind, body)
			}
		}
	}
	raw := RawPlayer{ID: "0816", Score: "NaN"}
	if err := raw.Validate(Projections); err == nil {
		t.Fatal("NaN accepted")
	}
	raw.Score = "-2.5"
	if err := raw.Validate(Projections); err != nil {
		t.Fatal(err)
	}
	if err := (RawPlayer{ID: "bad", Score: "1"}).Validate(Projections); err == nil {
		t.Fatal("bad id")
	}
	for _, body := range []string{
		`{"adp":{"totalDrafts":"0","totalPicks":"0","player":[]}}`,
		`{"adp":{"totalDrafts":"0","totalPicks":"0"}}`,
	} {
		feed, err := Parse(ADP, []byte(body))
		if err != nil || len(feed.Players) != 0 {
			t.Fatalf("empty ADP: %+v %v", feed, err)
		}
	}
	bad := strings.Replace(string(sample(t, ADP)), `"averagePick":"1.85"`, `"averagePick":"NaN"`, 1)
	if _, err := Parse(ADP, []byte(bad)); err == nil {
		t.Fatal("nonfinite ADP")
	}
}

type fixtureTransport struct {
	t         *testing.T
	requested map[string]int
}

func (f fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	kind := req.URL.Query().Get("TYPE")
	f.requested[kind]++
	host := "api.myfantasyleague.com"
	if kind == Projections {
		host = "www47.myfantasyleague.com"
	}
	if req.URL.Host != host {
		f.t.Errorf("%s host: %s, want %s", kind, req.URL.Host, host)
	}
	var body []byte
	if kind == "league" {
		body = []byte(`{"league":{"baseURL":"https://www47.myfantasyleague.com/2026/"}}`)
	} else {
		body = sample(f.t, kind)
	}
	if f.requested[kind] == 1 && kind != Projections && kind != "league" && req.URL.Query().Get("L") != "" {
		f.t.Error("market feed must be league-less")
	}
	if kind == ADP && (req.URL.Query().Get("PERIOD") != "ALL") {
		f.t.Error("ADP scope")
	}
	if kind == Projections && (req.URL.Query().Get("W") != "6" || req.URL.Query().Get("L") != "14432") {
		f.t.Error("projection scope")
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(body))), Header: http.Header{}}, nil
}
func TestFetchHostsAndParameters(t *testing.T) {
	requested := map[string]int{}
	client, err := mfl.New("www47", 10000, mfl.WithTransport(fixtureTransport{t, requested}))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{Projections, Ranks, ADP, AAV} {
		feed, err := Fetch(context.Background(), client, kind, "2026", "14432", 6)
		if err != nil || len(feed.Players) == 0 {
			t.Fatalf("%s: %v", kind, err)
		}
		if requested[kind] != 1 {
			t.Fatalf("%s requests: %d", kind, requested[kind])
		}
	}
	if requested["league"] != 1 {
		t.Fatal("unnecessary host discovery")
	}
	// Even a caller carrying L cannot accidentally route an MFL-wide export to the league host.
	for _, kind := range []string{Ranks, ADP, AAV} {
		if _, err := client.Do(context.Background(), mfl.Request{Type: kind, Year: "2026", Params: map[string]string{"L": "14432", "PERIOD": "ALL"}}); err != nil {
			t.Fatal(err)
		}
	}
}
