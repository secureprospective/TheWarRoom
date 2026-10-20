package main

import (
	"context"
	"fmt"
	"slices"
	"time"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type RawWhatIfRequest struct {
	Mine   string `json:"mine"`
	Action string `json:"action"`
	Player string `json:"player"`
	Move   string `json:"move"`
}

func (r RawWhatIfRequest) Validate() error {
	return contractsapp.WhatIf(r).Validate() //nolint:wrapcheck // the service's errors already start "what-if"
}

type WhatIfMoveView struct {
	ID          string                  `json:"id"`
	Player      string                  `json:"player"`
	Tag         string                  `json:"tag"`
	Status      string                  `json:"status"`
	Label       string                  `json:"label"`
	Charge      snapshot.ContractAmount `json:"charge"`
	Explanation string                  `json:"explanation"`
	Plan        string                  `json:"plan"`
}
type WhatIfFigure struct {
	Label  string `json:"label"`
	Before string `json:"before"`
	After  string `json:"after"`
	Change string `json:"change"`
}
type WhatIfChoice struct {
	Player        string `json:"player"`
	CutLabel      string `json:"cutLabel"`
	BuyoutLabel   string `json:"buyoutLabel"`
	BuyoutAllowed bool   `json:"buyoutAllowed"`
	Reason        string `json:"reason"`
}
type WhatIfView struct {
	Stale   bool             `json:"stale"`
	Banner  string           `json:"banner"`
	Empty   string           `json:"empty"`
	Moves   []WhatIfMoveView `json:"moves"`
	Figures []WhatIfFigure   `json:"figures"`
	Choices []WhatIfChoice   `json:"choices"`
	Notice  string           `json:"notice"`
}

// TargetWhatIf reads or changes only the persisted simulation; never a plan or MFL request.
func (a *App) TargetWhatIf(raw RawWhatIfRequest) Reading[WhatIfView] {
	if err := raw.Validate(); err != nil {
		return whatIfReading(WhatIfView{}, state.WhatIfSession{}, err)
	}
	if err := a.ready(); err != nil {
		return whatIfReading(WhatIfView{}, state.WhatIfSession{}, err)
	}
	a.whatifMu.Lock()
	defer a.whatifMu.Unlock()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	session, store, err := a.whatIfSession(ctx, raw)
	if err != nil {
		return whatIfReading(WhatIfView{}, session, err)
	}
	stamps, err := a.league.Stamps(ctx)
	if err != nil {
		return whatIfReading(WhatIfView{}, session, fmt.Errorf("what-if current stamps: %w", err))
	}
	stale := stamps.NewerThan(session.Stamps)
	if stale && len(session.Moves) == 0 {
		session, store, err = a.whatIfSession(ctx, RawWhatIfRequest{Mine: raw.Mine, Action: "reset"})
		if err != nil {
			return whatIfReading(WhatIfView{}, session, err)
		}
		stale = false
	}
	var notice string
	if raw.Action != "read" && raw.Action != "reset" {
		if stale {
			notice = "MFL changed since this what-if started. Nothing changed; Reset first."
		} else {
			next, nextStore, changeErr := a.changeWhatIf(ctx, session, raw)
			if changeErr != nil {
				notice = changeErr.Error()
			} else {
				session, store = next, nextStore
			}
		}
	}
	view, err := a.whatIfView(ctx, session, store, stale)
	view.Notice = notice
	return whatIfReading(view, session, err)
}

func whatIfReading(view WhatIfView, s state.WhatIfSession, err error) Reading[WhatIfView] {
	r := contractReading(view, s.Stamps.Rosters, "Your what-if, started from MFL rosters as of "+stampText(s.Stamps.Rosters), "Cut = 35% of each paid cell after this season, or 50% for a known restructured contract; claimed and final-year cuts cost $0. Sum exact cents, round half-up once, then snap once to $10,000. Buyout = mean of remaining paid cells times 60% / 75% / 90% for 2 / 3 / 4 years, then one snap; at most two per season. Future salaries carry today’s MFL annual salary through contract expiration; MFL does not report those years separately. The exports do not establish restructure history: seeded contracts assume 35%, not a proven unrestructured verdict. This season's used begins at the mirror total; the engine's ledger delta is applied without changing the original MFL adjustments. MFL adjustments remain literal adjustments, not proven dead cap. The whole modelled charge lands in the cut year; future seasons have no modelled dead cap. Buyouts here model offseason terms without asserting that today's window is open; offseason start is not held. No submission or plan is created.", len(s.Moves), err)
	r.Baseline = s.Stamps.Rosters
	r.HowCounted += fmt.Sprintf(" Separate contract-cell update time: %s. Separate dead-cap update time: %s.", stampText(s.Stamps.Contract), stampText(s.Stamps.DeadCap))
	r.Method = "Coordinator handlers on a generation-scoped whatif.db ledger; integer cents"
	r.Provenance.Freshness.Note = "Baseline held until Reset; survives restart. The mirror is never written."
	return r
}
func stampText(s string) string {
	if s == "" {
		return "not held by this mirror"
	}
	at, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return "date unavailable"
	}
	return reportTime(at, time.Local)
}

