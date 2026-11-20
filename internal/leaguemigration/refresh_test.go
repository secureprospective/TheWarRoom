package leaguemigration_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type refreshFixture struct {
	svc    *leaguemigration.Service
	store  *history.Store
	ledger leagueservice.MigrationLedger
	src    *migrationtest.Source
	league string
	now    time.Time
}

type testLineupEnv struct {
	facts nativecheck.Facts
	locks map[int]map[string]time.Time
}

func (e testLineupEnv) PlayerFacts() nativecheck.Facts { return e.facts }

func (e testLineupEnv) LineupFacts(_ context.Context, _, week int) (commissioner.LineupFacts, error) {
	locks, held := e.locks[week]
	return commissioner.LineupFacts{Facts: e.facts, Locks: locks, ScheduleHeld: held, Zone: time.UTC}, nil
}

func newRefreshFixture(t *testing.T, export mflsource.Export, facts nativecheck.Facts) *refreshFixture {
	t.Helper()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "league.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	s := history.New(pools, registry)
	if err := s.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	f := &refreshFixture{
		store:  s,
		src:    &migrationtest.Source{Export: export},
		league: leaguemigration.AppLeagueID(export.LeagueID, export.Season),
		now:    migrationtest.ReadAt.Add(time.Hour),
	}
	s.SetClock(func() time.Time { return f.now })
	f.ledger = leagueservice.MigrationLedger{
		Store: s,
		Facts: func() nativecheck.Facts { return facts },
	}
	f.svc = &leaguemigration.Service{
		Source: func(string) (leaguemigration.Source, error) { return f.src, nil },
		Ledger: f.ledger,
		Now:    func() time.Time { return f.now },
	}
	if _, err := f.svc.Import(t.Context(), export.LeagueID, "Initial MFL load", "creator"); err != nil {
		t.Fatal(err)
	}
	review, err := f.svc.Review(t.Context(), leaguemigration.Request{MFLLeague: export.LeagueID, League: f.league})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Cutover(t.Context(), leaguemigration.Request{MFLLeague: export.LeagueID, League: f.league},
		review.Digest, "Cutover to hosted server", "creator"); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Hour)
	return f
}

func (f *refreshFixture) exec(t *testing.T, raw commissioner.RawCommand, principal string, facts nativecheck.Facts) commissioner.Receipt {
	t.Helper()
	records, err := f.ledger.Records(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	raw.League = f.league
	raw.Expected = int64(len(records))
	receipt, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, principal, nil, facts, time.UTC)
	if err != nil {
		t.Fatalf("execute %s: %v", raw.Command, err)
	}
	f.now = f.now.Add(time.Minute)
	return receipt
}

