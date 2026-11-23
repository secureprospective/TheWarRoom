package leagueservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
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

type dueInjuryFeedStub struct {
	feed      NFLInjuryFeed
	onRefresh func(season int) error
	refreshes int
}

func (s *dueInjuryFeedStub) ReadNFLInjuryFeed(context.Context) (NFLInjuryFeed, error) {
	return s.feed, nil
}

func (s *dueInjuryFeedStub) ResolveNFLReturnWeeks(
	rows []leaguefeed.Injury, _ []leagueweek.Week,
) []leaguefeed.Injury {
	return rows
}

func (s *dueInjuryFeedStub) Refresh(
	_ context.Context, season int, _ http.RoundTripper, _ io.Writer,
) error {
	s.refreshes++
	if s.onRefresh != nil {
		return s.onRefresh(season)
	}
	return nil
}

func mustInjuryRow(t *testing.T, id, status string) leaguefeed.Injury {
	t.Helper()
	pid, err := playerid.New(id)
	if err != nil {
		t.Fatalf("playerid %q: %v", id, err)
	}
	return leaguefeed.Injury{ID: pid, Status: status}
}

func TestDueWorkerAutoLineupUsesLiveInjuriesAndFallback(t *testing.T) {
	kickoff := time.Date(2026, 10, 25, 17, 0, 0, 0, time.UTC)
	waveAt := kickoff.Add(-time.Hour)
	vacant := "franchise.04"
	extra := make([]commissioner.NativePlayer, 0, 5)
	for _, id := range []string{"7001", "7002", "7003", "7004", "7005"} {
		extra = append(extra, commissioner.NativePlayer{
			ID: id, Franchise: vacant, Status: domain.RosterActive,
			Salary: 100000000, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA,
		})
	}
	f := newDueFixtureWithPlayers(t, extra)
	facts := nativecheck.Facts{
		"0099": {Name: "QB One", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true},
		"7001": {Name: "WR Live Out", Position: leaguerules.WR, Team: "DAL", DesignationHeld: true},
		"7002": {Name: "WR Live Doubtful", Position: leaguerules.WR, Team: "DAL", DesignationHeld: true},
		"7003": {Name: "WR Live Q", Position: leaguerules.WR, Team: "DAL", DesignationHeld: true},
		"7004": {Name: "WR Facts Out", Position: leaguerules.WR, Team: "DAL",
			Designation: leaguerules.Out, DesignationHeld: true},
		"7005": {Name: "WR Facts Healthy", Position: leaguerules.WR, Team: "DAL", DesignationHeld: true},
	}
	f.store.SetLineupEnvironment(&dueLineupEnv{
		facts: facts,
		weeks: []leagueweek.Week{{
			Number: 1,
			Games:  []leagueweek.Game{{Teams: [2]string{"DAL", "PHI"}, Kickoff: kickoff}},
		}},
	})
	stub := &dueInjuryFeedStub{
		feed: NFLInjuryFeed{
			Season:    2026,
			FetchedAt: waveAt.Add(-10 * time.Minute),
			Available: true,
			Injuries: []leaguefeed.Injury{
				mustInjuryRow(t, "7001", "Out"),
				mustInjuryRow(t, "7002", "Doubtful"),
				mustInjuryRow(t, "7003", "Questionable"),
			},
		},
	}
	f.worker.WithNFLInjuryFeed(stub)
	f.clock.instant.Store(waveAt.UnixNano())
	if _, err := f.worker.processLeague(t.Context(), f.league); err != nil {
		t.Fatalf("processLeague: %v", err)
	}
	records, err := f.store.SharedCommissionerRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	got, found, err := commissioner.RecordedLineupFor(records, vacant, 2026, 1)
	if err != nil || !found {
		t.Fatalf("vacant lineup missing: found=%v err=%v", found, err)
	}
	want := []string{"7003", "7005"}
	if !slices.Equal(got.Starters, want) {
		t.Fatalf("starters = %v, want %v", got.Starters, want)
	}
}

