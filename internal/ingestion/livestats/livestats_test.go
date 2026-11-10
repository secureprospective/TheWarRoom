package livestats_test

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livestats"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

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

func TestADefendersTacklesAreSoloAndAssistedNeverBoth(t *testing.T) {
	read := func(labels, stats []string) map[leaguerules.StatID]leaguerules.Decimal {
		t.Helper()
		lines, err := livestats.Lines(livegame.Game{Box: []livegame.Box{{Category: "defensive", Labels: labels,
			Rows: []livegame.BoxRow{{ID: "1", Stats: stats}}}}})
		if err != nil || len(lines) != 1 {
			t.Fatalf("%+v %v", lines, err)
		}
		return lines[0].Stats.Quantities
	}
	got := read([]string{"TOT", "SOLO", "SACKS"}, []string{"10", "6", "1"})
	if got["tackle"] != 60000 || got["assist"] != 40000 || got["sack"] != 10000 {
		t.Fatalf("total and solo: %+v", got)
	}
	got = read([]string{"TOT", "SOLO", "AST"}, []string{"10", "6", "3"})
	if got["tackle"] != 60000 || got["assist"] != 30000 {
		t.Fatalf("the box score's own assists were replaced: %+v", got)
	}
	got = read([]string{"TOT"}, []string{"10"})
	if _, held := got["tackle"]; held || len(got) != 0 {
		t.Fatalf("a total was counted as tackles: %+v", got)
	}
}
