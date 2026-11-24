package mflread

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/mflmirror"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/mflref/lineup"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/mflref/moves"
)

// Plans live in the league's own database through the desktop's moves store. Every read and write is keyed
// by league and the seated franchise; nothing here is sent to MFL.

// planMu orders plan changes: a plan is loaded, changed and saved as one step.
var planMu sync.Mutex //nolint:gochecknoglobals // one lock orders plan changes across every copy of the fetcher

type bidBody struct {
	Add    string   `json:"add"`
	Amount float64  `json:"amount"`
	Years  int      `json:"years"`
	Drop   []string `json:"drop"`
	Snipe  bool     `json:"snipe"`
}

type lineupBody struct {
	Week     int      `json:"week"`
	Starters []string `json:"starters"`
}

func planned(s envelope.State) bool {
	return s == envelope.Draft || s == envelope.Blocked || s == envelope.Ready
}

func stateWords(s envelope.State) string {
	switch s { //nolint:exhaustive // every other state reads as its own name
	case envelope.Ready:
		return "Planned. Nothing has been done on MFL yet."
	case envelope.Draft, envelope.Blocked:
		return "Planned, but not ready."
	case envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified:
		return "You said you did it. The app has not verified it."
	case envelope.Withdrawn:
		return "Withdrawn."
	}
	return strings.ReplaceAll(string(s), "_", " ")
}

func (v view) store(ctx context.Context) (*moves.Store, error) {
	store := moves.New(v.Pools)
	if err := store.EnsureTables(ctx); err != nil {
		return nil, fmt.Errorf("MFL plans storage: %w", err)
	}
	return store, nil
}

func planID(prefix string) (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("plan id: %w", err)
	}
	return prefix + "-" + hex.EncodeToString(random[:]), nil
}

func decode(body json.RawMessage, into any) error {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return fmt.Errorf("the plan's details are not readable: %w", err)
	}
	return nil
}

// mflRefused marks an owner-facing refusal; leagueservice adds the "MFL moves:" prefix once.
func mflRefused(msg string) error {
	return mflmirror.Refused(msg) //nolint:wrapcheck // the refusal is the owner-facing words; leagueservice adds the prefix
}

// PlanSave re-runs the check on the server and saves the plan for the seated franchise only.
func (Fetcher) PlanSave(ctx context.Context, s mflmirror.Seat, kind string, body json.RawMessage) (mflmirror.Plan, error) {
	v := view{s}
	store, err := v.store(ctx)
	if err != nil {
		return mflmirror.Plan{}, err
	}
	planMu.Lock()
	defer planMu.Unlock()
	switch kind {
	case "bid":
		return v.saveBid(ctx, store, body)
	case "lineup":
		return v.saveLineup(ctx, store, body)
	}
	return mflmirror.Plan{}, mflRefused("a plan is a bid or a lineup")
}

func (v view) saveBid(ctx context.Context, store *moves.Store, body json.RawMessage) (mflmirror.Plan, error) {
	var b bidBody
	if err := decode(body, &b); err != nil {
		return mflmirror.Plan{}, err
	}
	if len(b.Drop) > 1 {
		return mflmirror.Plan{}, mflRefused("a bid plan holds one conditional drop")
	}
	player, err := playerid.New(b.Add)
	if err != nil || b.Amount <= 0 || b.Years < 0 || b.Years > bid.MaxYears {
		return mflmirror.Plan{}, mflRefused(fmt.Sprintf("choose the player, one annual salary above $0 and 1 to %d years", bid.MaxYears))
	}
	check := v.bidCheck(mflmirror.BidInput{Add: b.Add, Amount: b.Amount, Years: b.Years, Drop: b.Drop})
	if !check.OK {
		return mflmirror.Plan{}, mflRefused(fmt.Sprintf("this bid has problems, so it was not saved: %s", strings.Join(check.Problems, " ")))
	}
	if err := v.requireNoOpenBid(ctx, store, player, check.Player); err != nil {
		return mflmirror.Plan{}, err
	}
	points, err := bid.Score(check.Salary, check.Years)
	if err != nil {
		return mflmirror.Plan{}, fmt.Errorf("score the bid: %w", err)
	}
	facts := v.bidFacts(check, b, player, points)
	id, err := planID("bid")
	if err != nil {
		return mflmirror.Plan{}, err
	}
	e, err := envelope.DraftBid(v.Now.UTC(), envelope.BidRequest{LeagueID: v.League, FranchiseID: v.Franchise, Facts: facts},
		func() string { return id })
	if err != nil {
		return mflmirror.Plan{}, fmt.Errorf("plan the bid: %w", err)
	}
	if err := store.SaveBid(ctx, e, facts); err != nil {
		return mflmirror.Plan{}, fmt.Errorf("save the plan: %w", err)
	}
	return v.planView(e.Receipt(), facts), nil
}

