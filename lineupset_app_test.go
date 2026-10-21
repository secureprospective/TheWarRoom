package main

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/store/moves"
)

func lineupDraftApp(t *testing.T) (*App, *seasonTransport, []string) {
	t.Helper()
	a, transport, _, _ := isolatedSeasonTestApp(t)
	lineupTestRules(t, a, false)
	ids := lineupTestSnapshot(t, a)
	now := time.Date(2026, 10, 7, 15, 0, 0, 0, time.UTC)
	a.core.MovesNow = func() time.Time { return now }
	a.core.MovesLog = func(string) {}
	a.core.MovesChanged = func(context.Context) {}
	a.core.Week = leagueweek.Week{Number: 5, Games: []leagueweek.Game{{Kickoff: now.Add(24 * time.Hour)}}}
	a.refreshSeasonLineups(a.core.Ctx)
	a.core.SeasonLineups.FetchedAt = now.Add(-time.Minute)
	transport.requests = nil
	return a, transport, ids
}

func changedStarters(ids []string) []string {
	changed := slices.Clone(ids)
	for i, id := range changed {
		if id == "15754" {
			changed[i] = "16387"
		}
	}
	return changed
}

func TestTargetCheckLineupLocal(t *testing.T) {
	t.Parallel()
	a, transport, ids := lineupDraftApp(t)
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	result, err := a.contracts().TargetCheckLineup("0025", ids)
	if err != nil || !result.Full || !result.Legal {
		t.Fatalf("full: %+v %v", result, err)
	}
	result, err = a.contracts().TargetCheckLineup("0025", append(slices.Clone(ids), "15252"))
	if err != nil || result.Legal {
		t.Fatalf("extra QB: %+v %v", result, err)
	}
	// Toggle checks only positions; roster status is blocked at draft, not during editing.
	ir := slices.Clone(ids)
	ir[20] = "17211"
	result, err = a.contracts().TargetCheckLineup("0025", ir)
	if err != nil || !result.Legal {
		t.Fatalf("IR position check: %+v %v", result, err)
	}
	for _, bad := range [][]string{{"bad"}, {"99", "0099"}} {
		if _, err := a.contracts().TargetCheckLineup("0025", bad); err == nil {
			t.Fatal("invalid selection accepted")
		}
	}
	if _, err := a.contracts().TargetCheckLineup("nope", ids); err == nil {
		t.Fatal("unknown franchise")
	}
	if len(transport.requests) != 0 {
		t.Fatal("local binding fetched")
	}
	listed, err := a.TargetMoves("0025")
	if err != nil || len(listed) != 0 {
		t.Fatalf("check saved: %+v %v", listed, err)
	}
	a.core.HasTargetSnapshot = false
	if _, err := a.contracts().TargetCheckLineup("0025", ids); err == nil {
		t.Fatal("no snapshot")
	}
}

func TestTargetCheckLineupRulesAndUnknown(t *testing.T) {
	t.Parallel()
	a, _, ids := lineupDraftApp(t)
	a.core.TargetSnapshot.Players.Value = nil
	result, err := a.contracts().TargetCheckLineup("0025", ids)
	if err != nil || result.Legal || result.Problems[0].Kind != "short" {
		t.Fatalf("unknown: %+v %v", result, err)
	}
	cfg := a.core.Rulebook.ActiveConfig()
	cfg.Starters.Count = "unreadable"
	if _, err := a.core.Rulebook.Sync(a.core.Ctx, cfg); err != nil {
		t.Fatal(err)
	}
	result, err = a.contracts().TargetCheckLineup("0025", ids)
	reading := mustLineup(t, a)
	if err != nil || len(result.Problems) != 1 || result.Problems[0] != reading.Check.Problems[0] {
		t.Fatalf("rules: %+v %v", result, err)
	}
	bad := NewApp()
	bad.core.StartupErr = errors.New("offline startup")
	close(bad.core.Started)
	if _, err := bad.contracts().TargetCheckLineup("0025", ids); err == nil {
		t.Fatal("not ready")
	}
}

