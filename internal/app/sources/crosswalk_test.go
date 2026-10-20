package sources

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/store/history"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

func TestCrosswalkReportRatesAndReasons(t *testing.T) {
	t.Parallel()
	const dp = "mfl_id,gsis_id,espn_id,name\n" +
		"13294,00-0011000,1,Pat Veteran\n" +
		"816,,2,Dre' Bly\n" +
		"17471,,3,Diego Pavia\n" +
		"20000,00-0020000,,Free Agent\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(dp)) }))
	t.Cleanup(srv.Close)
	cw, err := crosswalk.Fetch(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	lk, err := normalize.NewLookup([]players.RawPlayer{
		{ID: "13294", Name: "Veteran, Pat", Position: "QB"},
		{ID: "0816", Name: "Gosnell, Stephen", Position: "WR", Team: "MIN"},
		{ID: "17471", Name: "Pavia, Diego", Position: "QB"},
		{ID: "0844", Name: "Wood, Julius", Position: "S"},
		{ID: "20000", Name: "Agent, Free", Position: "WR"},
		{ID: "20001", Name: "Unknown, Free", Position: "WR"},
		{ID: "0500", Name: "Chiefs D", Position: "Def"},
	})
	if err != nil {
		t.Fatal(err)
	}
	links, rejected := cw.Links(lk.Name)
	linked := map[string]bool{}
	for _, l := range links {
		if l.IDType == matchIDType {
			linked[l.MFLID] = true
		}
	}
	rostered := map[string]string{"13294": "0001", "0816": "0001", "17471": "0002", "0844": "0002"}
	rep := BuildCrosswalk(rostered, lk, linked, cw, rejected, map[string]string{"0001": "Buffalo Bills"})

	want := map[string]CrosswalkRate{
		"QB": {Position: "QB", Rostered: 2, RosteredMatched: 1},
		"WR": {Position: "WR", Rostered: 1, FreeAgents: 2, FreeAgentsMatched: 1},
		"S":  {Position: "S", Rostered: 1},
	}
	if len(rep.Rates) != len(want) {
		t.Fatalf("rates = %+v", rep.Rates)
	}
	for _, r := range rep.Rates {
		if r != want[r.Position] {
			t.Errorf("rate %s = %+v, want %+v", r.Position, r, want[r.Position])
		}
	}
	if rep.Rates[0].Position != "QB" || rep.Total.Rostered != 4 || rep.Total.RosteredMatched != 1 || rep.Total.FreeAgents != 2 {
		t.Errorf("order or total wrong: %+v / %+v", rep.Rates, rep.Total)
	}
	reasons := map[string]string{}
	for _, m := range rep.Unmatched {
		reasons[m.MFLID] = m.Reason
	}
	for id, why := range map[string]string{
		"0816": crosswalk.ReasonOtherPlayer, "17471": reasonNoGSIS, "0844": reasonNotInDP,
	} {
		if reasons[id] != why {
			t.Errorf("reason for %s = %q, want %q", id, reasons[id], why)
		}
	}
	if len(rep.Unmatched) != 3 || rep.Unmatched[0].Franchise != "Buffalo Bills" || rep.Unmatched[0].Team != "MIN" {
		t.Errorf("unmatched = %+v", rep.Unmatched)
	}
}

type crosswalkHistoryStub struct {
	calls   *[]string
	linkErr error
}

func (h crosswalkHistoryStub) LinkPlayerIDs(context.Context, string, []history.PlayerIDLink) (int, error) {
	*h.calls = append(*h.calls, "link")
	return 0, h.linkErr
}
func (h crosswalkHistoryStub) LinkedPlayers(context.Context, string) (map[string]bool, error) {
	*h.calls = append(*h.calls, "read")
	return map[string]bool{}, nil
}
func (h crosswalkHistoryStub) LoadFailed(context.Context, string, error) error { return nil }

func TestCrosswalkReadsFollowSuccessfulLink(t *testing.T) {
	t.Parallel()
	for _, fail := range []bool{false, true} {
		var calls []string
		stub := crosswalkHistoryStub{calls: &calls}
		if fail {
			stub.linkErr = errors.New("fixture link failed")
		}
		snapshot := func(name string) func() map[string]string {
			return func() map[string]string { calls = append(calls, name); return map[string]string{} }
		}
		rep, err := (Service{Crosswalk: stub}).LinkCrosswalk(context.Background(), crosswalk.Map{}, normalize.Lookup{}, snapshot("rosters"), snapshot("names"))
		want := "link,read,rosters,names"
		if fail {
			want = "link"
			if !errors.Is(err, stub.linkErr) || err.Error() != "app: link crosswalk: fixture link failed" {
				t.Fatalf("link failure = %v", err)
			}
		} else if err != nil || !rep.OK {
			t.Fatalf("link = %+v, %v", rep, err)
		}
		if got := strings.Join(calls, ","); got != want {
			t.Errorf("read order = %s; want %s", got, want)
		}
	}
}
