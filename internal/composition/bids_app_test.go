package composition

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/offers/offerstest"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type capSource = offerstest.CapSource

func poolLookup(t *testing.T, extra ...players.RawPlayer) normalize.Lookup {
	return offerstest.Lookup(t, extra...)
}
func bidTestApp(t *testing.T) *App {
	t.Helper()
	a := targetTestApp(t)
	bidSetup(t, a)
	return a
}

func bidSetup(t *testing.T, a *App) {
	t.Helper()
	if err := a.core.Rulebook.Initialize(a.core.Ctx, capSource{}); err != nil {
		t.Fatal(err)
	}
	setRosters(t, a)
	a.core.Lookup, a.core.HasLookup, a.core.LookupAt = poolLookup(t), true, time.Now()
	a.core.Week, a.core.WeekFetchedAt = leagueweek.Week{Number: 5}, time.Now()
	a.core.MovesChanged = func(context.Context) {}
	a.core.MovesLog = func(string) {}
}

// setRosters replaces the mirror: franchise 0001 holds 0042 ($20M) and extra players on top.
func setRosters(t *testing.T, a *App, extra ...state.MirrorPlayer) {
	offerstest.SetRosters(a.core.Ctx, t, a.core.League, extra...)
}
func rawBid(salary string, years int) RawBidInput {
	return RawBidInput{FranchiseID: "0001", Player: "13593", Salary: salary, Years: years}
}

func ruleWords(plan BidPlan, rule string) string {
	for _, r := range plan.Rules {
		if r.Rule == rule {
			return r.Mark + ": " + r.Reason
		}
	}
	return ""
}

func TestBidDraftSavesPlanAndFactsAndLeavesTheMoveRailAlone(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	events := 0
	a.core.MovesChanged = func(context.Context) { events++ }
	raw := rawBid("5", 3)
	raw.Snipe, raw.Beat, raw.Evidence = true, "5.500", "https://example.invalid/thread"
	plan, err := offersTestAPI(a).TargetBidDraft(raw)
	if err != nil || plan.Receipt.State != envelope.Ready || events != 1 {
		t.Fatalf("draft = %+v, %v, events %d", plan.Receipt, err, events)
	}
	if plan.State != "Ready" || plan.Points != "7.000" || plan.BeatText != "Bid to beat 5.500 · Snipe" ||
		!strings.HasSuffix(plan.Post, "\nSnipe") || plan.Receipt.Spec.FranchiseID != "0001" {
		t.Errorf("plan words = %+v", plan)
	}
	if plan.Contract != "3 years at $5,000,000 a year" || plan.Form.Salary != "5" || plan.Form.Beat != "5.500" {
		t.Errorf("form = %+v", plan.Form)
	}
	ids := map[string]bool{}
	for _, action := range plan.Actions {
		ids[action.ID] = !action.Disabled
	}
	if !ids["hand-off"] || !ids["revise"] || !ids["withdraw"] || ids["posted"] {
		t.Errorf("actions = %+v", plan.Actions)
	}
	facts, err := a.core.Moves.BidFacts(a.core.Ctx, plan.Receipt.CorrelationID)
	if err != nil || facts.Evidence != raw.Evidence || facts.Beat == nil || *facts.Beat != 5500 || !facts.Snipe {
		t.Fatalf("facts = %+v, %v", facts, err)
	}
	moves, err := a.platformAPI().TargetMoves("0001")
	if err != nil || len(moves) != 0 {
		t.Errorf("a bid plan appeared on the move rail: %v, %v", moves, err)
	}
	board := offersTestAPI(a).TargetBidPlans("0001")
	if board.Status != ReadingOK || board.Count != 1 || board.Value.Summary != "1 open of 1 bid plans" {
		t.Errorf("board = %+v", board)
	}
	if other := offersTestAPI(a).TargetBidPlans("0002"); other.Count != 0 {
		t.Errorf("another franchise's board = %+v", other)
	}
	if got := ruleWords(plan, "increment"); !strings.HasPrefix(got, "OK: 7.000 beats 5.500 by a full point (Snipe)") {
		t.Errorf("increment = %q", got)
	}
}

