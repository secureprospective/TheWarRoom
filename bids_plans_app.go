package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// BidAction is one thing the owner can do with a plan. Go decides what is offered and why a
// button is off; the screen only draws it.
type BidAction struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Disabled bool   `json:"disabled"`
	Reason   string `json:"reason"`
}

// BidPlan is a bid plan as the screens show it: the envelope, its facts and Go's words for both.
type BidPlan struct {
	Receipt  envelope.Receipt        `json:"receipt"`
	Facts    envelope.BidFacts       `json:"facts"`
	Player   string                  `json:"player"`
	Salary   snapshot.ContractAmount `json:"salary"`
	Points   string                  `json:"points"`
	Contract string                  `json:"contract"`
	BeatText string                  `json:"beatText"`
	// Form is the plan's figures as the bid form takes them back: millions, points, ids.
	Form   BidForm      `json:"form"`
	State  string       `json:"state"`
	Status string       `json:"status"`
	Rules  []bid.Result `json:"rules"`
	// RuleSummary is Go's one line for the rules that pass; the others are shown in full.
	RuleSummary string      `json:"ruleSummary"`
	Post        string      `json:"post"`
	Actions     []BidAction `json:"actions"`
}

type BidForm struct {
	Salary string `json:"salary"`
	Beat   string `json:"beat"`
}

type BidBoard struct {
	Plans   []BidPlan `json:"plans"`
	Summary string    `json:"summary"`
}

type bidPlanBrief struct{ ID, State string }

func bidStateWord(s envelope.State) string {
	words := map[envelope.State]string{
		envelope.Draft: "Draft", envelope.Blocked: "Blocked", envelope.Ready: "Ready",
		envelope.HandedOff: "Handed off", envelope.NotYetDone: "Not yet done", envelope.BidPending: "Bid pending",
		envelope.NotVerified: "Not verified", envelope.Landed: "Landed", envelope.AwardedOffApp: "Awarded off-app",
		envelope.Withdrawn: "Withdrawn", envelope.Stale: "Stale", envelope.Failed: "Failed",
	}
	if w, ok := words[s]; ok {
		return w
	}
	return string(s)
}

// bidOpen: a plan that is not over.
func bidOpen(s envelope.State) bool {
	return !slices.Contains([]envelope.State{envelope.Landed, envelope.AwardedOffApp, envelope.Withdrawn,
		envelope.Stale, envelope.Failed}, s)
}

// formOf writes the figures the way the form takes them: "5.25" millions, "6.500" points.
func formOf(f envelope.BidFacts) BidForm {
	whole, frac := int64(f.Salary)/1e8, int64(f.Salary)%1e8
	form := BidForm{Salary: strconv.FormatInt(whole, 10)}
	if frac != 0 {
		form.Salary += "." + strings.TrimRight(fmt.Sprintf("%08d", frac), "0")
	}
	if f.Beat != nil {
		form.Beat = f.Beat.String()
	}
	return form
}

// sentenceEnd ends a status line or reason with a full stop, unless it already ends a sentence.
func sentenceEnd(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s[len(s)-1:], ".!?") {
		return s
	}
	return s + "."
}

// noted returns the receipt with every history note ending in a full stop; older plans were
// saved without one.
func noted(r envelope.Receipt) envelope.Receipt {
	r.Audit = slices.Clone(r.Audit)
	for i := range r.Audit {
		r.Audit[i].Note = sentenceEnd(r.Audit[i].Note)
	}
	return r
}

func beatText(f envelope.BidFacts) string {
	text := "No bid to beat entered"
	if f.Beat != nil {
		text = "Bid to beat " + f.Beat.String()
	}
	if f.Snipe {
		text += " · Snipe"
	}
	return text
}

