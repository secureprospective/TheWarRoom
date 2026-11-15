package livestats

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Week 10 2026, WSH at NYG: Likely fell on his own fumble (OFC); Reaves recovered a Giants fumble (FC),
// written WAS in the play text; Martin's recovery came on a kickoff, so his REC stays unsplit.
func TestFumbleRecoveriesSplitOwnFromOpponent(t *testing.T) {
	likely := gainLine("Isaiah Likely", "NYG", map[leaguerules.StatID]leaguerules.Decimal{"reception": 10000, "receiving_yards": 60000})
	reaves := gainLine("Jeremy Reaves", "WSH", map[leaguerules.StatID]leaguerules.Decimal{})
	martin := gainLine("Chandler Martin", "NYG", map[leaguerules.StatID]leaguerules.Decimal{})
	lines := map[string]*Line{"1": likely, "2": reaves, "3": martin}
	nyg, wsh := &livegame.State{Possession: "19"}, &livegame.State{Possession: "28"}
	game := livegame.Game{
		Home: livegame.Team{ID: "19", Abbreviation: "NYG"}, Away: livegame.Team{ID: "28", Abbreviation: "WSH"},
		Box: []livegame.Box{{Team: "NYG", Category: "fumbles", Labels: []string{"FUM", "LOST", "REC"},
			Rows: []livegame.BoxRow{{ID: "1", Stats: []string{"1", "0", "1"}}, {ID: "3", Stats: []string{"0", "0", "1"}}}},
			{Team: "WSH", Category: "fumbles", Labels: []string{"FUM", "LOST", "REC"},
				Rows: []livegame.BoxRow{{ID: "2", Stats: []string{"0", "0", "1"}}}}},
		Plays: []livegame.Play{
			{Start: nyg, Text: "(Shotgun) J.McCarthy pass short middle to I.Likely to NYG 29 for 6 yards (J.Reaves, P.Butler). FUMBLES (P.Butler), and recovers at NYG 29."},
			{Start: nyg, Text: "(Shotgun) T.Tracy up the middle to WAS 16 for 3 yards (F.Luvu). FUMBLES (F.Luvu), RECOVERED by WAS-J.Reaves at WAS 20."},
			{Start: wsh, Text: "M.Gay kicks 65 yards from WAS 35 to NYG 0. K.Medrano to NYG 27 for 26 yards (K.Medrano). FUMBLES (K.Medrano), recovered by NYG-C.Martin at NYG 31."},
		},
	}
	attachFumbleRecoveries(game, lines)
	for _, tc := range []struct {
		line          *Line
		own, opponent leaguerules.Decimal
	}{{likely, 10000, 0}, {reaves, 0, 10000}} {
		if tc.line.Stats.Quantities["own_fumble_recovery"] != tc.own || tc.line.Stats.Quantities["opponent_fumble_recovery"] != tc.opponent {
			t.Fatalf("%s: %v", tc.line.Name, tc.line.Stats.Quantities)
		}
	}
	if _, ok := martin.Stats.Quantities["own_fumble_recovery"]; ok {
		t.Fatal("a kickoff recovery was split by possession")
	}
}
