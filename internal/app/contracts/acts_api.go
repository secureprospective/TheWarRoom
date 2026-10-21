package contracts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
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
	return Act(r).Validate()
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

func (a *API) TargetContractActs(raw RawContractAct) ContractActsReading {
	if err := raw.Validate(); err != nil {
		return a.contractActsReading(nil, err)
	}
	if err := a.host.Ready(); err != nil {
		return a.contractActsReading(nil, err)
	}
	var out ContractActsReading
	a.host.SerializeContractActs(func() {
		ctx, cancel := context.WithTimeout(a.host.ContractsContext(), timeout)
		defer cancel()
		out = a.contractActs(ctx, raw)
	})
	return out
}

// contractActs runs under the refresh and plan-ledger serializers.
func (a *API) contractActs(ctx context.Context, raw RawContractAct) ContractActsReading {
	st := a.stores()
	if err := RevalidateContractPlans(ctx, a.host); err != nil {
		return a.contractActsReading(nil, err)
	}
	if raw.Action == "options" {
		return a.contractActOptions(ctx, raw)
	}
	if raw.Action == "preview" {
		f, err := a.contractActFacts(ctx, raw)
		if err != nil {
			return a.contractActsReading(nil, err)
		}
		return a.contractActsReading([]ContractActView{contractActView(envelope.Receipt{}, f)}, nil)
	}
	if err := a.contractActAction(ctx, raw); err != nil {
		return a.contractActsReading(nil, err)
	}
	receipts, err := st.Moves.List(ctx, ingestion.LeagueID, raw.Mine)
	if err != nil {
		return a.contractActsReading(nil, fmt.Errorf("contract plans: %w", err))
	}
	out := []ContractActView{}
	for _, r := range receipts {
		if !envelope.IsContract(r.Spec.Intent) {
			continue
		}
		f, err := st.Moves.ContractFacts(ctx, r.CorrelationID)
		if err != nil {
			return a.contractActsReading(nil, err)
		}
		out = append(out, contractActView(r, f))
	}
	return a.contractActsReading(out, nil)
}
func (a *API) contractActsReading(v []ContractActView, err error) ContractActsReading {
	return ContractActsReading(contractReading(a, v, "", "local contract plans; MFL mirror", "Rulebook §§7,9–11; Math Rules §§0,3,5–7. All calculations in exact cents, then nearest $10,000. Future baseline cells are flat-filled inferences, not observations. Tag, extension and restructure follow the coordinator phase policy. Tender and re-sign are offseason only. Unknown history is for commissioner verification, not a block. Plans never write MFL or post to ProBoards.", len(v), err))
}

func (a *API) contractActFacts(ctx context.Context, r RawContractAct) (envelope.ContractFacts, error) {
	st := a.stores()
	m, _, _ := st.League.ContractsView()
	p, found := Player(m, r.Mine, r.Player)
	if !found {
		return envelope.ContractFacts{}, fmt.Errorf("contract Act: player is not on my roster")
	}
	phase, err := st.Whatif.CurrentPhase(ctx)
	if err != nil {
		return envelope.ContractFacts{}, fmt.Errorf("contract Act phase: %w", err)
	}
	directory, _ := a.host.ContractsDirectory()
	rows := snapshot.ContractRows(m, directory, st.Rulebook.FranchiseNames())
	f, priced, err := ActFacts(m, p, Act(r), envelope.ContractWindow(r.Intent, phase), rows)
	if err != nil {
		return f, err
	}
	if priced {
		if err := a.contractCapFacts(ctx, r.Mine, &f); err != nil {
			return f, err
		}
	}

	return f, nil
}

// contractActView is an Act as the owner reads it, as its wire type.
func contractActView(r envelope.Receipt, f envelope.ContractFacts) ContractActView {
	return wireContractAct(DescribeAct(r, f))
}

