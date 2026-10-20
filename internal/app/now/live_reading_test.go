package now

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/race/racetest"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

type liveCrosswalkTransport struct{}

func (liveCrosswalkTransport) RoundTrip(*http.Request) (*http.Response, error) {
	csv := "name,mfl_id,gsis_id,espn_id\nJacoby Brissett,13130,00-0033119,2578570\nTeagan Quitoriano,16001,00-0039000,4374045\n"
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(csv)), Header: http.Header{}}, nil
}
func TestSpineCrosswalkOnlyAndNewestFirst(t *testing.T) {
	t.Parallel()
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
	entries, unmatched := MatchSpine([]livegame.Game{{Plays: []livegame.Play{td}}}, known, players)
	if len(entries) != 2 || unmatched != 1 || !entries[0].Player.Mine || entries[1].Player.Mine {
		t.Fatalf("entries %+v unmatched %d", entries, unmatched)
	}
	newer := td
	newer.ID = "later"
	newer.Sequence++
	entries, _ = MatchSpine([]livegame.Game{{Plays: []livegame.Play{td, newer}}}, known, players)
	if entries[0].Play.ID != "later" || entries[2].Play.ID != td.ID {
		t.Fatal("spine not newest first")
	}
	entries, unmatched = MatchSpine([]livegame.Game{{Plays: []livegame.Play{{Text: "Jacoby Brissett", Participants: []string{"unmapped"}}}}}, known, players)
	if len(entries) != 0 || unmatched != 1 {
		t.Fatal("matched by name")
	}
}

func TestMatchupStartersExcludeBench(t *testing.T) {
	t.Parallel()
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
	ids := MatchupStarterIDs(l, "0001")
	if len(ids) != 2 || !ids["13130"] || ids["16001"] {
		t.Fatalf("starters %+v", ids)
	}
	if len(MatchupStarterIDs(l, "")) != 0 {
		t.Fatal("anonymous matchup inferred")
	}
}

func liveFixture(t *testing.T, name string) []byte {
	return []byte(racetest.Sample(t, "livegame", name+".json"))
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
	held := HeldWeekGames(LiveSnapshot{Details: adapter.Details})
	if len(held) != 2 {
		t.Fatal("held games counted twice or empty games counted", len(held))
	}
	players := map[string]SpinePlayer{"m": {ID: "1", Mine: true}, "o": {ID: "2"}}
	entries, _ := MatchSpine(held, map[string]bool{"m": true, "o": true}, players)
	order := ""
	for _, e := range entries {
		order += e.Play.ID
	}
	if order != "dcba" || entries[0].Game != "BUF at LAR" || entries[1].Game != "DET at ARI" ||
		!entries[0].Player.Mine || entries[1].Player.Mine {
		t.Fatalf("order %s entries %+v", order, entries)
	}
}
