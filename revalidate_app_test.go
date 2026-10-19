package main

import (
	"context"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/eligibility"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// readyIRPlan stores a Ready roster.ir plan for player 0042, checked against a snapshot whose
// held eligibility passes, so revalidation has exactly the plan under test.
func readyIRPlan(t *testing.T, a *App, at time.Time) envelope.Envelope {
	t.Helper()
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	spec := envelope.Spec{
		Intent: "roster.ir", LeagueID: ingestion.LeagueID, FranchiseID: "0001",
		Subject:  envelope.Subject{Players: []playerid.PlayerID{id}},
		Expected: envelope.ExpectedChange{Player: id, RosterStatus: domain.RosterIR},
		Gravity:  envelope.G2, Undo: envelope.Reversible,
		Target: envelope.Target{Kind: envelope.Mapped,
			URL: "https://www47.myfantasyleague.com/2026/options?L=14432&O=18"},
	}
	e, err := envelope.New(at, spec, func() string { return "revalidate-test" })
	if err != nil {
		t.Fatal(err)
	}
	fresh := domain.Freshness{State: domain.FreshLive, FetchedAt: at.Format(time.RFC3339)}
	snap := snapshot.Snapshot{
		Rosters: snapshot.Sourced[[]snapshot.Roster]{Value: []snapshot.Roster{{
			FranchiseID: "0001", Players: []snapshot.RosterPlayer{{ID: id, RosterStatus: domain.RosterActive}},
		}}, Provenance: snapshot.Provenance{Freshness: fresh}},
		Eligibility: snapshot.Eligibility{
			Players: map[string]eligibility.PlayerVerdicts{id.String(): {
				IR: eligibility.Decision{Direction: eligibility.ToIR, Verdict: eligibility.Eligible,
					Source: eligibility.SourceInjuries},
			}},
			Feeds: map[string]snapshot.Provenance{
				eligibility.SourceRosters:  {Freshness: fresh},
				eligibility.SourceInjuries: {Freshness: fresh},
				eligibility.SourcePlayers:  {Freshness: fresh},
			},
		},
	}
	ready, err := e.Check(at, snap, envelope.IRCheck{})
	if err != nil || ready.State() != envelope.Ready {
		t.Fatalf("plan did not draft ready: %+v %v", ready.Receipt(), err)
	}
	if err := a.moves.Save(a.ctx, ready); err != nil {
		t.Fatal(err)
	}
	return ready
}

func TestRevalidateReadyRosterMovesOnIneligibility(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	targetOfflineClient(t, a)
	events := 0
	a.movesChanged = func(context.Context) { events++ }
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	a.movesNow = func() time.Time { return now }

	// The held injuries feed is live but no longer designates the player, so the fresh
	// verdict is ineligible and the Ready plan returns to Draft through Invalidate and Rebase.
	ready := readyIRPlan(t, a, now)
	other, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	a.seasonMu.Lock()
	a.seasonInjuries = heldSeasonFeed[[]leaguefeed.Injury]{
		value: []leaguefeed.Injury{{ID: other, Status: "Out"}}, fetchedAt: now.Add(-time.Minute),
	}
	a.seasonMu.Unlock()
	a.revalidateReadyRosterMoves(a.ctx)
	stored, err := a.moves.Get(a.ctx, ready.ID())
	if err != nil {
		t.Fatal(err)
	}
	if stored.State() != envelope.Draft {
		t.Fatalf("state = %s, want draft after revalidation", stored.State())
	}
	reason := "player no longer eligible: not on NFL injured reserve"
	audit := stored.Receipt().Audit
	if len(audit) != 3 || audit[1].From != envelope.Ready || audit[1].To != envelope.Stale ||
		audit[2].From != envelope.Stale || audit[2].To != envelope.Draft ||
		audit[1].Note != reason || audit[2].Note != reason {
		t.Fatalf("audit: %+v", audit)
	}
	if events != 1 {
		t.Fatalf("moves events = %d, want one revalidation signal", events)
	}
}

func TestRevalidateKeepsPlansWhenEligibilityUnknown(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	targetOfflineClient(t, a)
	a.movesChanged = func(context.Context) { t.Fatal("no event expected") }
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	a.movesNow = func() time.Time { return now }
	ready := readyIRPlan(t, a, now)

	// A stale injuries feed reads unknown; unknown never yanks a plan — MFL decides.
	a.seasonMu.Lock()
	a.seasonInjuries = heldSeasonFeed[[]leaguefeed.Injury]{
		fetchedAt: now.Add(-time.Hour), refreshError: "live fetch failed, showing last known data: offline",
	}
	a.seasonMu.Unlock()
	a.revalidateReadyRosterMoves(a.ctx)
	stored, err := a.moves.Get(a.ctx, ready.ID())
	if err != nil {
		t.Fatal(err)
	}
	if stored.State() != envelope.Ready {
		t.Fatalf("stale input must not yank a Ready plan: %+v", stored.Receipt())
	}

	// The taxi direction follows its own slot: a taxi plan yanks on the taxi verdict.
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	a.seasonMu.Lock()
	a.seasonInjuries = heldSeasonFeed[[]leaguefeed.Injury]{}
	a.seasonMu.Unlock()
	taxiSpec := envelope.Spec{
		Intent: "roster.taxi", LeagueID: ingestion.LeagueID, FranchiseID: "0001",
		Subject:  envelope.Subject{Players: []playerid.PlayerID{id}},
		Expected: envelope.ExpectedChange{Player: id, RosterStatus: domain.RosterTaxi},
		Gravity:  envelope.G2, Undo: envelope.Reversible,
		Target: envelope.Target{Kind: envelope.Mapped,
			URL: "https://www47.myfantasyleague.com/2026/options?L=14432&O=98"},
	}
	e, err := envelope.New(now, taxiSpec, func() string { return "revalidate-taxi" })
	if err != nil {
		t.Fatal(err)
	}
	if err := a.moves.Save(a.ctx, e); err != nil {
		t.Fatal(err)
	}
	// An empty players directory leaves the taxi verdict unknown, so the plan stays Ready.
	a.revalidateReadyRosterMoves(a.ctx)
	storedTaxi, err := a.moves.Get(a.ctx, e.ID())
	if err != nil {
		t.Fatal(err)
	}
	if storedTaxi.State() != envelope.Draft {
		t.Fatalf("a Draft taxi plan must not be revalidated: %+v", storedTaxi.Receipt())
	}
}
