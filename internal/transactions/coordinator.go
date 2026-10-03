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
	now    func() time.Time // injectable for tests
}

// New fails on a nil writer, so a miswired Coordinator can't silently no-op. A nil policy is
// allowed and turns off roster-limit enforcement.
func New(w state.Writer, p RosterPolicy) (*Coordinator, error) {
	if w == nil {
		return nil, fmt.Errorf("transactions: nil state.Writer — the coordinator is the sole runtime mutator and requires it")
	}
	return &Coordinator{writer: w, policy: p, now: time.Now}, nil
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
	if req == nil {
		return Receipt{}, fmt.Errorf("transactions: Execute called with a nil request")
	}
	if err := req.validate(); err != nil {
		return Receipt{}, err
	}

	var res applyResult
	err := c.writer.WriteTx(ctx, func(w state.TxWriter) error {
		// The phase gate runs first inside the transaction, before any mutation; an unmapped op is
		// denied.
		if perr := gatePhase(ctx, w, req.Kind()); perr != nil {
			return perr
		}
		// Roster limits: project the request's roster effect against the policy before applying, so a
		// violation is rejected atomically and names the franchise and limit.
		if c.policy != nil {
			if ra, ok := req.(rosterAware); ok {
				if rerr := ra.enforceRosterLimits(ctx, c.writer, c.policy); rerr != nil {
					return rerr
				}
			}
		}
		r, aerr := req.apply(ctx, w)
		res = r
		return aerr
	})
	if err != nil {
		return Receipt{}, fmt.Errorf("transactions: execute %s: %w", req.Kind(), err)
	}
	return Receipt{Kind: req.Kind(), PlayersAffected: res.PlayersAffected, At: c.now().UTC(), CapDeltas: res.Deltas}, nil
}

// Preview runs a request exactly as Execute does, through the same handler, then always rolls
// back: "would this commit, and if not, why?" The commit that follows recomputes from scratch;
// the preview is never fed back as input.
func (c *Coordinator) Preview(ctx context.Context, req Request) (Receipt, error) {
	if req == nil {
		return Receipt{}, fmt.Errorf("transactions: Preview called with a nil request")
	}
	if err := req.validate(); err != nil {
		return Receipt{}, err
	}

	var res applyResult
	err := c.writer.WriteTx(ctx, func(w state.TxWriter) error {
		if perr := gatePhase(ctx, w, req.Kind()); perr != nil {
			return perr
		}
		// Same roster-limit gate as Execute, so a preview rejects exactly as the commit would.
		if c.policy != nil {
			if ra, ok := req.(rosterAware); ok {
				if rerr := ra.enforceRosterLimits(ctx, c.writer, c.policy); rerr != nil {
					return rerr
				}
			}
		}
		r, aerr := req.apply(ctx, w)
		if aerr != nil {
			return aerr
		}
		res = r
		return errDryRun // applied and valid: roll back
	})
	if err != nil && !errors.Is(err, errDryRun) {
		return Receipt{}, fmt.Errorf("transactions: preview %s: %w", req.Kind(), err)
	}
	return Receipt{Kind: req.Kind(), PlayersAffected: res.PlayersAffected, At: c.now().UTC(), CapDeltas: res.Deltas}, nil
}

// ExecuteTag runs a §9 franchise tag. The price (top-5 average at the position, floored at 120%
// of last year's salary) is resolved here from authoritative state; the frontend sends only the
// player id. dir is per call because the app builds the players directory lazily.
func (c *Coordinator) ExecuteTag(ctx context.Context, mflID string, dir Directory) (Receipt, error) {
	tag, err := c.resolveTag(mflID, dir)
	if err != nil {
		return Receipt{}, err
	}
	return c.Execute(ctx, tag)
}

// PreviewTag dry-runs a tag through the same resolver as ExecuteTag, so the prices never differ.
func (c *Coordinator) PreviewTag(ctx context.Context, mflID string, dir Directory) (Receipt, error) {
	tag, err := c.resolveTag(mflID, dir)
	if err != nil {
		return Receipt{}, err
	}
	return c.Preview(ctx, tag)
}

