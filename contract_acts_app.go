package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type RawContractAct struct {
	Mine        string `json:"mine"`
	Player      string `json:"player"`
	Action      string `json:"action"`
	Intent      string `json:"intent"`
	ID          string `json:"id"`
	Level       int    `json:"level"`
	Years       int    `json:"years"`
	Source      int    `json:"source"`
	Destination int    `json:"destination"`
	Move        string `json:"move"`
}

func (r RawContractAct) Validate() error {
	return contractsapp.Act(r).Validate() //nolint:wrapcheck // the service's errors already start "contract Act"
}

type ContractActFigure struct {
	Season int    `json:"season"`
	Before string `json:"before"`
	After  string `json:"after"`
	Change string `json:"change"`
}
type ContractActChoice struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}
type ContractActView struct {
	ID          string              `json:"id"`
	Intent      string              `json:"intent"`
	Act         string              `json:"act"`
	Levels      []ContractActChoice `json:"levels"`
	Years       []int               `json:"years"`
	TeamFigures []WhatIfFigure      `json:"teamFigures"`
	Player      string              `json:"player"`
	Subject     string              `json:"subject"`
	Label       string              `json:"label"`
	Price       string              `json:"price"`
	Post        string              `json:"post"`
	How         string              `json:"how"`
	Reasons     []string            `json:"reasons"`
	Figures     []ContractActFigure `json:"figures"`
	Receipt     envelope.Receipt    `json:"receipt"`
	CanHandOff  bool                `json:"canHandOff"`
	CanPost     bool                `json:"canPost"`
	CanWithdraw bool                `json:"canWithdraw"`
	CanRebase   bool                `json:"canRebase"`
}

func (a *App) TargetContractActs(raw RawContractAct) Reading[[]ContractActView] {
	if err := raw.Validate(); err != nil {
		return contractActsReading(nil, err)
	}
	if err := a.ready(); err != nil {
		return contractActsReading(nil, err)
	}
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	if err := a.revalidateContractPlans(ctx); err != nil {
		return contractActsReading(nil, err)
	}
	if raw.Action == "options" {
		return a.contractActOptions(ctx, raw)
	}
	if raw.Action == "preview" {
		f, err := a.contractActFacts(ctx, raw)
		if err != nil {
			return contractActsReading(nil, err)
		}
		return contractActsReading([]ContractActView{contractActView(envelope.Receipt{}, f)}, nil)
	}
	if err := a.contractActAction(ctx, raw); err != nil {
		return contractActsReading(nil, err)
	}
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, raw.Mine)
	if err != nil {
		return contractActsReading(nil, fmt.Errorf("contract plans: %w", err))
	}
	out := []ContractActView{}
	for _, r := range receipts {
		if !envelope.IsContract(r.Spec.Intent) {
			continue
		}
		f, err := a.moves.ContractFacts(ctx, r.CorrelationID)
		if err != nil {
			return contractActsReading(nil, err)
		}
		out = append(out, contractActView(r, f))
	}
	return contractActsReading(out, nil)
}
func contractActsReading(v []ContractActView, err error) Reading[[]ContractActView] {
	return contractReading(v, "", "local contract plans; MFL mirror", "Rulebook §§7,9–11; Math Rules §§0,3,5–7. All calculations in exact cents, then nearest $10,000. Future baseline cells are flat-filled inferences, not observations. Tag, extension and restructure follow the coordinator phase policy. Tender and re-sign are offseason only. Unknown history is for commissioner verification, not a block. Plans never write MFL or post to ProBoards.", len(v), err)
}

func (a *App) contractActFacts(ctx context.Context, r RawContractAct) (envelope.ContractFacts, error) {
	m, _, _ := a.league.ContractsView()
	p, found := contractsapp.Player(m, r.Mine, r.Player)
	if !found {
		return envelope.ContractFacts{}, fmt.Errorf("contract Act: player is not on my roster")
	}
	phase, err := a.whatif.CurrentPhase(ctx)
	if err != nil {
		return envelope.ContractFacts{}, fmt.Errorf("contract Act phase: %w", err)
	}
	a.targetSnapshotMu.Lock()
	directory := a.targetSnapshot
	a.targetSnapshotMu.Unlock()
	rows := snapshot.ContractRows(m, directory, a.rulebook.FranchiseNames())
	f, priced, err := contractsapp.ActFacts(m, p, contractsapp.Act(r), envelope.ContractWindow(r.Intent, phase), rows)
	if err != nil {
		return f, err //nolint:wrapcheck // the service's errors already start "contract Act"
	}
	if priced {
		if err := a.contractCapFacts(ctx, r.Mine, &f); err != nil {
			return f, err
		}
	}

	return f, nil
}