func TestTargetDraftLineupLanding(t *testing.T) {
	t.Parallel()
	for _, outcome := range []struct {
		name  string
		state envelope.State
	}{
		{"drafted", envelope.Landed}, {"baseline", envelope.NotYetDone}, {"third", envelope.NotVerified},
	} {
		t.Run(outcome.name, func(t *testing.T) {
			t.Parallel()
			a, transport, ids := lineupDraftApp(t)
			drafted := changedStarters(ids)
			r, err := a.contracts().TargetDraftLineup("0025", drafted)
			if err != nil || r.State != envelope.Ready || r.Audit[0].Note != "legal" {
				t.Fatalf("draft: %+v %v", r, err)
			}
			const target = "https://www47.myfantasyleague.com/2026/lineup?L=14432&WEEK=5&F=0025"
			if err := validateMoveURL(target); err != nil {
				t.Fatal(err)
			}
			opens := 0
			a.core.OpenURL = func(ctx context.Context, got string) {
				opens++
				if got != target {
					t.Fatalf("URL: %s", got)
				}
				stored, err := a.core.Moves.Get(ctx, r.CorrelationID)
				if err != nil || stored.State() != envelope.HandedOff {
					t.Fatal("open before save", err)
				}
			}
			handed := a.core.MovesNow().Add(time.Minute)
			a.core.MovesNow = func() time.Time { return handed }
			if _, err := a.TargetHandOff(r.CorrelationID); err != nil {
				t.Fatal(err)
			}
			if opens != 1 {
				t.Fatal("missing hand-off")
			}
			saved := drafted
			if outcome.name == "baseline" {
				saved = ids
			}
			if outcome.name == "third" {
				saved = drafted[:1]
			}
			transport.bodies["liveScoring"] = lineupFeedBody(t, saved)
			observed := handed.Add(time.Minute)
			a.core.MovesNow = func() time.Time { return observed }
			a.core.MovesRefresh = func(ctx context.Context, source envelope.Source) error {
				if source != envelope.Lineups {
					t.Fatalf("wrong feed: %s", source)
				}
				a.refreshSeasonLineups(ctx)
				a.core.SeasonMu.Lock()
				a.core.SeasonLineups.FetchedAt = observed
				a.core.SeasonMu.Unlock()
				return nil
			}
			if err := a.checkMoves(a.core.Ctx); err != nil {
				t.Fatal(err)
			}
			stored, err := a.core.Moves.Get(a.core.Ctx, r.CorrelationID)
			if err != nil || stored.State() != outcome.state {
				t.Fatalf("observed: %s %v", stored.State(), err)
			}
			if len(transport.requests) != 1 || transport.requests[0] != "liveScoring" {
				t.Fatalf("watcher requests: %v", transport.requests)
			}
		})
	}
}

func lineupFeedBody(t *testing.T, ids []string) string {
	t.Helper()
	raw, err := livescoring.Parse([]byte(seasonFixture(t, "livescoring", "liveScoring-w5.json")))
	if err != nil {
		t.Fatal(err)
	}
	for i := range raw.Matchups {
		for j := range raw.Matchups[i].Franchises {
			f := &raw.Matchups[i].Franchises[j]
			if f.ID != "0025" {
				continue
			}
			f.Players.Player = nil
			for _, id := range ids {
				f.Players.Player = append(f.Players.Player, livescoring.RawPlayer{ID: id, Status: "starter"})
			}
		}
	}
	body, err := json.Marshal(struct {
		Live livescoring.RawLineups `json:"liveScoring"`
	}{Live: raw})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestTargetDraftLineupBlocks(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"same", "week", "baseline", "absent feed", "lock", "locked", "host", "IR", "rules",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a, transport, ids := lineupDraftApp(t)
			starters, note := changedStarters(ids), ""
			switch name {
			case "same":
				starters, note = ids, "already your saved week 5 lineup"
			case "week":
				a.core.SeasonLineups.Value.Week = 4
				note = "MFL's saved lineups are for week 4, the league is in week 5"
			case "baseline":
				a.core.SeasonLineups.RefreshError = "failed feed"
				note = "saved lineup is unknown"
			case "absent feed":
				a.core.SeasonLineups = heldSeasonFeed[leaguefeed.Lineups]{}
				note = "saved lineup is unknown"
			case "lock":
				a.core.Week.Games = nil
				note = "lineup lock unknown"
			case "locked":
				last := leagueweek.LastLock(a.core.Week)
				a.core.MovesNow = func() time.Time { return last }
				note = "week 5 lineups are locked"
			case "host":
				a.core.MFLClient = nil
				note = "MFL host not yet known"
			case "IR":
				starters[7] = "17211"
				note = "ROSTER-status"
			case "rules":
				cfg := a.core.Rulebook.ActiveConfig()
				cfg.Starters.Count = "unreadable"
				if _, err := a.core.Rulebook.Sync(a.core.Ctx, cfg); err != nil {
					t.Fatal(err)
				}
				note = "rules could not be read"
			}
			r, err := a.contracts().TargetDraftLineup("0025", starters)
			if err != nil || r.State != envelope.Blocked || !strings.Contains(r.Audit[0].Note, note) {
				t.Fatalf("blocked: %+v %v", r, err)
			}
			if (name == "baseline" || name == "absent feed") && len(r.Spec.Expected.Lineup.Baseline) != 0 {
				t.Fatal("unknown baseline stored as known")
			}
			if len(transport.requests) != 0 {
				t.Fatal("draft fetched")
			}
		})
	}
}