// bidPlan assembles one plan. The rules are read live while the plan can still change; once it
// is handed off they were what the owner posted against, so none are shown.
func (a *App) bidPlan(ctx context.Context, r envelope.Receipt, f envelope.BidFacts) BidPlan {
	line := "Player " + f.Player.String()
	rules := []bid.Result{}
	var problem string
	if r.State == envelope.Blocked || r.State == envelope.Ready || r.State == envelope.Draft {
		in, found, err := a.bidInput(ctx, bidEntry{franchise: r.Spec.FranchiseID, facts: f}, r.CorrelationID)
		if err != nil {
			problem = err.Error()
		} else {
			line, rules = found, bid.Check(in)
		}
	} else if lk, err := a.directory(ctx); err == nil {
		if facts, ok := lk.Facts(f.Player.String()); ok {
			line = fmt.Sprintf("%s (%s, %s)", facts.Name, facts.Position, facts.Team)
		}
	}
	status := ""
	if n := len(r.Audit); n > 0 {
		status = r.Audit[n-1].Note
	}
	summary := ""
	if len(rules) > 0 {
		summary = bid.Summary(rules)
	}
	return BidPlan{Receipt: noted(r), Facts: f, Player: line, Salary: amountOf(f.Salary), Points: f.Points.String(),
		Contract: bid.ContractText(f.Salary, f.Years), BeatText: beatText(f), Form: formOf(f),
		State: bidStateWord(r.State), Status: sentenceEnd(status), Rules: rules, RuleSummary: summary,
		Post: postText(line, f), Actions: bidActions(r.State, status, rules, problem)}
}

func bidActions(s envelope.State, status string, rules []bid.Result, problem string) []BidAction {
	copyPost := BidAction{ID: "hand-off", Label: "Copy ProBoards post"}
	again := BidAction{ID: "copy", Label: "Copy post again"}
	switch {
	case slices.Contains([]envelope.State{envelope.Blocked, envelope.Ready, envelope.Draft}, s):
		reason := bidBlockReason(rules, problem)
		if reason == "" && s == envelope.Blocked {
			reason = status + "; change the figures to check it again"
		}
		if reason == "" && s == envelope.Draft {
			reason = "This plan is not checked yet"
		}
		if reason != "" {
			copyPost.Disabled, copyPost.Reason = true, sentenceEnd(reason)
		}
		out := []BidAction{copyPost, {ID: "revise", Label: "Change figures"}}
		if s == envelope.Ready {
			out = append(out, BidAction{ID: "withdraw", Label: "Withdraw plan"})
		}
		return out
	case slices.Contains([]envelope.State{envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified}, s):
		out := []BidAction{again, {ID: "posted", Label: "I posted it"}}
		if s != envelope.NotVerified { // a bid never posted can be taken back; a landing in doubt cannot
			out = append(out, BidAction{ID: "unposted", Label: "I didn't post it"})
		}
		return out
	case s == envelope.BidPending:
		return []BidAction{again}
	}
	return []BidAction{}
}

func bidBlockReason(rules []bid.Result, problem string) string {
	if problem != "" {
		return problem
	}
	if bid.Blocked(rules) {
		return bid.Note(rules)
	}
	return ""
}

func (a *App) bidPlans(ctx context.Context, franchise string) ([]BidPlan, error) {
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, franchise)
	if err != nil {
		return nil, fmt.Errorf("bid plans: list: %w", err)
	}
	facts, err := a.moves.AllBidFacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("bid plans: %w", err)
	}
	plans := []BidPlan{}
	for _, r := range receipts {
		if f, ok := facts[r.CorrelationID]; ok && r.Spec.Intent == envelope.BidIntent {
			plans = append(plans, a.bidPlan(ctx, r, f))
		}
	}
	return plans, nil
}

// openBidPlansByPlayer maps a player id to the owner's open bid plan on him, for the pool rows.
func (a *App) openBidPlansByPlayer(ctx context.Context, franchise string) map[string]bidPlanBrief {
	out := map[string]bidPlanBrief{}
	if franchise == "" {
		return out
	}
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, franchise)
	if err != nil {
		return out
	}
	for _, r := range receipts { // newest first: the newest open plan wins
		if r.Spec.Intent != envelope.BidIntent || !bidOpen(r.State) {
			continue
		}
		if _, taken := out[r.Spec.Expected.Player.String()]; !taken {
			out[r.Spec.Expected.Player.String()] = bidPlanBrief{ID: r.CorrelationID, State: bidStateWord(r.State)}
		}
	}
	return out
}