func (a *App) whatIfSession(ctx context.Context, raw RawWhatIfRequest) (state.WhatIfSession, *state.Store, error) {
	sessions := state.NewWhatIfSessions(a.whatifPools)
	session, found, err := sessions.Load(ctx, raw.Mine)
	if err != nil {
		return session, nil, fmt.Errorf("what-if session: %w", err)
	}
	if !found || raw.Action == "reset" {
		m, _, used := a.league.ContractsView()
		amount, ok := used[raw.Mine]
		if !ok {
			return session, nil, fmt.Errorf("what-if: franchise %s is not in the mirror", raw.Mine)
		}
		stamps, err := a.league.Stamps(ctx)
		if err != nil {
			return session, nil, fmt.Errorf("what-if baseline: %w", err)
		}
		session = state.WhatIfSession{Mine: raw.Mine, Snapshot: m, Stamps: stamps, Used: amount, Moves: []state.WhatIfMove{}, TaxiPercent: a.rulebook.TaxiCapPercent(), IRPercent: a.rulebook.IRCapPercent()}
		if capText := a.rulebook.GetSalaryCap(); capText != "" {
			capAmount, err := domain.ParseMoneyMillions(capText)
			if err != nil {
				return session, nil, fmt.Errorf("what-if cap: %w", err)
			}
			session.Cap = &capAmount
		}
		return a.replayWhatIf(ctx, session)
	}
	store, err := a.loadWhatIf(ctx, session)
	return session, store, err
}
func (a *App) loadWhatIf(ctx context.Context, s state.WhatIfSession) (*state.Store, error) {
	store := state.New(a.whatifPools, s.Generation, s.Snapshot.Season, s)
	if err := store.Initialize(ctx, state.SnapshotSource{Snapshot: s.Snapshot}); err != nil {
		return nil, fmt.Errorf("what-if ledger: %w", err)
	}
	if err := store.Err(); err != nil {
		return nil, fmt.Errorf("what-if ledger: %w", err)
	}
	return store, nil
}

func (a *App) changeWhatIf(ctx context.Context, s state.WhatIfSession, raw RawWhatIfRequest) (state.WhatIfSession, *state.Store, error) {
	previous := s
	s.Moves = slices.Clone(s.Moves)
	if raw.Action == "undo" {
		index := slices.IndexFunc(s.Moves, func(m state.WhatIfMove) bool { return m.ID == raw.Move })
		if index < 0 {
			return previous, nil, fmt.Errorf("what-if: move no longer exists")
		}
		s.Moves = slices.Delete(s.Moves, index, index+1)
	} else {
		if slices.ContainsFunc(s.Moves, func(m state.WhatIfMove) bool { return m.Player == raw.Player }) {
			return previous, nil, fmt.Errorf("what-if: player already removed; Undo first")
		}
		kind := string(transactions.KindWaiver)
		if raw.Action == "buyout" {
			kind = string(transactions.KindBuyout)
		}
		name, found := "", false
		for _, p := range s.Snapshot.Players {
			if p.MFLID == raw.Player && p.FranchiseID == s.Mine {
				found = true
				name = "Player " + p.MFLID
			}
		}
		if !found {
			return previous, nil, fmt.Errorf("what-if: only your baseline roster can be modelled")
		}
		a.targetSnapshotMu.Lock()
		for _, p := range a.targetSnapshot.Players.Value {
			if p.ID.String() == raw.Player && p.Name != "" {
				name = p.Name
			}
		}
		a.targetSnapshotMu.Unlock()
		s.Moves = append(s.Moves, state.WhatIfMove{ID: fmt.Sprint(time.Now().UnixNano()), Kind: kind, Player: raw.Player, Name: name})
	}
	next, store, err := a.replayWhatIf(ctx, s)
	if err != nil {
		return previous, nil, err
	}
	return next, store, nil
}

