package assembly

// schooltier.go is the S-Phase 1 assembler — the SECOND scouting signal, cloning the
// leaf shape S-Phase 0 (RAS) established: fetch a Layer-1 source, join it to rostered
// MFL ids, and emit a typed scouting value keyed by playerid.PlayerID. It imports the
// Layer-1 schooltier fetcher + the scouting/playerid types and PRODUCES a typed value;
// it imports no engine, no store, and no database/sql (depguard-legal leaf).
//
// WHY THIS SHAPE DIFFERS FROM BuildRAS: RAS is a bare float that needs per-position
// normalization, so BuildRAS carries a PositionLookup and runs the §3 math. SchoolTier
// is POSITION-INDEPENDENT and the fetcher already emits a FINAL scouting.SchoolTier
// (its own deliberate deviation — see the schooltier package doc), so there is no math
// here: the assembler's whole job is the JOIN — MFL id → MFL college name → CFBD school
// → tier. It therefore needs a SchoolLookup (college by id), not a PositionLookup.
//
// THE NAME RECONCILIATION (v1): MFL's college vocabulary and CFBD's school vocabulary
// agree on ~94% of rostered skill-position players by exact string, but disagree on a
// well-known tail of Power-Four / Group-of-Five aliases ("Miami (FL)" vs "Miami",
// "Mississippi" vs "Ole Miss", "Brigham Young" vs "BYU", …). A player left unmatched
// falls to SchoolUnset → Data-Parity neutral, which for a real Power-Four player is a
// visibly WRONG neutral, so v1 carries a small, deterministic alias map covering the
// observed FBS mismatches (collegeAliases, every target verified against the live CFBD
// /teams set 2026-07-20). This lifts the match to ~98%+ for FBS players; the residual
// misses are legitimately non-FBS/D2/D3 schools CFBD does not classify, which SHOULD be
// neutral. A comprehensive fuzzy reconciliation is deliberately deferred — a v1 alias
// table is auditable where a fuzzy matcher is not, and the neutral fallback is safe.
//
// ZERO-LEAK (hard constraint): a competition tier carries no fantasy points, projected
// volume, or MFL scoring — a leak is structurally unrepresentable here.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/schooltier"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// SchoolLookup resolves a rostered MFL id to its raw MFL college name. A narrow
// injected port mirroring PositionLookup: it keeps the players-DB read out of this
// package (the app wires it over the existing normalize.Lookup) so the assembler is
// fake-testable with no live DB. ok=false means MFL has no college for that id
// (team-defense rows, some deep-database players) — an ordinary skip, never an error.
type SchoolLookup interface {
	College(mflID string) (string, bool)
}

// BuildSchoolTier fetches the CFBD school→tier map for the season, then joins each
// rostered MFL id to a SchoolTier via its MFL college name (exact match + the
// collegeAliases reconciliation). It returns a map[playerid.PlayerID]scouting.SchoolTier
// holding ONLY players whose college resolved to a CFBD-classified tier; a player absent
// from the map has no school-tier signal, which the rankings side treats identically to
// SchoolUnset (Data-Parity neutral). Every value in the returned map is a real classified
// tier — the fetcher never emits SchoolUnset (it skips unclassified schools).
//
// Failures: a genuine fetch failure (network/HTTP/parse, or a fetch that classified zero
// schools) is surfaced loudly — matching BuildRAS's fail-loud posture and the app's stance
// that a signal-less league should be VISIBLE, not silent. A player-level miss (no college,
// an unmatched name, an unclassified school) is ordinary and never an error. The MISSING-KEY
// case (CFBD not configured for this environment) is NOT handled here — the caller decides
// whether to skip the signal or fail; this assembler is only reached with a key present.
//
// CONTEXT DISCIPLINE: ctx is the cancellation handle; client, teamsURL, apiKey, and year
// are injected (not package globals) so tests can point at a fixture server and a future
// config can override the source or season.
func BuildSchoolTier(
	ctx context.Context,
	client *http.Client,
	teamsURL, apiKey string,
	year int,
	rosterMFLIDs []string,
	sl SchoolLookup,
) (map[playerid.PlayerID]scouting.SchoolTier, error) {
	if client == nil {
		return nil, fmt.Errorf("assembly: BuildSchoolTier requires a non-nil *http.Client")
	}
	if sl == nil {
		return nil, fmt.Errorf("assembly: BuildSchoolTier requires a non-nil SchoolLookup")
	}

	tiers, err := schooltier.Fetch(ctx, client, teamsURL, apiKey, year)
	if err != nil {
		return nil, fmt.Errorf("assembly: fetch school tiers: %w", err)
	}

	aliases := collegeAliases()
	out := make(map[playerid.PlayerID]scouting.SchoolTier, len(rosterMFLIDs))
	for _, mfl := range rosterMFLIDs {
		pid, err := playerid.New(mfl)
		if err != nil {
			continue // malformed — an upstream layer should have caught this; skip
		}
		college, ok := sl.College(mfl)
		if !ok || college == "" {
			continue // MFL has no college for this player — ordinary miss
		}
		school := college
		if canon, ok := aliases[college]; ok {
			school = canon // MFL→CFBD name reconciliation (v1 alias table)
		}
		tier, ok := tiers[school]
		if !ok {
			continue // unmatched name or a school CFBD did not classify — neutral downstream
		}
		out[pid] = tier
	}
	return out, nil
}

// collegeAliases maps the MFL college spelling to the CFBD /teams "school" spelling for
// the observed FBS mismatches. Returned by a constructor (not a package global) to
// respect gochecknoglobals, mirroring normalize.newPositionMap. Every target was verified
// present in the live CFBD 2026 /teams set (2026-07-20). Identity mappings are omitted —
// an exact match needs no alias. Extend this table as new FBS mismatches surface; a name
// NOT here falls through to exact match, then to a neutral miss.
func collegeAliases() map[string]string {
	return map[string]string{
		"Miami (FL)":               "Miami",
		"North Carolina State":     "NC State",
		"Mississippi":              "Ole Miss",
		"Brigham Young":            "BYU",
		"Appalachian State":        "App State",
		"Central Florida":          "UCF",
		"Middle Tennessee State":   "Middle Tennessee",
		"San Jose State":           "San José State",
		"Southern California":      "USC",
		"Connecticut":              "UConn",
		"Hawaii":                   "Hawai'i",
		"Louisiana-Lafayette":      "Louisiana",
		"Texas-San Antonio":        "UTSA",
		"Southern Mississippi":     "Southern Miss",
		"UMass":                    "Massachusetts",
		"North Carolina-Charlotte": "Charlotte",
	}
}
