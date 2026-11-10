package livescoring

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func ids(t *testing.T, list string) []playerid.PlayerID {
	t.Helper()
	out := make([]playerid.PlayerID, 0)
	for _, s := range strings.Split(list, ",") {
		id, err := playerid.New(s)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	return out
}

func TestSyntheticLineups(t *testing.T) {
	// Synthetic: a singleton matchup/player and a nonstarter.
	body := `{"liveScoring":{"week":"7","matchup":{"franchise":[` +
		`{"id":"0001","score":"1.25","gameSecondsRemaining":"10",` +
		`"playersCurrentlyPlaying":"1","playersYetToPlay":"2",` +
		`"players":{"player":{"id":"16289","status":"nonstarter",` +
		`"score":"0.5","gameSecondsRemaining":"3"}}},` +
		`{"id":"0002","score":"0","gameSecondsRemaining":"0",` +
		`"playersCurrentlyPlaying":"0","playersYetToPlay":"0","players":{}}]}}}`
	raw, err := Parse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ToLineups(raw)
	if err != nil || len(rows.Franchises[0].Starters) != 0 ||
		!reflect.DeepEqual(rows.Franchises[0].NonStarters, ids(t, "16289")) {
		t.Fatalf("nonstarter: %+v %v", rows, err)
	}
	for _, tc := range []struct{ old, replacement, want string }{
		{"nonstarter", "benched", "benched"},
		{`"1.25"`, `"NaN"`, "non-finite"},
		{`"1.25"`, `"oops"`, "score"},
		{`"10"`, `"-1"`, "negative"},
		{`"16289"`, `"bad"`, "player 0"},
		{`"0002"`, `"0001"`, "duplicate franchise"},
		{`"7"`, `"0"`, "positive week"},
	} {
		bad := strings.Replace(body, tc.old, tc.replacement, 1)
		if _, err := Parse([]byte(bad)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: %v", tc.want, err)
		}
	}
	for _, bad := range []string{`{}`, `{`, `{"error":{"$t":"denied"}}`} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	raw.Matchups[0].Franchises[0].Players.Player[0].Status = "unknown"
	if _, err := ToLineups(raw); err == nil {
		t.Fatal("conversion skipped validation")
	}
}

func TestMatchupAndPlayerFields(t *testing.T) {
	body, err := os.ReadFile("testdata/liveScoring-w5.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	for _, homes := range [][2]string{{"1", "0"}, {"0", "1"}, {"0", "0"}, {"1", "1"}} {
		raw.Matchups[0].Franchises[0].IsHome = homes[0]
		raw.Matchups[0].Franchises[1].IsHome = homes[1]
		rows, err := ToLineups(raw)
		if err != nil {
			t.Fatal(err)
		}
		home, away := raw.Matchups[0].Franchises[0].ID, raw.Matchups[0].Franchises[1].ID
		if homes == [2]string{"0", "1"} {
			home, away = away, home
		}
		if rows.Matchups[0] != (leaguefeed.Matchup{Home: home, Away: away}) {
			t.Fatal(rows.Matchups[0])
		}
	}
	for _, field := range []string{"playing", "yet", "player score", "player seconds"} {
		for _, bad := range []string{"oops", "NaN", "+Inf"} {
			t.Run(field+bad, func(t *testing.T) {
				copyRaw, err := Parse(body)
				if err != nil {
					t.Fatal(err)
				}
				f := &copyRaw.Matchups[0].Franchises[0]
				want := ""
				switch field {
				case "playing":
					f.Playing, want = bad, "playersCurrentlyPlaying"
				case "yet":
					f.YetToPlay, want = bad, "playersYetToPlay"
				case "player score":
					f.Players.Player[0].Score, want = bad, "score"
				case "player seconds":
					f.Players.Player[0].SecondsRemaining, want = bad, "gameSecondsRemaining"
				}
				if _, err := ToLineups(copyRaw); err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("%s: %v", want, err)
				}
			})
		}
	}
	raw.Matchups[0].Franchises[0].Players.Player = nil
	rows, err := ToLineups(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(rows.Franchises[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"players":[]`) {
		t.Fatal(string(encoded))
	}
}