// requireNoOpenBid refuses a second open plan on the same player.
func (v view) requireNoOpenBid(ctx context.Context, store *moves.Store, player playerid.PlayerID, name string) error {
	receipts, err := store.List(ctx, v.League, v.Franchise)
	if err != nil {
		return fmt.Errorf("read your plans: %w", err)
	}
	for _, r := range receipts {
		if r.Spec.Intent == envelope.BidIntent && r.Spec.Expected.Player == player &&
			(r.State == envelope.Ready || r.State == envelope.HandedOff) {
			return mflRefused(fmt.Sprintf("you already have an open bid plan on %s", name))
		}
	}
	return nil
}

func (v view) bidFacts(check mflmirror.BidCheck, b bidBody, player playerid.PlayerID, points bid.Points) envelope.BidFacts {
	facts := envelope.BidFacts{Player: player, Salary: check.Salary, Years: check.Years, Points: points, Snipe: b.Snipe}
	if len(b.Drop) == 1 {
		facts.ConditionalDrop, _ = playerid.New(b.Drop[0])
		facts.ConditionalName = v.name(b.Drop[0])
	}
	return facts
}

func (v view) saveLineup(ctx context.Context, store *moves.Store, body json.RawMessage) (mflmirror.Plan, error) {
	var b lineupBody
	if err := decode(body, &b); err != nil {
		return mflmirror.Plan{}, err
	}
	if b.Week < 1 || b.Week > 22 {
		return mflmirror.Plan{}, mflRefused("choose the NFL week, 1 to 22")
	}
	ids, err := starterIDs(b.Starters)
	if err != nil {
		return mflmirror.Plan{}, err
	}
	check, err := v.lineupCheck(b.Week, ids)
	if err != nil {
		return mflmirror.Plan{}, err
	}
	if !check.OK {
		return mflmirror.Plan{}, lineupRefusal(check)
	}
	config, err := v.config()
	if err != nil {
		return mflmirror.Plan{}, err
	}
	rules, err := lineup.ParseRules(config)
	if err != nil {
		return mflmirror.Plan{}, fmt.Errorf("MFL's starting lineup rules could not be read: %w", err)
	}
	snap, target := v.seatSnapshot(), v.lineupTarget(b.Week)
	id, err := planID("lineup")
	if err != nil {
		return mflmirror.Plan{}, err
	}
	e, err := envelope.DraftLineup(v.Now.UTC(), envelope.LineupRequest{LeagueID: v.League, FranchiseID: v.Franchise,
		Week: b.Week, Starters: ids, Target: target}, snap, envelope.LineupCheck{Rules: rules, BaselineKnown: true},
		func() string { return id })
	if err != nil {
		return mflmirror.Plan{}, fmt.Errorf("plan the lineup: %w", err)
	}
	if e.State() != envelope.Ready {
		note := ""
		if audit := e.Receipt().Audit; len(audit) > 0 {
			note = audit[len(audit)-1].Note
		}
		return mflmirror.Plan{}, mflRefused(fmt.Sprintf("this lineup is not ready, so it was not saved: %s", note))
	}
	// One live lineup plan per week, as on the desktop: this one supersedes the owner's older planned one.
	if err := v.supersedeLineups(ctx, store, e); err != nil {
		return mflmirror.Plan{}, fmt.Errorf("replace the older lineup plan: %w", err)
	}
	if err := store.Save(ctx, e); err != nil {
		return mflmirror.Plan{}, fmt.Errorf("save the plan: %w", err)
	}
	return v.planView(e.Receipt(), envelope.BidFacts{}), nil
}

func lineupRefusal(check mflmirror.LineupCheck) error {
	words := []string{}
	for _, p := range check.Problems {
		words = append(words, p.Message)
	}
	return mflRefused(fmt.Sprintf("this lineup has problems, so it was not saved: %s", strings.Join(words, " ")))
}

