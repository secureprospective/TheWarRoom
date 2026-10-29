package trades

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type PlanReceipts interface {
	Get(context.Context, string) (envelope.Envelope, error)
	Save(context.Context, envelope.Envelope) error
}

// PlanWorkflow preserves the local plan/receipt save order; it never submits to MFL.
type PlanWorkflow struct {
	Receipts     PlanReceipts
	Request      func(string, leaguefeed.PendingTrade) envelope.TradeRequest
	Now          func() time.Time
	Lock, Unlock func()
	SavePlan     func(context.Context, state.TradePlan) error
	Plans        func(context.Context, string) ([]state.TradePlan, error)
	Own          func(string) error
	Project      func(context.Context, RawTradeBuilder, snapshot.Snapshot) (TradeBuilderPlan, error)
	Supersede    func(context.Context, envelope.Envelope, time.Time) error
	Audit        func(envelope.Envelope)
	Changed      func(context.Context)
	State        func(string) (snapshot.Snapshot, snapshot.Sourced[[]leaguefeed.PendingTrade], error)
	ValidateURL  func(string) error
	OpenURL      func(context.Context, string)
	Wake         func()
	HandOff      func(context.Context, RawTradeBuilder, snapshot.Snapshot) error
}

func PlanID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("trade ID: %w", err)
	}
	return "trade-" + hex.EncodeToString(bytes[:]), nil
}

func (s PlanWorkflow) Keep(ctx context.Context, r RawTradeBuilder, blocks []string) (envelope.Receipt, error) {
	s.Lock()
	defer s.Unlock()
	id, err := PlanID()
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	req := s.Request(r.Mine, leaguefeed.PendingTrade{Offering: r.Mine, OfferedTo: r.Other})
	req.Trade = envelope.ExpectedTrade{TradeID: id, Offering: r.Mine, Accepting: r.Other, OfferingGives: r.Give, AcceptingGives: r.Get}
	if req.Target.Kind == envelope.Unmapped {
		blocks = append(blocks, "MFL host not yet known")
	}
	e, err := envelope.DraftTradeAct(s.Now(), "trade.propose", req, blocks, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	if err := s.SavePlan(ctx, state.TradePlan{ID: id, Mine: r.Mine, Other: r.Other, Give: r.Give, Get: r.Get, Rationale: r.Rationale}); err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	if err := s.Receipts.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade plan receipt save: %w", err)
	}
	s.Changed(ctx)
	return e.Receipt(), nil
}

func (s PlanWorkflow) Exit(ctx context.Context, r RawTradeBuilder,
	held snapshot.Sourced[[]leaguefeed.PendingTrade]) (envelope.Receipt, error) {
	index := slices.IndexFunc(held.Value, func(p leaguefeed.PendingTrade) bool { return p.ID == r.ID })
	if index < 0 {
		return envelope.Receipt{}, fmt.Errorf("offer no longer pending on MFL")
	}
	p := held.Value[index]
	if (r.Action == "reject" && p.OfferedTo != r.Mine) || (r.Action == "revoke" && p.Offering != r.Mine) {
		return envelope.Receipt{}, fmt.Errorf("trade action belongs to the other owner")
	}
	blocks := []string{}
	if reason := envelope.TradeFeedReason(held.Provenance); reason != "" {
		blocks = append(blocks, "pending trades unavailable: "+reason)
	}
	req := s.Request(r.Mine, p)
	if req.Target.Kind == envelope.Unmapped {
		blocks = append(blocks, "MFL host not yet known")
	}
	id, err := PlanID()
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	now := s.Now()
	e, err := envelope.DraftTradeAct(now, "trade."+r.Action, req, blocks, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	s.Lock()
	defer s.Unlock()
	// One live exit plan per offer, as with accept plans: a new one supersedes the old.
	if err := s.Supersede(ctx, e, now); err != nil {
		return envelope.Receipt{}, err
	}
	if err := s.Receipts.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade exit save: %w", err)
	}
	s.Changed(ctx)
	return e.Receipt(), nil
}

