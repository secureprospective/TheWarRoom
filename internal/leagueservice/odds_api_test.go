package leagueservice

import (
	"errors"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/appvalue"
)

func TestOddsValuesSayWhyTheyAreMissing(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		reading appvalue.Reading
		err     error
		words   string
		players int
	}{
		{"failed read", appvalue.Reading{}, errors.New("no record"), "Player values could not be read: no record", 0},
		{"still preparing", appvalue.Reading{Preparing: true}, nil, "Player values are still being prepared", 0},
		{"every franchise's players", appvalue.Reading{Players: []appvalue.Player{{Player: "1", Franchise: "franchise.01", Position: "QB",
			NowPPG: 20, Available: true}, {Player: "2", Franchise: "franchise.02", Position: "RB", NowPPG: 12}}}, nil, "", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := OddsValuesOf(tc.reading, tc.err)
			if got.Words != tc.words || len(got.Players) != tc.players {
				t.Fatalf("got %+v", got)
			}
			if tc.players > 0 && (got.Players[0].NowPPG != 20 || !got.Players[0].Available || got.Players[1].Available) {
				t.Fatalf("values must carry over unchanged: %+v", got.Players)
			}
		})
	}
}
