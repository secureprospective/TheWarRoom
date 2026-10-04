package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

func TestCrosswalkReportRatesAndReasons(t *testing.T) {
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
		{ID: "0816", Name: "Gosnell, Stephen", Position: "WR"},
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
	rep := crosswalkReport(rostered, lk, linked, cw, rejected, map[string]string{"0001": "Buffalo Bills"})

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
	if len(rep.Unmatched) != 3 || rep.Unmatched[0].Franchise != "Buffalo Bills" {
		t.Errorf("unmatched = %+v", rep.Unmatched)
	}
}
