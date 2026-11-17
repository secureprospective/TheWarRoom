package livestats

import (
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestWeek10PlayStatGaps(t *testing.T) {
	cases := []struct {
		name, player, team, category, possession string
		labels, stats, plays                     []string
		event                                    leaguerules.StatID
		wantQuantity                             leaguerules.Decimal
		wantEvents                               []leaguerules.Decimal
	}{
		{
			name: "Rodgers recovers own fumble and Frazier fumble", player: "Aaron Rodgers", team: "PIT", category: "fumbles", possession: "1",
			labels: []string{"FUM", "LOST", "REC"}, stats: []string{"1", "0", "2"}, event: "own_fumble_recovery", wantQuantity: 20000,
			plays: []string{
				"(Shotgun) A.Rodgers FUMBLES (Aborted) at PIT 17, and recovers at PIT 16. A.Rodgers to PIT 17 for 1 yard (D.Lawrence).",
				"(Shotgun) Z.Frazier to CIN 32 for -3 yards. FUMBLES, recovered by PIT-A.Rodgers at CIN 33. A.Rodgers to CIN 32 for 1 yard. Handoff to R.Dowdle to CIN 22 for 10 yards (C.Johnson; B.Carter).",
			},
		},
		{
			name: "Laube muffed punt", player: "Dylan Laube", team: "LV", category: "fumbles", possession: "1",
			labels: []string{"FUM", "LOST", "REC"}, stats: []string{"1", "0", "1"}, event: "own_fumble_recovery", wantQuantity: 10000,
			plays: []string{"M.Dickson punts 63 yards to LV 9, Center-C.Stoll. D.Laube MUFFS catch, and recovers at LV 9."},
		},
		{
			name: "Downs muffed punt", player: "Josh Downs", team: "CIN", category: "fumbles", possession: "1",
			labels: []string{"FUM", "LOST", "REC"}, stats: []string{"1", "0", "1"}, event: "own_fumble_recovery", wantQuantity: 10000,
			plays: []string{"J.Bailey punts 50 yards to KC 5, Center-L.McCullough. J.Downs MUFFS catch, and recovers at KC 5. J.Downs to KC 4 for -1 yards (S.Brown)."},
		},
		{
			name: "Jones recovers teammate fumble", player: "Daniel Jones", team: "IND", category: "fumbles", possession: "1",
			labels: []string{"FUM", "LOST", "REC"}, stats: []string{"0", "0", "1"}, event: "own_fumble_recovery", wantQuantity: 10000,
			plays: []string{"(Shotgun) D.Jones Aborted. T.Bortolini FUMBLES at IND 27, recovered by IND-D.Jones at IND 24. D.Jones to IND 26 for 2 yards."},
		},
		{
			name: "Brown recovers lateral fumble", player: "Camden Brown", team: "DAL", category: "fumbles", possession: "1",
			labels: []string{"FUM", "LOST", "REC"}, stats: []string{"1", "0", "1"}, event: "own_fumble_recovery", wantQuantity: 10000,
			plays: []string{"(Shotgun) D.Prescott pass short middle to R.Flournoy to DAL 21 for 12 yards [K.White]. Lateral to C.Beebe to DAL 15 for -6 yards. Lateral to T.Bass to DAL 20 for 5 yards. FUMBLES, recovered by DAL-C.Brown at DAL 16. C.Brown to DAL 13 for -3 yards (M.Sigle). FUMBLES (M.Sigle), recovered by DAL-R.Flournoy at DAL 12. R.Flournoy to DAL 11 for -1 yards. Lateral to D.Prescott to DAL 7 for -4 yards. Lateral to C.Brown to DAL 2 for -5 yards. Lateral to T.Steele to DAL 23 for 21 yards (E.Prysock). FUMBLES (E.Prysock), RECOVERED by SF-M.Sigle at DAL 23."},
		},
		{
			name: "Black 25-yard rush", player: "Kaelon Black", team: "CHI", category: "rushing", possession: "1",
			labels: []string{"CAR", "YDS", "TD"}, stats: []string{"11", "53", "1"}, event: "long_rush", wantEvents: []leaguerules.Decimal{250000},
			plays: []string{"K.Black left end for 25 yards, TOUCHDOWN. E.Pineiro extra point is GOOD, Center-J.Weeks, Holder-C.Waitman."},
		},
		{
			name: "Nabers 37-yard catch", player: "Malik Nabers", team: "MIN", category: "receiving", possession: "1",
			labels: []string{"REC", "YDS", "TD"}, stats: []string{"5", "44", "0"}, event: "long_reception", wantEvents: []leaguerules.Decimal{370000},
			plays: []string{"(Shotgun) J.McCarthy pass deep left to M.Nabers pushed ob at WAS 36 for 37 yards (A.Robertson)."},
		},
		{
			name: "Adebawore fumble-recovery touchdown", player: "Adetomiwa Adebawore", team: "IND", category: "defensive", possession: "2",
			labels: []string{"TD"}, stats: []string{"1"}, event: "fumble_recovery_tds", wantQuantity: 10000,
			plays: []string{"(Shotgun) K.McCord sacked at MIA 2 for -8 yards (J.Walley). FUMBLES (J.Walley) [J.Walley], RECOVERED by IND-A.Adebawore at MIA -6. TOUCHDOWN. S.Shrader extra point is GOOD, Center-J.Bobenmoyer, Holder-R.Sanchez."},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			box := livegame.Box{Team: tc.team, Category: tc.category, Labels: tc.labels,
				Rows: []livegame.BoxRow{{ID: "player", Name: tc.player, Stats: tc.stats}}}
			game := livegame.Game{
				Home: livegame.Team{ID: "1", Abbreviation: tc.team}, Away: livegame.Team{ID: "2", Abbreviation: "MIA"},
				Box: []livegame.Box{box},
			}
			for _, text := range tc.plays {
				game.Plays = append(game.Plays, livegame.Play{Text: text, Start: &livegame.State{Possession: tc.possession}})
			}
			lines, err := Lines(game)
			if err != nil {
				t.Fatal(err)
			}
			got := lines[0].Stats
			if tc.wantQuantity != 0 && got.Quantities[tc.event] != tc.wantQuantity {
				t.Errorf("%s = %d, want %d; line=%+v", tc.event, got.Quantities[tc.event], tc.wantQuantity, lines[0])
			}
			if tc.wantEvents != nil && !slices.Equal(got.Events[tc.event], tc.wantEvents) {
				t.Errorf("%s events = %v, want %v", tc.event, got.Events[tc.event], tc.wantEvents)
			}
		})
	}
}
