package assembly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/madden"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// BuildIDPFilm fetches the crosswalk-keyed Madden defense ratings and returns the ONE
// engine-ready [0,1] IDP film anchor (higher = better) that scouting.Profile.IDPFilm
// carries for each rostered DT/DE/LB/CB/S whose Madden record resolves. A player NOT in
// the returned map has no Madden signal (an ordinary miss — the Madden name+birthdate
// match resolves ~77% of the feed — and his film composite stays Data-Parity neutral).
//
// This is FILM Thread C's C-4 step 2: the Madden defense composite (K1), the primary
// live term of the IDP film composite. The recipe is LOCKED (C-3 expert-panel gate):
// the composite is the EQUAL-WEIGHT MEAN of the position's curated Madden sub-attributes
// (each _rating normalized /maddenRatingMax), with man + zone coverage AVERAGED into ONE
// coverage term (K1 — counting both would double-weight coverage, r≈0.85–0.95 collinear).
// The curated per-position sub-attribute set is the UNION of every attribute named in
// that position's rubric Madden-mapping table (Christopher, 2026-07-20: "full union from
// rubric tables" — the tables ARE the existing curation; a wider mean is robust to any
// single stale attr). At CB/S the Madden coverage term is KEPT even though a separate PFR
// coverage anchor rides upstream: C-1 proved the two are statistically INDEPENDENT
// (|r|<0.30 — Madden coverage is a scout grade of technique, the PFR anchor is outcome
// allowed), so they are additive, not a double-count (Christopher, 2026-07-20).
//
// The upstream blend (the Madden composite's SHARE of the film budget, and the K4=0.20
// CB/S coverage anchor) lives in rankings.applyScouting; this leaf owns only the raw
// Madden ratings -> normalized composite, so the engine keeps consuming a single [0,1]
// FilmComposite and this package imports no engine/store.
//
// HARD BOUNDARY: the IDP film composite applies at DT/DE/LB/CB/S ONLY. A non-IDP
// defender or an offense player is skipped (maddenComposite returns ok=false for a
// position with no curated set) — the composite must not bleed to any other position.
//
// ZERO-LEAK (hard constraint): RawMaddenRating carries only integer athletic/skill
// grades — no fantasy points, projected volume, or MFL scoring column exists on it to
// bind.
//
// Failures: a genuine fetch failure (network/HTTP/parse, or a Madden fetch that resolved
// zero gsis-keyed records) is surfaced loudly — a Madden-less league should be visible,
// matching BuildRAS/BuildCoverage. A player-level miss (no gsis, no Madden row, a non-IDP
// position, or a record carrying none of the curated attributes) is ordinary, never an
// error.
func BuildIDPFilm(
	ctx context.Context,
	client *http.Client,
	sourceURL string,
	cw crosswalk.Map,
	rosterMFLIDs []string,
	pos PositionLookup,
) (map[playerid.PlayerID]float64, error) {
	if client == nil {
		return nil, fmt.Errorf("assembly: BuildIDPFilm requires a non-nil *http.Client")
	}
	if pos == nil {
		return nil, fmt.Errorf("assembly: BuildIDPFilm requires a non-nil PositionLookup")
	}

	raw, err := madden.Fetch(ctx, client, sourceURL, cw.MaddenResolver())
	if err != nil {
		return nil, fmt.Errorf("assembly: fetch madden: %w", err)
	}

	out := make(map[playerid.PlayerID]float64, len(rosterMFLIDs))
	for _, mfl := range rosterMFLIDs {
		pid, err := playerid.New(mfl)
		if err != nil {
			continue // malformed — an upstream layer should have caught this; skip
		}
		gsis, ok := cw.Lookup(pid)
		if !ok {
			continue // no MFL->gsis mapping — ordinary miss
		}
		rc, ok := raw[gsis]
		if !ok {
			continue // no Madden row — ordinary miss (name+birthdate resolves ~77%)
		}
		position, ok := pos.Position(mfl)
		if !ok {
			continue // no resolved position → cannot apply the IDP boundary — ordinary miss
		}
		norm, ok := maddenComposite(rc, position)
		if !ok {
			continue // non-IDP position, or none of the curated attrs present — no signal
		}
		out[pid] = norm
	}
	return out, nil
}

// maddenRatingMax is the Madden rating ceiling (EA rates every attribute on [0,99]); a
// raw attribute is normalized to [0,1] by dividing by it, then clamped defensively.
const maddenRatingMax = 99.0