// mirrorContractEvidence is a held contract as an Act sees it; the moves watcher reads it too.
func mirrorContractEvidence(m state.MirrorSnapshot, mine, player string) envelope.ContractEvidence {
	return contractsapp.Evidence(m, mine, player)
}

// contractActView is an Act as the owner reads it, as its wire type.
func contractActView(r envelope.Receipt, f envelope.ContractFacts) ContractActView {
	return wireContractAct(contractsapp.DescribeAct(r, f))
}

// wireContractAct converts the service's Act to its wire twin; TestContractsWireTwinsMatch proves it.
func wireContractAct(v contractsapp.ActView) ContractActView {
	out := ContractActView{ID: v.ID, Intent: v.Intent, Act: v.Act, Levels: make([]ContractActChoice, len(v.Levels)), Years: v.Years,
		TeamFigures: wireWhatIfFigures(v.TeamFigures), Player: v.Player, Subject: v.Subject, Label: v.Label, Price: v.Price, Post: v.Post,
		How: v.How, Reasons: v.Reasons, Figures: make([]ContractActFigure, len(v.Figures)), Receipt: v.Receipt, CanHandOff: v.CanHandOff,
		CanPost: v.CanPost, CanWithdraw: v.CanWithdraw, CanRebase: v.CanRebase}
	for i, l := range v.Levels {
		out.Levels[i] = ContractActChoice(l)
	}
	for i, f := range v.Figures {
		out.Figures[i] = ContractActFigure(f)
	}
	return out
}

func (a *App) contractActAction(ctx context.Context, r RawContractAct) error {
	if r.Action == "read" {
		return nil
	}
	if r.Action == "draft" {
		f, err := a.contractActFacts(ctx, r)
		if err != nil {
			return err
		}
		if err := a.refuseSecondContractPlan(ctx, r.Mine, f); err != nil {
			return err
		}
		var bytes [16]byte
		if _, err := rand.Read(bytes[:]); err != nil {
			return fmt.Errorf("contract plan id: %w", err)
		}
		spec := envelope.Spec{Intent: r.Intent, LeagueID: ingestion.LeagueID, FranchiseID: r.Mine, Subject: envelope.Subject{Players: []playerid.PlayerID{f.Player}}, Expected: envelope.ExpectedChange{Player: f.Player}, Gravity: envelope.G3, Undo: envelope.Irreversible, Target: envelope.Target{Kind: envelope.Manual}}
		e, err := envelope.DraftContract(a.movesNow(), spec, f, func() string { return "contract-" + hex.EncodeToString(bytes[:]) })
		if err != nil {
			return fmt.Errorf("contract draft: %w", err)
		}
		if err := a.moves.SaveContract(ctx, e, f); err != nil {
			return fmt.Errorf("contract draft save: %w", err)
		}
		a.movesChanged(ctx)
		return nil
	}
	e, err := a.moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("contract plan: %w", err)
	}
	if !envelope.IsContract(e.Receipt().Spec.Intent) || e.Receipt().Spec.FranchiseID != r.Mine {
		return fmt.Errorf("contract plan outside scope")
	}
	f, err := a.moves.ContractFacts(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("contract facts: %w", err)
	}
	next, err := a.contractActTransition(ctx, r, e, f)
	if err != nil {
		return err
	}
	if err := a.moves.Save(ctx, next); err != nil {
		return fmt.Errorf("contract action save: %w", err)
	}
	a.movesChanged(ctx)
	a.wakeMoves()
	return nil
}
func (a *App) refuseSecondContractPlan(ctx context.Context, mine string, f envelope.ContractFacts) error {
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, mine)
	if err != nil {
		return fmt.Errorf("contract plans: %w", err)
	}
	for _, r := range receipts {
		if r.Spec.Intent == f.Intent && r.Spec.Expected.Player == f.Player && envelope.ReleaseOpen(r.State) {
			return fmt.Errorf("%s · %s is already an open plan in My moves; remove it there before keeping another", contractsapp.IntentLabel(f.Intent), f.Name)
		}
	}
	return nil
}
func (a *App) contractActTransition(ctx context.Context, r RawContractAct, e envelope.Envelope, f envelope.ContractFacts) (envelope.Envelope, error) {
	now := a.movesNow()
	switch r.Action {
	case "handoff":
		next, err := e.HandOffManual(now, "Contract request copied for the commissioner on ProBoards; nothing posted")
		return contractTransitionResult(next, err)
	case "posted":
		next, err := e.ContractPosted(now)
		return contractTransitionResult(next, err)
	case "withdraw":
		next, err := e.WithdrawContract(now)
		return contractTransitionResult(next, err)
	case "rebase":
		fresh, err := a.contractActFacts(ctx, RawContractAct{Mine: r.Mine, Player: f.Player.String(), Intent: f.Intent, Level: f.Level, Years: f.Years, Source: f.Source, Destination: f.Destination, Move: fmt.Sprintf("%d.%08d", f.Move/contractacts.Million, f.Move%contractacts.Million)})
		if err != nil {
			return envelope.Envelope{}, err
		}
		draft, err := e.Rebase(now, "Repriced from today's MFL mirror; rules and missing evidence checked again")
		if err != nil {
			return envelope.Envelope{}, fmt.Errorf("contract rebase: %w", err)
		}
		checked, err := draft.Check(now, snapshot.Snapshot{}, envelope.ContractCheck{Reasons: fresh.Reasons})
		if err != nil {
			return envelope.Envelope{}, fmt.Errorf("contract recheck: %w", err)
		}
		if err := a.moves.SaveContract(ctx, checked, fresh); err != nil {
			return envelope.Envelope{}, fmt.Errorf("contract rebase save: %w", err)
		}
		return checked, nil
	default:
		return envelope.Envelope{}, fmt.Errorf("unsupported contract action")
	}
}

