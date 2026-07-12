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
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// errDryRun is the sentinel a Preview returns from inside WriteTx to force a rollback AFTER
// the request has been fully validated, phase-gated, and applied. WriteTx rolls back on ANY
// closure error, so returning this after a clean apply proves the transaction WOULD commit —
// while persisting nothing. It never escapes to a caller (Preview maps it to success).
var errDryRun = errors.New("transactions: dry-run rollback (not an error)")

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
		// Season-phase eligibility is the FIRST step inside the tx (Vision-2026 D3): read the
		// committed current phase and reject an op the phase disallows BEFORE any mutation or
		// counter read runs — fail-fast, atomic with the write, race-free under the single
		// writer. Default-deny an unmapped op_kind.
		if perr := gatePhase(ctx, w, req.Kind()); perr != nil {
			return perr
		}
		n, aerr := req.apply(ctx, w)
		affected = n
		return aerr
	})
	if err != nil {
		return Receipt{}, fmt.Errorf("transactions: execute %s: %w", req.Kind(), err)
	}
	return Receipt{Kind: req.Kind(), PlayersAffected: affected, At: c.now().UTC()}, nil
}

// Preview runs a request exactly as Execute would — same validation, same phase gate, same
// apply, through the same single write path — but ALWAYS rolls back, so nothing is persisted.
// It answers "would this transaction commit, and if not, why?" for the staged-and-confirm UI,
// reusing the REAL handler so a preview can never drift from the commit (design D5). A rejection
// returns the authoritative error (eligibility, phase gate, signing window, min-salary floor, …);
// success returns a Receipt whose PlayersAffected the confirm step displays. The commit that
// FOLLOWS a preview re-sends the SAME intent and recomputes authoritatively — the preview is
// advisory only, never fed back in as input (design D4).
func (c *Coordinator) Preview(ctx context.Context, req Request) (Receipt, error) {
	if req == nil {
		return Receipt{}, fmt.Errorf("transactions: Preview called with a nil request")
	}
	if err := req.validate(); err != nil {
		return Receipt{}, err // rejected before a transaction is even opened
	}

	var affected int
	err := c.writer.WriteTx(ctx, func(w state.TxWriter) error {
		if perr := gatePhase(ctx, w, req.Kind()); perr != nil {
			return perr
		}
		n, aerr := req.apply(ctx, w)
		if aerr != nil {
			return aerr
		}
		affected = n
		return errDryRun // fully applied and valid — roll it all back, persist nothing
	})
	if err != nil && !errors.Is(err, errDryRun) {
		return Receipt{}, fmt.Errorf("transactions: preview %s: %w", req.Kind(), err)
	}
	return Receipt{Kind: req.Kind(), PlayersAffected: affected, At: c.now().UTC()}, nil
}

// ExecuteTag runs a §9 franchise tag. It RESOLVES the tag price here — the top-5-by-position
// league-wide average, floored at 120% of the player's prior-year salary — from the
// Coordinator's own authoritative Reader plus the supplied Directory (the players-DB
// position join), then delegates to Execute. The price is computed in this trusted core, not
// carried across the IPC boundary: the frontend sends only a player id. The Directory is
// passed per-call because the app builds its players-DB Lookup lazily (it is not available at
// startup when the Coordinator is constructed). Fails loud before opening any transaction if
// the player is unrostered or has no resolvable position.
func (c *Coordinator) ExecuteTag(ctx context.Context, mflID string, dir Directory) (Receipt, error) {
	tag, err := c.resolveTag(mflID, dir)
	if err != nil {
		return Receipt{}, err
	}
	return c.Execute(ctx, tag)
}

// PreviewTag is the dry-run counterpart of ExecuteTag (design D5): it resolves the §9 price through
// the SAME resolveTag path, then previews (rolls back) instead of committing, so the staged-confirm
// UI can surface an "unrostered" / "no position" / any in-tx rejection BEFORE any write. Because both
// verbs build the Tag from one resolver, a preview's price can never drift from the commit's.
func (c *Coordinator) PreviewTag(ctx context.Context, mflID string, dir Directory) (Receipt, error) {
	tag, err := c.resolveTag(mflID, dir)
	if err != nil {
		return Receipt{}, err
	}
	return c.Preview(ctx, tag)
}