func (f *refreshFixture) records(t *testing.T) []commissioner.Record {
	t.Helper()
	recs, err := f.ledger.Records(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func mustPID(t *testing.T, raw string) *playerid.PlayerID {
	t.Helper()
	id, err := playerid.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	return &id
}

func TestRefreshRostersContractsAdjustmentsAndPicks(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Assets league", migrationtest.Specs()["8 teams"])
	f := newRefreshFixture(t, export, nil)

	// Record a server-only salary adjustment that must survive refresh.
	serverAmt := domain.Money(2500000)
	serverAdj := f.exec(t, commissioner.RawCommand{
		Command: commissioner.CapAdjust, Request: "server-cap-adj", Reason: "In-app fine",
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Override: &commissioner.RawNativeOverride{
			Franchise: "franchise.01", Season: 2026, Amount: &serverAmt,
		}},
	}, "creator", nil)

	// Mutate MFL export across rosters, statuses, contracts, adjustments, and picks.
	next := f.src.Export
	next.Players = map[string]string{}
	for k, v := range f.src.Export.Players {
		next.Players[k] = v
	}
	next.Players["9901"] = "Free Agent Signee"
	next.Players["9902"] = "Drafted Rookie"

	rosters := slices.Clone(f.src.Export.Rosters)
	// Drop rosters[0], trade rosters[1] to franchise 0002, move rosters[2] to IR, edit rosters[3] contract.
	droppedID := rosters[0].Player
	tradedID := rosters[1].Player
	irID := rosters[2].Player
	editedID := rosters[3].Player
	rosters = rosters[1:] // drop original [0]
	rosters[0].Franchise = "0002"
	rosters[1].Status = "INJURED_RESERVE"
	rosters[2].Salary = "12.50"
	rosters[2].ContractYear = "2030"
	rosters[2].ContractStatus = "EXT (2026)"
	rosters[2].ContractInfo = "12.5M / 4yr"
	// Add a brand-new player directly onto TAXI_SQUAD with contract.
	rosters = append(rosters, mflsource.Rostered{
		Franchise: "0003", Player: "9901", Status: "TAXI_SQUAD",
		Salary: "1.25", ContractYear: "2029", ContractStatus: "Rookie", ContractInfo: "1.25M / 3yr",
	})
	next.Rosters = rosters

	// Adjustments: edit adj-1, remove adj-2, add adj-3.
	next.Adjustments = []mflsource.Adjustment{
		{ID: "adj-1", Franchise: "0002", Amount: "3.75", Description: "Updated dead money"},
		{ID: "adj-3", Franchise: "0004", Amount: "-1.00", Description: "Comp credit"},
	}

	// Picks: select draft slot 0, skip draft slot 1, change future pick 0 owner, remove future pick 1, add 2029 future pick.
	draft := slices.Clone(f.src.Export.Draft)
	draft[0].Player = mustPID(t, "9902")
	draft[0].Skipped = false
	draft[1].Player = nil
	draft[1].Skipped = true
	next.Draft = draft

	future := slices.Clone(f.src.Export.Future)
	future[0].Owner = "0005"
	removedFuture := future[1]
	future = append(future[:1], future[2:]...)
	future = append(future, draftreports.FuturePick{Year: 2029, Round: 1, Original: "0001", Owner: "0006"})
	next.Future = future
	f.src.Export = next

	report, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Refresh from MFL", Principal: "creator",
	})
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if len(report.Receipts) == 0 || report.Summary.RosterRemoves < 2 || report.Summary.RosterAdds < 2 ||
		report.Summary.Contracts < 1 || report.Summary.Adjustments != 3 || report.Summary.Picks < 5 {
		t.Fatalf("unexpected summary: %+v (receipts=%d)", report.Summary, len(report.Receipts))
	}

	app, err := leaguemigration.ReadApp(f.league, f.records(t), f.now)
	if err != nil {
		t.Fatal(err)
	}
	snap := app.Snapshot
	if snap.PlayerOf("franchise.01", droppedID) >= 0 {
		t.Fatalf("dropped player %s still on roster", droppedID)
	}
	if snap.PlayerOf("franchise.02", tradedID) < 0 {
		t.Fatalf("traded player %s missing on franchise.02", tradedID)
	}
	if idx := snap.PlayerOf("franchise.01", irID); idx < 0 || snap.Players[idx].Status != domain.RosterIR {
		t.Fatalf("IR status not updated for %s", irID)
	}
	if idx := snap.PlayerOf("franchise.01", editedID); idx < 0 || snap.Players[idx].Salary != 1250000000 ||
		snap.Players[idx].ExpirationYear != 2030 || snap.Players[idx].HeldContractStatus != "EXT (2026)" ||
		snap.Players[idx].ContractInfo != "12.5M / 4yr" {
		t.Fatalf("edited contract mismatch: %+v", snap.Players[idx])
	}
	if idx := snap.PlayerOf("franchise.03", "9901"); idx < 0 || snap.Players[idx].Status != domain.RosterTaxi ||
		snap.Players[idx].Salary != 125000000 || snap.Players[idx].ExpirationYear != 2029 {
		t.Fatalf("added taxi player mismatch: %+v", snap.Players[idx])
	}

	// Server-only adjustment preserved, MFL adjustments reconciled.
	adjByID := map[string]commissioner.NativeCharge{}
	for _, a := range snap.Adjustments {
		adjByID[a.ID] = a
	}
	if _, ok := adjByID[serverAdj.Entry]; !ok {
		t.Fatalf("server-only adjustment %s was dropped", serverAdj.Entry)
	}
	if _, ok := adjByID["adj-2"]; ok {
		t.Fatal("removed MFL adjustment adj-2 still present")
	}
	if adjByID["adj-1"].Amount != 375000000 || adjByID["adj-3"].Amount != -100000000 {
		t.Fatalf("MFL adjustments mismatch: %+v", adjByID)
	}

	// Verify picks.
	var found2029, foundRemoved bool
	for _, pk := range snap.Picks {
		if pk.Season == 2026 && pk.Round == draft[0].Round && pk.Slot == draft[0].Slot && pk.Player != "9902" {
			t.Fatalf("draft selection not synced: %+v", pk)
		}
		if pk.Season == 2026 && pk.Round == draft[1].Round && pk.Slot == draft[1].Slot && !pk.Skipped {
			t.Fatalf("draft skip not synced: %+v", pk)
		}
		if pk.Season == 2029 && pk.Round == 1 && pk.Original == "franchise.01" && pk.Owner == "franchise.06" {
			found2029 = true
		}
		if pk.Season == removedFuture.Year && pk.Round == removedFuture.Round && pk.Original == "franchise."+removedFuture.Original[2:] {
			foundRemoved = true
		}
	}
	if !found2029 || foundRemoved {
		t.Fatalf("future picks mismatch: found2029=%t foundRemoved=%t", found2029, foundRemoved)
	}

	// Idempotent second run writes nothing.
	beforeCount := len(f.records(t))
	again, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Refresh from MFL", Principal: "creator",
	})
	if err != nil || len(again.Commands) != 0 || len(again.Receipts) != 0 || len(f.records(t)) != beforeCount {
		t.Fatalf("second refresh was not a no-op: report=%+v err=%v", again, err)
	}
}

