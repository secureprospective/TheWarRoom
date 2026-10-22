package contracts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type RawReleaseRequest struct {
	Mine      string `json:"mine"`
	Action    string `json:"action"`
	Move      string `json:"move"`
	ID        string `json:"id"`
	Confirmed bool   `json:"confirmed"`
}

func (r RawReleaseRequest) Validate() error {
	return Release(r).Validate()
}

// ReleasePlanView is a release plan as the owner reads it.
type ReleasePlanView ReleasePlan

func (a *API) TargetReleasePlans(raw RawReleaseRequest) ReleasePlansReading {
	if err := raw.Validate(); err != nil {
		return a.releaseReading(nil, err)
	}
	if err := a.host.Ready(); err != nil {
		return a.releaseReading(nil, err)
	}
	if err := a.host.RequireOwnFranchise(raw.Mine); err != nil {
		return a.releaseReading(nil, err)
	}
	var out ReleasePlansReading
	a.host.SerializeReleasePlans(func() {
		ctx, cancel := context.WithTimeout(a.host.ContractsContext(), timeout)
		defer cancel()
		out = a.releasePlans(ctx, raw)
	})
	return out
}

// releasePlans runs under the what-if, refresh and plan-ledger serializers.
func (a *API) releasePlans(ctx context.Context, raw RawReleaseRequest) ReleasePlansReading {
	st := a.stores()
	if err := a.requirePlanFranchise(ctx, raw.ID, "release plan"); err != nil {
		return a.releaseReading(nil, err)
	}
	if err := a.releaseAction(ctx, raw); err != nil {
		return a.releaseReading(nil, err)
	}
	if err := RevalidateReleasePlans(ctx, a.host); err != nil {
		return a.releaseReading(nil, err)
	}
	receipts, err := st.Moves.List(ctx, ingestion.LeagueID, raw.Mine)
	if err != nil {
		return a.releaseReading(nil, fmt.Errorf("release list: %w", err))
	}
	out := []ReleasePlanView{}
	for _, r := range receipts {
		if !envelope.IsRelease(r.Spec.Intent) {
			continue
		}
		f, err := st.Moves.ReleaseFacts(ctx, r.CorrelationID)
		if err != nil {
			return a.releaseReading(nil, err)
		}
		out = append(out, ReleasePlanView(Plan(r, f)))
	}
	return a.releaseReading(out, nil)
}
func (a *API) releaseReading(v []ReleasePlanView, err error) ReleasePlansReading {
	return ReleasePlansReading(contractReading(a, v, "", "local envelopes; MFL mirror", "Plans are drafted from a pinned what-if move, with the engine charge in cents and its baseline stamps. Ready is not accepted. Buyouts require offseason, 2–4 years left and fewer than two other landed/open plans this season. MFL is never submitted to or written. Only explicitly player-labelled dead-cap adjustments can verify a charge; franchise totals cannot.", len(v), err))
}

func (a *API) releaseAction(ctx context.Context, r RawReleaseRequest) error {
	st := a.stores()
	if r.Action == "read" {
		return nil
	}
	if r.Action == "draft" {
		return a.draftRelease(ctx, r)
	}
	e, err := st.Moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("release plan: %w", err)
	}
	spec := e.Receipt().Spec
	if !envelope.IsRelease(spec.Intent) || spec.FranchiseID != r.Mine {
		return fmt.Errorf("release: plan outside scope")
	}
	if err := RevalidateReleasePlans(ctx, a.host); err != nil {
		return err
	}
	e, err = st.Moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("release reload: %w", err)
	}
	if r.Action == "rebase" {
		return a.rebaseRelease(ctx, e)
	}
	if r.Action == "withdraw" {
		return a.withdrawRelease(ctx, e)
	}
	next, err := e.HandOffRelease(a.host.ContractsNow(), r.Confirmed)
	if err != nil {
		return fmt.Errorf("release hand-off: %w", err)
	}
	if spec.Intent == envelope.CutIntent {
		if err := a.host.ValidateMoveURL(spec.Target.URL); err != nil {
			return err //nolint:wrapcheck // The host returns the moves feature's existing refusal words.
		}
	}
	if err := st.Moves.Save(ctx, next); err != nil {
		return fmt.Errorf("release hand-off save: %w", err)
	}
	if spec.Intent == envelope.CutIntent {
		a.host.OpenMoveURL(ctx, spec.Target.URL)
	}
	a.host.NotifyMoves(ctx)
	a.host.WakeMoves()
	return nil
}
func (a *API) draftRelease(ctx context.Context, r RawReleaseRequest) error {
	st := a.stores()
	s, found, err := state.NewWhatIfSessions(st.WhatifPools).Load(ctx, r.Mine)
	if err != nil {
		return fmt.Errorf("release baseline: %w", err)
	}
	index := slices.IndexFunc(s.Moves, func(m state.WhatIfMove) bool { return m.ID == r.Move })
	if !found || index < 0 {
		return fmt.Errorf("release: what-if move no longer exists")
	}
	m := s.Moves[index]
	f, err := ReleaseFacts(s, m)
	if err != nil {
		return err
	}
	intent, target := envelope.CutIntent, envelope.Target{Kind: envelope.Manual}
	if m.Kind == string(transactions.KindBuyout) {
		intent = envelope.BuyoutIntent
	} else {
		target = a.host.ReleaseTarget()
	}
	check, err := releaseCheck(ctx, st, r.Mine, "", f)
	if err != nil {
		return err
	}
	spec := envelope.Spec{Intent: intent, LeagueID: ingestion.LeagueID, FranchiseID: r.Mine,
		Subject: envelope.Subject{Players: []playerid.PlayerID{f.Player}}, Expected: envelope.ExpectedChange{Player: f.Player}, Gravity: envelope.G3, Undo: envelope.Irreversible, Target: target}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Errorf("release id: %w", err)
	}
	id := "release-" + hex.EncodeToString(random[:])
	e, err := envelope.DraftRelease(a.host.ContractsNow(), spec, check, func() string { return id })
	if err != nil {
		return fmt.Errorf("release draft: %w", err)
	}
	if err := st.Moves.SaveRelease(ctx, e, f); err != nil {
		return fmt.Errorf("release save: %w", err)
	}
	a.host.NotifyMoves(ctx)
	return nil
}
func releaseCheck(ctx context.Context, st Stores, mine, id string, f envelope.ReleaseFacts) (envelope.ReleaseCheck, error) {
	m, _, _ := st.League.ContractsView()
	p, found := Player(m, mine, f.Player.String())
	phase, err := st.Whatif.CurrentPhase(ctx)
	if err != nil {
		return envelope.ReleaseCheck{}, fmt.Errorf("release phase: %w", err)
	}
	count, err := ReleaseBuyoutCount(ctx, st, mine, id, m.Season)
	if err != nil {
		return envelope.ReleaseCheck{}, err
	}
	return envelope.ReleaseCheck{OnRoster: found, Offseason: phase == domain.PhaseOffseason, OtherBuyouts: count, Remaining: p.ContractYear - m.Season, ChargeText: snapshot.ContractAmountOf(f.Charge).Text}, nil
}

