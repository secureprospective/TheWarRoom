package trades

import (
	"context"
	"errors"
	"fmt"
	"time"

	mflwhatif "github.com/secureprospective/TheWarRoom/internal/mflref/whatif"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// SimulationWorkflow projects a trade on throwaway ledgers. Scratch opens one temporary pool per
// projection; Open and Load must write only to the pool they are given, so a projection (including
// the live re-projection of every saved plan on each builder read) never writes whatif.db.
type SimulationWorkflow struct {
	Scratch func(context.Context) (*db.Pools, error)
	Open    func(context.Context, *db.Pools, string) (mflwhatif.WhatIfSession, *state.Store, error)
	Mirror  func(context.Context, string) (mflwhatif.WhatIfSession, error)
	Load    func(context.Context, *db.Pools, mflwhatif.WhatIfSession) (*state.Store, error)
	Now     func() time.Time
}

type ProjectionWorkflow struct {
	Simulate     func(context.Context, string, string) (Simulation, error)
	Stamps       func(context.Context) (state.MirrorStamps, error)
	Possessive   func(string) string
	ClockReasons func(context.Context) []string
	PickReason   func(RawTradeBuilder) string
	Teams        func(context.Context, snapshot.Snapshot) []TradeBuilderTeam
	Figure       func(string, domain.Money, domain.Money) Figure
}

// Simulation's Store lives in its scratch pool; Close discards both.
type Simulation struct {
	Sessions []mflwhatif.WhatIfSession
	Store    *state.Store
	Before   map[string]domain.Money
	scratch  *db.Pools
}

// Close discards the scratch pool. It is safe on a zero Simulation.
func (s Simulation) Close() error {
	if s.scratch == nil {
		return nil
	}
	return s.scratch.Close() //nolint:wrapcheck // db.Close names the pool.
}

func (s SimulationWorkflow) Run(ctx context.Context, mine, other string) (out Simulation, err error) {
	out = Simulation{Before: map[string]domain.Money{}}
	scratch, err := s.Scratch(ctx)
	if err != nil {
		return out, fmt.Errorf("trade scratch ledger: %w", err)
	}
	out.scratch = scratch
	defer func() {
		if err != nil {
			err = errors.Join(err, out.Close())
		}
	}()
	merged := state.MirrorSnapshot{}
	for _, fid := range []string{mine, other} {
		var session mflwhatif.WhatIfSession
		var store *state.Store
		var err error
		if fid == mine {
			session, store, err = s.Open(ctx, scratch, fid)
		} else {
			session, err = s.Mirror(ctx, fid)
			if err == nil {
				session.Generation = "trade-counterparty:" + fmt.Sprint(s.Now().UnixNano())
				store, err = s.Load(ctx, scratch, session)
				if err == nil {
					session.SeedUsed, _ = store.CapUsed(fid)
				}
			}
		}
		if err != nil {
			return out, err
		}
		out.Sessions = append(out.Sessions, session)
		used, _ := store.CapUsed(fid)
		out.Before[fid] = session.Used + used - session.SeedUsed
		merged.Season = session.Snapshot.Season
		for _, p := range session.Snapshot.Players {
			if p.FranchiseID != fid {
				continue
			}
			if live, ok := store.Player(p.MFLID); !ok || live.FranchiseID != fid {
				continue
			}
			merged.Players = append(merged.Players, p)
		}
		for _, adjust := range session.Snapshot.Adjustments {
			if adjust.FranchiseID == fid {
				merged.Adjustments = append(merged.Adjustments, adjust)
			}
		}
	}
	seed := out.Sessions[0]
	seed.Generation = "trade-preview:" + fmt.Sprint(s.Now().UnixNano())
	seed.Snapshot = merged
	store, err := s.Load(ctx, scratch, seed)
	if err != nil {
		return out, err
	}
	out.Store = store
	return out, nil
}
func (s ProjectionWorkflow) Run(ctx context.Context, r RawTradeBuilder, snap snapshot.Snapshot) (TradeBuilderPlan, error) {
	out := TradeBuilderPlan{Reasons: []string{}, Teams: []TradeProjection{}}
	sim, err := s.Simulate(ctx, r.Mine, r.Other)
	if err != nil {
		return out, err
	}
	defer func() { _ = sim.Close() }() // a throwaway scratch pool; nothing to keep
	stamps, err := s.Stamps(ctx)
	if err != nil {
		return out, fmt.Errorf("trade baseline stamps: %w", err)
	}
	for _, session := range sim.Sessions {
		if stamps.NewerThan(session.Stamps) {
			name := FranchiseName(snap, session.Mine)
			out.Reasons = append(out.Reasons, "MFL changed since "+s.Possessive(name)+" what-if started; reset it to "+
				"check this deal against MFL as it stands.")
			out.Resets = append(out.Resets, TradeReset{Franchise: session.Mine, Label: "Reset " + s.Possessive(name) + " what-if"})
		}
	}
	out.Reasons = append(out.Reasons, s.ClockReasons(ctx)...)
	if reason := envelope.TradeFeedReason(snap.Rosters.Provenance); reason != "" {
		out.Reasons = append(out.Reasons, "Roster unavailable: "+reason)
	}
	if reason := s.PickReason(r); reason != "" {
		out.Reasons = append(out.Reasons, reason)
	}
	catalog := s.Teams(ctx, snap)
	moves, blocks := Legs(Input(r), ServiceTeams(catalog), sim.Store.Player)
	out.Reasons = append(out.Reasons, blocks...)
	before := map[string]domain.Money{}
	for _, session := range sim.Sessions {
		before[session.Mine], _ = sim.Store.CapUsed(session.Mine)
	}
	if len(blocks) == 0 && len(moves) > 0 {
		if err := executeProjection(ctx, sim.Store, moves); err != nil {
			return out, err
		}
	}
	players := map[string]snapshot.Player{}
	for _, p := range snap.Players.Value {
		players[p.ID.String()] = p
	}
	for _, session := range sim.Sessions {
		projected, blocks := ProjectedTeam(simulationInput(sim, session, before[session.Mine], players, FranchiseName(snap, session.Mine)), s.Figure)
		projected.Roster = RichRoster(projected.Roster, ServiceTeams(catalog))
		team := WireProjection(projected)
		out.Teams, out.Reasons = append(out.Teams, team), append(out.Reasons, blocks...)
	}
	out.Post = DealText(Input(r), ServiceTeams(catalog), sim.Sessions[0].Snapshot)
	return out, nil
}

// executeProjection makes the trade on the simulation's scratch ledger.
func executeProjection(ctx context.Context, store *state.Store, moves []transactions.PlayerMove) error {
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		return fmt.Errorf("trade preview coordinator: %w", err)
	}
	if _, err := coord.Execute(ctx, transactions.Trade{Moves: moves, Rationale: "Local what-if trade projection"}); err != nil {
		return fmt.Errorf("trade preview: %w", err)
	}
	return nil
}

func simulationInput(sim Simulation, s mflwhatif.WhatIfSession, seed domain.Money,
	players map[string]snapshot.Player, name string) ProjectionInput {
	used, _ := sim.Store.CapUsed(s.Mine)
	original := SimulationPlayers(sim.Sessions)
	live := make(map[string]state.PlayerState, len(original))
	for _, p := range original {
		if value, ok := sim.Store.Player(p.MFLID); ok {
			live[p.MFLID] = value
		}
	}
	return ProjectionInput{Session: s, Name: name, Before: sim.Before[s.Mine], Seed: seed,
		Used: used, Players: original, Live: live, Names: players}
}