func TestRefreshScheduleAndMonotonicCutoverWeek(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Schedule league", migrationtest.Specs()["8 teams"])
	var err error
	export.Schedule, err = leagueschedule.Parse([]byte(`{"schedule":{"weeklySchedule":[
		{"week":"1","matchup":{"franchise":[{"id":"0001","isHome":"1","score":"105.25"},{"id":"0002","isHome":"0","score":"99.00"}]}},
		{"week":"2","matchup":{"franchise":[{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}},
		{"week":"3","matchup":{"franchise":[{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}}
	]}}`))
	if err != nil {
		t.Fatal(err)
	}
	export.Results, err = weeklyresults.Parse([]byte(`{"allWeeklyResults":{"weeklyResults":[
		{"week":"1","matchup":{"regularSeason":"1","franchise":[{"id":"0001","isHome":"1","score":"105.25","result":"W"},{"id":"0002","isHome":"0","score":"99.00","result":"L"}]}},
		{"week":"2","matchup":{"regularSeason":"1","franchise":[{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}}
	]}}`))
	if err != nil {
		t.Fatal(err)
	}
	f := newRefreshFixture(t, export, nil)
	if held := commissioner.RecordedResults(f.records(t), 2026); held == nil || held.CutoverWeek != 2 {
		t.Fatalf("initial cutover = %+v, want 2", held)
	}

	// Week 2 and Week 3 finish on MFL -> CutoverWeek advances from 2 to 4.
	next := f.src.Export
	next.Results, err = weeklyresults.Parse([]byte(`{"allWeeklyResults":{"weeklyResults":[
		{"week":"1","matchup":{"regularSeason":"1","franchise":[{"id":"0001","isHome":"1","score":"105.25","result":"W"},{"id":"0002","isHome":"0","score":"99.00","result":"L"}]}},
		{"week":"2","matchup":{"regularSeason":"1","franchise":[{"id":"0001","isHome":"1","score":"120.50","result":"W"},{"id":"0002","isHome":"0","score":"110.00","result":"L"}]}},
		{"week":"3","matchup":{"regularSeason":"1","franchise":[{"id":"0001","isHome":"1","score":"88.00","result":"L"},{"id":"0002","isHome":"0","score":"95.00","result":"W"}]}}
	]}}`))
	if err != nil {
		t.Fatal(err)
	}
	f.src.Export = next

	report, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Refresh scores", Principal: "creator",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Summary.ScheduleUpdated || report.Summary.CutoverWeek != 4 || report.Summary.RecordedWeeks != 3 {
		t.Fatalf("unexpected schedule summary: %+v", report.Summary)
	}
	held := commissioner.RecordedResults(f.records(t), 2026)
	if held == nil || held.CutoverWeek != 4 || len(held.Teams) != 6 {
		t.Fatalf("recorded results after advance: %+v", held)
	}

	// If MFL feed later omits week 2-3 (smaller CutoverWeek 2), CutoverWeek must NEVER move backwards.
	regressed := f.src.Export
	regressed.Results = export.Results
	f.src.Export = regressed
	report2, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Refresh monotonic check", Principal: "creator",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report2.Commands) != 0 || report2.Summary.CutoverWeek != 4 {
		t.Fatalf("cutover moved backwards or rewrote unnecessarily: %+v", report2)
	}
	if heldAfter := commissioner.RecordedResults(f.records(t), 2026); heldAfter == nil || heldAfter.CutoverWeek != 4 {
		t.Fatalf("cutover regressed: %+v", heldAfter)
	}
}

