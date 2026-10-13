package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type capSource struct{}

func (capSource) Fetch(context.Context) (league.RawConfig, error) {
	return league.RawConfig{Source: "test", SalaryCapAmount: "125", RosterSize: "80", TaxiSquad: "8",
		ScoringRules: []league.PositionRuleSet{{Positions: "CB|S", Rules: []league.ScoringRule{
			{Event: "TK", Points: "*0.5", Range: "0-99"}}}},
		RosterLimits: []league.PositionLimit{{Name: "QB", Limit: "0-0"}}}, nil
}

// poolLookup is a small players directory: id 13593 is a 2022 draftee, 13594 has no draft year.
func poolLookup(t *testing.T, extra ...players.RawPlayer) normalize.Lookup {
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

// bidTestApp is an app with a $125M cap, a $20M roster, a players directory and NFL week 5.
func bidTestApp(t *testing.T) *App {
	t.Helper()
	a := targetTestApp(t)
	bidSetup(t, a)
	return a
}

func bidSetup(t *testing.T, a *App) {
	t.Helper()
	if err := a.rulebook.Initialize(a.ctx, capSource{}); err != nil {
		t.Fatal(err)
	}
	setRosters(t, a)
	a.lookup, a.hasLookup, a.lookupAt = poolLookup(t), true, time.Now()
	a.week, a.weekFetchedAt = leagueweek.Week{Number: 5}, time.Now()
	a.movesChanged = func(context.Context) {}
	a.movesLog = func(string) {}
}

// setRosters replaces the mirror: franchise 0001 holds 0042 ($20M) and extra players on top.
func setRosters(t *testing.T, a *App, extra ...state.MirrorPlayer) {
	t.Helper()
	mine := state.MirrorPlayer{MFLID: "0042", FranchiseID: "0001", RosterStatus: domain.RosterActive,
		Salary: 2_000_000_000, ContractYear: 2027}
	if _, err := a.league.Replace(a.ctx, state.MirrorSnapshot{Season: 2026,
		Players: append([]state.MirrorPlayer{mine}, extra...)}); err != nil {
		t.Fatal(err)
	}
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

func TestBidPreviewShowsGoFigures(t *testing.T) {
	a := bidTestApp(t)
	p := a.TargetBidPreview(rawBid("5", 3))
	if p.Problem != "" || p.Points != "7.000" || p.Salary.Cents != 500_000_000 || p.Blocked ||
		p.Player != "Smith, John (WR, DAL)" || p.Snapped != "" || len(p.Rules) != 11 {
		t.Fatalf("preview = %+v", p)
	}
	if want := "Player: Smith, John (WR, DAL)\nContract: 3 years at $5,000,000 a year, flat\n" +
		"Cap hit: $5,000,000 a year\nPoints: 7.000"; p.Post != want {
		t.Errorf("post text:\n%s", p.Post)
	}
	snapped := a.TargetBidPreview(rawBid("5.004", 1))
	if snapped.Salary.Cents != 500_000_000 || !strings.Contains(snapped.Snapped, "Entered $5,004,000") ||
		snapped.Points != "5.000" {
		t.Errorf("snapped preview = %+v", snapped)
	}
	long := a.TargetBidPreview(rawBid("5", 5))
	if !long.Blocked || long.Points != "Not scored" || !strings.Contains(long.Note, "A bid runs 1 to 4 years, not 5") {
		t.Errorf("5-year preview = %+v", long)
	}
	for _, bad := range []RawBidInput{rawBid("", 1), rawBid("abc", 1), {FranchiseID: "1", Player: "13593", Salary: "5", Years: 1},
		{FranchiseID: "0001", Player: "99999", Salary: "5", Years: 1},
		{FranchiseID: "0001", Player: "13593", Salary: "5", Years: 1, Beat: "6.5551"},
		{FranchiseID: "0001", Player: "13593", Salary: "5", Years: 1, Closes: "tomorrow"}} {
		if got := a.TargetBidPreview(bad); got.Problem == "" || got.Rules == nil {
			t.Errorf("bad input %+v gave %+v", bad, got)
		}
	}
}

func TestBidPreviewMinimumAndCapRoom(t *testing.T) {
	a := bidTestApp(t)
	// 13594 has no draft year: the rookie minimum applies, and the preview says why.
	low := a.TargetBidPreview(RawBidInput{FranchiseID: "0001", Player: "13594", Salary: "0.32", Years: 1})
	if !low.Blocked || !strings.Contains(low.Note, "Below the $330,000 minimum salary (no draft year on file") {
		t.Errorf("rookie minimum: %+v", low)
	}
	// 13595 has 16 years: $630,000.
	vet := a.TargetBidPreview(RawBidInput{FranchiseID: "0001", Player: "13595", Salary: "0.6", Years: 1})
	if !vet.Blocked || !strings.Contains(vet.Note, "Below the $630,000 minimum") {
		t.Errorf("veteran minimum: %+v", vet)
	}
	room := a.TargetBidPreview(rawBid("5", 2))
	last := room.Rules[len(room.Rules)-1]
	if last.Rule != "cap-room" || !strings.Contains(last.Reason, "$5,000,000 a year against $105,000,000 of room leaves $100,000,000") {
		t.Errorf("cap room = %+v", last)
	}
	over := a.TargetBidPreview(rawBid("110", 3))
	if !over.Blocked || !strings.Contains(over.Note, "room") {
		t.Errorf("cap room did not block the plan: %s", over.Note)
	}
}

func TestBidDraftSavesPlanAndFactsAndLeavesTheMoveRailAlone(t *testing.T) {
	a := bidTestApp(t)
	events := 0
	a.movesChanged = func(context.Context) { events++ }
	raw := rawBid("5", 3)
	raw.Snipe, raw.Beat, raw.Evidence = true, "5.500", "https://example.invalid/thread"
	plan, err := a.TargetBidDraft(raw)
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
	facts, err := a.moves.BidFacts(a.ctx, plan.Receipt.CorrelationID)
	if err != nil || facts.Evidence != raw.Evidence || facts.Beat == nil || *facts.Beat != 5500 || !facts.Snipe {
		t.Fatalf("facts = %+v, %v", facts, err)
	}
	moves, err := a.TargetMoves("0001")
	if err != nil || len(moves) != 0 {
		t.Errorf("a bid plan appeared on the move rail: %v, %v", moves, err)
	}
	board := a.TargetBidPlans("0001")
	if board.Status != ReadingOK || board.Count != 1 || board.Value.Summary != "1 open of 1 bid plans" {
		t.Errorf("board = %+v", board)
	}
	if other := a.TargetBidPlans("0002"); other.Count != 0 {
		t.Errorf("another franchise's board = %+v", other)
	}
	if got := ruleWords(plan, "increment"); !strings.HasPrefix(got, "OK: 7.000 beats 5.500 by a full point (Snipe)") {
		t.Errorf("increment = %q", got)
	}
}

func TestBidBlockedByPlayoffStatusKeepsTheActOff(t *testing.T) {
	a := bidTestApp(t)
	a.week = leagueweek.Week{Number: 13}
	plan, err := a.TargetBidDraft(rawBid("5", 1))
	if err != nil || plan.Receipt.State != envelope.Blocked {
		t.Fatalf("draft = %v, %v", plan.Receipt.State, err)
	}
	if !strings.Contains(plan.Status, "this app cannot tell whether yours is one") {
		t.Errorf("status = %q", plan.Status)
	}
	if got := ruleWords(plan, "late-season"); !strings.HasPrefix(got, "Can't tell:") {
		t.Errorf("late-season = %q", got)
	}
	var handOff BidAction
	for _, action := range plan.Actions {
		if action.ID == "hand-off" {
			handOff = action
		}
	}
	if !handOff.Disabled || !strings.Contains(handOff.Reason, "cannot tell whether yours is one") {
		t.Errorf("hand-off action = %+v", handOff)
	}
	if _, err := a.TargetBidHandOff(plan.Receipt.CorrelationID); err == nil {
		t.Error("a blocked plan was handed off")
	}
}

func TestBidWithoutAHeldScheduleCannotCheckTheWeekLimits(t *testing.T) {
	a := bidTestApp(t)
	a.weekFetchedAt = time.Time{}
	plan, err := a.TargetBidDraft(rawBid("5", 1))
	if err != nil || plan.Receipt.State != envelope.Blocked || !strings.Contains(plan.Status, "holds no NFL week") {
		t.Fatalf("plan = %v %q, %v", plan.Receipt.State, plan.Status, err)
	}
}

func TestBidActiveLimitOfThirty(t *testing.T) {
	a := bidTestApp(t)
	var extra []players.RawPlayer // one player per plan: a second open plan on a player is refused
	for i := range 32 {
		extra = append(extra, players.RawPlayer{ID: fmt.Sprint(20000 + i), Name: fmt.Sprintf("Player, N%d", i),
			Position: "WR", Team: "DAL", DraftYear: "2015"})
	}
	a.lookup = poolLookup(t, extra...)
	on := func(i int) RawBidInput { r := rawBid("1", 1); r.Player = fmt.Sprint(20000 + i); return r }
	var first string
	for i := range 30 {
		plan, err := a.TargetBidDraft(on(i))
		if err != nil || plan.Receipt.State != envelope.Ready {
			t.Fatalf("plan %d = %v, %v", i, plan.Receipt.State, err)
		}
		if i == 0 {
			first = plan.Receipt.CorrelationID
		}
	}
	over, err := a.TargetBidDraft(on(30))
	if err != nil || over.Receipt.State != envelope.Blocked ||
		!strings.Contains(over.Status, "You already have 30 active bid plans") {
		t.Fatalf("31st plan = %v %q, %v", over.Receipt.State, over.Status, err)
	}
	if _, err := a.TargetBidWithdraw(first); err != nil {
		t.Fatal(err)
	}
	again, err := a.TargetBidDraft(on(31))
	if err != nil || again.Receipt.State != envelope.Ready {
		t.Fatalf("plan after a withdrawal = %v, %v", again.Receipt.State, err)
	}
}

func TestBidReviseAndWithdraw(t *testing.T) {
	a := bidTestApp(t)
	plan, err := a.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	id := plan.Receipt.CorrelationID
	revised, err := a.TargetBidRevise(id, rawBid("6", 2))
	if err != nil || revised.Receipt.State != envelope.Ready || revised.Points != "7.200" {
		t.Fatalf("revised = %+v, %v", revised.Receipt.State, err)
	}
	var states []string
	for _, entry := range revised.Receipt.Audit {
		states = append(states, string(entry.To))
	}
	if strings.Join(states, ",") != "ready,stale,draft,ready" {
		t.Errorf("revision route = %v", states)
	}
	if _, err := a.TargetBidRevise(id, RawBidInput{FranchiseID: "0001", Player: "13594", Salary: "6", Years: 2}); err == nil {
		t.Error("a plan changed its player")
	}
	broken, err := a.TargetBidRevise(id, rawBid("6", 5))
	if err != nil || broken.Receipt.State != envelope.Blocked {
		t.Fatalf("revised into a block = %v, %v", broken.Receipt.State, err)
	}
	fixed, err := a.TargetBidRevise(id, rawBid("6", 4))
	if err != nil || fixed.Receipt.State != envelope.Ready || fixed.Facts.Years != 4 {
		t.Fatalf("fixed = %v, %v", fixed.Receipt.State, err)
	}
	gone, err := a.TargetBidWithdraw(id)
	if err != nil || gone.Receipt.State != envelope.Withdrawn || gone.State != "Withdrawn" ||
		gone.Status != "Removed by you before it was handed off." || len(gone.Actions) != 0 {
		t.Fatalf("withdrawn = %+v, %v", gone, err)
	}
	if _, err := a.TargetBidRevise(id, rawBid("6", 4)); err == nil {
		t.Error("a withdrawn plan was revised")
	}
}

func TestBidFactsFreezeOnceHandedOff(t *testing.T) {
	a := bidTestApp(t)
	plan, err := a.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	id := plan.Receipt.CorrelationID
	handed, err := a.TargetBidHandOff(id)
	if err != nil || handed.Receipt.State != envelope.HandedOff {
		t.Fatalf("hand-off = %v, %v", handed.Receipt.State, err)
	}
	if !strings.Contains(handed.Status, "nothing was posted for you") || len(handed.Rules) != 0 {
		t.Errorf("handed off: %q rules %d", handed.Status, len(handed.Rules))
	}
	for _, try := range []func() error{
		func() error { _, err := a.TargetBidRevise(id, rawBid("9", 3)); return err },
		func() error { _, err := a.TargetBidHandOff(id); return err },
	} {
		if err := try(); err == nil {
			t.Error("a handed-off plan was changed")
		}
	}
	posted, err := a.TargetBidPosted(id)
	if err != nil || posted.Receipt.State != envelope.BidPending || !strings.Contains(posted.Status, "owner reports") {
		t.Fatalf("posted = %v, %v", posted.Receipt.State, err)
	}
	select {
	case <-a.movesWake:
	default:
		t.Error("the watcher was not woken")
	}
}

// landedBid drafts, hands off and posts a 3-year $5M bid on 13593, then lets the watcher look at
// a roster refresh that sets the mirror to extra.
func landedBid(t *testing.T, closes string, extra ...state.MirrorPlayer) BidPlan {
	t.Helper()
	a := bidTestApp(t)
	past := time.Now().Add(-time.Hour)
	a.movesNow = func() time.Time { return past }
	raw := rawBid("5", 3)
	raw.Closes = closes
	plan, err := a.TargetBidDraft(raw)
	if err != nil {
		t.Fatal(err)
	}
	id := plan.Receipt.CorrelationID
	if _, err := a.TargetBidHandOff(id); err != nil {
		t.Fatal(err)
	}
	if _, err := a.TargetBidPosted(id); err != nil {
		t.Fatal(err)
	}
	a.movesNow = func() time.Time { return time.Now().Add(time.Minute) }
	a.movesRefresh = func(context.Context, envelope.Source) error { setRosters(t, a, extra...); return nil }
	if err := a.checkMoves(a.ctx); err != nil {
		t.Fatal(err)
	}
	board := a.TargetBidPlans("0001")
	if board.Count != 1 {
		t.Fatalf("board = %+v", board)
	}
	return board.Value.Plans[0]
}

func TestBidWatcherLandsAMatchingContractFromTheMirror(t *testing.T) {
	// Expiring in 2028 of a 2026 mirror is three years remaining.
	plan := landedBid(t, "", state.MirrorPlayer{MFLID: "13593", FranchiseID: "0001",
		RosterStatus: domain.RosterActive, Salary: 500_000_000, ContractYear: 2028})
	if plan.Receipt.State != envelope.Landed || plan.State != "Landed" ||
		!strings.Contains(plan.Status, "Landed: on your roster at $5,000,000 for 3 years") {
		t.Fatalf("plan = %v %q", plan.Receipt.State, plan.Status)
	}
}

func TestBidWatcherShowsAContractMismatchPlainly(t *testing.T) {
	plan := landedBid(t, "", state.MirrorPlayer{MFLID: "13593", FranchiseID: "0001",
		RosterStatus: domain.RosterActive, Salary: 450_000_000, ContractYear: 2027})
	if plan.Receipt.State != envelope.NotVerified || plan.State != "Not verified" ||
		!strings.Contains(plan.Status, "Contract mismatch: on your roster at $4,500,000 for 2 years, not the planned $5,000,000 for 3 years") {
		t.Fatalf("plan = %v %q", plan.Receipt.State, plan.Status)
	}
}

func TestBidWatcherAwardedOffAppWhenAnotherRosterHoldsHim(t *testing.T) {
	plan := landedBid(t, "", state.MirrorPlayer{MFLID: "13593", FranchiseID: "0002",
		RosterStatus: domain.RosterActive, Salary: 600_000_000, ContractYear: 2028})
	if plan.Receipt.State != envelope.AwardedOffApp || plan.State != "Awarded off-app" ||
		strings.Contains(plan.Status, "0002") || len(plan.Actions) != 0 {
		t.Fatalf("plan = %v %q %+v", plan.Receipt.State, plan.Status, plan.Actions)
	}
}

func TestBidWatcherAwardedOffAppWhenTheWindowClosesAndNothingLanded(t *testing.T) {
	plan := landedBid(t, time.Now().Add(-30*time.Minute).UTC().Format(time.RFC3339))
	if plan.Receipt.State != envelope.AwardedOffApp || !strings.Contains(plan.Status, "the bidding window closed") {
		t.Fatalf("plan = %v %q", plan.Receipt.State, plan.Status)
	}
	open := landedBid(t, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
	if open.Receipt.State != envelope.BidPending {
		t.Errorf("an open window ended the plan: %v", open.Receipt.State)
	}
}

func TestFormOfWritesMillionsExactly(t *testing.T) {
	for cents, want := range map[domain.Money]string{500_000_000: "5", 525_000_000: "5.25", 33_000_000: "0.33",
		1_001_000_000: "10.01", 100_000_000: "1"} {
		if got := formOf(envelope.BidFacts{Salary: cents}).Salary; got != want {
			t.Errorf("formOf(%d) = %q, want %q", cents, got, want)
		}
	}
}

func TestBlockedPlanThatWouldPassNowStillExplainsItself(t *testing.T) {
	a := bidTestApp(t)
	a.week = leagueweek.Week{Number: 13}
	plan, err := a.TargetBidDraft(rawBid("5", 1))
	if err != nil || plan.Receipt.State != envelope.Blocked {
		t.Fatalf("plan = %v, %v", plan.Receipt.State, err)
	}
	a.week = leagueweek.Week{Number: 5}
	board := a.TargetBidPlans("0001").Value.Plans[0]
	var handOff BidAction
	for _, action := range board.Actions {
		if action.ID == "hand-off" {
			handOff = action
		}
	}
	if !handOff.Disabled || !strings.Contains(handOff.Reason, "cannot tell whether yours is one") ||
		!strings.HasSuffix(handOff.Reason, "change the figures to check it again.") {
		t.Errorf("hand-off = %+v", handOff)
	}
}
