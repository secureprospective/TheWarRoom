// Package assembly is the composition-class leaf that wires the RAS scouting
// signal for the first time: it fetches combine measurables + the MFL↔gsis
// crosswalk, joins each rostered MFL id to its gsis, computes the per-position
// RAS-equivalent (provisional v1 method, see ras_math.go), and returns a map
// of scouting.Profile keyed by playerid.PlayerID with only the RAS field
// populated (every other Profile field stays zero this phase — the template
// for every later scouting signal to clone).
//
// It is a COMPOSITION leaf (Layer 2-adjacent): it imports Layer-1 fetchers
// (ras, crosswalk) + the scouting/domain/playerid types and PRODUCES a typed
// domain value, but it does NOT import the engine, any store, the transaction
// coordinator, the output store, normalize's write side, or database/sql. The
// PositionLookup port (defined here) is what keeps the players-DB read out of
// this package — the app wires it over an existing normalize.Lookup so the
// assembler is fake-testable with no live DB. The RAS math itself is total and
// network-free (see ras_math.go) so it is unit-tested directly.
//
// ZERO-LEAK (hard constraint): combine measurables are pure athletic testing —
// no fantasy points / projected volume / MFL scoring column is bound here, and
// none could be (ras.RawCombine has no such field). The Profile this builds
// carries RAS only; every other scouting sub-signal stays absent this phase.
package assembly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/ras"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// PositionLookup resolves a rostered MFL id to its players-DB position. A
// narrow injected port so this assembler is fake-testable with no live DB and
// so the package imports neither normalize (Layer 1) nor any store (Layer 2).
// A miss (ok=false) means the player cannot be cohorrted and is an ordinary
// skip — never an error.
type PositionLookup interface {
	Position(mflID string) (domain.Position, bool)
}

// BuildRAS fetches combine measurables + the crosswalk, joins each rostered
// MFL id to its gsis, computes the per-position RAS-equivalent (§3), and
// returns a map[playerid.PlayerID]scouting.Profile with ONLY RAS populated
// (every other Profile field zero this phase). rosterMFLIDs is the set of ids
// to score; a player NOT in the returned map has no RAS signal (clean miss —
// the rankings side treats "absent from map" and "HasRAS=false" identically,
// and L1 imputes DefaultRASFallback = 5.0 downstream).
//
// Failures: a genuine fetch failure (network/HTTP/parse, or a crosswalk that
// resolved zero entries) is surfaced loudly — a RAS-less league should be
// visible, matching the app's fail-loud posture. A player-level miss (no gsis,
// no combine row, no measurable, no resolved position) is ordinary and never
// an error.
//
// CONTEXT DISCIPLINE: ctx is the cancellation handle; client and the URLs are
// injected (not package globals) so tests can point at fixture servers, and a
// future config can override the source.
func BuildRAS(
	ctx context.Context,
	client *http.Client,
	combineURL string,
	cw crosswalk.Map,
	rosterMFLIDs []string,
	pos PositionLookup,
) (map[playerid.PlayerID]scouting.Profile, error) {
	if client == nil {
		return nil, fmt.Errorf("assembly: BuildRAS requires a non-nil *http.Client")
	}
	if pos == nil {
		return nil, fmt.Errorf("assembly: BuildRAS requires a non-nil PositionLookup")
	}

	combine, err := ras.Fetch(ctx, client, combineURL, cw.PFRMap())
	if err != nil {
		return nil, fmt.Errorf("assembly: fetch combine: %w", err)
	}

	// Restrict to rostered players: join each roster MFL id → gsis → RawCombine
	// and resolve its position. The cohort (§3) is exactly "rostered players at
	// a position with ≥1 measurable present" — a non-rostered combine row never
	// enters the math.
	rawByGSIS := make(map[string]ras.RawCombine, len(rosterMFLIDs))
	posByGSIS := make(map[string]domain.Position, len(rosterMFLIDs))
	mflToGSIS := make(map[string]string, len(rosterMFLIDs)) // join back to playerid.PlayerID
	for _, mfl := range rosterMFLIDs {
		pid, err := playerid.New(mfl)
		if err != nil {
			continue // malformed — an upstream layer should have caught this; skip
		}
		gsis, ok := cw.Lookup(pid)
		if !ok {
			continue // no MFL→gsis mapping — ordinary miss
		}
		rc, ok := combine[gsis]
		if !ok {
			continue // no combine row — ordinary miss
		}
		position, ok := pos.Position(mfl)
		if !ok {
			continue // no resolved position → cannot cohort — ordinary miss
		}
		rawByGSIS[gsis] = rc
		posByGSIS[gsis] = position
		mflToGSIS[gsis] = pid.String()
	}

	scores := scoreRAS(rawByGSIS, posByGSIS)
	out := make(map[playerid.PlayerID]scouting.Profile, len(scores))
	for gsis, r := range scores {
		mflStr, ok := mflToGSIS[gsis]
		if !ok {
			continue // defensive — mflToGSIS was built from the same gsis set
		}
		pid, err := playerid.New(mflStr)
		if err != nil {
			continue // defensive — same id was validated above
		}
		out[pid] = scouting.Profile{MFLID: pid, RAS: r, HasRAS: true}
	}
	return out, nil
}
