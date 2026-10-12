package livegame

import (
	"encoding/json"
	"os"
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
