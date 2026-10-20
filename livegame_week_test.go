package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

func TestHeldFinalsSurviveRestartThroughHistory(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	var endpoints livegame.Endpoints
	if err := json.Unmarshal(liveGameSources, &endpoints); err != nil {
		t.Fatal(err)
	}
	finals := heldFinals{store: a.history, endpoints: endpoints}
	ctx := context.Background()
	at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	summary := liveFixture(t, "espn-summary-401872991")
	if err := finals.Keep(ctx, "summary", "401872991", summary, at); err != nil {
		t.Fatal(err)
	}
	if err := finals.Keep(ctx, "plays", "401872991", liveFixture(t, "espn-core-plays"), at); err != nil {
		t.Fatal(err)
	}
	s, err := livegame.ParseSummary(summary)
	if err != nil {
		t.Fatal(err)
	}
	seeded := livegame.Game{ID: "DET-ARI", Kickoff: s.Kickoff, Status: "post",
		Home: livegame.Team{Abbreviation: "ARI"}, Away: livegame.Team{Abbreviation: "DET"}}
	adapter := livegame.Adapter{Finals: finals, Board: livegame.Held{Games: []livegame.Game{seeded}},
		Fetch: func(_ context.Context, kind, id string) ([]byte, error) {
			t.Fatalf("restart fetched %s %s", kind, id)
			return nil, nil
		}}
	adapter.Final(ctx, at.Add(time.Hour), "DET-ARI")
	d := adapter.Details["DET-ARI"]
	if !d.Final || len(d.Game.Plays) == 0 || !d.At.Equal(at) {
		t.Fatalf("kept final not read: final %v plays %d at %v", d.Final, len(d.Game.Plays), d.At)
	}
	view := liveGameReading(adapter, "DET-ARI", false)
	if view.DetailAsOf == "" || len(view.Scoring) == 0 || view.Followed.Last == nil || view.Box.Status != "ok" {
		t.Fatalf("final shape: asOf %q scoring %d last %v box %s", view.DetailAsOf, len(view.Scoring), view.Followed.Last, view.Box.Status)
	}
}