// ReleaseBuyoutCount is how many other buyout plans of the franchise are open or landed in season.
func ReleaseBuyoutCount(ctx context.Context, st Stores, mine, except string, season int) (int, error) {
	receipts, err := st.Moves.List(ctx, ingestion.LeagueID, mine)
	if err != nil {
		return 0, fmt.Errorf("release buyout count: %w", err)
	}
	n := 0
	for _, r := range receipts {
		if r.CorrelationID == except || r.Spec.Intent != envelope.BuyoutIntent || (!envelope.ReleaseOpen(r.State) && r.State != envelope.Landed) {
			continue
		}
		f, err := st.Moves.ReleaseFacts(ctx, r.CorrelationID)
		if err != nil {
			return 0, fmt.Errorf("release count facts: %w", err)
		}
		if f.Season == season {
			n++
		}
	}
	return n, nil
}

func (a *API) withdrawRelease(ctx context.Context, e envelope.Envelope) error {
	note := WithdrawNote(e.Receipt())
	next, err := e.WithdrawRelease(a.host.ContractsNow(), note)
	if err != nil {
		return fmt.Errorf("release withdraw: %w", err)
	}
	if err := a.stores().Moves.Save(ctx, next); err != nil {
		return fmt.Errorf("release withdraw save: %w", err)
	}
	a.host.NotifyMoves(ctx)
	return nil
}

func (a *API) rebaseRelease(ctx context.Context, e envelope.Envelope) error {
	st := a.stores()
	if e.State() != envelope.Stale {
		return fmt.Errorf("release rebase: only a stale plan can be repriced")
	}
	r := e.Receipt()
	old, err := st.Moves.ReleaseFacts(ctx, e.ID())
	if err != nil {
		return fmt.Errorf("release facts: %w", err)
	}
	m, _, _ := st.League.ContractsView()
	stamps, err := st.League.Stamps(ctx)
	if err != nil {
		return fmt.Errorf("release rebase stamps: %w", err)
	}
	kind := string(transactions.KindWaiver)
	if r.Spec.Intent == envelope.BuyoutIntent {
		kind = string(transactions.KindBuyout)
	}
	s := state.WhatIfSession{Mine: r.Spec.FranchiseID, Snapshot: m, Stamps: stamps, Generation: "release-rebase:" + e.ID() + ":" + fmt.Sprint(time.Now().UnixNano()), TaxiPercent: st.Rulebook.TaxiCapPercent(), IRPercent: st.Rulebook.IRCapPercent()}
	move := state.WhatIfMove{Kind: kind, Player: old.Player.String(), Name: old.Name}
	f, err := ReleaseFacts(s, move)
	if err != nil {
		return err
	}
	f.Charge, err = releaseCharge(ctx, st, s, move)
	if err != nil {
		return err
	}
	check, err := releaseCheck(ctx, st, r.Spec.FranchiseID, e.ID(), f)
	if err != nil {
		return err
	}
	note := snapshot.ContractAmountOf(old.Charge).Text + " → " + snapshot.ContractAmountOf(f.Charge).Text + " dead cap on current mirror"
	check.ChargeText = note
	e, err = e.Rebase(a.host.ContractsNow(), note)
	if err == nil {
		e, err = e.Check(a.host.ContractsNow(), snapshot.Snapshot{}, check)
	}
	if err != nil {
		return fmt.Errorf("release rebase: %w", err)
	}
	if err := st.Moves.SaveRelease(ctx, e, f); err != nil {
		return fmt.Errorf("release rebase save: %w", err)
	}
	a.host.NotifyMoves(ctx)
	return nil
}
func releaseCharge(ctx context.Context, st Stores, s state.WhatIfSession, m state.WhatIfMove) (domain.Money, error) {
	store, err := LoadWhatIf(ctx, st.WhatifPools, s)
	if err != nil {
		return 0, err
	}
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		return 0, fmt.Errorf("release rebase coordinator: %w", err)
	}
	req, err := buildRequest(TransactionRequest{Kind: m.Kind, MFLID: m.Player})
	if err != nil {
		return 0, fmt.Errorf("release rebase request: %w", err)
	}
	receipt, err := coord.Execute(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("release rebase engine: %w", err)
	}
	if len(receipt.CapDeltas) != 1 {
		return 0, fmt.Errorf("release rebase: engine charge missing")
	}
	return receipt.CapDeltas[0].Cents, nil
}

