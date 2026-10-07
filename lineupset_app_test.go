package main

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/store/moves"
)

func lineupDraftApp(t *testing.T) (*App, *seasonTransport, []string) {
	t.Helper()
	a, transport, _, _ := seasonTestApp(t)
	lineupTestRules(t, a, false)
	ids := lineupTestSnapshot(t, a)
	now := time.Date(2026, 10, 7, 15, 0, 0, 0, time.UTC)
	a.movesNow = func() time.Time { return now }
	a.movesLog = func(string) {}
	a.movesChanged = func(context.Context) {}
	a.week = leagueweek.Week{Number: 5, Games: []leagueweek.Game{{Kickoff: now.Add(24 * time.Hour)}}}
	a.refreshSeasonLineups(a.ctx)
	a.seasonLineups.fetchedAt = now.Add(-time.Minute)
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
	a, transport, ids := lineupDraftApp(t)
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	result, err := a.TargetCheckLineup("0025", ids)
	if err != nil || !result.Full || !result.Legal {
		t.Fatalf("full: %+v %v", result, err)
	}
	result, err = a.TargetCheckLineup("0025", append(slices.Clone(ids), "15252"))
	if err != nil || result.Legal {
		t.Fatalf("extra QB: %+v %v", result, err)
	}
	// Toggle checks only positions; roster status is blocked at draft, not during editing.
	ir := slices.Clone(ids)
	ir[20] = "17211"
	result, err = a.TargetCheckLineup("0025", ir)
	if err != nil || !result.Legal {
		t.Fatalf("IR position check: %+v %v", result, err)
	}
	for _, bad := range [][]string{{"bad"}, {"99", "0099"}} {
		if _, err := a.TargetCheckLineup("0025", bad); err == nil {
			t.Fatal("invalid selection accepted")
		}
	}
	if _, err := a.TargetCheckLineup("nope", ids); err == nil {
		t.Fatal("unknown franchise")
	}
	if len(transport.requests) != 0 {
		t.Fatal("local binding fetched")
	}
	listed, err := a.TargetMoves("0025")
	if err != nil || len(listed) != 0 {
		t.Fatalf("check saved: %+v %v", listed, err)
	}
	a.hasTargetSnapshot = false
	if _, err := a.TargetCheckLineup("0025", ids); err == nil {
		t.Fatal("no snapshot")
	}
}

func TestTargetCheckLineupRulesAndUnknown(t *testing.T) {
	a, _, ids := lineupDraftApp(t)
	a.targetSnapshot.Players.Value = nil
	result, err := a.TargetCheckLineup("0025", ids)
	if err != nil || result.Legal || result.Problems[0].Kind != "short" {
		t.Fatalf("unknown: %+v %v", result, err)
	}
	cfg := a.rulebook.ActiveConfig()
	cfg.Starters.Count = "unreadable"
	if _, err := a.rulebook.Sync(a.ctx, cfg); err != nil {
		t.Fatal(err)
	}
	result, err = a.TargetCheckLineup("0025", ids)
	reading := mustLineup(t, a)
	if err != nil || len(result.Problems) != 1 || result.Problems[0] != reading.Check.Problems[0] {
		t.Fatalf("rules: %+v %v", result, err)
	}
	bad := NewApp()
	bad.startupErr = errors.New("offline startup")
	close(bad.started)
	if _, err := bad.TargetCheckLineup("0025", ids); err == nil {
		t.Fatal("not ready")
	}
}

