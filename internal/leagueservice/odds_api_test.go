package leagueservice

import (
	"context"
	"errors"
	"strings"
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

// valuesOnly is a host that answers only the value read.
type valuesOnly struct {
	Host
	reading appvalue.Reading
	err     error
}

func (h valuesOnly) NativeValues(context.Context, string, string, int) (appvalue.Reading, error) {
	return h.reading, h.err
}

// Odds never go blank for want of player values: a host without them, a pass still preparing and a
// failed pass each give results-only odds that say why (R11); a completed pass gives value-based odds.
func TestOddsFallBackToResultsWhenValuesAreMissing(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		host    Host
		reduced string
	}{
		{"no values on the host", nil, "player values are not on this server yet"},
		{"still preparing", valuesOnly{reading: appvalue.Reading{Preparing: true}}, "player values are still being prepared"},
		{"failed pass", valuesOnly{err: errors.New("no weekly statistics")}, "player values could not be read (no weekly statistics)"},
		{"completed pass", valuesOnly{reading: appvalue.Reading{Players: []appvalue.Player{{Player: "1", Franchise: "franchise.01"}}}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := New(tc.host).oddsValues(t.Context(), "native.l", 2026)
			if got.Words != "" || !strings.HasPrefix(got.Reduced, tc.reduced) || (tc.reduced == "") != (got.Reduced == "") {
				t.Fatalf("got %+v", got)
			}
			if tc.reduced == "" && len(got.Players) != 1 {
				t.Fatalf("a completed pass must give value-based odds: %+v", got)
			}
		})
	}
}
