package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func liveFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("internal/ingestion/livegame/testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type liveCrosswalkTransport struct{}

func (liveCrosswalkTransport) RoundTrip(*http.Request) (*http.Response, error) {
	csv := "name,mfl_id,gsis_id,espn_id\nJacoby Brissett,13130,00-0033119,2578570\nTeagan Quitoriano,16001,00-0039000,4374045\n"
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(csv)), Header: http.Header{}}, nil
}
func TestSpineCrosswalkOnlyAndNewestFirst(t *testing.T) {
	cw, err := crosswalk.Fetch(context.Background(), &http.Client{Transport: liveCrosswalkTransport{}}, "https://fixture.invalid/crosswalk.csv")
	if err != nil {
		t.Fatal(err)
	}
	links, _ := cw.Links(func(string) (string, bool) { return "", false })
	known := map[string]bool{}
	players := map[string]SpinePlayer{}
	for _, link := range links {
		if link.IDType == "espn" {
			known[link.IDValue] = true
			players[link.IDValue] = SpinePlayer{ID: link.MFLID, Mine: link.MFLID == "13130"}
		}
	}
	plays, err := livegame.ParseCore(liveFixture(t, "espn-core-plays"))
	if err != nil {
		t.Fatal(err)
	}
	var td livegame.Play
	for _, p := range plays {
		if strings.HasPrefix(p.Text, "J.Brissett pass short right to T.Quitoriano for 2 yards, TOUCHDOWN") {
			td = p
		}
	}
	entries, unmatched := matchSpine([]livegame.Play{td}, known, players)
	if len(entries) != 2 || unmatched != 1 || !entries[0].Player.Mine || entries[1].Player.Mine {
		t.Fatalf("entries %+v unmatched %d", entries, unmatched)
	}
	newer := td
	newer.ID = "later"
	newer.Sequence++
	entries, _ = matchSpine([]livegame.Play{td, newer}, known, players)
	if entries[0].Play.ID != "later" || entries[2].Play.ID != td.ID {
		t.Fatal("spine not newest first")
	}
	entries, unmatched = matchSpine([]livegame.Play{{Text: "Jacoby Brissett", Participants: []string{"unmapped"}}}, known, players)
	if len(entries) != 0 || unmatched != 1 {
		t.Fatal("matched by name")
	}
}
func TestMatchupStartersExcludeBench(t *testing.T) {
	id := func(s string) playerid.PlayerID {
		p, e := playerid.New(s)
		if e != nil {
			t.Fatal(e)
		}
		return p
	}
	l := leaguefeed.Lineups{Matchups: []leaguefeed.Matchup{{Home: "0001", Away: "0002"}}, Franchises: []leaguefeed.Lineup{
		{Franchise: "0001", Starters: []playerid.PlayerID{id("13130")}, NonStarters: []playerid.PlayerID{id("16002")}},
		{Franchise: "0002", Starters: []playerid.PlayerID{id("16001")}},
		{Franchise: "0003", Starters: []playerid.PlayerID{id("16003")}},
	}}
	ids := matchupStarterIDs(l, "0001")
	if len(ids) != 2 || !ids["13130"] || ids["16001"] {
		t.Fatalf("starters %+v", ids)
	}
	if len(matchupStarterIDs(l, "")) != 0 {
		t.Fatal("anonymous matchup inferred")
	}
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
	view.Unmatched = 1
	return view
}
func TestLiveWireMatchesGo(t *testing.T) {
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