func (a *App) revalidateContractPlans(ctx context.Context) error {
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, "")
	if err != nil {
		return fmt.Errorf("contract validation list: %w", err)
	}
	m, _, _ := a.league.ContractsView()
	phase, err := a.whatif.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("contract validation phase: %w", err)
	}
	for _, r := range receipts {
		if !envelope.IsContract(r.Spec.Intent) || !envelope.ReleaseOpen(r.State) || r.State == envelope.Stale {
			continue
		}
		f, err := a.moves.ContractFacts(ctx, r.CorrelationID)
		if err != nil {
			return fmt.Errorf("contract validation facts: %w", err)
		}
		window := ""
		if r.State == envelope.Ready {
			window = envelope.ContractWindow(f.Intent, phase)
		}
		reason := envelope.ContractInvalidation(f, mirrorContractEvidence(m, r.Spec.FranchiseID, f.Player.String()), window)
		if !contractsapp.NeedsInvalidation(r.State, reason) {
			continue
		}
		e, err := a.moves.Get(ctx, r.CorrelationID)
		if err != nil {
			return fmt.Errorf("contract validation load: %w", err)
		}
		next, err := e.Invalidate(a.movesNow(), reason)
		if err != nil {
			return fmt.Errorf("contract invalidate: %w", err)
		}
		if err := a.moves.Save(ctx, next); err != nil {
			return fmt.Errorf("contract invalidate save: %w", err)
		}
		a.movesChanged(ctx)
	}
	return nil
}

func contractTransitionResult(next envelope.Envelope, err error) (envelope.Envelope, error) {
	if err != nil {
		return next, fmt.Errorf("contract transition: %w", err)
	}
	return next, nil
}
func (a *App) contractActOptions(ctx context.Context, raw RawContractAct) Reading[[]ContractActView] {
	out := []ContractActView{}
	for _, intent := range []string{"rfa.tender", "fa.resign", "contract.tag", "contract.extend", "contract.restructure"} {
		request := RawContractAct{Mine: raw.Mine, Player: raw.Player, Intent: intent, Level: 1, Years: 1, Move: "1"}
		m, _, _ := a.league.ContractsView()
		request.Source = m.Season
		if p, ok := contractsapp.Player(m, raw.Mine, raw.Player); ok {
			request.Destination = p.ContractYear
		}
		f, err := a.contractActFacts(ctx, request)
		if err != nil {
			return contractActsReading(nil, err)
		}
		if !slices.Contains(f.Reasons, contractsapp.Ineligible) {
			out = append(out, contractActView(envelope.Receipt{}, f))
		}
	}
	return contractActsReading(out, nil)
}