func TestRefreshLineupsPlayedVsOwnerCurrentWeek(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Lineups league", migrationtest.Specs()["8 teams"])
	// Simplify lineup requirements to 1 QB so an owner can legally submit a 1-player lineup in-app.
	export.Config.Starters = migrationtest.Synthetic("tmp", migrationtest.Specs()["8 teams"]).Config.Starters
	export.Config.Starters.Count = "1"
	export.Config.Starters.IOPStarters = "1"
	export.Config.Starters.IDPStarters = "0"
	export.Config.Starters.Positions = export.Config.Starters.Positions[:1] // QB: 1

	var err error
	export.Schedule, err = leagueschedule.Parse([]byte(`{"schedule":{"weeklySchedule":[
		{"week":"1","matchup":{"franchise":[{"id":"0001","isHome":"1","score":"100"},{"id":"0002","isHome":"0","score":"90"}]}},
		{"week":"2","matchup":{"franchise":[{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}}
	]}}`))
	if err != nil {
		t.Fatal(err)
	}
	// At initial load, week 1 is NOT yet finished (CutoverWeek 1).
	export.Results, err = weeklyresults.Parse([]byte(`{"allWeeklyResults":{"weeklyResults":[
		{"week":"1","matchup":{"regularSeason":"1","franchise":[{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}}
	]}}`))
	if err != nil {
		t.Fatal(err)
	}

	// Find two active players on franchise 0001 and one on franchise 0002.
	var f1p1, f1p2, f2p1 string
	for _, r := range export.Rosters {
		if r.Franchise == "0001" && r.Status == "ROSTER" {
			if f1p1 == "" {
				f1p1 = r.Player
			} else if f1p2 == "" {
				f1p2 = r.Player
			}
		}
		if r.Franchise == "0002" && r.Status == "ROSTER" && f2p1 == "" {
			f2p1 = r.Player
		}
	}
	facts := nativecheck.Facts{
		f1p1: {Name: "QB One", Position: leaguerules.QB, Team: "KC"},
		f1p2: {Name: "QB Two", Position: leaguerules.QB, Team: "BUF"},
		f2p1: {Name: "QB Three", Position: leaguerules.QB, Team: "SF"},
	}

	f := newRefreshFixture(t, export, facts)
	futureKickoff := f.now.Add(24 * time.Hour)
	f.store.SetLineupEnvironment(testLineupEnv{
		facts: facts,
		locks: map[int]map[string]time.Time{
			2: {"KC": futureKickoff, "BUF": futureKickoff, "SF": futureKickoff},
		},
	})

	// Seat owner1 on franchise.01 so they can submit an in-app lineup.
	f.exec(t, commissioner.RawCommand{
		Command: "commish.seat.replace", Request: "seat-owner1", Reason: "Seat owner 1",
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: "owner1@example.com"}},
	}, "creator", facts)
	// Owner 1 sets their week-2 (current week) lineup in-app to f1p2.
	f.exec(t, commissioner.RawCommand{
		Command: commissioner.LineupSet, Request: "owner-w2-lineup", Reason: "Owner sets week 2",
		Effective: "now", Visibility: "private",
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{
			Season: 2026, Week: 2, Franchise: "franchise.01", Starters: []string{f1p2},
		}},
	}, "owner1@example.com", facts)

	// Now MFL finishes week 1 (f1p1 started week 1 and was THEN dropped from MFL's current roster!),
	// and MFL liveScoring has week 2 starters for franchise.01 (f1p1? no, use another active player) and franchise.02 (f2p1).
	var f1p3 string
	for _, r := range export.Rosters {
		if r.Franchise == "0001" && r.Status == "ROSTER" && r.Player != f1p1 && r.Player != f1p2 {
			f1p3 = r.Player
			break
		}
	}
	next := f.src.Export
	// Drop f1p1 from MFL's current roster after he played in week 1.
	next.Rosters = slices.DeleteFunc(slices.Clone(export.Rosters), func(r mflsource.Rostered) bool {
		return r.Player == f1p1
	})
	next.Results = []leaguefeed.WeekResult{{
		Week: 1,
		Games: []leaguefeed.Game{{
			Regular: true,
			Home: leaguefeed.Side{
				Franchise: "0001", Scored: true, Points: 10500, Result: "W",
				Players: []leaguefeed.WeekPlayer{{ID: *mustPID(t, f1p1), Starter: true}},
			},
			Away: leaguefeed.Side{
				Franchise: "0002", Scored: true, Points: 9500, Result: "L",
				Players: []leaguefeed.WeekPlayer{{ID: *mustPID(t, f2p1), Starter: true}},
			},
		}},
	}}
	next.LiveLineups = &leaguefeed.Lineups{
		Week: 2,
		Franchises: []leaguefeed.Lineup{
			{Franchise: "0001", Starters: []playerid.PlayerID{*mustPID(t, f1p3)}},
			{Franchise: "0002", Starters: []playerid.PlayerID{*mustPID(t, f2p1)}},
		},
	}
	f.src.Export = next

	report, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Refresh week 1+2", Principal: "creator",
	})
	if err != nil {
		t.Fatalf("refresh lineups failed: %v", err)
	}
	// Played week 1 (2 franchises) + current week 2 franchise.02 (1 franchise) = 3 lineups synced.
	// Current week 2 franchise.01 MUST be skipped (owner set in-app).
	if report.Summary.Lineups != 3 || report.Summary.CutoverWeek != 2 {
		t.Fatalf("unexpected lineup summary: %+v", report.Summary)
	}
	recs := f.records(t)
	w1f1, ok1, err := commissioner.RecordedLineupFor(recs, "franchise.01", 2026, 1)
	if err != nil || !ok1 || !slices.Equal(w1f1.Starters, []string{f1p1}) {
		t.Fatalf("played week 1 lineup not synced (dropped starter): %+v %v", w1f1, err)
	}
	w2f1, ok2, err := commissioner.RecordedLineupFor(recs, "franchise.01", 2026, 2)
	if err != nil || !ok2 || w2f1.Corrected || !slices.Equal(w2f1.Starters, []string{f1p2}) {
		t.Fatalf("owner's current-week lineup was overwritten: %+v %v", w2f1, err)
	}
	w2f2, ok3, err := commissioner.RecordedLineupFor(recs, "franchise.02", 2026, 2)
	if err != nil || !ok3 || !slices.Equal(w2f2.Starters, []string{f2p1}) {
		t.Fatalf("unset franchise.02 week 2 lineup not synced: %+v %v", w2f2, err)
	}

	// Second refresh keeps owner's week 2 lineup and writes nothing.
	again, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Refresh again", Principal: "creator",
	})
	if err != nil || len(again.Commands) != 0 {
		t.Fatalf("second refresh wrote commands: %+v %v", again, err)
	}
}

