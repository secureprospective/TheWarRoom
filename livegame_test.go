package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

func liveFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("internal/ingestion/livegame/testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func liveWire(t *testing.T) LiveGameView {
	t.Helper()
	g, e := livegame.ParseSummary(liveFixture(t, "espn-summary-401872991"))
	if e != nil {
		t.Fatal(e)
	}
	plays, e := livegame.ParseCore(liveFixture(t, "espn-core-plays"))
	if e != nil {
		t.Fatal(e)
	}
	g.Plays = plays
	at := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	a := livegame.Adapter{Board: livegame.Held{Games: []livegame.Game{g}, Source: "ESPN", At: at}, Details: map[string]livegame.Detail{g.ID: {Game: g, At: at}}}
	view := liveGameReading(a, g.ID, false)
	for _, p := range plays {
		if strings.HasPrefix(p.Text, "J.Brissett pass short right to T.Quitoriano for 2 yards, TOUCHDOWN") {
			view.Spine = []SpineEntry{
				{Player: SpinePlayer{ID: "13130", Name: "Jacoby Brissett", Team: "ARI", Mine: true, Photo: photoView("13130", "Jacoby Brissett", true)}, Play: p},
				{Player: SpinePlayer{ID: "16001", Name: "Teagan Quitoriano", Team: "ARI", Photo: photoView("16001", "Teagan Quitoriano", true)}, Play: p},
			}
		}
	}
	for i := range view.Spine {
		view.Spine[i].Game = "DET at ARI"
	}
	view.Unmatched = 1
	view.WeekSpine, view.WeekHeld = view.Spine, 1
	if len(view.Scoring) > 2 {
		view.Scoring = view.Scoring[:2]
	}
	return view
}

func TestLiveWireMatchesGo(t *testing.T) {
	t.Parallel()
	view := liveWire(t)
	// Keep the IPC proof small: the adapter fixtures hold the complete game.
	view.Games[0].Plays = []livegame.Play{}
	view.Followed.Plays = []livegame.Play{}
	body, e := json.MarshalIndent(view, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	body = append(body, '\n')
	file := "frontend/src/app/now/live.testdata.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if e := os.WriteFile(file, body, 0o644); e != nil { //nolint:gosec // committed IPC fixture, not a secret
			t.Fatal(e)
		}
		return
	}
	want, e := os.ReadFile(file)
	if e != nil {
		t.Fatal(e)
	}
	if string(body) != string(want) {
		t.Fatal("live wire stale; regenerate with TWR_WRITE_WIRE=1")
	}
}

func TestLivePregameWireMatchesGo(t *testing.T) {
	t.Parallel()
	g, e := livegame.ParseSummary(liveFixture(t, "espn-summary-401872994"))
	if e != nil {
		t.Fatal(e)
	}
	at := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	a := livegame.Adapter{Board: livegame.Held{Games: []livegame.Game{g}, Source: "ESPN", At: at}, Details: map[string]livegame.Detail{g.ID: {Game: g, At: at}}}
	view := liveGameReading(a, g.ID, false)
	body, e := json.MarshalIndent(view, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	body = append(body, '\n')
	file := "frontend/src/app/now/live-pregame.testdata.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if e := os.WriteFile(file, body, 0o644); e != nil { //nolint:gosec // committed IPC fixture, not a secret
			t.Fatal(e)
		}
		return
	}
	want, e := os.ReadFile(file)
	if e != nil {
		t.Fatal(e)
	}
	if string(body) != string(want) {
		t.Fatal("pregame live wire stale; regenerate with TWR_WRITE_WIRE=1")
	}
}

func TestHeldLiveReadingKeepsTimesAndFailure(t *testing.T) {
	t.Parallel()
	view := liveWire(t)
	g := *view.Followed
	old := time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC)
	now := old.Add(time.Hour)
	a := livegame.Adapter{Board: livegame.Held{Games: []livegame.Game{g}, At: now, Source: "Yahoo failover", Failure: "ESPN down"}, Details: map[string]livegame.Detail{g.ID: {Game: g, At: old, Failure: "participant feed down"}}}
	got := liveGameReading(a, g.ID, true)
	if got.AsOf == got.DetailAsOf || got.Failure == "" || got.DetailFailure == "" || got.Box.Status != "stale" || len(got.Followed.Plays) == 0 {
		t.Fatalf("held detail %+v", got)
	}
}

func TestMFLDemandPollingAndBackoff(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	a.week.Number = 5
	if err := a.keyStore.Set(a.ctx, "synthetic-stored-key"); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC)
	a.pollLiveMFL(a.ctx, at, false)
	if len(transport.requests) != 0 {
		t.Fatal("inactive MFL poll fetched")
	}
	a.pollLiveMFL(a.ctx, at, true)
	if len(transport.requests) != 1 || transport.requests[0] != "liveScoring" || transport.weeks[0] != "5" {
		t.Fatal(transport.requests)
	}
	a.pollLiveMFL(a.ctx, at.Add(29*time.Second), true)
	if len(transport.requests) != 1 {
		t.Fatal("MFL polled early")
	}
	heldAt := a.seasonLineups.fetchedAt
	transport.failure = "liveScoring"
	a.pollLiveMFL(a.ctx, at.Add(30*time.Second), true)
	if len(transport.requests) != 2 || !a.seasonLineups.fetchedAt.Equal(heldAt) || a.seasonLineups.refreshError == "" {
		t.Fatal("MFL lost held time/failure")
	}
	a.pollLiveMFL(a.ctx, at.Add(89*time.Second), true)
	if len(transport.requests) != 2 {
		t.Fatal("MFL did not back off")
	}
	a.pollLiveMFL(a.ctx, at.Add(90*time.Second), false)
	if len(transport.requests) != 2 {
		t.Fatal("closed or off-game MFL refreshed")
	}
}