// wireContractAct converts the service's Act to its wire twin; TestWireTwinsMatch proves it.
func wireContractAct(v ActView) ContractActView {
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

func (a *API) contractActAction(ctx context.Context, r RawContractAct) error {
	st := a.stores()
	if r.Action == "read" {
		return nil
	}
	if r.Action == "draft" {
		f, err := a.contractActFacts(ctx, r)
		if err != nil {
			return err
		}
		if err := refuseSecondContractPlan(ctx, st, r.Mine, f); err != nil {
			return err
		}
		var bytes [16]byte
		if _, err := rand.Read(bytes[:]); err != nil {
			return fmt.Errorf("contract plan id: %w", err)
		}
		spec := envelope.Spec{Intent: r.Intent, LeagueID: ingestion.LeagueID, FranchiseID: r.Mine, Subject: envelope.Subject{Players: []playerid.PlayerID{f.Player}}, Expected: envelope.ExpectedChange{Player: f.Player}, Gravity: envelope.G3, Undo: envelope.Irreversible, Target: envelope.Target{Kind: envelope.Manual}}
		e, err := envelope.DraftContract(a.host.ContractsNow(), spec, f, func() string { return "contract-" + hex.EncodeToString(bytes[:]) })
		if err != nil {
			return fmt.Errorf("contract draft: %w", err)
		}
		if err := st.Moves.SaveContract(ctx, e, f); err != nil {
			return fmt.Errorf("contract draft save: %w", err)
		}
		a.host.NotifyMoves(ctx)
		return nil
	}
	e, err := st.Moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("contract plan: %w", err)
	}
	if !envelope.IsContract(e.Receipt().Spec.Intent) || e.Receipt().Spec.FranchiseID != r.Mine {
		return fmt.Errorf("contract plan outside scope")
	}
	f, err := st.Moves.ContractFacts(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("contract facts: %w", err)
	}
	next, err := a.contractActTransition(ctx, r, e, f)
	if err != nil {
		return err
	}
	if err := st.Moves.Save(ctx, next); err != nil {
		return fmt.Errorf("contract action save: %w", err)
	}
	a.host.NotifyMoves(ctx)
	a.host.WakeMoves()
	return nil
}
func refuseSecondContractPlan(ctx context.Context, st Stores, mine string, f envelope.ContractFacts) error {
	receipts, err := st.Moves.List(ctx, ingestion.LeagueID, mine)
	if err != nil {
		return fmt.Errorf("contract plans: %w", err)
	}
	for _, r := range receipts {
		if r.Spec.Intent == f.Intent && r.Spec.Expected.Player == f.Player && envelope.ReleaseOpen(r.State) {
			return fmt.Errorf("%s · %s is already an open plan in My moves; remove it there before keeping another", IntentLabel(f.Intent), f.Name)
		}
	}
	return nil
}
func (a *API) contractActTransition(ctx context.Context, r RawContractAct, e envelope.Envelope, f envelope.ContractFacts) (envelope.Envelope, error) {
	now := a.host.ContractsNow()
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
		if err := a.stores().Moves.SaveContract(ctx, checked, fresh); err != nil {
			return envelope.Envelope{}, fmt.Errorf("contract rebase save: %w", err)
		}
		return checked, nil
	default:
		return envelope.Envelope{}, fmt.Errorf("unsupported contract action")
	}
}

// RevalidateContractPlans moves open contract plans whose held evidence changed to Stale. The caller
// holds the plan-ledger serializer (and the refresh serializer, from a binding or a refresh).
func RevalidateContractPlans(ctx context.Context, h PlanLedger) error {
	st := StoresOf(h)
	receipts, err := st.Moves.List(ctx, ingestion.LeagueID, "")
	if err != nil {
		return fmt.Errorf("contract validation list: %w", err)
	}
	m, _, _ := st.League.ContractsView()
	phase, err := st.Whatif.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("contract validation phase: %w", err)
	}
	for _, r := range receipts {
		if !envelope.IsContract(r.Spec.Intent) || !envelope.ReleaseOpen(r.State) || r.State == envelope.Stale {
			continue
		}
		f, err := st.Moves.ContractFacts(ctx, r.CorrelationID)
		if err != nil {
			return fmt.Errorf("contract validation facts: %w", err)
		}
		window := ""
		if r.State == envelope.Ready {
			window = envelope.ContractWindow(f.Intent, phase)
		}
		reason := envelope.ContractInvalidation(f, Evidence(m, r.Spec.FranchiseID, f.Player.String()), window)
		if !NeedsInvalidation(r.State, reason) {
			continue
		}
		e, err := st.Moves.Get(ctx, r.CorrelationID)
		if err != nil {
			return fmt.Errorf("contract validation load: %w", err)
		}
		next, err := e.Invalidate(h.ContractsNow(), reason)
		if err != nil {
			return fmt.Errorf("contract invalidate: %w", err)
		}
		if err := st.Moves.Save(ctx, next); err != nil {
			return fmt.Errorf("contract invalidate save: %w", err)
		}
		h.NotifyMoves(ctx)
	}
	return nil
}