func landedBid(t *testing.T, closes string, extra ...state.MirrorPlayer) BidPlan {
	t.Helper()
	a := bidTestApp(t)
	past := time.Now().Add(-time.Hour)
	a.core.MovesNow = func() time.Time { return past }
	raw := rawBid("5", 3)
	raw.Closes = closes
	plan, err := offersTestAPI(a).TargetBidDraft(raw)
	if err != nil {
		t.Fatal(err)
	}
	id := plan.Receipt.CorrelationID
	if _, err := offersTestAPI(a).TargetBidHandOff(id); err != nil {
		t.Fatal(err)
	}
	if _, err := offersTestAPI(a).TargetBidPosted(id); err != nil {
		t.Fatal(err)
	}
	a.core.MovesNow = func() time.Time { return time.Now().Add(time.Minute) }
	a.core.MovesRefresh = func(context.Context, envelope.Source) error { setRosters(t, a, extra...); return nil }
	if err := a.checkMoves(a.core.Ctx); err != nil {
		t.Fatal(err)
	}
	board := offersTestAPI(a).TargetBidPlans("0001")
	if board.Count != 1 {
		t.Fatalf("board = %+v", board)
	}
	return board.Value.Plans[0]
}

func TestBidWatcherLandsAMatchingContractFromTheMirror(t *testing.T) {
	t.Parallel()
	// Expiring in 2028 of a 2026 mirror is three years remaining.
	plan := landedBid(t, "", state.MirrorPlayer{MFLID: "13593", FranchiseID: "0001",
		RosterStatus: domain.RosterActive, Salary: 500_000_000, ContractYear: 2028})
	if plan.Receipt.State != envelope.Landed || plan.State != "Landed" ||
		!strings.Contains(plan.Status, "Landed: on your roster at $5,000,000 for 3 years") {
		t.Fatalf("plan = %v %q", plan.Receipt.State, plan.Status)
	}
}

func TestBidWatcherShowsAContractMismatchPlainly(t *testing.T) {
	t.Parallel()
	plan := landedBid(t, "", state.MirrorPlayer{MFLID: "13593", FranchiseID: "0001",
		RosterStatus: domain.RosterActive, Salary: 450_000_000, ContractYear: 2027})
	if plan.Receipt.State != envelope.NotVerified || plan.State != "Not verified" ||
		!strings.Contains(plan.Status, "Contract mismatch: on your roster at $4,500,000 for 2 years, not the planned $5,000,000 for 3 years") {
		t.Fatalf("plan = %v %q", plan.Receipt.State, plan.Status)
	}
}

func TestBidWatcherAwardedOffAppWhenAnotherRosterHoldsHim(t *testing.T) {
	t.Parallel()
	plan := landedBid(t, "", state.MirrorPlayer{MFLID: "13593", FranchiseID: "0002",
		RosterStatus: domain.RosterActive, Salary: 600_000_000, ContractYear: 2028})
	if plan.Receipt.State != envelope.AwardedOffApp || plan.State != "Awarded off-app" ||
		strings.Contains(plan.Status, "0002") || len(plan.Actions) != 0 {
		t.Fatalf("plan = %v %q %+v", plan.Receipt.State, plan.Status, plan.Actions)
	}
}

func TestBidWatcherAwardedOffAppWhenTheWindowClosesAndNothingLanded(t *testing.T) {
	t.Parallel()
	plan := landedBid(t, time.Now().Add(-30*time.Minute).UTC().Format(time.RFC3339))
	if plan.Receipt.State != envelope.AwardedOffApp || !strings.Contains(plan.Status, "the bidding window closed") {
		t.Fatalf("plan = %v %q", plan.Receipt.State, plan.Status)
	}
	open := landedBid(t, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
	if open.Receipt.State != envelope.BidPending {
		t.Errorf("an open window ended the plan: %v", open.Receipt.State)
	}
}

func TestBidWireSerializerAndEventOrder(t *testing.T) {
	a := bidTestApp(t)
	api := offersTestAPI(a)
	var order []string
	a.core.MovesLog = func(string) { order = append(order, "log") }
	a.core.MovesChanged = func(ctx context.Context) {
		if a.core.MovesMu.TryLock() {
			a.core.MovesMu.Unlock()
			t.Error("event escaped move serializer")
		}
		rows, err := a.core.Moves.List(ctx, ingestion.LeagueID, "0001")
		if err != nil || len(rows) != 1 {
			t.Errorf("event before plan saved: %v %v", rows, err)
		}
		select {
		case <-a.core.MovesWake:
			t.Error("wake happened before event")
		default:
		}
		order = append(order, "event")
	}
	plan, err := api.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.TargetBidHandOff(plan.Receipt.CorrelationID); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(order, []string{"log", "event", "log", "event"}) {
		t.Fatal(order)
	}
	select {
	case <-a.core.MovesWake:
	default:
		t.Fatal("hand-off did not wake watcher after event")
	}
	if !a.core.MovesMu.TryLock() {
		t.Fatal("move did not release serializer")
	}
	a.core.MovesMu.Unlock()
}