func (a *App) replayWhatIf(ctx context.Context, s state.WhatIfSession) (state.WhatIfSession, *state.Store, error) {
	s.Generation = "whatif:" + s.Mine + ":" + fmt.Sprint(time.Now().UnixNano())
	store, err := a.loadWhatIf(ctx, s)
	if err != nil {
		return s, nil, err
	}
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		return s, nil, fmt.Errorf("what-if coordinator: %w", err)
	}
	s.SeedUsed, _ = store.CapUsed(s.Mine)
	for i, m := range s.Moves {
		player, ok := store.Player(m.Player)
		if !ok || player.FranchiseID != s.Mine {
			return s, nil, fmt.Errorf("what-if replay: player not on your roster")
		}
		req, err := buildRequest(TransactionRequest{Kind: m.Kind, MFLID: m.Player})
		if err != nil {
			return s, nil, fmt.Errorf("what-if request: %w", err)
		}
		receipt, err := coord.Execute(ctx, req)
		if err != nil {
			return s, nil, fmt.Errorf("what-if replay: %w", err)
		}
		if len(receipt.CapDeltas) > 1 {
			return s, nil, fmt.Errorf("what-if replay: missing charge")
		}
		s.Moves[i].Charge = 0
		if len(receipt.CapDeltas) == 1 {
			s.Moves[i].Charge = receipt.CapDeltas[0].Cents
		}
		s.Moves[i].Explanation = contractsapp.Rule(m.Kind, player, s.Snapshot.Season)
	}
	if err := state.NewWhatIfSessions(a.whatifPools).Save(ctx, s); err != nil {
		return s, nil, fmt.Errorf("what-if publish: %w", err)
	}
	return s, store, nil
}
func (a *App) whatIfView(ctx context.Context, s state.WhatIfSession, store *state.Store, stale bool) (WhatIfView, error) {
	return a.whatIfViewThrough(ctx, s, store, stale, s.Snapshot.Season)
}

func (a *App) whatIfViewThrough(ctx context.Context, s state.WhatIfSession, store *state.Store, stale bool, horizon int) (WhatIfView, error) {
	var ledger contractsapp.Ledger
	if store != nil {
		ledger = store
	}
	v, err := contractsapp.DescribeWhatIf(ctx, s, ledger, stale, horizon, func(moves []contractsapp.WhatIfMove) error {
		return a.whatIfPlanLabels(ctx, s.Mine, moves)
	})
	return wireWhatIf(v), err
}

// wireWhatIf converts the service's what-if to its wire twin; TestContractsWireTwinsMatch proves it.
func wireWhatIf(v contractsapp.WhatIfView) WhatIfView {
	out := WhatIfView{Stale: v.Stale, Banner: v.Banner, Empty: v.Empty, Moves: make([]WhatIfMoveView, len(v.Moves)),
		Figures: wireWhatIfFigures(v.Figures), Choices: make([]WhatIfChoice, len(v.Choices)), Notice: v.Notice}
	for i, m := range v.Moves {
		out.Moves[i] = WhatIfMoveView(m)
	}
	for i, c := range v.Choices {
		out.Choices[i] = WhatIfChoice(c)
	}
	return out
}

// wireWhatIfFigures converts the service's figures to their wire twins.
func wireWhatIfFigures(figures []contractsapp.WhatIfFigure) []WhatIfFigure {
	out := make([]WhatIfFigure, len(figures))
	for i, f := range figures {
		out[i] = WhatIfFigure(f)
	}
	return out
}

// whatIfFigure is a money figure before and after, as its wire type.
func whatIfFigure(label string, before, after domain.Money) WhatIfFigure {
	return WhatIfFigure(contractsapp.Figure(label, before, after))
}

func (a *App) whatIfPlanLabels(ctx context.Context, mine string, moves []contractsapp.WhatIfMove) error {
	if a.moves != nil {
		receipts, err := a.moves.List(ctx, ingestion.LeagueID, mine)
		if err != nil {
			return fmt.Errorf("what-if plans: %w", err)
		}
		for i, move := range moves {
			for _, receipt := range receipts {
				if envelope.IsRelease(receipt.Spec.Intent) && receipt.Spec.Expected.Player.String() == move.Player && envelope.ReleaseOpen(receipt.State) {
					moves[i].Plan = "Plan: " + contractsapp.StateLabel(receipt.State)
					break
				}
			}
		}
	}
	return nil
}