func contractTransitionResult(next envelope.Envelope, err error) (envelope.Envelope, error) {
	if err != nil {
		return next, fmt.Errorf("contract transition: %w", err)
	}
	return next, nil
}
func (a *API) contractActOptions(ctx context.Context, raw RawContractAct) ContractActsReading {
	out := []ContractActView{}
	for _, intent := range []string{"rfa.tender", "fa.resign", "contract.tag", "contract.extend", "contract.restructure"} {
		request := RawContractAct{Mine: raw.Mine, Player: raw.Player, Intent: intent, Level: 1, Years: 1, Move: "1"}
		m, _, _ := a.stores().League.ContractsView()
		request.Source = m.Season
		if p, ok := Player(m, raw.Mine, raw.Player); ok {
			request.Destination = p.ContractYear
		}
		f, err := a.contractActFacts(ctx, request)
		if err != nil {
			return a.contractActsReading(nil, err)
		}
		if !slices.Contains(f.Reasons, Ineligible) {
			out = append(out, contractActView(envelope.Receipt{}, f))
		}
	}
	return a.contractActsReading(out, nil)
}

func (a *API) contractTeamCap(ctx context.Context, mine string, f envelope.ContractFacts) (contractCap, error) {
	st := a.stores()
	m, _, used := st.League.ContractsView()
	players := []state.MirrorPlayer{}
	for _, p := range m.Players {
		if p.FranchiseID == mine {
			players = append(players, p)
		}
	}
	m.Players = players
	s := state.WhatIfSession{Mine: mine, Snapshot: m, Used: used[mine], Generation: "contract-preview:" + mine + ":" + fmt.Sprint(time.Now().UnixNano()), TaxiPercent: st.Rulebook.TaxiCapPercent(), IRPercent: st.Rulebook.IRCapPercent()}
	if text := st.Rulebook.GetSalaryCap(); text != "" {
		capAmount, err := domain.ParseMoneyMillions(text)
		if err != nil {
			return contractCap{}, fmt.Errorf("contract cap: %w", err)
		}
		s.Cap = &capAmount
	}
	store, err := LoadWhatIf(ctx, st.WhatifPools, s)
	if err != nil {
		return contractCap{}, err
	}
	s.SeedUsed, _ = store.CapUsed(mine)
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		return contractCap{}, fmt.Errorf("contract coordinator: %w", err)
	}
	if err := coord.ProjectContract(ctx, f.Player.String(), f.Terms.Cells); err != nil {
		return contractCap{}, fmt.Errorf("contract projection: %w", err)
	}
	// Extend the reporting horizon without changing the held baseline commitments.
	horizon := f.Terms.Cells[len(f.Terms.Cells)-1].Season
	view, err := a.whatIfViewThrough(ctx, s, store, false, horizon)
	if err != nil {
		return contractCap{}, err
	}
	out := contractCap{Figures: []envelope.ContractFigure{}, Cap: s.Cap, After: map[int]domain.Money{}}
	for _, row := range view.Figures {
		out.Figures = append(out.Figures, envelope.ContractFigure{Label: row.Label, Before: row.Before, After: row.After, Change: row.Change})
	}
	for _, p := range players {
		if err := Committed(ctx, store, p, m.Season, out.After); err != nil {
			return contractCap{}, err
		}
	}
	projected, _ := store.CapUsed(mine)
	out.After[m.Season] = s.Used + projected - s.SeedUsed
	return out, nil
}

// contractCap is the team cap table after an Act, and each season's commitments after it.
type contractCap struct {
	Figures []envelope.ContractFigure
	Cap     *domain.Money
	After   map[int]domain.Money
}

func (a *API) contractCapFacts(ctx context.Context, mine string, f *envelope.ContractFacts) error {
	if len(f.Terms.Cells) == 0 || slices.Contains(f.Reasons, Ineligible) {
		return nil
	}
	team, err := a.contractTeamCap(ctx, mine, *f)
	if err != nil {
		return err
	}
	f.TeamFigures = team.Figures
	if team.Cap == nil {
		f.Reasons = append(f.Reasons, "Not checked: team salary cap; no cap setting is held")
		return nil
	}
	if last := f.Terms.Cells[len(f.Terms.Cells)-1].Season; last > f.Season {
		f.Reasons = append(f.Reasons, fmt.Sprintf("Not checked: salary caps after %d; later seasons are judged against the held %d cap", f.Season, f.Season))
	}
	if reason := CapReason(*f, *team.Cap, team.After); reason != "" {
		f.Reasons = append(f.Reasons, reason)
	}
	return nil
}