func TestRefreshPreservesSeatsRolesRulesNotesAndInAppFranchiseEdits(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("32-team league", migrationtest.Specs()["32 teams"])
	f := newRefreshFixture(t, export, nil)

	// Seat commissioner on franchise.25, claim owner seat on franchise.02, grant co-commissioner + deputy,
	// edit a rule, post a league note, and edit franchise.02's name in-app.
	f.exec(t, commissioner.RawCommand{
		Command: "commish.seat.replace", Request: "seat-commish-25", Reason: "Seat commissioner on 25",
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.25", Principal: "creator"}},
	}, "creator", nil)
	f.exec(t, commissioner.RawCommand{
		Command: "commish.seat.invite", Request: "invite-f02", Reason: "Invite owner 2",
		Effective: "now", Visibility: "private",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{ID: "inv.02", Franchise: "franchise.02", Principal: "owner2@example.com"}},
	}, "creator", nil)
	f.exec(t, commissioner.RawCommand{
		Command: "commish.seat.invite.redeem", Request: "redeem-f02", Reason: "Claim seat",
		Effective: "now", Visibility: "private",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{ID: "inv.02", Principal: "owner2@example.com"}},
	}, "owner2@example.com", nil)
	f.exec(t, commissioner.RawCommand{
		Command: "commish.role.grant", Request: "grant-cocommish", Reason: "Co-commissioner",
		Effective: "now", Visibility: "private",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{Principal: "co@example.com", Role: governance.CoCommissioner}},
	}, "creator", nil)
	f.exec(t, commissioner.RawCommand{
		Command: "commish.role.grant", Request: "grant-deputy", Reason: "Deputy",
		Effective: "now", Visibility: "private",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{Principal: "dep@example.com", Role: governance.Deputy}},
	}, "creator", nil)
	f.exec(t, commissioner.RawCommand{
		Command: commissioner.RulesSet, Request: "edit-cap-rule", Reason: "Raise cap in app",
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Field: "cap.amount", Value: json.RawMessage(`25000000000`)},
	}, "creator", nil)
	f.exec(t, commissioner.RawCommand{
		Command: commissioner.NoticePublish, Request: "in-app-note", Reason: "League announcement",
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Note: &commissioner.RawNote{Text: "Season kickoff", Pin: true}},
	}, "creator", nil)
	f.exec(t, commissioner.RawCommand{
		Command: "commish.franchise.edit", Request: "in-app-rename-f02", Reason: "Owner custom name",
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.02", Name: "Server Custom Name", Abbreviation: "SCN"}},
	}, "creator", nil)

	// Mutate MFL: rename franchise 0001 (unedited) AND 0002 (edited in-app), and change a salary.
	next := f.src.Export
	next.Config.Franchises = slices.Clone(export.Config.Franchises)
	next.Config.Franchises[0].Name = "MFL Renamed One"
	next.Config.Franchises[0].Abbrev = "MR1"
	next.Config.Franchises[1].Name = "MFL Renamed Two"
	next.Config.Franchises[1].Abbrev = "MR2"
	next.Rosters = slices.Clone(export.Rosters)
	next.Rosters[0].Salary = "8.88"
	f.src.Export = next

	report, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Refresh governance test", Principal: "creator",
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Franchises != 1 {
		t.Fatalf("expected exactly 1 franchise renamed (unedited franchise.01), got %+v", report.Summary)
	}

	recs := f.records(t)
	state, err := commissioner.GovernanceState(recs)
	if err != nil {
		t.Fatal(err)
	}
	if !state.HoldsSeat("franchise.25", "creator") || !state.HoldsSeat("franchise.02", "owner2@example.com") {
		t.Fatal("claimed seats were not preserved")
	}
	if !state.HasRole("co@example.com", governance.CoCommissioner) || !state.HasRole("dep@example.com", governance.Deputy) {
		t.Fatal("roles/deputies were not preserved")
	}
	rules, err := commissioner.RulesNow(recs, f.now)
	if err != nil || rules.Cap.Amount.Value != 25000000000 {
		t.Fatalf("in-app rule edit was not preserved: %+v %v", rules.Cap.Amount, err)
	}
	notes, err := commissioner.NotesReading(f.league, recs)
	if err != nil || len(notes.Notices) != 1 || notes.Notices[0].Text != "Season kickoff" {
		t.Fatalf("in-app note was not preserved: %+v %v", notes, err)
	}
	dir, err := commissioner.NativeDirectoryReading(f.league, recs, f.now)
	if err != nil {
		t.Fatal(err)
	}
	if got := dir.Seats().Of("franchise.01"); got.Name != "MFL Renamed One" || got.Abbreviation != "MR1" {
		t.Fatalf("unedited franchise.01 did not take MFL rename: %+v", got)
	}
	if got := dir.Seats().Of("franchise.02"); got.Name != "Server Custom Name" || got.Abbreviation != "SCN" {
		t.Fatalf("in-app edited franchise.02 was overwritten: %+v", got)
	}

	// A second MFL rename of franchise.01 (previously renamed by refresh) MUST still take MFL's newer name.
	next2 := f.src.Export
	next2.Config.Franchises = slices.Clone(next.Config.Franchises)
	next2.Config.Franchises[0].Name = "MFL Renamed Again"
	f.src.Export = next2
	report2, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Second rename", Principal: "creator",
	})
	if err != nil || report2.Summary.Franchises != 1 {
		t.Fatalf("second MFL rename failed: %+v %v", report2, err)
	}
	dir2, err := commissioner.NativeDirectoryReading(f.league, f.records(t), f.now)
	if err != nil || dir2.Seats().Of("franchise.01").Name != "MFL Renamed Again" {
		t.Fatalf("second MFL rename not applied: %+v", dir2.Seats().Of("franchise.01"))
	}
}