// supersedeLineups moves the owner's older planned lineup for the same week to Stale, as the desktop does.
// Only a plan still waiting to be done competes; a step already marked done stays on record. (The desktop's
// own SupersedeMoves lives in a package this one may not import.)
func (v view) supersedeLineups(ctx context.Context, store *moves.Store, next envelope.Envelope) error {
	spec := next.Receipt().Spec
	receipts, err := store.List(ctx, v.League, v.Franchise)
	if err != nil {
		return fmt.Errorf("list your plans: %w", err)
	}
	for _, r := range receipts {
		if r.Spec.Intent != spec.Intent || r.State != envelope.Ready || r.Spec.Expected.Lineup.Week != spec.Expected.Lineup.Week {
			continue
		}
		old, err := store.Get(ctx, r.CorrelationID)
		if err != nil {
			return fmt.Errorf("load the older plan: %w", err)
		}
		stale, err := old.Invalidate(v.Now.UTC(), "superseded by "+next.ID())
		if err != nil {
			return fmt.Errorf("replace the older plan: %w", err)
		}
		if err := store.Save(ctx, stale); err != nil {
			return fmt.Errorf("save the replaced plan: %w", err)
		}
	}
	return nil
}

// seatSnapshot is the held snapshot with the picked MFL team's roster scoped to the seat, as plans are.
func (v view) seatSnapshot() snapshot.Snapshot {
	snap := v.heldSnapshot()
	for i := range snap.Rosters.Value {
		if snap.Rosters.Value[i].FranchiseID == v.Team {
			snap.Rosters.Value[i].FranchiseID = v.Franchise
		}
	}
	return snap
}

// lineupTarget is the MFL lineup page for the week: an address the owner's browser opens.
func (v view) lineupTarget(week int) envelope.Target {
	return envelope.Target{Kind: envelope.Mapped, URL: fmt.Sprintf("https://www.myfantasyleague.com/%d/lineup?L=%s&WEEK=%d&F=%s",
		v.Held.Season, url.QueryEscape(v.MFLLeague), week, url.QueryEscape(v.Team))}
}

func (v view) planView(r envelope.Receipt, f envelope.BidFacts) mflmirror.Plan {
	out := mflmirror.Plan{ID: r.CorrelationID, State: string(r.State), Words: stateWords(r.State),
		CanDone: r.State == envelope.Ready, CanWithdraw: planned(r.State)}
	if r.Spec.Intent == envelope.BidIntent {
		out.Kind = "bid"
		out.Player, out.PlayerName = f.Player.String(), v.name(f.Player.String())
		out.Salary, out.Years, out.Points, out.Snipe = f.Salary, f.Years, f.Points.String(), f.Snipe
		if f.ConditionalDrop.String() != "" {
			out.Drop, out.DropName = f.ConditionalDrop.String(), f.ConditionalName
		}
		return out
	}
	out.Kind = "lineup"
	if l := r.Spec.Expected.Lineup; l != nil {
		out.Week = l.Week
		for _, id := range l.Starters {
			out.Starters = append(out.Starters, id.String())
		}
	}
	return out
}

func isPlanIntent(intent string) bool {
	return intent == envelope.BidIntent || intent == "mfl.lineup.set"
}

// PlansRead lists the seated franchise's own plans, newest first. Results are never cached.
func (Fetcher) PlansRead(ctx context.Context, s mflmirror.Seat) (mflmirror.Plans, error) {
	v := view{s}
	store, err := v.store(ctx)
	if err != nil {
		return mflmirror.Plans{}, err
	}
	receipts, err := store.List(ctx, v.League, v.Franchise)
	if err != nil {
		return mflmirror.Plans{}, fmt.Errorf("read your plans: %w", err)
	}
	out := mflmirror.Plans{League: v.League, Plans: []mflmirror.Plan{}}
	for _, r := range receipts {
		if r.Spec.LeagueID != v.League || r.Spec.FranchiseID != v.Franchise || !isPlanIntent(r.Spec.Intent) {
			continue
		}
		var f envelope.BidFacts
		if r.Spec.Intent == envelope.BidIntent {
			if f, err = store.BidFacts(ctx, r.CorrelationID); err != nil {
				return mflmirror.Plans{}, fmt.Errorf("read a bid plan: %w", err)
			}
		}
		out.Plans = append(out.Plans, v.planView(r, f))
	}
	count := len(out.Plans)
	label := "plans"
	if count == 1 {
		label = "plan"
	}
	out.Words = fmt.Sprintf("%d %s. Nothing here was sent to MFL.", count, label)
	return out, nil
}

