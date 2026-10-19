package livegame

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestCapturedParsers(t *testing.T) {
	games, err := ParseScoreboard(fixture(t, "espn-scoreboard"))
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 15 {
		t.Fatalf("games %d", len(games))
	}
	finals := 0
	for _, g := range games {
		if g.Status == "post" {
			finals++
		}
	}
	if finals != 14 || games[0].Status != "pre" || games[0].Home.Abbreviation != "LAR" {
		t.Fatalf("scoreboard %+v", games[0])
	}
	g, err := ParseSummary(fixture(t, "espn-summary-401872991"))
	if err != nil {
		t.Fatal(err)
	}
	if g.Status != "post" || g.Home.Abbreviation != "ARI" || g.Away.Abbreviation != "DET" || len(g.Box) != 8 {
		t.Fatalf("summary %+v", g)
	}
	checkTouchdown(t, g.Plays)
	plays, err := ParseCore(fixture(t, "espn-core-plays"))
	if err != nil {
		t.Fatal(err)
	}
	if len(plays) != 191 {
		t.Fatalf("plays %d", len(plays))
	}
	checkTouchdown(t, plays)
	found := false
	for _, p := range plays {
		if strings.HasPrefix(p.Text, "J.Brissett pass short right to T.Quitoriano for 2 yards, TOUCHDOWN") {
			if len(p.Participants) < 2 {
				t.Fatal("no participants")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("touchdown absent")
	}
	pre, err := ParseSummary(fixture(t, "espn-summary-401872994"))
	if err != nil {
		t.Fatal(err)
	}
	if pre.Status != "pre" || pre.Last != nil || len(pre.Box) != 0 || len(pre.Plays) != 0 {
		t.Fatalf("pregame %+v", pre)
	}
	yahoo, err := ParseYahoo(fixture(t, "yahoo-scoreboard"))
	if err != nil {
		t.Fatal(err)
	}
	if len(yahoo) != 15 || yahoo[0].Last == nil || yahoo[0].Last.Text != "End of Game" {
		t.Fatalf("Yahoo %+v", yahoo)
	}
}
func checkTouchdown(t *testing.T, plays []Play) {
	t.Helper()
	for _, p := range plays {
		if strings.HasPrefix(p.Text, "J.Brissett pass short right to T.Quitoriano for 2 yards, TOUCHDOWN") {
			if p.Clock != "5:06" || p.Period != 1 || p.HomeScore != 7 || p.AwayScore != 0 {
				t.Fatalf("TD %+v", p)
			}
			return
		}
	}
	t.Fatal("touchdown absent")
}
func TestMalformedFields(t *testing.T) {
	cases := []struct {
		name        string
		parse       func([]byte) error
		body, field string
	}{
		{"scoreboard", func(b []byte) error { _, e := ParseScoreboard(b); return e }, `{"events":[{"id":"1","competitions":[]}]}`, "competitions"},
		{"summary", func(b []byte) error { _, e := ParseSummary(b); return e }, `{}`, "header"},
		{"core", func(b []byte) error { _, e := ParseCore(b); return e }, `{"count":2,"pageCount":2,"items":[]}`, "pageCount"},
		{"yahoo", func(b []byte) error { _, e := ParseYahoo(b); return e }, `{}`, "scoreboard"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if e := c.parse([]byte(c.body)); e == nil || !strings.Contains(e.Error(), c.field) {
				t.Fatalf("error %v", e)
			}
		})
	}
	var raw RawCore
	if e := json.Unmarshal(fixture(t, "espn-core-plays"), &raw); e != nil {
		t.Fatal(e)
	}
	raw.Items[1].Participants[0].Athlete.Ref = "https://host/athletes/not-an-id"
	b, e := json.Marshal(raw)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = ParseCore(b); e == nil || !strings.Contains(e.Error(), "participants") {
		t.Fatalf("error %v", e)
	}
	raw.Items[1].Participants = nil
	raw.Items[0].Start.Distance = nil
	b, e = json.Marshal(raw)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = ParseCore(b); e == nil || !strings.Contains(e.Error(), "distance") {
		t.Fatalf("error %v", e)
	}
}

// liveBoard puts the fixture's first game in progress with the given situation, the shape ESPN
// sent on 2026-10-12 during BUF at LAR.
func liveBoard(t *testing.T, situation map[string]any) []byte {
	t.Helper()
	var board map[string]any
	if err := json.Unmarshal(fixture(t, "espn-scoreboard"), &board); err != nil {
		t.Fatal(err)
	}
	event := board["events"].([]any)[0].(map[string]any)
	inPlay := map[string]any{"displayClock": "12:40", "period": 1, "type": map[string]any{"state": "in", "completed": false}}
	event["status"] = inPlay
	competition := event["competitions"].([]any)[0].(map[string]any)
	competition["status"] = inPlay
	for _, c := range competition["competitors"].([]any) {
		c.(map[string]any)["score"] = "3"
	}
	competition["situation"] = situation
	b, err := json.Marshal(board)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestScoreboardKeepsTheBoardWhenASituationHasNoPossession(t *testing.T) {
	last := map[string]any{"id": "401872994183", "text": "Kickoff to the LAR 30."}
	games, err := ParseScoreboard(liveBoard(t, map[string]any{"possession": "", "lastPlay": last}))
	if err != nil {
		t.Fatal("a kickoff situation failed the whole board:", err)
	}
	if games[0].State != nil || games[0].Last == nil || games[0].Last.Text != "Kickoff to the LAR 30." {
		t.Fatalf("no field state, last play kept: %+v", games[0])
	}
	games, err = ParseScoreboard(liveBoard(t, map[string]any{"possession": "14", "down": 2, "distance": 7,
		"yardLine": 33, "lastPlay": last}))
	if err != nil || games[0].State == nil || games[0].State.Possession != "14" || games[0].State.Down != 2 {
		t.Fatalf("a full situation gives the field state: %v %+v", err, games)
	}
}

func TestYahooReadsAGameInProgress(t *testing.T) {
	var board map[string]any
	if err := json.Unmarshal(fixture(t, "yahoo-scoreboard"), &board); err != nil {
		t.Fatal(err)
	}
	games := board["service"].(map[string]any)["scoreboard"].(map[string]any)["games"].(map[string]any)
	// Flip the first finished game by ID: the pregame game has no points yet, and map order is random.
	ids := make([]string, 0, len(games))
	for id := range games {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if g := games[id].(map[string]any); g["status_type"] == "status.type.final" {
			g["status_type"] = "status.type.in_progress"
			break
		}
	}
	b, err := json.Marshal(board)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseYahoo(b)
	if err != nil {
		t.Fatal("a live game failed the Yahoo board:", err)
	}
	live := 0
	for _, g := range parsed {
		if g.Status == "in" {
			live++
		}
	}
	if live != 1 {
		t.Fatalf("one game in progress, got %d", live)
	}
}

func TestSummaryCountsTheDriveInProgressOnce(t *testing.T) {
	var summary map[string]any
	if err := json.Unmarshal(fixture(t, "espn-summary-401872991"), &summary); err != nil {
		t.Fatal(err)
	}
	drives := summary["drives"].(map[string]any)
	previous := drives["previous"].([]any)
	want, err := ParseSummary(fixture(t, "espn-summary-401872991"))
	if err != nil {
		t.Fatal(err)
	}
	// ESPN's live shape: the drive in progress is also the last previous drive, same id.
	last := previous[len(previous)-1].(map[string]any)
	last["id"] = "4018729942"
	drives["current"] = last
	b, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseSummary(b)
	if err != nil {
		t.Fatal("the drive in progress failed the game details:", err)
	}
	if len(got.Plays) != len(want.Plays) {
		t.Fatalf("plays %d, want %d", len(got.Plays), len(want.Plays))
	}
}