// resolveTag computes the §9 price and returns the Tag request. Unrostered or no position fails.
func (c *Coordinator) resolveTag(mflID string, dir Directory) (Tag, error) {
	if dir == nil {
		return Tag{}, fmt.Errorf("transactions: tag %q: nil directory (position join required for the §9 price)", mflID)
	}
	ps, ok := c.writer.Player(mflID)
	if !ok {
		return Tag{}, fmt.Errorf("transactions: tag %q: player not on any roster", mflID)
	}
	facts, ok := dir.Facts(mflID)
	if !ok {
		return Tag{}, fmt.Errorf("transactions: tag %q: no players-DB record — cannot resolve position for the §9 top-5 average", mflID)
	}
	price := tagFloorPrice(tagPrice(c.writer, dir, facts.Position), ps.Salary)
	return Tag{MFLID: mflID, price: price}, nil
}

// ExecuteExtension runs a §10 extension. The position floor needs the players directory, so it
// is resolved here; the 150%-of-top-year price is resolved in the transaction from the player's
// cells.
func (c *Coordinator) ExecuteExtension(ctx context.Context, mflID string, addedYears int, dir Directory) (Receipt, error) {
	ext, err := c.resolveExtension(mflID, addedYears, dir)
	if err != nil {
		return Receipt{}, err
	}
	return c.Execute(ctx, ext)
}

// PreviewExtension dry-runs an extension through the same resolver as ExecuteExtension.
func (c *Coordinator) PreviewExtension(ctx context.Context, mflID string, addedYears int, dir Directory) (Receipt, error) {
	ext, err := c.resolveExtension(mflID, addedYears, dir)
	if err != nil {
		return Receipt{}, err
	}
	return c.Preview(ctx, ext)
}

// resolveExtension finds the §10 position floor and returns the Extension request. Unrostered,
// or a position with no floor, fails.
func (c *Coordinator) resolveExtension(mflID string, addedYears int, dir Directory) (Extension, error) {
	if dir == nil {
		return Extension{}, fmt.Errorf("transactions: extension %q: nil directory (position join required for the §10 floor)", mflID)
	}
	if _, ok := c.writer.Player(mflID); !ok {
		return Extension{}, fmt.Errorf("transactions: extension %q: player not on any roster", mflID)
	}
	facts, ok := dir.Facts(mflID)
	if !ok {
		return Extension{}, fmt.Errorf("transactions: extension %q: no players-DB record — cannot resolve position for the §10 floor", mflID)
	}
	floor, ok := PositionFloor(facts.Position)
	if !ok {
		return Extension{}, fmt.Errorf("transactions: extension %q: position %q has no §10 floor", mflID, facts.Position)
	}
	return Extension{MFLID: mflID, AddedYears: addedYears, floor: floor}, nil
}

// ExecuteSign runs a §6 signing. The player's draft year, which sets the min-salary floor, is
// resolved here from the players directory. A player with no real draft year (commissioner-created
// or missing data) gets the rookie floor, per Christopher's ruling. Eligibility is judged in the
// transaction from status events.
func (c *Coordinator) ExecuteSign(ctx context.Context, sign Sign, dir Directory) (Receipt, error) {
	if dir == nil {
		return Receipt{}, fmt.Errorf("transactions: sign %q: nil directory (draft-year join required for the §6 min-salary floor)", sign.MFLID)
	}
	if facts, ok := dir.Facts(sign.MFLID); ok && facts.HasDraftYear {
		sign.draftYear, sign.hasDraftYear = facts.DraftYear, true
	}
	return c.Execute(ctx, sign)
}

// PreviewSign dry-runs a signing with the same draft-year input as ExecuteSign.
func (c *Coordinator) PreviewSign(ctx context.Context, sign Sign, dir Directory) (Receipt, error) {
	if dir == nil {
		return Receipt{}, fmt.Errorf("transactions: preview sign %q: nil directory (draft-year join required for the §6 min-salary floor)", sign.MFLID)
	}
	if facts, ok := dir.Facts(sign.MFLID); ok && facts.HasDraftYear {
		sign.draftYear, sign.hasDraftYear = facts.DraftYear, true
	}
	return c.Preview(ctx, sign)
}
