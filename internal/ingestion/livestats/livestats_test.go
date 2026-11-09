package livestats_test

import (
	"os"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livestats"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func heldGame(t *testing.T) livegame.Game {
	t.Helper()
	body, err := os.ReadFile("../livegame/testdata/espn-summary-401872991.json")
	if err != nil {
		t.Fatal(err)
	}
	game, err := livegame.ParseSummary(body)
	if err != nil {
		t.Fatal(err)
	}
	return game
}

func find(t *testing.T, lines []livestats.Line, name string) livestats.Line {
	t.Helper()
	for _, line := range lines {
		if line.Name == name {
			return line
		}
	}
	t.Fatalf("%s is not in the live box score", name)
	return livestats.Line{}
}

// The hand calc: Goff's recorded ESPN line is 19/30, 271 yards, 3 TD, 0 INT, sacked 3 times for 22.
func TestLiveLinesRecordWhatTheBoxScoreReports(t *testing.T) {
	t.Parallel()
	lines, err := livestats.Lines(heldGame(t))
	if err != nil {
		t.Fatal(err)
	}
	goff := find(t, lines, "Jared Goff")
	for id, want := range map[leaguerules.StatID]leaguerules.Decimal{
		"pass_completion": 190000, "pass_attempt": 300000, "passing_yards": 2710000,
		"passing_td": 30000, "interception_thrown": 0, "times_sacked": 30000,
		"times_sacked_yards": 220000,
	} {
		got, ok := goff.Stats.Quantities[id]
		if !ok || got != want {
			t.Fatalf("%s: got %d (held %v) want %d", id, got, ok, want)
		}
	}
	// Nothing the live box score does not report is recorded as a zero.
	for _, id := range []leaguerules.StatID{"fumble_lost", "passing_two_point", "assist"} {
		if _, ok := goff.Stats.Quantities[id]; ok {
			t.Fatalf("%s was invented from a live box score", id)
		}
	}
	gibbs := find(t, lines, "Jahmyr Gibbs")
	for id, want := range map[leaguerules.StatID]leaguerules.Decimal{
		"rushing_attempt": 250000, "rushing_yards": 1510000, "rushing_td": 20000,
	} {
		if got := gibbs.Stats.Quantities[id]; got != want {
			t.Fatalf("%s: got %d want %d", id, got, want)
		}
	}
	laporta := find(t, lines, "Sam LaPorta")
	for id, want := range map[leaguerules.StatID]leaguerules.Decimal{
		"reception": 50000, "receiving_yards": 880000, "receiving_td": 10000, "target": 50000,
	} {
		if got := laporta.Stats.Quantities[id]; got != want {
			t.Fatalf("%s: got %d want %d", id, got, want)
		}
	}
	baker := find(t, lines, "Budda Baker")
	for id, want := range map[leaguerules.StatID]leaguerules.Decimal{
		"tackle": 110000, "sack": 10000, "tackle_for_loss": 10000, "qb_hit": 10000,
		"pass_defensed": 0, "def_tds": 0,
	} {
		got, ok := baker.Stats.Quantities[id]
		if !ok || got != want {
			t.Fatalf("%s: got %d (held %v) want %d", id, got, ok, want)
		}
	}
	if baker.Team == "" || baker.SourceID == "" || baker.Words == "" {
		t.Fatalf("a live line needs its team, source id and words: %+v", baker)
	}
}

// A player who both ran and caught the ball has one line, not two.
func TestLiveLinesMergeOneAthletesCategories(t *testing.T) {
	t.Parallel()
	game := livegame.Game{Box: []livegame.Box{
		{Team: "DET", Category: "rushing", Labels: []string{"CAR", "YDS", "TD"},
			Rows: []livegame.BoxRow{{ID: "1", Name: "Both Ways", Stats: []string{"3", "20", "1"}}}},
		{Team: "DET", Category: "receiving", Labels: []string{"REC", "YDS", "TD", "TGTS"},
			Rows: []livegame.BoxRow{{ID: "1", Name: "Both Ways", Stats: []string{"2", "30", "0", "4"}}}},
	}}
	lines, err := livestats.Lines(game)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Fatalf("lines %+v", lines)
	}
	if lines[0].Stats.Quantities["rushing_yards"] != 200000 || lines[0].Stats.Quantities["receiving_yards"] != 300000 {
		t.Fatalf("merged line %+v", lines[0].Stats.Quantities)
	}
}

// An absent cell stays absent; a cell the source cannot be read from is an error, never a guess.
func TestLiveLinesKeepAbsenceAndRefuseAnUnreadableCell(t *testing.T) {
	t.Parallel()
	game := livegame.Game{Box: []livegame.Box{{Team: "DET", Category: "rushing",
		Labels: []string{"CAR", "YDS", "TD"},
		Rows:   []livegame.BoxRow{{ID: "1", Name: "Dashed", Stats: []string{"3", "--", "0"}}}}}}
	lines, err := livestats.Lines(game)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lines[0].Stats.Quantities["rushing_yards"]; ok {
		t.Fatal("a dash was read as a number")
	}
	if lines[0].Stats.Quantities["rushing_td"] != 0 {
		t.Fatal("a recorded zero was dropped")
	}
	game.Box[0].Rows[0].Stats[1] = "lots"
	if _, err := livestats.Lines(game); err == nil {
		t.Fatal("an unreadable cell was accepted")
	}
}

