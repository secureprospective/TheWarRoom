package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

// A final kept through the real history archive is read back after a restart with no fetch.
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

func TestWeekSpineNewestFirstMarkedAndCounted(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	play := func(id string, minutes int, who string) livegame.Play {
		return livegame.Play{ID: id, Sequence: minutes, At: at.Add(time.Duration(minutes) * time.Minute), Participants: []string{who}}
	}
	early := livegame.Game{ID: "1", Away: livegame.Team{Abbreviation: "DET"}, Home: livegame.Team{Abbreviation: "ARI"},
		Plays: []livegame.Play{play("a", 1, "m"), play("c", 30, "o")}}
	late := livegame.Game{ID: "2", Away: livegame.Team{Abbreviation: "BUF"}, Home: livegame.Team{Abbreviation: "LAR"},
		Plays: []livegame.Play{play("b", 10, "o"), play("d", 40, "m")}}
	adapter := livegame.Adapter{Details: map[string]livegame.Detail{
		"DET-ARI": {Game: early, At: at, Final: true}, "1": {Game: early, At: at, Final: true},
		"2": {Game: late, At: at}, "3": {At: at}}}
	held := heldWeekGames(adapter)
	if len(held) != 2 {
		t.Fatal("held games counted twice or empty games counted", len(held))
	}
	players := map[string]SpinePlayer{"m": {ID: "1", Mine: true}, "o": {ID: "2"}}
	entries, _ := matchSpine(held, map[string]bool{"m": true, "o": true}, players)
	order := ""
	for _, e := range entries {
		order += e.Play.ID
	}
	if order != "dcba" || entries[0].Game != "BUF at LAR" || entries[1].Game != "DET at ARI" ||
		!entries[0].Player.Mine || entries[1].Player.Mine {
		t.Fatalf("order %s entries %+v", order, entries)
	}
}