// RevalidateReleasePlans re-checks every open release plan against the mirror. The caller holds the
// refresh and plan-ledger serializers; this touches only the local plan ledger.
func RevalidateReleasePlans(ctx context.Context, h PlanLedger) error {
	st := StoresOf(h)
	receipts, err := st.Moves.List(ctx, ingestion.LeagueID, "")
	if err != nil {
		return fmt.Errorf("release validation list: %w", err)
	}
	for _, r := range receipts {
		if !envelope.IsRelease(r.Spec.Intent) {
			continue
		}
		if err := revalidateRelease(ctx, h, st, r); err != nil {
			return err
		}
	}
	return nil
}
func revalidateRelease(ctx context.Context, h PlanLedger, st Stores, r envelope.Receipt) error {
	f, err := st.Moves.ReleaseFacts(ctx, r.CorrelationID)
	if err != nil {
		return fmt.Errorf("release facts: %w", err)
	}
	m, _, _ := st.League.ContractsView()
	if r.State == envelope.Landed {
		return releaseChargeNote(ctx, h, st, r, f, m)
	}
	if !NeedsValidation(r.State) {
		return nil
	}
	p, onRoster := Player(m, r.Spec.FranchiseID, f.Player.String())
	stamps, err := st.League.ReleaseStamps(ctx, f.Player.String(), r.Spec.FranchiseID)
	if err != nil {
		return fmt.Errorf("release validation stamps: %w", err)
	}
	adjustments, err := Adjustments(m, r.Spec.FranchiseID)
	if err != nil {
		return err
	}
	phase, err := st.Whatif.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("release validation phase: %w", err)
	}
	contractChanged := ContractChanged(p, f) ||
		(state.MirrorStamps{Contract: stamps.Contract}).NewerThan(state.MirrorStamps{Contract: f.ContractStamp})
	adjustmentChanged := adjustments != f.Adjustments || (state.MirrorStamps{DeadCap: stamps.DeadCap}).NewerThan(state.MirrorStamps{DeadCap: f.DeadCapStamp})
	reason := envelope.ReleaseInvalidation(r.Spec.Intent, r.State != envelope.Ready, onRoster, phase == domain.PhaseOffseason, contractChanged, adjustmentChanged)
	if reason == "" {
		return nil
	}
	e, err := st.Moves.Get(ctx, r.CorrelationID)
	if err != nil {
		return fmt.Errorf("release invalidate load: %w", err)
	}
	next, err := e.Invalidate(h.ContractsNow(), reason)
	if err != nil {
		return fmt.Errorf("release invalidate: %w", err)
	}
	if err := st.Moves.Save(ctx, next); err != nil {
		return fmt.Errorf("release invalidate save: %w", err)
	}
	h.NotifyMoves(ctx)
	return nil
}
func releaseChargeNote(ctx context.Context, h PlanLedger, st Stores, r envelope.Receipt, f envelope.ReleaseFacts, m state.MirrorSnapshot) error {
	note := ChargeNote(f, r.Spec.FranchiseID, m)
	if len(r.Audit) > 0 && r.Audit[len(r.Audit)-1].Note == note {
		return nil
	}
	e, err := st.Moves.Get(ctx, r.CorrelationID)
	if err != nil {
		return fmt.Errorf("release charge note load: %w", err)
	}
	next, err := e.Invalidate(h.ContractsNow(), note)
	if err != nil {
		return fmt.Errorf("release charge note: %w", err)
	}
	if err := st.Moves.Save(ctx, next); err != nil {
		return fmt.Errorf("release charge note save: %w", err)
	}
	h.NotifyMoves(ctx)
	return nil
}
