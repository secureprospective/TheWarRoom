package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// MoveDTO is one leg of a trade as it crosses the IPC boundary.
type MoveDTO struct {
	MFLID         string `json:"mflID"`
	ToFranchiseID string `json:"toFranchiseID"`
}

// TransactionRequest is the typed IPC payload the dev transaction form sends. Kind
// selects which fields are read: TRADE reads Moves; ROSTER_STATUS reads MFLID + Status.
// It stays fully typed (no any/interface{}) so the ifaceguard boundary holds; the App
// method translates it into the sealed transactions.Request the Coordinator executes.
type TransactionRequest struct {
	Kind   string    `json:"kind"`
	Moves  []MoveDTO `json:"moves"`
	MFLID  string    `json:"mflID"`
	Status string    `json:"status"`
}

// TransactionResult is the typed IPC outcome: OK plus the committed Receipt fields, or
// OK=false with a human-readable Detail. A failed transaction changed nothing (the
// Coordinator rolls back), so the frontend can safely re-read state after any result.
type TransactionResult struct {
	OK              bool   `json:"ok"`
	Kind            string `json:"kind"`
	PlayersAffected int    `json:"playersAffected"`
	At              string `json:"at"`
	Detail          string `json:"detail"`
}

// ExecuteTransaction is the dev-surface IPC method (functional gate for B7a): it runs a
// trade or roster-status change through the Coordinator and returns a typed result. The
// UI is deliberately minimal — debuggability over polish; a full transaction UI is B7b.
func (a *App) ExecuteTransaction(req TransactionRequest) TransactionResult {
	if a.startupErr != nil {
		return TransactionResult{Detail: a.startupErr.Error()}
	}
	if a.coordinator == nil {
		return TransactionResult{Detail: "transaction coordinator not initialized"}
	}

	txn, err := buildRequest(req)
	if err != nil {
		return TransactionResult{Detail: err.Error()}
	}

	// Bounded context: an IPC method must never block the frontend indefinitely if the
	// data layer stalls (the Ping/M1 pattern).
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	rec, err := a.coordinator.Execute(ctx, txn)
	if err != nil {
		return TransactionResult{Kind: req.Kind, Detail: err.Error()}
	}
	return TransactionResult{
		OK:              true,
		Kind:            string(rec.Kind),
		PlayersAffected: rec.PlayersAffected,
		At:              rec.At.Format(time.RFC3339),
	}
}

// FranchisePlayerDTO is one player's live state as the dev surface renders it.
type FranchisePlayerDTO struct {
	MFLID          string  `json:"mflID"`
	RosterStatus   string  `json:"rosterStatus"`
	Salary         float64 `json:"salary"`
	AdjustedSalary float64 `json:"adjustedSalary"`
}

// FranchiseStateResult is a read of one franchise's runtime state: its cap usage and
// roster. It backs the "load → transact → reload" flow that IS the B7a functional gate.
type FranchiseStateResult struct {
	OK          bool                 `json:"ok"`
	FranchiseID string               `json:"franchiseID"`
	CapUsed     float64              `json:"capUsed"`
	Players     []FranchisePlayerDTO `json:"players"`
	Detail      string               `json:"detail"`
}

// GetFranchiseState reads one franchise's current roster + derived cap through the
// read-only surface (never the writer), so the dev form can confirm a transaction's
// effect with a fresh read after it commits.
func (a *App) GetFranchiseState(franchiseID string) FranchiseStateResult {
	if a.startupErr != nil {
		return FranchiseStateResult{Detail: a.startupErr.Error()}
	}
	if a.state == nil {
		return FranchiseStateResult{Detail: "state store not initialized"}
	}
	fs, ok := a.state.Reader().FranchiseState(franchiseID)
	if !ok {
		return FranchiseStateResult{FranchiseID: franchiseID, Detail: "no such franchise (or it holds no players)"}
	}
	players := make([]FranchisePlayerDTO, len(fs.Players))
	for i, p := range fs.Players {
		players[i] = FranchisePlayerDTO{
			MFLID:          p.MFLID,
			RosterStatus:   string(p.RosterStatus),
			Salary:         p.Salary,
			AdjustedSalary: p.AdjustedSalary,
		}
	}
	return FranchiseStateResult{OK: true, FranchiseID: franchiseID, CapUsed: fs.CapUsed, Players: players}
}

// buildRequest maps the wire DTO onto a sealed transactions.Request. An unknown Kind is
// rejected here, at the boundary, before the Coordinator is touched.
func buildRequest(req TransactionRequest) (transactions.Request, error) {
	switch req.Kind {
	case string(transactions.KindTrade):
		moves := make([]transactions.PlayerMove, len(req.Moves))
		for i, m := range req.Moves {
			moves[i] = transactions.PlayerMove{MFLID: m.MFLID, ToFranchiseID: m.ToFranchiseID}
		}
		return transactions.Trade{Moves: moves}, nil
	case string(transactions.KindRosterStatus):
		return transactions.RosterStatusChange{
			MFLID:  req.MFLID,
			Status: domain.RosterStatus(req.Status),
		}, nil
	default:
		return nil, fmt.Errorf("unknown transaction kind %q", req.Kind)
	}
}