func TestLiveLinesAdditionalESPNCategories(t *testing.T) {
	for _, tc := range []struct {
		category, label, value string
		stat                   leaguerules.StatID
		want                   leaguerules.Decimal
	}{
		{"defensive", "AST", "3", "assist", 30000},
		{"fumbles", "FUM", "1", "fumbles_total", 10000},
		{"fumbles", "LOST", "0", "fumble_lost", 0},
		{"interceptions", "INT", "2", "interception_caught", 20000},
		{"interceptions", "YDS", "30", "interception_return_yards", 300000},
		{"kickReturns", "YDS", "45", "kickoff_return_yards", 450000},
		{"puntReturns", "YDS", "-2", "punt_return_yards", -20000},
		{"kicking", "XP", "3/4", "pat_made", 30000},
		{"kicking", "XP", "3/4", "pat_missed", 10000},
		{"punting", "NO", "3", "punt", 30000},
		{"punting", "IN 20", "2", "punt_inside_20", 20000},
	} {
		t.Run(tc.category+"_"+tc.label, func(t *testing.T) {
			lines, err := livestats.Lines(livegame.Game{Box: []livegame.Box{{Category: tc.category, Labels: []string{tc.label}, Rows: []livegame.BoxRow{{ID: "1", Stats: []string{tc.value}}}}}})
			if err != nil || len(lines) != 1 {
				t.Fatalf("%+v %v", lines, err)
			}
			if got, ok := lines[0].Stats.Quantities[tc.stat]; !ok || got != tc.want {
				t.Fatalf("%s=%d, present %t", tc.stat, got, ok)
			}
		})
	}
}

func TestLiveKickingDistancesReconcileWithBox(t *testing.T) {
	for _, tc := range []struct {
		name, attempts string
		plays          []string
		present        bool
	}{
		{"complete", "2/3", []string{"J.Bates 39 yard field goal is GOOD, Center-X.", "J.Bates 60 yard field goal is GOOD, Center-X.", "J.Bates 29 yard field goal is No Good, Wide Left."}, true},
		{"missing play", "2/3", []string{"J.Bates 39 yard field goal is GOOD, Center-X."}, false},
		{"unknown name", "1/1", []string{"C.Ryland 53 yard field goal is GOOD, Center-X."}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			game := livegame.Game{Box: []livegame.Box{{Category: "kicking", Labels: []string{"FG"}, Rows: []livegame.BoxRow{{ID: "1", Name: "Jake Bates", Stats: []string{tc.attempts}}}}}}
			for _, text := range tc.plays {
				game.Plays = append(game.Plays, livegame.Play{Text: text})
			}
			lines, err := livestats.Lines(game)
			if err != nil {
				t.Fatal(err)
			}
			_, exists := lines[0].Stats.Quantities["fg_made_0_39"]
			if exists != tc.present {
				t.Fatal(lines)
			}
			if tc.present && (lines[0].Stats.Quantities["fg_made_60_99"] != 10000 || lines[0].Stats.Quantities["fg_missed_0_29"] != 10000 || len(lines[0].Stats.Events["fg_made_distance"]) != 2) {
				t.Fatal(lines)
			}
		})
	}
}

func TestLiveReturnAndRecoveryTouchdownsStayDistinct(t *testing.T) {
	for _, tc := range []struct {
		category, text, possession string
		stat                       leaguerules.StatID
	}{
		{"kickReturns", "", "", "kickoff_return_td"},
		{"puntReturns", "", "", "punt_return_td"},
		{"interceptions", "", "", "interception_return_td"},
		{"rushing", "FUMBLES, RECOVERED by DET-J.Gibbs at ARZ 0. TOUCHDOWN.", "1", "offensive_fumble_recovery_td"},
		{"defensive", "FUMBLES, RECOVERED by DET-J.Gibbs at ARZ 0. TOUCHDOWN.", "2", "fumble_recovery_tds"},
	} {
		t.Run(string(tc.stat), func(t *testing.T) {
			game := livegame.Game{Home: livegame.Team{ID: "1", Abbreviation: "DET"}, Away: livegame.Team{ID: "2", Abbreviation: "ARZ"}, Box: []livegame.Box{{Team: "DET", Category: tc.category, Labels: []string{"TD"}, Rows: []livegame.BoxRow{{ID: "1", Name: "Jahmyr Gibbs", Stats: []string{"1"}}}}}}
			if tc.text != "" {
				game.Plays = []livegame.Play{{Text: tc.text, Start: &livegame.State{Possession: tc.possession}}}
			}
			lines, err := livestats.Lines(game)
			if err != nil || lines[0].Stats.Quantities[tc.stat] != 10000 {
				t.Fatalf("%+v %v", lines, err)
			}
		})
	}
}

// A live line never carries a stat scoring counts from its parts; scoring refuses a recorded total, so one
// would fail every reading of a league that scores it (11-02: pat_missed_or_blocked broke standings).
func TestLiveLinesNeverRecordAPartsTotal(t *testing.T) {
	lines, err := livestats.Lines(livegame.Game{Box: []livegame.Box{
		{Category: "kicking", Labels: []string{"XP", "FG"}, Rows: []livegame.BoxRow{{ID: "1", Stats: []string{"3/4", "1/2"}}}},
		{Category: "rushing", Labels: []string{"YDS"}, Rows: []livegame.BoxRow{{ID: "1", Stats: []string{"40"}}}},
		{Category: "receiving", Labels: []string{"YDS"}, Rows: []livegame.BoxRow{{ID: "1", Stats: []string{"12"}}}},
	}})
	if err != nil || len(lines) != 1 {
		t.Fatalf("%+v %v", lines, err)
	}
	for id := range lines[0].Stats.Quantities {
		if s, ok := leaguerules.StatByID(id); ok && len(s.Parts) > 0 {
			t.Fatalf("%s is a parts total and must not be recorded", id)
		}
	}
	if lines[0].Stats.Quantities["pat_missed"] != 10000 || lines[0].Stats.Quantities["pat_blocked"] != 0 {
		t.Fatalf("missed PAT parts: %+v", lines[0].Stats.Quantities)
	}
}