func TestRefreshRefusesOpenInAppTradeOffers(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Trade refusal league", migrationtest.Specs()["8 teams"])
	f := newRefreshFixture(t, export, nil)

	// Enable trades (trade rules + open trade window) and seat two owners.
	for _, rule := range [][2]string{
		{"trades.review", `"commissioner"`},
		{"trades.decideAt", `1`},
		{"trades.futureYears", `3`},
		{"trades.rounds", `{"min":1,"max":5}`},
		{"trades.rosterGate", `false`},
		{"trades.offerExpiry", `172800000000000`},
		{"trades.maximumParties", `2`},
		{"cap.enforcement", `"hard"`},
		{"cap.quantum", `100000`},
		{"calendar.timezone", `"UTC"`},
		{"calendar.tradeDeadline", `"2026-12-01T00:00:00Z"`},
	} {
		f.exec(t, commissioner.RawCommand{
			Command: commissioner.RulesSet, Request: "rule-" + rule[0], Reason: "Configure trade rule",
			Effective: "now", Visibility: "league",
			Args: commissioner.RawArgs{Field: rule[0], Value: json.RawMessage(rule[1])},
		}, "creator", nil)
	}
	f.exec(t, commissioner.RawCommand{
		Command: commissioner.WindowSet, Request: "open-trades", Reason: "Open trade window",
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Window: &commissioner.RawWindowArgs{
			ID: "trades", Start: &commissioner.RawDate{Month: 1, Day: 1}, End: &commissioner.RawDate{Month: 12, Day: 31},
			Opens: []leaguerules.Action{"trade"},
		}},
	}, "creator", nil)
	f.exec(t, commissioner.RawCommand{
		Command: "commish.seat.replace", Request: "seat-1", Reason: "Seat 1",
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: "o1@example.com"}},
	}, "creator", nil)
	f.exec(t, commissioner.RawCommand{
		Command: "commish.seat.replace", Request: "seat-2", Reason: "Seat 2",
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.02", Principal: "o2@example.com"}},
	}, "creator", nil)

	// Offer a pick-for-pick trade between franchise.01 and franchise.02 using picks each actually owns.
	app, err := leaguemigration.ReadApp(f.league, f.records(t), f.now)
	if err != nil {
		t.Fatal(err)
	}
	var pick1, pick2 commissioner.NativePick
	for _, pk := range app.Snapshot.Picks {
		if pk.Season == 2027 && pk.Original != "" {
			if pk.Owner == "franchise.01" && pick1.Original == "" {
				pick1 = pk
			}
			if pk.Owner == "franchise.02" && pick2.Original == "" {
				pick2 = pk
			}
		}
	}
	offerReceipt := f.exec(t, commissioner.RawCommand{
		Command: commissioner.TradeOffer, Request: "trade-offer-1", Reason: "Offer trade",
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{
			Franchise: "franchise.01",
			Parties:   []string{"franchise.01", "franchise.02"},
			Assets: []commissioner.TradeAsset{
				{From: "franchise.01", To: "franchise.02", Season: pick1.Season, Round: pick1.Round, Original: pick1.Original},
				{From: "franchise.02", To: "franchise.01", Season: pick2.Season, Round: pick2.Round, Original: pick2.Original},
			},
		}},
	}, "o1@example.com", nil)

	// Mutate MFL so refresh would otherwise have work to do.
	next := f.src.Export
	next.Rosters = slices.Clone(export.Rosters)
	next.Rosters[0].Salary = "9.50"
	f.src.Export = next

	before := f.records(t)
	for _, dryRun := range []bool{true, false} {
		_, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
			MFLLeague: export.LeagueID, League: f.league, Reason: "Refresh blocked", Principal: "creator", DryRun: dryRun,
		})
		var openErr *leaguemigration.OpenTradesError
		if !errors.As(err, &openErr) || len(openErr.Offers) != 1 || openErr.Offers[0].ID != offerReceipt.Entry {
			t.Fatalf("dryRun=%t: expected OpenTradesError naming %s, got %v", dryRun, offerReceipt.Entry, err)
		}
		if !strings.Contains(err.Error(), offerReceipt.Entry) ||
			!strings.Contains(err.Error(), "franchise.01") ||
			!strings.Contains(err.Error(), "franchise.02") {
			t.Fatalf("error did not list offer id and franchises: %v", err)
		}
	}
	if !reflect.DeepEqual(before, f.records(t)) {
		t.Fatal("refused refresh modified the ledger")
	}

	// Withdraw the trade offer -> settled -> refresh now succeeds!
	f.exec(t, commissioner.RawCommand{
		Command: commissioner.TradeWithdraw, Request: "trade-withdraw-1", Reason: "Withdraw",
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offerReceipt.Entry, Franchise: "franchise.01"}},
	}, "o1@example.com", nil)

	report, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Refresh after settle", Principal: "creator",
	})
	if err != nil || report.Summary.Contracts != 1 {
		t.Fatalf("refresh after settling trade failed: report=%+v err=%v", report, err)
	}
}

