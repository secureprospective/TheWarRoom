// Package transactions is the only code that mutates league state at runtime. The Coordinator
// holds the process's one state.Writer and runs every request inside a single state transaction,
// so a multi-leg trade commits whole or not at all. Request types live in this package, so callers
// never import a handler subpackage.
package transactions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// errDryRun makes Preview roll back after a fully validated, applied request: proof it would
// commit, with nothing stored. Preview turns it into success; it never reaches a caller.
var errDryRun = errors.New("transactions: dry-run rollback (not an error)")

// Coordinator is the sole runtime mutator of league state. Construct with New.
type Coordinator struct {
	writer state.Writer
	policy RosterPolicy     // nil disables roster-limit enforcement
	dirs   DirectorySource  // players directory, for requests that resolve facts first
	now    func() time.Time // injectable for tests
}

// DirectorySource returns the players directory. The app builds it lazily with one MFL fetch, so
// the coordinator asks only when a request needs it (tag price, extension floor, signing floor).
type DirectorySource func(context.Context) (Directory, error)

// resolvable is a request that needs players-directory facts before it can validate. resolve
// returns the request with those facts filled in, computed from authoritative state.
type resolvable interface {
	resolve(c *Coordinator, dir Directory) (Request, error)
}

// New fails on a nil writer, so a miswired Coordinator can't silently no-op. A nil policy turns
// off roster-limit enforcement; a nil dirs makes every resolvable request fail.
func New(w state.Writer, p RosterPolicy, dirs DirectorySource) (*Coordinator, error) {
	if w == nil {
		return nil, fmt.Errorf("transactions: nil state.Writer — the coordinator is the sole runtime mutator and requires it")
	}
	return &Coordinator{writer: w, policy: p, dirs: dirs, now: time.Now}, nil
}

// Receipt is what a committed transaction did. A failure returns a zero Receipt and the error.
type Receipt struct {
	Kind            Kind      `json:"kind"`
	PlayersAffected int       `json:"playersAffected"`
	At              time.Time `json:"at"`
	// CapDeltas is the handler's signed cap impact (dead-cap charges, relief credits), mostly for
	// Preview. After a commit, the reloaded franchise cap is the authority.
	CapDeltas []CapDelta `json:"capDeltas"`
}

// Execute validates the request and runs it in one transaction. Any error persists nothing and
// returns a zero Receipt.
func (c *Coordinator) Execute(ctx context.Context, req Request) (Receipt, error) {
	return c.run(ctx, req, false)
}

// Preview runs a request exactly as Execute does, through the same handler, then always rolls
// back: "would this commit, and if not, why?" The commit that follows recomputes from scratch;
// the preview is never fed back as input.
func (c *Coordinator) Preview(ctx context.Context, req Request) (Receipt, error) {
	return c.run(ctx, req, true)
}

func (c *Coordinator) run(ctx context.Context, req Request, dryRun bool) (Receipt, error) {
	verb := "execute"
	if dryRun {
		verb = "preview"
	}
	if req == nil {
		return Receipt{}, fmt.Errorf("transactions: %s called with a nil request", verb)
	}
	req, err := c.resolve(ctx, req, verb)
	if err != nil {
		return Receipt{}, err
	}
	if err := req.validate(); err != nil {
		return Receipt{}, err
	}
	res, err := c.inTx(ctx, req, dryRun)
	if err != nil {
		return Receipt{}, fmt.Errorf("transactions: %s %s: %w", verb, req.Kind(), err)
	}
	return Receipt{Kind: req.Kind(), PlayersAffected: res.PlayersAffected, At: c.now().UTC(), CapDeltas: res.Deltas}, nil
}

// resolve fills in the players-directory facts a resolvable request needs; others pass through.
func (c *Coordinator) resolve(ctx context.Context, req Request, verb string) (Request, error) {
	r, ok := req.(resolvable)
	if !ok {
		return req, nil
	}
	if c.dirs == nil {
		return nil, fmt.Errorf("transactions: %s %s: no players directory configured", verb, req.Kind())
	}
	dir, err := c.dirs(ctx)
	if err == nil && dir == nil {
		err = errors.New("source returned no directory")
	}
	if err != nil {
		return nil, fmt.Errorf("transactions: %s %s: players directory: %w", verb, req.Kind(), err)
	}
	return r.resolve(c, dir)
}

// inTx applies req inside one transaction, rolling it back after a successful apply when dryRun.
func (c *Coordinator) inTx(ctx context.Context, req Request, dryRun bool) (applyResult, error) {
	var res applyResult
	err := c.writer.WriteTx(ctx, func(w state.TxWriter) error {
		// The phase gate runs first, before any mutation; an unmapped op is denied.
		if err := gatePhase(ctx, w, req.Kind()); err != nil {
			return err
		}
		// Project the roster effect against the policy before applying, so a violation is
		// rejected atomically and names the franchise and limit.
		if ra, ok := req.(rosterAware); ok && c.policy != nil {
			if err := ra.enforceRosterLimits(ctx, c.writer, c.policy); err != nil {
				return err
			}
		}
		r, err := req.apply(ctx, w)
		if err != nil {
			return err
		}
		res = r
		if dryRun {
			return errDryRun // applied and valid: roll back
		}
		return nil
	})
	if dryRun && errors.Is(err, errDryRun) {
		err = nil
	}
	return res, err
}