// loadOwn returns the plan only when it belongs to this league and the seated franchise; a plan that is
// missing and a plan that is someone else's read the same.
func (v view) loadOwn(ctx context.Context, store *moves.Store, id string) (envelope.Envelope, error) {
	e, err := store.Get(ctx, id)
	if err != nil {
		return envelope.Envelope{}, mflRefused(mflmirror.PlanNotFound)
	}
	spec := e.Receipt().Spec
	if spec.FranchiseID != v.Franchise || spec.LeagueID != v.League || !isPlanIntent(spec.Intent) {
		return envelope.Envelope{}, mflRefused(mflmirror.PlanNotFound)
	}
	return e, nil
}

// PlanLink returns the MFL page for a plan. It only names the page; the server never opens it.
func (Fetcher) PlanLink(ctx context.Context, s mflmirror.Seat, plan string) (mflmirror.PlanLink, error) {
	v := view{s}
	store, err := v.store(ctx)
	if err != nil {
		return mflmirror.PlanLink{}, err
	}
	e, err := v.loadOwn(ctx, store, plan)
	if err != nil {
		return mflmirror.PlanLink{}, err
	}
	link, words := e.Receipt().Spec.Target.URL, "Open this page on MFL, then come back and say you did it."
	if link == "" {
		// A bid is posted by hand where the league takes bids; the nearest MFL page is the league home.
		link = fmt.Sprintf("https://www.myfantasyleague.com/%d/home/%s", mflmirror.Season(v.Now), url.PathEscape(v.MFLLeague))
		words = "A bid is posted by hand where your league takes bids. This is your league's MFL home page."
	}
	if err := envelope.ValidateMoveURL(link); err != nil {
		return mflmirror.PlanLink{}, fmt.Errorf("that plan has no safe MFL page to open: %w", err)
	}
	return mflmirror.PlanLink{League: v.League, Plan: plan, URL: link, Words: words}, nil
}

// PlanDone records the owner's word that they did the step. It is a local change only; nothing is sent to MFL.
func (Fetcher) PlanDone(ctx context.Context, s mflmirror.Seat, plan string) (mflmirror.Plan, error) {
	return view{s}.move(ctx, plan, func(e envelope.Envelope, at time.Time) (envelope.Envelope, error) {
		if e.State() != envelope.Ready {
			return envelope.Envelope{}, mflRefused(fmt.Sprintf("only a planned step can be marked done; this one is %s", strings.ToLower(stateWords(e.State()))))
		}
		if e.Receipt().Spec.Intent == envelope.BidIntent {
			return e.HandOffManual(at, "You said you posted this bid; nothing was sent from here.")
		}
		return e.HandOff(at)
	})
}

// PlanWithdraw ends a plan that is still only planned. A step already marked done stays on record.
func (Fetcher) PlanWithdraw(ctx context.Context, s mflmirror.Seat, plan string) (mflmirror.Plan, error) {
	return view{s}.move(ctx, plan, func(e envelope.Envelope, at time.Time) (envelope.Envelope, error) {
		if !planned(e.State()) {
			return envelope.Envelope{}, mflRefused(fmt.Sprintf("only a plan that is still planned can be withdrawn; this one is %s", strings.ToLower(stateWords(e.State()))))
		}
		if e.Receipt().Spec.Intent == envelope.BidIntent {
			return e.WithdrawBid(at, envelope.RemovePlanNote)
		}
		return e.WithdrawLineup(at)
	})
}

func (v view) move(ctx context.Context, plan string, change func(envelope.Envelope, time.Time) (envelope.Envelope, error)) (mflmirror.Plan, error) {
	store, err := v.store(ctx)
	if err != nil {
		return mflmirror.Plan{}, err
	}
	planMu.Lock()
	defer planMu.Unlock()
	e, err := v.loadOwn(ctx, store, plan)
	if err != nil {
		return mflmirror.Plan{}, err
	}
	next, err := change(e, v.Now.UTC())
	if err != nil {
		return mflmirror.Plan{}, err
	}
	if err := store.Save(ctx, next); err != nil {
		return mflmirror.Plan{}, fmt.Errorf("save the plan: %w", err)
	}
	var f envelope.BidFacts
	if next.Receipt().Spec.Intent == envelope.BidIntent {
		if f, err = store.BidFacts(ctx, plan); err != nil {
			return mflmirror.Plan{}, fmt.Errorf("read the bid plan: %w", err)
		}
	}
	return v.planView(next.Receipt(), f), nil
}