func TestLineupSupersedeLivePlansOnly(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"ready", "awaiting", "blocked", "unmapped"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			a, _, ids := lineupDraftApp(t)
			if kind == "unmapped" {
				a.core.MFLClient = nil
			}
			starters := changedStarters(ids)
			if kind == "blocked" {
				starters = ids
			}
			first, err := a.contracts().TargetDraftLineup("0025", starters)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "awaiting" {
				a.core.OpenURL = func(context.Context, string) {}
				if _, err := a.TargetHandOff(first.CorrelationID); err != nil {
					t.Fatal(err)
				}
			}
			second, err := a.contracts().TargetDraftLineup("0025", changedStarters(ids))
			if err != nil {
				t.Fatal(err)
			}
			old, err := a.core.Moves.Get(a.core.Ctx, first.CorrelationID)
			if kind == "blocked" || kind == "unmapped" {
				// Blocked plans cannot be handed off or land; they stay as drafted.
				if err != nil || old.State() != envelope.Blocked {
					t.Fatalf("inert plan changed: %+v %v", old, err)
				}
				return
			}
			if err != nil || old.State() != envelope.Stale {
				t.Fatalf("superseded: %+v %v", old, err)
			}
			audit := old.Receipt().Audit
			if audit[len(audit)-1].Note != "superseded by "+second.CorrelationID {
				t.Fatal(audit)
			}
		})
	}
}

func TestKnownEmptyLineupBaseline(t *testing.T) {
	t.Parallel()
	a, _, ids := lineupDraftApp(t)
	a.core.SeasonLineups.Value.Franchises = nil
	r, err := a.contracts().TargetDraftLineup("0025", ids)
	if err != nil || r.State != envelope.Ready || len(r.Spec.Expected.Lineup.Baseline) != 0 {
		t.Fatalf("known empty: %+v %v", r, err)
	}
}

func TestMoveSeasonFeedSkip(t *testing.T) {
	t.Parallel()
	a, _, ids := lineupDraftApp(t)
	r, err := a.contracts().TargetDraftLineup("0025", changedStarters(ids))
	if err != nil {
		t.Fatal(err)
	}
	a.core.OpenURL = func(context.Context, string) {}
	if _, err := a.TargetHandOff(r.CorrelationID); err != nil {
		t.Fatal(err)
	}
	handed := a.core.MovesNow()
	now := handed.Add(30 * time.Second)
	a.core.MovesNow = func() time.Time { return now }
	sources := map[envelope.Source]bool{
		envelope.Transactions: true, envelope.Lineups: true, envelope.PendingTrades: true, envelope.Rosters: true,
	}
	requests := []envelope.Source{}
	a.core.MovesRefresh = func(_ context.Context, s envelope.Source) error {
		requests = append(requests, s)
		return nil
	}
	for _, name := range []string{"after", "before", "failed", "old"} {
		at, failure := handed.Add(time.Second), ""
		if name == "before" {
			at = handed.Add(-time.Second)
		}
		if name == "failed" {
			failure = "failed"
		}
		if name == "old" {
			at = now.Add(-time.Minute)
		}
		a.core.SeasonTransactions.FetchedAt, a.core.SeasonTransactions.RefreshError = at, failure
		a.core.SeasonLineups.FetchedAt, a.core.SeasonLineups.RefreshError = at, failure
		a.core.SeasonPendingTrades.FetchedAt, a.core.SeasonPendingTrades.RefreshError = at, failure
		requests = nil
		if err := a.refreshMoveSources(a.core.Ctx, sources); err != nil {
			t.Fatal(err)
		}
		want := 4
		if name == "after" {
			want = 1
		}
		if len(requests) != want || requests[0] != envelope.Rosters {
			t.Fatalf("%s: %v", name, requests)
		}
	}
	later := constructedMove(t, handed.Add(15*time.Second), moveSpec(t), false)
	later, err = later.HandOff(handed.Add(20 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	saveMove(t, a, later)
	at := handed.Add(10 * time.Second)
	a.core.SeasonTransactions.FetchedAt, a.core.SeasonTransactions.RefreshError = at, ""
	a.core.SeasonLineups.FetchedAt, a.core.SeasonLineups.RefreshError = at, ""
	a.core.SeasonPendingTrades.FetchedAt, a.core.SeasonPendingTrades.RefreshError = at, ""
	requests = nil
	if err := a.refreshMoveSources(a.core.Ctx, sources); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 4 {
		t.Fatalf("feed predating latest hand-off skipped: %v", requests)
	}

}

func TestSupersedeSavedBeforeNewEnvelope(t *testing.T) {
	t.Parallel()
	a, _, ids := lineupDraftApp(t)
	first, err := a.contracts().TargetDraftLineup("0025", changedStarters(ids))
	if err != nil {
		t.Fatal(err)
	}
	now := a.core.MovesNow()
	req, check := contractsapp.LineupPlan(a.contracts(), "0025", first.Spec.Subject.Players, now)
	next, err := envelope.DraftLineup(now, req, a.core.TargetSnapshot, check, func() string { return "next" })
	if err != nil {
		t.Fatal(err)
	}
	a.core.MovesMu.Lock()
	err = contractsapp.SupersedeMoves(a.core.Ctx, a.core.Moves, next, now)
	a.core.MovesMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	previous, err := a.core.Moves.Get(a.core.Ctx, first.CorrelationID)
	if err != nil || previous.State() != envelope.Stale {
		t.Fatalf("old: %s %v", previous.State(), err)
	}
	if _, err := a.core.Moves.Get(a.core.Ctx, next.ID()); !errors.Is(err, moves.ErrNotFound) {
		t.Fatalf("new exists before old saved: %v", err)
	}
}
