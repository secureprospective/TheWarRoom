package offers

import (
	"context"
	"fmt"
	"strings"
	"time"

	bidsapp "github.com/secureprospective/TheWarRoom/internal/app/offers/bids"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
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

// bidOpen: a plan that is not over.

// formOf writes the figures the way the form takes them: "5.25" millions, "6.500" points.

// sentenceEnd ends a status line or reason with a full stop, unless it already ends a sentence.

// noted returns the receipt with every history note ending in a full stop; older plans were
// saved without one.

// bidPlan assembles one plan. The rules are read live while the plan can still change; once it
// is handed off they were what the owner posted against, so none are shown.
func (a *API) bidPlan(ctx context.Context, r envelope.Receipt, f envelope.BidFacts) BidPlan {
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
	} else if lk, err := a.host.OffersDirectory(ctx); err == nil {
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

func (a *API) bidPlans(ctx context.Context, franchise string) ([]BidPlan, error) {
	receipts, err := a.host.OffersMoves().List(ctx, ingestion.LeagueID, franchise)
	if err != nil {
		return nil, fmt.Errorf("bid plans: list: %w", err)
	}
	facts, err := a.host.OffersMoves().AllBidFacts(ctx)
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
func (a *API) openBidPlansByPlayer(ctx context.Context, franchise string) map[string]bidPlanBrief {
	out := map[string]bidPlanBrief{}
	if franchise == "" {
		return out
	}
	receipts, err := a.host.OffersMoves().List(ctx, ingestion.LeagueID, franchise)
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
func (a *API) TargetBidPlans(franchiseID string) report.Reading[BidBoard] {
	board := BidBoard{Plans: []BidPlan{}}
	now := time.Now().UTC().Format(time.RFC3339)
	p := snapshot.Provenance{Source: "TheWarRoom bid plans (this computer)", Kind: "live",
		Freshness: domain.Freshness{State: domain.FreshLive, FetchedAt: now, Note: "Local plans; nothing is read from MFL here"}}
	read := report.Reading[BidBoard]{Sourced: snapshot.Sourced[BidBoard]{Value: board, Provenance: p}, Status: report.ReadingOK,
		Baseline: "Plans saved on this computer", Cutoff: now, Method: "Envelope plans plus the bid_facts table; rules read live",
		BuildStamp: a.host.OffersBuildStamp(),
		HowCounted: "Your own bid plans only. Nothing is stored about other franchises; the bid to beat is points you typed. A plan lands only when MFL's rosters show the player on your roster with the planned contract."}
	fail := func(err error) report.Reading[BidBoard] {
		reason := err.Error()
		read.Status, read.AbsenceReason = report.ReadingFailed, &reason
		read.Provenance.Freshness.State = domain.FreshFail
		return read
	}
	if err := a.host.Ready(); err != nil {
		return fail(err)
	}
	if err := ingestion.FeedFranchise(franchiseID); err != nil {
		return fail(fmt.Errorf("bid plans: %w", err))
	}
	ctx, cancel := context.WithTimeout(a.host.OffersContext(), readTimeout)
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
func (a *API) loadBid(ctx context.Context, id string) (envelope.Envelope, envelope.BidFacts, error) {
	e, err := a.host.OffersMoves().Get(ctx, id)
	if err != nil {
		return envelope.Envelope{}, envelope.BidFacts{}, fmt.Errorf("load: %w", err)
	}
	if e.Receipt().Spec.Intent != envelope.BidIntent {
		return envelope.Envelope{}, envelope.BidFacts{}, fmt.Errorf("load: %s is not a bid plan", id)
	}
	f, err := a.host.OffersMoves().BidFacts(ctx, id)
	if err != nil {
		return envelope.Envelope{}, envelope.BidFacts{}, fmt.Errorf("load: %w", err)
	}
	return e, f, nil
}

// bidMove runs one locked change to a saved bid plan: load, change, save, tell the screens.
func (a *API) bidMove(id, what string, change func(context.Context, envelope.Envelope, envelope.BidFacts) (envelope.Envelope, error)) (BidPlan, error) {
	if err := a.host.Ready(); err != nil {
		return BidPlan{}, fmt.Errorf("bid %s: startup: %w", what, err)
	}
	ctx, cancel := context.WithTimeout(a.host.OffersContext(), readTimeout)
	defer cancel()
	return a.serializedMove(func() (BidPlan, error) {
		e, f, err := a.loadBid(ctx, id)
		if err != nil {
			return BidPlan{}, fmt.Errorf("bid %s: %w", what, err)
		}
		next, err := change(ctx, e, f)
		if err != nil {
			return BidPlan{}, fmt.Errorf("bid %s: %w", what, err)
		}
		if err := a.host.OffersMoves().Save(ctx, next); err != nil {
			return BidPlan{}, fmt.Errorf("bid %s: save: %w", what, err)
		}
		a.host.OffersLog(next)
		a.host.OffersChanged()
		a.host.OffersWake()
		return a.bidPlan(ctx, next.Receipt(), f), nil
	})
}

// TargetBidHandOff records that the owner copied the ProBoards post. Nothing is posted: the
// plan must still pass every rule now, or the hand-off is refused with the reason.
func (a *API) TargetBidHandOff(id string) (BidPlan, error) {
	return a.bidMove(id, "hand-off", func(ctx context.Context, e envelope.Envelope, f envelope.BidFacts) (envelope.Envelope, error) {
		spec := e.Receipt().Spec
		in, _, err := a.bidInput(ctx, bidEntry{franchise: spec.FranchiseID, facts: f}, id)
		if err != nil {
			return envelope.Envelope{}, err
		}
		if rules := bid.Check(in); bid.Blocked(rules) {
			return envelope.Envelope{}, fmt.Errorf("the rules no longer pass: %s", bid.Note(rules))
		}
		return e.HandOffManual(a.host.OffersNow(), "post text copied for ProBoards; nothing was posted for you")
	})
}

// TargetBidPosted records the owner's word that the bid is up on ProBoards; the landing
// watcher then waits for the award.
func (a *API) TargetBidPosted(id string) (BidPlan, error) {
	return a.bidMove(id, "posted", func(_ context.Context, e envelope.Envelope, _ envelope.BidFacts) (envelope.Envelope, error) {
		return e.Await(a.host.OffersNow(), envelope.AwaitBid)
	})
}

// TargetBidWithdraw ends the local plan without removing a ProBoards post.
func (a *API) TargetBidWithdraw(id string) (BidPlan, error) {
	return a.bidMove(id, "withdraw", func(_ context.Context, e envelope.Envelope, _ envelope.BidFacts) (envelope.Envelope, error) {
		s := e.State()
		if s == envelope.Draft || s == envelope.Blocked || s == envelope.Ready {
			return e.WithdrawBid(a.host.OffersNow(), envelope.RemovePlanNote)
		}
		if s == envelope.HandedOff || s == envelope.NotYetDone {
			return e.WithdrawBid(a.host.OffersNow(), "You said you did not post it, so the plan is withdrawn.")
		}
		if s == envelope.BidPending || s == envelope.NotVerified || s == envelope.Stale {
			return e.WithdrawBid(a.host.OffersNow(), "Withdrawn: you said this bid was not processed; the app cannot remove a ProBoards post.")
		}
		return envelope.Envelope{}, fmt.Errorf("only a plan not yet posted can be withdrawn; this one is %s",
			strings.ToLower(bidStateWord(e.State())))
	})
}

// TargetBidRevise changes the figures of a plan not yet handed off. A Ready plan goes back
// through Stale and Draft first, the envelope's only route, and is checked again.
func (a *API) TargetBidRevise(id string, raw RawBidInput) (BidPlan, error) {
	if err := a.host.Ready(); err != nil {
		return BidPlan{}, fmt.Errorf("bid revise: startup: %w", err)
	}
	entry, err := raw.entry()
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid revise: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.OffersContext(), readTimeout)
	defer cancel()
	return a.serializedMove(func() (BidPlan, error) {
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
		entry.facts = withConditionalName(entry.facts, a.bidDropOptions(entry.franchise, entry.facts.Salary))
		checked, err := next.Check(a.host.OffersNow(), snapshot.Snapshot{}, envelope.BidCheck{Input: in})
		if err != nil {
			return BidPlan{}, fmt.Errorf("bid revise: check: %w", err)
		}
		if err := a.host.OffersMoves().SaveBid(ctx, checked, entry.facts); err != nil {
			return BidPlan{}, fmt.Errorf("bid revise: save: %w", err)
		}
		a.host.OffersLog(checked)
		a.host.OffersChanged()
		return a.bidPlan(ctx, checked.Receipt(), entry.facts), nil
	})
}

// reopened returns a Ready or Blocked plan to a state that can be checked again.
func (a *API) reopened(e envelope.Envelope) (envelope.Envelope, error) {
	if e.State() == envelope.Blocked || e.State() == envelope.Draft {
		return e, nil
	}
	if e.State() != envelope.Ready {
		return envelope.Envelope{}, fmt.Errorf("a %s plan cannot be changed", strings.ToLower(bidStateWord(e.State())))
	}
	now := a.host.OffersNow()
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

func bidStateWord(s envelope.State) string      { return bidsapp.BidStateWord(s) }
func bidOpen(s envelope.State) bool             { return bidsapp.BidOpen(s) }
func formOf(f envelope.BidFacts) BidForm        { return BidForm(bidsapp.FormOf(f)) }
func sentenceEnd(s string) string               { return bidsapp.SentenceEnd(s) }
func noted(r envelope.Receipt) envelope.Receipt { return bidsapp.Noted(r) }
func beatText(f envelope.BidFacts) string       { return bidsapp.BeatText(f) }
func bidActions(s envelope.State, status string, rules []bid.Result, problem string) []BidAction {
	actions := bidsapp.BidActions(s, status, rules, problem)
	out := make([]BidAction, len(actions))
	for i, a := range actions {
		out[i] = BidAction(a)
	}
	return out
}