func TestTargetDraftLineupLanding(t *testing.T) {
	for _, outcome := range []struct {
		name  string
		state envelope.State
	}{
		{"drafted", envelope.Landed}, {"baseline", envelope.NotYetDone}, {"third", envelope.NotVerified},
	} {
		t.Run(outcome.name, func(t *testing.T) {
			a, transport, ids := lineupDraftApp(t)
			drafted := changedStarters(ids)
			r, err := a.TargetDraftLineup("0025", drafted)
			if err != nil || r.State != envelope.Ready || r.Audit[0].Note != "legal" {
				t.Fatalf("draft: %+v %v", r, err)
			}
			const target = "https://www47.myfantasyleague.com/2026/lineup?L=14432&WEEK=5&F=0025"
			if err := validateMoveURL(target); err != nil {
				t.Fatal(err)
			}
			opens := 0
			a.openURL = func(ctx context.Context, got string) {
				opens++
				if got != target {
					t.Fatalf("URL: %s", got)
				}
				stored, err := a.moves.Get(ctx, r.CorrelationID)
				if err != nil || stored.State() != envelope.HandedOff {
					t.Fatal("open before save", err)
				}
			}
			handed := a.movesNow().Add(time.Minute)
			a.movesNow = func() time.Time { return handed }
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
			a.movesNow = func() time.Time { return observed }
			a.movesRefresh = func(ctx context.Context, source envelope.Source) error {
				if source != envelope.Lineups {
					t.Fatalf("wrong feed: %s", source)
				}
				a.refreshSeasonLineups(ctx)
				a.seasonMu.Lock()
				a.seasonLineups.fetchedAt = observed
				a.seasonMu.Unlock()
				return nil
			}
			if err := a.checkMoves(a.ctx); err != nil {
				t.Fatal(err)
			}
			stored, err := a.moves.Get(a.ctx, r.CorrelationID)
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
	for _, name := range []string{
		"same", "week", "baseline", "absent feed", "lock", "locked", "host", "IR", "rules",
	} {
		t.Run(name, func(t *testing.T) {
			a, transport, ids := lineupDraftApp(t)
			starters, note := changedStarters(ids), ""
			switch name {
			case "same":
				starters, note = ids, "already your saved week 5 lineup"
			case "week":
				a.seasonLineups.value.Week = 4
				note = "MFL's saved lineups are for week 4, the league is in week 5"
			case "baseline":
				a.seasonLineups.refreshError = "failed feed"
				note = "saved lineup is unknown"
			case "absent feed":
				a.seasonLineups = heldSeasonFeed[leaguefeed.Lineups]{}
				note = "saved lineup is unknown"
			case "lock":
				a.week.Games = nil
				note = "lineup lock unknown"
			case "locked":
				last := leagueweek.LastLock(a.week)
				a.movesNow = func() time.Time { return last }
				note = "week 5 lineups are locked"
			case "host":
				a.mflClient = nil
				note = "MFL host not yet known"
			case "IR":
				starters[7] = "17211"
				note = "ROSTER-status"
			case "rules":
				cfg := a.rulebook.ActiveConfig()
				cfg.Starters.Count = "unreadable"
				if _, err := a.rulebook.Sync(a.ctx, cfg); err != nil {
					t.Fatal(err)
				}
				note = "rules could not be read"
			}
			r, err := a.TargetDraftLineup("0025", starters)
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
	for _, kind := range []string{"ready", "awaiting", "blocked", "unmapped"} {
		t.Run(kind, func(t *testing.T) {
			a, _, ids := lineupDraftApp(t)
			if kind == "unmapped" {
				a.mflClient = nil
			}
			starters := changedStarters(ids)
			if kind == "blocked" {
				starters = ids
			}
			first, err := a.TargetDraftLineup("0025", starters)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "awaiting" {
				a.openURL = func(context.Context, string) {}
				if _, err := a.TargetHandOff(first.CorrelationID); err != nil {
					t.Fatal(err)
				}
			}
			second, err := a.TargetDraftLineup("0025", changedStarters(ids))
			if err != nil {
				t.Fatal(err)
			}
			old, err := a.moves.Get(a.ctx, first.CorrelationID)
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
	a, _, ids := lineupDraftApp(t)
	a.seasonLineups.value.Franchises = nil
	r, err := a.TargetDraftLineup("0025", ids)
	if err != nil || r.State != envelope.Ready || len(r.Spec.Expected.Lineup.Baseline) != 0 {
		t.Fatalf("known empty: %+v %v", r, err)
	}
}

func TestMoveSeasonFeedSkip(t *testing.T) {
	a, _, ids := lineupDraftApp(t)
	r, err := a.TargetDraftLineup("0025", changedStarters(ids))
	if err != nil {
		t.Fatal(err)
	}
	a.openURL = func(context.Context, string) {}
	if _, err := a.TargetHandOff(r.CorrelationID); err != nil {
		t.Fatal(err)
	}
	handed := a.movesNow()
	now := handed.Add(30 * time.Second)
	a.movesNow = func() time.Time { return now }
	sources := map[envelope.Source]bool{
		envelope.Transactions: true, envelope.Lineups: true, envelope.PendingTrades: true, envelope.Rosters: true,
	}
	requests := []envelope.Source{}
	a.movesRefresh = func(_ context.Context, s envelope.Source) error {
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
		a.seasonTransactions.fetchedAt, a.seasonTransactions.refreshError = at, failure
		a.seasonLineups.fetchedAt, a.seasonLineups.refreshError = at, failure
		a.seasonPendingTrades.fetchedAt, a.seasonPendingTrades.refreshError = at, failure
		requests = nil
		if err := a.refreshMoveSources(a.ctx, sources); err != nil {
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
	a.seasonTransactions.fetchedAt, a.seasonTransactions.refreshError = at, ""
	a.seasonLineups.fetchedAt, a.seasonLineups.refreshError = at, ""
	a.seasonPendingTrades.fetchedAt, a.seasonPendingTrades.refreshError = at, ""
	requests = nil
	if err := a.refreshMoveSources(a.ctx, sources); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 4 {
		t.Fatalf("feed predating latest hand-off skipped: %v", requests)
	}

}

func TestSupersedeSavedBeforeNewEnvelope(t *testing.T) {
	a, _, ids := lineupDraftApp(t)
	first, err := a.TargetDraftLineup("0025", changedStarters(ids))
	if err != nil {
		t.Fatal(err)
	}
	now := a.movesNow()
	req, check := a.lineupRequest("0025", first.Spec.Subject.Players, now)
	next, err := envelope.DraftLineup(now, req, a.targetSnapshot, check, func() string { return "next" })
	if err != nil {
		t.Fatal(err)
	}
	a.movesMu.Lock()
	err = a.supersedeLineups(a.ctx, next, now)
	a.movesMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	previous, err := a.moves.Get(a.ctx, first.CorrelationID)
	if err != nil || previous.State() != envelope.Stale {
		t.Fatalf("old: %s %v", previous.State(), err)
	}
	if _, err := a.moves.Get(a.ctx, next.ID()); !errors.Is(err, moves.ErrNotFound) {
		t.Fatalf("new exists before old saved: %v", err)
	}
}