func TestDueWorkerAutoLineupRefreshesStaleFeedOnceAndFallsBackOnError(t *testing.T) {
	kickoff := time.Date(2026, 10, 25, 17, 0, 0, 0, time.UTC)
	waveAt := kickoff.Add(-time.Hour)
	vacant := "franchise.04"
	setup := func(t *testing.T, stub *dueInjuryFeedStub) []string {
		t.Helper()
		extra := []commissioner.NativePlayer{
			{ID: "7001", Franchise: vacant, Status: domain.RosterActive,
				Salary: 100000000, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA},
			{ID: "7002", Franchise: vacant, Status: domain.RosterActive,
				Salary: 100000000, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA},
		}
		f := newDueFixtureWithPlayers(t, extra)
		f.store.SetLineupEnvironment(&dueLineupEnv{
			facts: nativecheck.Facts{
				"0099": {Name: "QB One", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true},
				"7001": {Name: "QB One", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true},
				"7002": {Name: "QB Two", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true},
			},
			weeks: []leagueweek.Week{{
				Number: 1,
				Games:  []leagueweek.Game{{Teams: [2]string{"DAL", "PHI"}, Kickoff: kickoff}},
			}},
		})
		f.worker.WithNFLInjuryFeed(stub)
		f.clock.instant.Store(waveAt.UnixNano())
		if _, err := f.worker.processLeague(t.Context(), f.league); err != nil {
			t.Fatalf("processLeague: %v", err)
		}
		records, err := f.store.SharedCommissionerRecords(t.Context(), f.league)
		if err != nil {
			t.Fatal(err)
		}
		got, found, err := commissioner.RecordedLineupFor(records, vacant, 2026, 1)
		if err != nil || !found {
			t.Fatalf("vacant lineup missing: found=%v err=%v", found, err)
		}
		return got.Starters
	}

	t.Run("stale triggers one refresh before reading", func(t *testing.T) {
		var stub *dueInjuryFeedStub
		stub = &dueInjuryFeedStub{
			feed: NFLInjuryFeed{
				Season:    2026,
				FetchedAt: waveAt.Add(-46 * time.Minute),
				Available: true,
			},
			onRefresh: func(season int) error {
				if season != 2026 {
					t.Errorf("refresh season = %d, want 2026", season)
				}
				stub.feed.FetchedAt = waveAt
				stub.feed.Injuries = []leaguefeed.Injury{mustInjuryRow(t, "7001", "Out")}
				return nil
			},
		}
		starters := setup(t, stub)
		if stub.refreshes != 1 {
			t.Fatalf("refreshes = %d, want 1", stub.refreshes)
		}
		if !slices.Equal(starters, []string{"7002"}) {
			t.Fatalf("starters = %v, want [7002]", starters)
		}
	})

	t.Run("fresh triggers no refresh", func(t *testing.T) {
		stub := &dueInjuryFeedStub{
			feed: NFLInjuryFeed{
				Season:    2026,
				FetchedAt: waveAt.Add(-20 * time.Minute),
				Available: true,
				Injuries:  []leaguefeed.Injury{mustInjuryRow(t, "7001", "Out")},
			},
		}
		starters := setup(t, stub)
		if stub.refreshes != 0 {
			t.Fatalf("refreshes = %d, want 0", stub.refreshes)
		}
		if !slices.Equal(starters, []string{"7002"}) {
			t.Fatalf("starters = %v, want [7002]", starters)
		}
	})

	t.Run("failed refresh still sets lineup from held data", func(t *testing.T) {
		stub := &dueInjuryFeedStub{
			feed: NFLInjuryFeed{
				Season:    2026,
				FetchedAt: waveAt.Add(-90 * time.Minute),
				Available: true,
				Injuries:  []leaguefeed.Injury{mustInjuryRow(t, "7001", "Out")},
			},
			onRefresh: func(int) error { return errors.New("upstream unavailable") },
		}
		starters := setup(t, stub)
		if stub.refreshes != 1 {
			t.Fatalf("refreshes = %d, want 1", stub.refreshes)
		}
		if !slices.Equal(starters, []string{"7002"}) {
			t.Fatalf("starters = %v, want [7002]", starters)
		}
	})
}
