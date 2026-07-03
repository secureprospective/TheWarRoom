// Package transactions is the B7a transaction layer: the ONE component that mutates
// league state at runtime (AD-02 single-writer law). Everything before it reads; the
// Coordinator writes. It holds the injected state.Writer — the SOLE holder in the
// process — and exposes exactly one verb, Execute(ctx, Request) (Receipt, error).
//
// Every transaction runs inside state's spanning transaction (Writer.WriteTx): the
// Coordinator hands the request's steps to one TxWriter, so a multi-leg trade commits
// as a unit or rolls back whole. Request types are defined in this root package (see
// request.go) so callers never import a handler subpackage; the handlers themselves
// live in acquisitions/ (and, once Money is locked, contracts/ + deadcap/) behind the
// transactions-only-through-coordinator depguard rule.
package transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Coordinator is the sole runtime mutator of league state. Construct with New.
type Coordinator struct {
	writer state.Writer
	now    func() time.Time // injectable clock so a Receipt's timestamp is testable
}

// New wires the Coordinator with the single state.Writer it will ever hold. A nil
// writer is a wiring error surfaced HERE, at construction — never a Coordinator that
// silently no-ops every transaction at run time (the rankings.New fail-loud rule).
func New(w state.Writer) (*Coordinator, error) {
	if w == nil {
		return nil, fmt.Errorf("transactions: nil state.Writer — the coordinator is the sole runtime mutator and requires it")
	}
	return &Coordinator{writer: w, now: time.Now}, nil
}

// Receipt is the outcome of one executed transaction: what kind ran, how many players
// it moved, and when it committed. It is returned only on success — a failed
// transaction returns a zero Receipt and the error, never a partial receipt.
type Receipt struct {
	Kind            Kind      `json:"kind"`
	PlayersAffected int       `json:"playersAffected"`
	At              time.Time `json:"at"`
}

// Execute validates the request, then runs its steps in ONE spanning transaction. On
// any error — validation, a failed step, or a commit failure — nothing is persisted
// (WriteTx rolls back) and a zero Receipt is returned with the error. On success the
// Receipt reflects the committed change.
func (c *Coordinator) Execute(ctx context.Context, req Request) (Receipt, error) {
	if req == nil {
		return Receipt{}, fmt.Errorf("transactions: Execute called with a nil request")
	}
	if err := req.validate(); err != nil {
		return Receipt{}, err // rejected before a transaction is even opened
	}

	var affected int
	err := c.writer.WriteTx(ctx, func(w state.TxWriter) error {
		n, aerr := req.apply(ctx, w)
		affected = n
		return aerr
	})
	if err != nil {
		return Receipt{}, fmt.Errorf("transactions: execute %s: %w", req.Kind(), err)
	}
	return Receipt{Kind: req.Kind(), PlayersAffected: affected, At: c.now().UTC()}, nil
}