func (s PlanWorkflow) Saved(ctx context.Context, mine string, snap snapshot.Snapshot) ([]TradeBuilderPlan, error) {
	out := []TradeBuilderPlan{}

	plans, err := s.Plans(ctx, mine)
	if err != nil {
		return nil, fmt.Errorf("saved trade plans: %w", err)
	}
	for _, plan := range plans {
		if err := s.Own(plan.Mine); err != nil {
			return nil, err
		}
		e, err := s.Receipts.Get(ctx, plan.ID)
		if err != nil {
			return nil, fmt.Errorf("saved trade receipt: %w", err)
		}
		if e.State() == envelope.Withdrawn {
			continue
		}
		p, err := s.Project(ctx, RawTradeBuilder{Mine: plan.Mine, Other: plan.Other, Give: plan.Give,
			Get: plan.Get, Rationale: plan.Rationale}, snap)
		if err != nil {
			return nil, err
		}
		receipt := e.Receipt()
		p.ID, p.Receipt = plan.ID, &receipt
		out = append(out, p)
	}
	return out, nil
}

func (s PlanWorkflow) Remove(ctx context.Context, r RawTradeBuilder) error {
	s.Lock()
	defer s.Unlock()
	e, err := s.Receipts.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("remove trade plan: %w", err)
	}
	if e.Receipt().Spec.FranchiseID != r.Mine {
		return fmt.Errorf("remove trade plan: belongs to another franchise")
	}
	next, err := e.WithdrawTrade(s.Now())
	if err != nil {
		return fmt.Errorf("remove trade plan: %w", err)
	}
	if err := s.Receipts.Save(ctx, next); err != nil {
		return fmt.Errorf("remove trade plan: save: %w", err)
	}
	s.Audit(next)
	s.Changed(ctx)
	return nil
}

func (s PlanWorkflow) Action(ctx context.Context, r RawTradeBuilder, snap snapshot.Snapshot) error {
	switch r.Action {
	case "remove":
		return s.Remove(ctx, r)
	case "handoff":
		return s.HandOff(ctx, r, snap)
	default:
		return nil
	}
}

func (s PlanWorkflow) RequireOwner(ctx context.Context, r RawTradeBuilder) error {
	if r.ID != "" && (r.Action == "remove" || r.Action == "handoff") {
		e, err := s.Receipts.Get(ctx, r.ID)
		if err != nil {
			return fmt.Errorf("trade plan: %w", err)
		}
		if err := s.Own(e.Receipt().Spec.FranchiseID); err != nil {
			return err
		}
	}
	return nil
}

// BuilderWorkflow runs under the caller's what-if then refresh locks.
type BuilderWorkflow struct {
	RequireOwner func(context.Context, RawTradeBuilder) error
	State        func(string) (snapshot.Snapshot, snapshot.Sourced[[]leaguefeed.PendingTrade], error)
	Teams        func(context.Context, snapshot.Snapshot) []TradeBuilderTeam
	Project      func(context.Context, RawTradeBuilder, snapshot.Snapshot) (TradeBuilderPlan, error)
	Keep         func(context.Context, RawTradeBuilder, []string) (envelope.Receipt, error)
	Exit         func(context.Context, RawTradeBuilder, snapshot.Sourced[[]leaguefeed.PendingTrade]) (envelope.Receipt, error)
	Action       func(context.Context, RawTradeBuilder, snapshot.Snapshot) error
	Saved        func(context.Context, string, snapshot.Snapshot) ([]TradeBuilderPlan, error)
}

func (s BuilderWorkflow) Run(ctx context.Context, r RawTradeBuilder, out TradeBuilderView) (TradeBuilderView, error) {
	if err := s.RequireOwner(ctx, r); err != nil {
		return out, err
	}

	snap, held, err := s.State(r.Mine)
	if err != nil {
		return out, err
	}
	out.Teams = s.Teams(ctx, snap)
	if r.Action == "preview" || r.Action == "save" {
		preview, err := s.Project(ctx, r, snap)
		if err != nil {
			return out, err
		}
		out.Preview = &preview
		if r.Action == "save" {
			receipt, err := s.Keep(ctx, r, preview.Reasons)
			if err != nil {
				return out, err
			}
			out.Preview.ID, out.Preview.Receipt = receipt.CorrelationID, &receipt
		}
	}
	if r.Action == "reject" || r.Action == "revoke" {
		receipt, err := s.Exit(ctx, r, held)
		if err != nil {
			return out, err
		}
		out.Preview = &TradeBuilderPlan{ID: receipt.CorrelationID, Reasons: []string{}, Teams: []TradeProjection{}, Receipt: &receipt}
	}
	if err := s.Action(ctx, r, snap); err != nil {
		return out, err
	}
	out.Plans, err = s.Saved(ctx, r.Mine, snap)
	return out, err
}

