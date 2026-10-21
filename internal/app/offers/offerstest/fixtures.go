// Package offerstest supplies shared, offline Offers-cluster fixtures.
package offerstest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type CapSource struct{}

func (CapSource) Fetch(context.Context) (league.RawConfig, error) {
	return league.RawConfig{Source: "test", SalaryCapAmount: "125", RosterSize: "80", TaxiSquad: "8",
		ScoringRules: []league.PositionRuleSet{{Positions: "CB|S", Rules: []league.ScoringRule{
			{Event: "TK", Points: "*0.5", Range: "0-99"}}}},
		RosterLimits: []league.PositionLimit{{Name: "QB", Limit: "0-0"}}}, nil
}

func Lookup(t *testing.T, extra ...players.RawPlayer) normalize.Lookup {
	t.Helper()
	born := fmt.Sprint(time.Date(1999, 3, 15, 0, 0, 0, 0, time.UTC).Unix())
	lk, err := normalize.NewLookup(append([]players.RawPlayer{
		{ID: "0042", Name: "Rostered, Rick", Position: "QB", Team: "BUF", DraftYear: "2015"},
		{ID: "13593", Name: "Smith, John", Position: "WR", Team: "DAL", DraftYear: "2022", Birthdate: born},
		{ID: "13594", Name: "Jones, Amy", Position: "QB", Team: "KCC", DraftYear: "0"},
		{ID: "13595", Name: "Adams, Zed", Position: "CB", Team: "MIA", DraftYear: "2010"},
	}, extra...))
	if err != nil {
		t.Fatal(err)
	}
	return lk
}

func SetRosters(ctx context.Context, t *testing.T, mirror *state.Mirror, extra ...state.MirrorPlayer) {
	t.Helper()
	mine := state.MirrorPlayer{MFLID: "0042", FranchiseID: "0001", RosterStatus: domain.RosterActive, Salary: 2_000_000_000, ContractYear: 2027}
	if _, err := mirror.Replace(ctx, state.MirrorSnapshot{Season: 2026, Players: append([]state.MirrorPlayer{mine}, extra...)}); err != nil {
		t.Fatal(err)
	}
}