// resolveTag resolves the §9 tag price — the top-5-by-position league-wide average, floored at 120%
// of the player's prior-year salary — from the Coordinator's authoritative Reader plus the supplied
// Directory (the players-DB position join), and returns the sealed Tag request the tx handler runs.
// The price is computed in this trusted core, never carried across the IPC boundary (the frontend
// sends only a player id). Fails loud if the player is unrostered or has no resolvable position.
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

// ExecuteExtension runs a §10 contract extension. It resolves the position FLOOR here — the
// one figure that needs the players-DB position join (via the supplied Directory) — and hands
// it to the handler, which resolves the 150%-of-highest-remaining price from the player's own
// committed cells inside the transaction. No money crosses the IPC boundary: the frontend sends
// only the player id and the added-year count. The Directory is passed per-call (the app builds
// its players-DB Lookup lazily, after the Coordinator is constructed). Fails loud before opening
// any transaction if the player is unrostered or his position has no §10 floor.
func (c *Coordinator) ExecuteExtension(ctx context.Context, mflID string, addedYears int, dir Directory) (Receipt, error) {
	ext, err := c.resolveExtension(mflID, addedYears, dir)
	if err != nil {
		return Receipt{}, err
	}
	return c.Execute(ctx, ext)
}

// PreviewExtension is the dry-run counterpart of ExecuteExtension (design D5): it resolves the same
// §10 position floor through the SAME resolveExtension path, then previews (rolls back) instead of
// committing, so the staged-confirm UI can surface an "unrostered" / "no floor" / any in-tx rejection
// BEFORE any write. One resolver for both verbs means a preview can never drift from the commit.
func (c *Coordinator) PreviewExtension(ctx context.Context, mflID string, addedYears int, dir Directory) (Receipt, error) {
	ext, err := c.resolveExtension(mflID, addedYears, dir)
	if err != nil {
		return Receipt{}, err
	}
	return c.Preview(ctx, ext)
}

// resolveExtension resolves the §10 position FLOOR (the one figure needing the players-DB position
// join via the supplied Directory) and returns the sealed Extension request; the handler resolves the
// 150%-of-highest-remaining price from the player's own committed cells inside the transaction. No
// money crosses the IPC boundary — the frontend sends only the id and the added-year count. Fails loud
// if the player is unrostered or his position has no §10 floor.
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

// ExecuteSign runs a §6 free-agency signing. It resolves the player's DRAFT YEAR here — the
// experience source for the §6 min-salary floor — from the supplied Directory (the players-DB
// join), stamps it on the unexported request fields, and delegates to Execute, which derives
// experience against the authoritative in-tx season and enforces the floor. Only the id, salary,
// and year count crossed the IPC boundary; the draft year is resolved server-side. Unlike a tag or
// extension the signee is NOT rostered, so there is no roster lookup — eligibility (must be a
// signable free agent) is judged against status events inside the tx. A player absent from the
// players DB (commissioner-created) or carrying no real draft year keeps hasDraftYear=false, so the
// handler applies the rookie floor (Christopher's missing-draft-data → rookie-floor ruling). The
// Directory is passed per-call because the app builds its players-DB Lookup lazily.
func (c *Coordinator) ExecuteSign(ctx context.Context, sign Sign, dir Directory) (Receipt, error) {
	if dir == nil {
		return Receipt{}, fmt.Errorf("transactions: sign %q: nil directory (draft-year join required for the §6 min-salary floor)", sign.MFLID)
	}
	if facts, ok := dir.Facts(sign.MFLID); ok && facts.HasDraftYear {
		sign.draftYear, sign.hasDraftYear = facts.DraftYear, true
	}
	return c.Execute(ctx, sign)
}

// PreviewSign is the dry-run counterpart of ExecuteSign: it resolves the same draft-year floor
// input, then previews (rolls back) instead of committing, so the staged-confirm UI can surface
// a "below §6 minimum" / "signing window closed" / "not a free agent" rejection BEFORE any write.
func (c *Coordinator) PreviewSign(ctx context.Context, sign Sign, dir Directory) (Receipt, error) {
	if dir == nil {
		return Receipt{}, fmt.Errorf("transactions: preview sign %q: nil directory (draft-year join required for the §6 min-salary floor)", sign.MFLID)
	}
	if facts, ok := dir.Facts(sign.MFLID); ok && facts.HasDraftYear {
		sign.draftYear, sign.hasDraftYear = facts.DraftYear, true
	}
	return c.Preview(ctx, sign)
}
