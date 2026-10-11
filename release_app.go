package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

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
	if err := (snapshot.RawContractScope{Mine: r.Mine, Sort: "name"}).Validate(); err != nil {
		return fmt.Errorf("release scope: %w", err)
	}
	if r.Mine == "" || !slices.Contains([]string{"read", "draft", "handoff", "rebase", "withdraw"}, r.Action) {
		return fmt.Errorf("release: franchise and known action required")
	}
	if r.Action == "draft" && r.Move == "" {
		return fmt.Errorf("release: what-if move required")
	}
	if (r.Action == "handoff" || r.Action == "rebase" || r.Action == "withdraw") && r.ID == "" {
		return fmt.Errorf("release: plan id required")
	}
	return nil
}

type ReleasePlanView struct {
	ID            string           `json:"id"`
	Player        string           `json:"player"`
	Label         string           `json:"label"`
	Charge        string           `json:"charge"`
	Confirmation  string           `json:"confirmation"`
	Post          string           `json:"post"`
	Channel       string           `json:"channel"`
	Receipt       envelope.Receipt `json:"receipt"`
	CanHandOff    bool             `json:"canHandOff"`
	CanRebase     bool             `json:"canRebase"`
	RebaseLabel   string           `json:"rebaseLabel"`
	WithdrawLabel string           `json:"withdrawLabel"`
	Cut           bool             `json:"cut"`
}

func (a *App) TargetReleasePlans(raw RawReleaseRequest) Reading[[]ReleasePlanView] {
	if err := raw.Validate(); err != nil {
		return releaseReading(nil, err)
	}
	if err := a.ready(); err != nil {
		return releaseReading(nil, err)
	}
	a.whatifMu.Lock()
	defer a.whatifMu.Unlock()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	if err := a.releaseAction(ctx, raw); err != nil {
		return releaseReading(nil, err)
	}
	if err := a.revalidateReleasePlans(ctx); err != nil {
		return releaseReading(nil, err)
	}
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, raw.Mine)
	if err != nil {
		return releaseReading(nil, fmt.Errorf("release list: %w", err))
	}
	out := []ReleasePlanView{}
	for _, r := range receipts {
		if !envelope.IsRelease(r.Spec.Intent) {
			continue
		}
		f, err := a.moves.ReleaseFacts(ctx, r.CorrelationID)
		if err != nil {
			return releaseReading(nil, err)
		}
		out = append(out, releasePlanView(r, f))
	}
	return releaseReading(out, nil)
}
func releaseReading(v []ReleasePlanView, err error) Reading[[]ReleasePlanView] {
	return contractReading(v, "", "local envelopes; MFL mirror", "Plans are drafted from a pinned what-if move, with the engine charge in cents and its baseline stamps. Ready is not accepted. Buyouts require offseason, 2–4 years left and fewer than two other landed/open plans this season. MFL is never submitted to or written. Only explicitly player-labelled dead-cap adjustments can verify a charge; franchise totals cannot.", len(v), err)
}
func releasePlanView(r envelope.Receipt, f envelope.ReleaseFacts) ReleasePlanView {
	cut := r.Spec.Intent == envelope.CutIntent
	verb, channel := "Buy out ", "Buyout channel not stated in §12; confirm with the commissioner before posting."
	if cut {
		verb, channel = "Cut ", "Release on MFL; post player, contract and cap hit on ProBoards (§8)."
	}
	cost := snapshot.ContractAmountOf(f.Charge).Text
	return ReleasePlanView{ID: r.CorrelationID, Player: f.Name, Label: verb + f.Name, Charge: cost,
		Confirmation: fmt.Sprintf("This releases %s; %s dead cap this season. It cannot be undone.", f.Name, cost),
		Post:         fmt.Sprintf("%s\n%s: annual salary %s; contract expires %d; %d paid years after %d.\nPlanned dead cap this season: %s. Please process this %s request.\nCannot %s this player for the rest of %s.", verb+f.Name, f.Name, snapshot.ContractAmountOf(f.Salary).Text, f.Expiration, max(0, f.Expiration-f.Season), f.Season, cost, strings.TrimSpace(verb), releaseLockVerb(cut), releaseLockSeason(cut)),
		Channel:      channel, Receipt: r, CanHandOff: r.State == envelope.Ready, CanRebase: r.State == envelope.Stale, RebaseLabel: "Reprice from today's MFL rosters", WithdrawLabel: releaseWithdrawLabel(r.State, cut), Cut: cut}
}
func releaseWithdrawLabel(s envelope.State, cut bool) string {
	if s == envelope.Ready || s == envelope.Blocked {
		return "Remove plan"
	}
	if s == envelope.HandedOff || s == envelope.NotYetDone {
		if cut {
			return "I didn't drop him"
		}
		return "I didn't post it"
	}
	return ""
}