// TargetBidPlans lists the owner's bid plans, newest first, from the local plan store.
func (a *App) TargetBidPlans(franchiseID string) Reading[BidBoard] {
	board := BidBoard{Plans: []BidPlan{}}
	now := time.Now().UTC().Format(time.RFC3339)
	p := snapshot.Provenance{Source: "TheWarRoom bid plans (this computer)", Kind: "live",
		Freshness: domain.Freshness{State: domain.FreshLive, FetchedAt: now, Note: "Local plans; nothing is read from MFL here"}}
	read := Reading[BidBoard]{Sourced: snapshot.Sourced[BidBoard]{Value: board, Provenance: p}, Status: ReadingOK,
		Baseline: "Plans saved on this computer", Cutoff: now, Method: "Envelope plans plus the bid_facts table; rules read live",
		BuildStamp: buildLabel() + " " + buildDate,
		HowCounted: "Your own bid plans only. Nothing is stored about other franchises; the bid to beat is points you typed. A plan lands only when MFL's rosters show the player on your roster with the planned contract."}
	fail := func(err error) Reading[BidBoard] {
		reason := err.Error()
		read.Status, read.AbsenceReason = ReadingFailed, &reason
		read.Provenance.Freshness.State = domain.FreshFail
		return read
	}
	if err := a.ready(); err != nil {
		return fail(err)
	}
	if err := ingestion.FeedFranchise(franchiseID); err != nil {
		return fail(fmt.Errorf("bid plans: %w", err))
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	plans, err := a.bidPlans(ctx, franchiseID)
	if err != nil {
		return fail(err)
	}
	read.Value, read.Count = BidBoard{Plans: plans, Summary: boardSummary(plans)}, len(plans)
	return read
}

func boardSummary(plans []BidPlan) string {
	open := 0
	for _, p := range plans {
		if bidOpen(p.Receipt.State) {
			open++
		}
	}
	return fmt.Sprintf("%d open of %d bid plans", open, len(plans))
}

// loadBid reads a plan's envelope and facts and checks it belongs to the owner's bids.
func (a *App) loadBid(ctx context.Context, id string) (envelope.Envelope, envelope.BidFacts, error) {
	e, err := a.moves.Get(ctx, id)
	if err != nil {
		return envelope.Envelope{}, envelope.BidFacts{}, fmt.Errorf("load: %w", err)
	}
	if e.Receipt().Spec.Intent != envelope.BidIntent {
		return envelope.Envelope{}, envelope.BidFacts{}, fmt.Errorf("load: %s is not a bid plan", id)
	}
	f, err := a.moves.BidFacts(ctx, id)
	if err != nil {
		return envelope.Envelope{}, envelope.BidFacts{}, fmt.Errorf("load: %w", err)
	}
	return e, f, nil
}

// bidMove runs one locked change to a saved bid plan: load, change, save, tell the screens.
func (a *App) bidMove(id, what string, change func(context.Context, envelope.Envelope, envelope.BidFacts) (envelope.Envelope, error)) (BidPlan, error) {
	if err := a.ready(); err != nil {
		return BidPlan{}, fmt.Errorf("bid %s: startup: %w", what, err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	e, f, err := a.loadBid(ctx, id)
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid %s: %w", what, err)
	}
	next, err := change(ctx, e, f)
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid %s: %w", what, err)
	}
	if err := a.moves.Save(ctx, next); err != nil {
		return BidPlan{}, fmt.Errorf("bid %s: save: %w", what, err)
	}
	a.logMove(next)
	a.movesChanged(a.ctx)
	a.wakeMoves()
	return a.bidPlan(ctx, next.Receipt(), f), nil
}

// TargetBidHandOff records that the owner copied the ProBoards post. Nothing is posted: the
// plan must still pass every rule now, or the hand-off is refused with the reason.
func (a *App) TargetBidHandOff(id string) (BidPlan, error) {
	return a.bidMove(id, "hand-off", func(ctx context.Context, e envelope.Envelope, f envelope.BidFacts) (envelope.Envelope, error) {
		spec := e.Receipt().Spec
		in, _, err := a.bidInput(ctx, bidEntry{franchise: spec.FranchiseID, facts: f}, id)
		if err != nil {
			return envelope.Envelope{}, err
		}
		if rules := bid.Check(in); bid.Blocked(rules) {
			return envelope.Envelope{}, fmt.Errorf("the rules no longer pass: %s", bid.Note(rules))
		}
		return e.HandOffManual(a.movesNow(), "post text copied for ProBoards; nothing was posted for you")
	})
}

// TargetBidPosted records the owner's word that the bid is up on ProBoards; the landing
// watcher then waits for the award.
func (a *App) TargetBidPosted(id string) (BidPlan, error) {
	return a.bidMove(id, "posted", func(_ context.Context, e envelope.Envelope, _ envelope.BidFacts) (envelope.Envelope, error) {
		return e.Await(a.movesNow(), envelope.AwaitBid)
	})
}

// TargetBidWithdraw ends a plan the owner will not post: a Ready one, or one handed off whose
// post was copied but never posted ("I didn't post it"). It frees its place in the 30 and
// leaves the plan's facts as they were. A posted bid is not withdrawn here: the app cannot
// take a post down from ProBoards.
func (a *App) TargetBidWithdraw(id string) (BidPlan, error) {
	return a.bidMove(id, "withdraw", func(_ context.Context, e envelope.Envelope, _ envelope.BidFacts) (envelope.Envelope, error) {
		s := e.State()
		if s == envelope.Ready {
			return e.WithdrawBid(a.movesNow(), "Withdrawn by you before it was handed off.")
		}
		if s == envelope.HandedOff || s == envelope.NotYetDone {
			return e.WithdrawBid(a.movesNow(), "You said you did not post it, so the plan is withdrawn.")
		}
		if s == envelope.BidPending {
			return envelope.Envelope{}, fmt.Errorf("a posted bid cannot be withdrawn here: " +
				"the app cannot take a post down from ProBoards")
		}
		return envelope.Envelope{}, fmt.Errorf("only a plan not yet posted can be withdrawn; this one is %s",
			strings.ToLower(bidStateWord(e.State())))
	})
}

// TargetBidRevise changes the figures of a plan not yet handed off. A Ready plan goes back
// through Stale and Draft first, the envelope's only route, and is checked again.
func (a *App) TargetBidRevise(id string, raw RawBidInput) (BidPlan, error) {
	if err := a.ready(); err != nil {
		return BidPlan{}, fmt.Errorf("bid revise: startup: %w", err)
	}
	entry, err := raw.entry()
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid revise: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	e, _, err := a.loadBid(ctx, id)
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid revise: %w", err)
	}
	spec := e.Receipt().Spec
	if spec.Expected.Player != entry.facts.Player || spec.FranchiseID != entry.franchise {
		return BidPlan{}, fmt.Errorf("bid revise: a plan keeps its player; draft a new one")
	}
	next, err := a.reopened(e)
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid revise: %w", err)
	}
	in, _, err := a.bidInput(ctx, entry, id)
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid revise: %w", err)
	}
	checked, err := next.Check(a.movesNow(), snapshot.Snapshot{}, envelope.BidCheck{Input: in})
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid revise: check: %w", err)
	}
	if err := a.moves.SaveBid(ctx, checked, entry.facts); err != nil {
		return BidPlan{}, fmt.Errorf("bid revise: save: %w", err)
	}
	a.logMove(checked)
	a.movesChanged(a.ctx)
	return a.bidPlan(ctx, checked.Receipt(), entry.facts), nil
}

// reopened returns a Ready or Blocked plan to a state that can be checked again.
func (a *App) reopened(e envelope.Envelope) (envelope.Envelope, error) {
	if e.State() == envelope.Blocked {
		return e, nil
	}
	if e.State() != envelope.Ready {
		return envelope.Envelope{}, fmt.Errorf("a %s plan cannot be changed", strings.ToLower(bidStateWord(e.State())))
	}
	now := a.movesNow()
	stale, err := e.Invalidate(now, "figures changed by you")
	if err != nil {
		return envelope.Envelope{}, fmt.Errorf("reopen: %w", err)
	}
	draft, err := stale.Rebase(now, "figures changed by you")
	if err != nil {
		return envelope.Envelope{}, fmt.Errorf("reopen: %w", err)
	}
	return draft, nil
}
