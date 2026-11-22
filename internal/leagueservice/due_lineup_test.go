package leagueservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type dueLineupEnv struct {
	facts nativecheck.Facts
	weeks []leagueweek.Week
}

func (e *dueLineupEnv) PlayerFacts() nativecheck.Facts { return e.facts }

func (e *dueLineupEnv) ScheduleDates(_ context.Context, _ int) ([]leagueweek.Week, error) {
	return e.weeks, nil
}

func (e *dueLineupEnv) LineupFacts(_ context.Context, _, week int) (commissioner.LineupFacts, error) {
	for _, w := range e.weeks {
		if w.Number == week {
			return commissioner.LineupFacts{
				Facts: e.facts, Locks: leagueweek.Locks(w), ScheduleHeld: true, Zone: time.UTC,
			}, nil
		}
	}
	return commissioner.LineupFacts{}, fmt.Errorf("week %d not held", week)
}

func TestDueWorkerSetsVacantFranchiseLineupAtLockMinus60Min(t *testing.T) {
	kickoff := time.Date(2026, 10, 25, 17, 0, 0, 0, time.UTC)
	// franchise.04 is vacant (only franchise.01 and franchise.02 are seated by newDueFixture).
	vacant := "franchise.04"
	extra := []commissioner.NativePlayer{{
		ID: "7001", Franchise: vacant, Status: domain.RosterActive,
		Salary: 100000000, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA,
	}}
	f := newDueFixtureWithPlayers(t, extra)
	facts := nativecheck.Facts{
		"0099": {Name: "QB One", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true},
		"7001": {Name: "Vacant QB", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true},
	}
	f.store.SetLineupEnvironment(&dueLineupEnv{
		facts: facts,
		weeks: []leagueweek.Week{{
			Number: 1,
			Games:  []leagueweek.Game{{Teams: [2]string{"DAL", "PHI"}, Kickoff: kickoff}},
		}},
	})

	// 61 minutes before kickoff: no lineup set yet, next deadline is kickoff - 60 min.
	f.clock.instant.Store(kickoff.Add(-61 * time.Minute).UnixNano())
	next, err := f.worker.processLeague(t.Context(), f.league)
	if err != nil {
		t.Fatalf("processLeague at -61m: %v", err)
	}
	if !next.Equal(kickoff.Add(-time.Hour)) {
		t.Fatalf("next deadline = %s, want %s", next, kickoff.Add(-time.Hour))
	}
	records, err := f.store.SharedCommissionerRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := commissioner.RecordedLineupFor(records, vacant, 2026, 1); err != nil || found {
		t.Fatalf("lineup set too early: found=%v err=%v", found, err)
	}

	// Exactly 60 minutes before kickoff: one processLeague pass sets the vacant franchise's lineup.
	f.clock.instant.Store(kickoff.Add(-time.Hour).UnixNano())
	if _, err := f.worker.processLeague(t.Context(), f.league); err != nil {
		t.Fatalf("processLeague at -60m: %v", err)
	}
	records, err = f.store.SharedCommissionerRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	got, found, err := commissioner.RecordedLineupFor(records, vacant, 2026, 1)
	if err != nil || !found {
		t.Fatalf("vacant franchise lineup missing at -60m: found=%v err=%v", found, err)
	}
	if got.By != commissioner.LineupClockPrincipal || len(got.Starters) != 1 || got.Starters[0] != "7001" {
		t.Fatalf("unexpected automatic lineup: by=%q starters=%v", got.By, got.Starters)
	}
}