func releaseLockVerb(cut bool) string {
	if cut {
		return "re-sign"
	}
	return "bid on"
}
func releaseLockSeason(cut bool) string {
	if cut {
		return "this season"
	}
	return "this season; eligible again the following offseason"
}

func (a *App) releaseAction(ctx context.Context, r RawReleaseRequest) error {
	if r.Action == "read" {
		return nil
	}
	if r.Action == "draft" {
		return a.draftRelease(ctx, r)
	}
	e, err := a.moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("release plan: %w", err)
	}
	spec := e.Receipt().Spec
	if !envelope.IsRelease(spec.Intent) || spec.FranchiseID != r.Mine {
		return fmt.Errorf("release: plan outside scope")
	}
	if err := a.revalidateReleasePlans(ctx); err != nil {
		return err
	}
	e, err = a.moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("release reload: %w", err)
	}
	if r.Action == "rebase" {
		return a.rebaseRelease(ctx, e)
	}
	if r.Action == "withdraw" {
		return a.withdrawRelease(ctx, e)
	}
	next, err := e.HandOffRelease(a.movesNow(), r.Confirmed)
	if err != nil {
		return fmt.Errorf("release hand-off: %w", err)
	}
	if spec.Intent == envelope.CutIntent {
		if err := validateMoveURL(spec.Target.URL); err != nil {
			return err
		}
	}
	if err := a.moves.Save(ctx, next); err != nil {
		return fmt.Errorf("release hand-off save: %w", err)
	}
	if spec.Intent == envelope.CutIntent {
		a.openURL(ctx, spec.Target.URL)
	}
	a.movesChanged(ctx)
	a.wakeMoves()
	return nil
}
func (a *App) draftRelease(ctx context.Context, r RawReleaseRequest) error {
	s, found, err := state.NewWhatIfSessions(a.whatifPools).Load(ctx, r.Mine)
	if err != nil {
		return fmt.Errorf("release baseline: %w", err)
	}
	index := slices.IndexFunc(s.Moves, func(m state.WhatIfMove) bool { return m.ID == r.Move })
	if !found || index < 0 {
		return fmt.Errorf("release: what-if move no longer exists")
	}
	m := s.Moves[index]
	f, err := releaseFacts(s, m)
	if err != nil {
		return err
	}
	intent, target := envelope.CutIntent, envelope.Target{Kind: envelope.Manual}
	if m.Kind == string(transactions.KindBuyout) {
		intent = envelope.BuyoutIntent
	} else {
		target, _ = a.rosterMoveTarget("257")
	}
	check, err := a.releaseCheck(ctx, r.Mine, intent, "", f)
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
	e, err := envelope.DraftRelease(a.movesNow(), spec, check, func() string { return id })
	if err != nil {
		return fmt.Errorf("release draft: %w", err)
	}
	if err := a.moves.SaveRelease(ctx, e, f); err != nil {
		return fmt.Errorf("release save: %w", err)
	}
	a.movesChanged(ctx)
	return nil
}
func releaseFacts(s state.WhatIfSession, m state.WhatIfMove) (envelope.ReleaseFacts, error) {
	pid, err := playerid.New(m.Player)
	if err != nil {
		return envelope.ReleaseFacts{}, fmt.Errorf("release player: %w", err)
	}
	p, found := releasePlayer(s.Snapshot, s.Mine, m.Player)
	if !found {
		return envelope.ReleaseFacts{}, fmt.Errorf("release: player not in what-if baseline")
	}
	adjustments, err := releaseAdjustments(s.Snapshot, s.Mine)
	if err != nil {
		return envelope.ReleaseFacts{}, err
	}
	return envelope.ReleaseFacts{Player: pid, Name: m.Name, Season: s.Snapshot.Season, Charge: m.Charge, Salary: p.Salary, Expiration: p.ContractYear, ContractInfo: p.ContractInfo,
		RostersStamp: s.Stamps.Rosters, ContractStamp: s.Stamps.Contract, DeadCapStamp: s.Stamps.DeadCap, Adjustments: adjustments}, nil
}
func releasePlayer(m state.MirrorSnapshot, mine, player string) (state.MirrorPlayer, bool) {
	for _, p := range m.Players {
		if p.MFLID == player && p.FranchiseID == mine {
			return p, true
		}
	}
	return state.MirrorPlayer{}, false
}
func releaseAdjustments(m state.MirrorSnapshot, mine string) (string, error) {
	rows := []state.Adjustment{}
	for _, r := range m.Adjustments {
		if r.FranchiseID == mine {
			rows = append(rows, r)
		}
	}
	body, err := json.Marshal(rows)
	if err != nil {
		return "", fmt.Errorf("release adjustments: %w", err)
	}
	return string(body), nil
}
func (a *App) releaseCheck(ctx context.Context, mine, _, id string, f envelope.ReleaseFacts) (envelope.ReleaseCheck, error) {
	m, _, _ := a.league.ContractsView()
	p, found := releasePlayer(m, mine, f.Player.String())
	phase, err := a.whatif.CurrentPhase(ctx)
	if err != nil {
		return envelope.ReleaseCheck{}, fmt.Errorf("release phase: %w", err)
	}
	count, err := a.releaseBuyoutCount(ctx, mine, id, m.Season)
	if err != nil {
		return envelope.ReleaseCheck{}, err
	}
	return envelope.ReleaseCheck{OnRoster: found, Offseason: phase == domain.PhaseOffseason, OtherBuyouts: count, Remaining: p.ContractYear - m.Season, ChargeText: snapshot.ContractAmountOf(f.Charge).Text}, nil
}
func (a *App) releaseBuyoutCount(ctx context.Context, mine, except string, season int) (int, error) {
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, mine)
	if err != nil {
		return 0, fmt.Errorf("release buyout count: %w", err)
	}
	n := 0
	for _, r := range receipts {
		if r.CorrelationID == except || r.Spec.Intent != envelope.BuyoutIntent || (!envelope.ReleaseOpen(r.State) && r.State != envelope.Landed) {
			continue
		}
		f, err := a.moves.ReleaseFacts(ctx, r.CorrelationID)
		if err != nil {
			return 0, fmt.Errorf("release count facts: %w", err)
		}
		if f.Season == season {
			n++
		}
	}
	return n, nil
}

func releaseStateLabel(s envelope.State) string {
	labels := map[envelope.State]string{envelope.Ready: "Ready", envelope.Blocked: "Blocked", envelope.Stale: "Stale", envelope.HandedOff: "Handed off", envelope.NotYetDone: "Not yet done", envelope.NotVerified: "Not verified", envelope.Draft: "Draft"}
	return labels[s]
}

func (a *App) withdrawRelease(ctx context.Context, e envelope.Envelope) error {
	note := "Withdrawn by you before it was handed off."
	if e.State() == envelope.HandedOff || e.State() == envelope.NotYetDone {
		note = "You said you did not post it, so the plan is withdrawn."
		if e.Receipt().Spec.Intent == envelope.CutIntent {
			note = "You said you did not drop him, so the plan is withdrawn."
		}
	}
	next, err := e.WithdrawRelease(a.movesNow(), note)
	if err != nil {
		return fmt.Errorf("release withdraw: %w", err)
	}
	if err := a.moves.Save(ctx, next); err != nil {
		return fmt.Errorf("release withdraw save: %w", err)
	}
	a.movesChanged(ctx)
	return nil
}