// idpMaddenTerms is the LOCKED K1 curation: for each IDP position, the list of terms the
// equal-weight composite averages (a func, not a package global — gochecknoglobals). Each
// term is a list of EA attribute names averaged together BEFORE the composite mean — a
// single-attribute term is that attribute on its own; the two-attribute {manCoverage,
// zoneCoverage} term is the K1 "average man+zone into one coverage term". The sets are the
// union of every attribute named in each position's rubric Madden-mapping table (see
// BuildIDPFilm doc). LB's coverage is zone-only in its table, so it is an ordinary single
// term (no man-coverage row for LB). ok=false marks a non-IDP position (the hard boundary).
//
// NOTE: the EA field names below are the suffix-stripped "_rating" keys the madden fetcher
// emits (manCoverage and speed are confirmed against live fixtures; the rest follow EA's
// camelCase convention). maddenComposite averages only the PRESENT terms, so a key that
// does not match a live attribute degrades that one term to absent rather than corrupting
// the composite — VERIFY at the live gate that composites are non-degenerate.
func idpMaddenTerms(pos domain.Position) ([][]string, bool) {
	switch pos {
	case domain.PosDT:
		return [][]string{
			{"powerMoves"}, {"strength"}, {"blockShedding"}, {"finesseMoves"},
			{"acceleration"}, {"tackle"}, {"playRecognition"},
		}, true
	case domain.PosDE:
		return [][]string{
			{"speed"}, {"acceleration"}, {"powerMoves"}, {"strength"}, {"finesseMoves"},
			{"agility"}, {"blockShedding"}, {"tackle"}, {"pursuit"}, {"playRecognition"},
		}, true
	case domain.PosLB:
		return [][]string{
			{"speed"}, {"pursuit"}, {"tackle"}, {"hitPower"}, {"strength"},
			{"zoneCoverage"}, {"playRecognition"}, {"blockShedding"}, {"awareness"}, {"powerMoves"},
		}, true
	case domain.PosCB:
		return [][]string{
			{"speed"}, {"acceleration"}, {"press"}, {"strength"},
			{"manCoverage", "zoneCoverage"}, // K1 coverage term
			{"agility"}, {"playRecognition"}, {"catching"}, {"jumping"},
		}, true
	case domain.PosS:
		return [][]string{
			{"speed"}, {"acceleration"}, {"tackle"}, {"hitPower"}, {"strength"},
			{"manCoverage", "zoneCoverage"}, // K1 coverage term
			{"playRecognition"}, {"pursuit"}, {"awareness"}, {"agility"}, {"catching"}, {"jumping"},
		}, true
	case domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE,
		domain.PosK, domain.PosFlag:
		return nil, false // non-IDP — the IDP film composite is DT/DE/LB/CB/S ONLY
	default:
		return nil, false
	}
}

// maddenComposite maps one defender's raw Madden ratings to the engine-ready [0,1] IDP
// film anchor, or reports ok=false when the row yields no usable signal: a position with
// no curated set (non-IDP — the hard boundary), or a record carrying NONE of the curated
// attributes (fully absent — no signal). It averages only the PRESENT terms (Data-Parity:
// an absent attribute is not faked as 0), so a partly-populated record still scores.
func maddenComposite(rc madden.RawMaddenRating, pos domain.Position) (float64, bool) {
	terms, ok := idpMaddenTerms(pos)
	if !ok {
		return 0, false // non-IDP position — the IDP film composite is DT/DE/LB/CB/S ONLY
	}
	var sum float64
	var n int
	for _, term := range terms {
		if tv, tok := termValue(rc, term); tok {
			sum += tv
			n++
		}
	}
	if n == 0 {
		return 0, false // none of the curated attributes present — absent, not 0.0
	}
	return sum / float64(n), true
}

// termValue averages the present attributes of one composite term to [0,1], or reports
// ok=false when the record carries none of them. A single-attribute term is just that
// attribute normalized; the two-attribute coverage term is the man+zone average (or
// whichever single one is present, Data-Parity).
func termValue(rc madden.RawMaddenRating, attrs []string) (float64, bool) {
	var sum float64
	var c int
	for _, a := range attrs {
		v, ok := rc.Attributes[a]
		if !ok {
			continue
		}
		norm := float64(v) / maddenRatingMax
		if norm < 0 {
			norm = 0
		} else if norm > 1 {
			norm = 1
		}
		sum += norm
		c++
	}
	if c == 0 {
		return 0, false
	}
	return sum / float64(c), true
}

// IDPFilmGroup wraps a normalized Madden composite as the non-nil scouting.IDPFilm group
// the merge step writes into Profile.IDPFilm (present at DT/DE/LB/CB/S when the Madden
// record resolved). Exported so the app's thin merge loop stays a copy — mirroring
// CoverageGroup and every other S-Phase signal.
func IDPFilmGroup(norm float64) *scouting.IDPFilm {
	return &scouting.IDPFilm{MaddenComposite: norm}
}
