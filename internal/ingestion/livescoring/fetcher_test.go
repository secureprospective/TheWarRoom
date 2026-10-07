package livescoring

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestRealLineups(t *testing.T) {
	cases := []struct {
		file     string
		week     int
		score    float64
		seconds  int
		starters string
		shorter  map[string]int
	}{
		{"liveScoring.json", 4, 292.90, 0,
			"13322,15850,14892,16303,16734,16846,16264,16694,15761,16617,16641," +
				"13813,15798,16460,15754,13133,16195,15836,16230,16267,16150",
			map[string]int{"0003": 20, "0014": 20, "0028": 19}},
		{"liveScoring-w5.json", 5, 0, 75600,
			"15836,16267,16230,16150,15798,13813,16460,15754,13133,16195,16694," +
				"16264,15761,16617,16641,13322,16303,15850,14892,16734,16846",
			map[string]int{"0003": 20, "0014": 20, "0028": 19, "0020": 20}},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			body, err := os.ReadFile("testdata/" + tc.file)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := Parse(body)
			if err != nil {
				t.Fatal(err)
			}
			rows, err := ToLineups(raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(raw.Matchups) != 16 || len(rows.Franchises) != 32 || rows.Week != tc.week {
				t.Fatalf("matchups %d franchises %d week %d", len(raw.Matchups), len(rows.Franchises), rows.Week)
			}
			found := false
			for _, row := range rows.Franchises {
				count := 21
				if shorter, ok := tc.shorter[row.Franchise]; ok {
					count = shorter
				}
				if len(row.Starters) != count || len(row.NonStarters) != 0 {
					t.Fatalf("%s starters %d nonstarters %d", row.Franchise, len(row.Starters), len(row.NonStarters))
				}
				if row.Franchise != "0025" {
					continue
				}
				found = true
				want := leaguefeed.Lineup{
					Franchise: "0025", Score: tc.score, SecondsRemaining: tc.seconds,
					Starters: ids(t, tc.starters), NonStarters: []playerid.PlayerID{},
				}
				if !reflect.DeepEqual(row, want) {
					t.Fatalf("0025 got %+v want %+v", row, want)
				}
			}
			if !found {
				t.Fatal("missing 0025")
			}
		})
	}
}

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
		`"players":{"player":{"id":"16289","status":"nonstarter"}}},` +
		`{"id":"0002","score":"0","gameSecondsRemaining":"0","players":{}}]}}}`
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
