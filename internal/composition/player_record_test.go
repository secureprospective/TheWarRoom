package composition

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// playersListing serves MFL's players export and counts the fetches, so a test can tell an
// ingest from a read of the listing already held.
type playersListing struct{ fetches int }

func (p *playersListing) RoundTrip(r *http.Request) (*http.Response, error) {
	kind := r.URL.Query().Get("TYPE")
	if kind == "league" { // host discovery, before any export
		return jsonResponse(r, `{"league":{"id":"14432","baseURL":"https://api.myfantasyleague.com"}}`), nil
	}
	if kind != "players" {
		return nil, http.ErrNotSupported
	}
	p.fetches++
	body := `{"players":{"player":[
		{"id":"0042","name":"Ingested, Wide","position":"WR","team":"BUF","draft_year":"2021","draft_round":"2","draft_pick":"7"},
		{"id":"0043","name":"Ingested, Quarter","position":"QB","team":"LAR","draft_year":"2019"}]}}`
	return jsonResponse(r, body), nil
}

func jsonResponse(r *http.Request, body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)),
		Header: http.Header{"Content-Type": []string{"application/json"}}, Request: r}
}

func listingApp(t *testing.T) (*App, *playersListing) {
	t.Helper()
	a := targetTestApp(t)
	transport := &playersListing{}
	client, err := mfl.New("api", 10000, mfl.WithTransport(transport))
	if err != nil {
		t.Fatal(err)
	}
	a.core.MFLClient = client
	return a, transport
}

func recordCount(t *testing.T, a *App) int {
	t.Helper()
	records, err := a.recordStore().Records(a.core.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	return len(records)
}

// TestPlayerRecordIsWrittenWhereItsFeedsLand pins R1: the record is a fact about a feed, so it
// is written where the feed arrives. Both feeds that supply it write it; a read of a listing
// already held does not refetch, and the record it wrote stays.
func TestPlayerRecordIsWrittenWhereItsFeedsLand(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		land    func(*App) error
		records int
		fetches int
	}{
		{"MFL's players listing lands", func(a *App) error {
			_, err := a.directory(a.core.Ctx)
			return err
		}, 2, 1},
		{"a signals load lands", func(a *App) error {
			a.writeRecordAfterSignals(a.core.Ctx)
			return nil
		}, 2, 1},
		{"the listing already held is read again", func(a *App) error {
			if _, err := a.directory(a.core.Ctx); err != nil {
				return err
			}
			_, err := a.directory(a.core.Ctx)
			return err
		}, 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a, transport := listingApp(t)
			if recordCount(t, a) != 0 {
				t.Fatal("a fresh league already holds player records")
			}
			if err := tc.land(a); err != nil {
				t.Fatal(err)
			}
			if got := recordCount(t, a); got != tc.records {
				t.Fatalf("records written: %d, want %d", got, tc.records)
			}
			if transport.fetches != tc.fetches {
				t.Fatalf("listing fetches: %d, want %d", transport.fetches, tc.fetches)
			}
		})
	}
}

// TestValuationReadsTheRecordItDoesNotWriteIt pins the other half of R1: a pass values what the
// feeds wrote. With no record written it refuses in plain words rather than resolving one of
// its own, which would value the league on a record nobody can read back.
func TestValuationReadsTheRecordItDoesNotWriteIt(t *testing.T) {
	t.Parallel()
	a, _ := listingApp(t)
	if err := a.core.Rulebook.Initialize(a.core.Ctx, weekRules{end: "17"}); err != nil {
		t.Fatal(err)
	}
	a.core.Params = params.New(a.core.Pools)
	if err := a.core.Params.Initialize(a.core.Ctx); err != nil {
		t.Fatal(err)
	}
	lk, err := normalize.NewLookup([]players.RawPlayer{{ID: "0042", Name: "Held, Wide", Position: "WR", Team: "BUF"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.runModelLocked(a.core.Ctx, a.core.Params.Snapshot())
	if err == nil || !strings.Contains(err.Error(), "MFL's players listing has not been read") {
		t.Fatalf("a pass valued the league with no record written: %v", err)
	}
	if recordCount(t, a) != 0 {
		t.Fatal("the pass wrote the player record")
	}
	ingestListing(t, a, lk)
	// The fixture holds no production, so the pass still has nothing to value; what it must not
	// say again is that no player has a record.
	_, err = a.runModelLocked(a.core.Ctx, a.core.Params.Snapshot())
	if err != nil && strings.Contains(err.Error(), "MFL's players listing has not been read") {
		t.Fatal("a pass refused a record the feeds had written", err)
	}
}

// ingestListing stands in for the players listing's ingest: it publishes the Lookup and writes
// the player record from it, as a real fetch does.
func ingestListing(t *testing.T, a *App, lk normalize.Lookup) {
	t.Helper()
	a.core.LookupMu.Lock()
	a.core.Lookup, a.core.HasLookup = lk, true
	a.core.LookupMu.Unlock()
	a.writeRecordOnIngest(context.WithoutCancel(a.core.Ctx), lk, "the fixture listing")
	if recordCount(t, a) == 0 {
		t.Fatal("the fixture listing wrote no player record")
	}
}