func (s PlanWorkflow) Accept(parent context.Context, timeout time.Duration, franchiseID string, offer leaguefeed.PendingTrade, snap snapshot.Snapshot, held snapshot.Sourced[[]leaguefeed.PendingTrade], id string) (envelope.Receipt, error) {
	now := s.Now()
	req := s.Request(franchiseID, offer)
	check := envelope.TradeCheck{PendingTrades: held, OfferingName: FranchiseName(snap, offer.Offering)}
	if req.Target.Kind == envelope.Unmapped {
		check.Blocks = append(check.Blocks, "MFL host not yet known")
	}
	e, err := envelope.DraftTrade(now, req, snap, check, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft trade accept: %w", err)
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	if err := s.Supersede(ctx, e, now); err != nil {
		return envelope.Receipt{}, err
	}
	if err := s.Receipts.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft trade accept: save: %w", err)
	}
	return e.Receipt(), nil
}

func (s PlanWorkflow) Handoff(ctx context.Context, r RawTradeBuilder, snap snapshot.Snapshot) error {
	plans, err := s.Plans(ctx, r.Mine)
	if err != nil {
		return fmt.Errorf("trade hand-off plans: %w", err)
	}
	blocks := []string{}
	for _, p := range plans {
		if err := s.Own(p.Mine); err != nil {
			return err
		}
		if p.ID != r.ID {
			continue
		}
		preview, err := s.Project(ctx, RawTradeBuilder{Mine: p.Mine, Other: p.Other, Give: p.Give, Get: p.Get, Rationale: p.Rationale}, snap)
		if err != nil {
			return err
		}
		blocks = preview.Reasons
	}
	if len(blocks) > 0 {
		return fmt.Errorf("trade hand-off blocked: %s", strings.Join(blocks, "; "))
	}
	s.Lock()
	defer s.Unlock()
	e, err := s.Receipts.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("trade hand-off load: %w", err)
	}
	spec := e.Receipt().Spec
	if spec.FranchiseID != r.Mine || !envelope.IsTrade(spec.Intent) || spec.Intent == "mfl.trade.accept" {
		return fmt.Errorf("trade hand-off outside scope")
	}
	if err := s.CheckExitPending(r.Mine, spec); err != nil {
		return err
	}
	if err := s.ValidateURL(spec.Target.URL); err != nil {
		return err
	}
	next, err := e.HandOff(s.Now())
	if err != nil {
		return fmt.Errorf("trade hand-off: %w", err)
	}
	if err := s.Receipts.Save(ctx, next); err != nil {
		return fmt.Errorf("trade hand-off save: %w", err)
	}
	s.OpenURL(ctx, spec.Target.URL)
	s.Changed(ctx)
	s.Wake()
	return nil
}

func (s PlanWorkflow) CheckExitPending(mine string, spec envelope.Spec) error {
	if spec.Intent == "trade.propose" {
		return nil
	}
	_, held, err := s.State(mine)
	if err != nil {
		return err
	}
	if envelope.TradeFeedReason(held.Provenance) != "" || !slices.ContainsFunc(held.Value, func(p leaguefeed.PendingTrade) bool {
		return p.ID == spec.Expected.Trade.TradeID && envelope.MatchesPendingTrade(*spec.Expected.Trade, p)
	}) {
		return fmt.Errorf("offer no longer available; refresh the trade desk")
	}
	return nil
}

func Request(franchiseID string, offer leaguefeed.PendingTrade, host func() string, season int) envelope.TradeRequest {
	req := envelope.TradeRequest{
		LeagueID: ingestion.LeagueID, FranchiseID: franchiseID,
		Trade: envelope.ExpectedTrade{
			TradeID: offer.ID, Offering: offer.Offering, Accepting: offer.OfferedTo,
			OfferingGives: leaguefeed.AssetTokens(offer.Gives), AcceptingGives: leaguefeed.AssetTokens(offer.Gets),
		},
		Target: envelope.Target{Kind: envelope.Unmapped},
	}
	if !offer.Expires.IsZero() {
		req.Deadline = &offer.Expires
	}
	if host() != "" {
		req.Target = envelope.Target{Kind: envelope.Mapped, URL: fmt.Sprintf(
			"https://%s/%d/options?L=%s&O=05", host(), season,
			url.QueryEscape(req.LeagueID))}
	}
	return req
}