func TestRefreshDryRunAndIdempotency(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("DryRun league", migrationtest.Specs()["8 teams"])
	f := newRefreshFixture(t, export, nil)

	next := f.src.Export
	next.Rosters = slices.Clone(export.Rosters)
	next.Rosters[0].Salary = "7.77"
	f.src.Export = next

	before := f.records(t)
	dry, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Dry run check", Principal: "creator", DryRun: true,
	})
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if !dry.DryRun || len(dry.Commands) != 1 || len(dry.Receipts) != 0 || dry.Summary.Contracts != 1 {
		t.Fatalf("unexpected dry run report: %+v", dry)
	}
	if !reflect.DeepEqual(before, f.records(t)) {
		t.Fatal("dry run wrote to the ledger")
	}

	// Real commit applies the planned batch.
	applied, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Apply refresh", Principal: "creator",
	})
	if err != nil || applied.DryRun || len(applied.Receipts) != 1 {
		t.Fatalf("apply failed: %+v %v", applied, err)
	}

	// Second run against the same MFL data is an idempotent no-op.
	afterCommit := f.records(t)
	second, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Second run", Principal: "creator",
	})
	if err != nil || len(second.Commands) != 0 || len(second.Receipts) != 0 || !reflect.DeepEqual(afterCommit, f.records(t)) {
		t.Fatalf("second run was not idempotent: %+v %v", second, err)
	}
}

type faultingLedger struct {
	leaguemigration.Ledger
	failBatch error
}

func (l faultingLedger) Batch(ctx context.Context, raws []commissioner.RawCommand, principal string) ([]commissioner.Receipt, error) {
	if l.failBatch != nil {
		return nil, l.failBatch
	}
	return l.Ledger.Batch(ctx, raws, principal)
}

func TestRefreshFaultLeavesLedgerUnchanged(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Fault league", migrationtest.Specs()["8 teams"])
	f := newRefreshFixture(t, export, nil)

	next := f.src.Export
	next.Rosters = slices.Clone(export.Rosters)
	next.Rosters[0].Salary = "6.50"
	f.src.Export = next

	before := f.records(t)
	f.svc.Ledger = faultingLedger{Ledger: f.ledger, failBatch: errors.New("simulated commit failure")}
	if _, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Faulty refresh", Principal: "creator",
	}); err == nil {
		t.Fatal("expected refresh error on batch fault")
	}
	if !reflect.DeepEqual(before, f.records(t)) {
		t.Fatal("failed refresh altered the ledger")
	}
}

func TestRefreshReplaysUnseenMFLTransactions(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Tx replay league", migrationtest.Specs()["8 teams"])
	f := newRefreshFixture(t, export, nil)

	// After initial load, MFL records a FREE_AGENT drop+add at txTime (after RosterAt, before now).
	txTime := f.now.Add(-5 * time.Minute)
	dropped := export.Rosters[0]
	next := f.src.Export
	next.Players = map[string]string{}
	for k, v := range export.Players {
		next.Players[k] = v
	}
	next.Players["9910"] = "Waiver Pickup"
	rosters := slices.Clone(export.Rosters[1:])
	rosters = append(rosters, mflsource.Rostered{
		Franchise: dropped.Franchise, Player: "9910", Status: "ROSTER",
		Salary: "2.00", ContractYear: "2028", ContractStatus: "UFA", ContractInfo: "2.0M / 2yr",
	})
	next.Rosters = rosters
	txJSON := `{"transactions":{"transaction":[{"type":"FREE_AGENT","franchise":"` + dropped.Franchise +
		`","timestamp":"` + jsonNumber(txTime.Unix()) + `","transaction":"9910,|` + dropped.Player + `,"}]}}`
	parsed, err := transactions.Parse([]byte(txJSON))
	if err != nil {
		t.Fatal(err)
	}
	next.Transactions = &parsed
	f.src.Export = next

	report, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Refresh tx", Principal: "creator",
	})
	if err != nil {
		t.Fatalf("refresh with transactions failed: %v", err)
	}
	if report.Summary.Transactions != 1 || report.Summary.Contracts != 1 {
		t.Fatalf("expected 1 replayed tx + 1 contract update, got %+v", report.Summary)
	}
	// Second run must be a 0-command no-op.
	again, err := f.svc.Refresh(t.Context(), leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID, League: f.league, Reason: "Refresh tx again", Principal: "creator",
	})
	if err != nil || len(again.Commands) != 0 {
		t.Fatalf("second run not idempotent: %+v %v", again, err)
	}
}

func jsonNumber(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
