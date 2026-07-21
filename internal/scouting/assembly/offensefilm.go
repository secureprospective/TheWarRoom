package assembly

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/madden"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/veteranfilm"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// BuildOffenseFilm fetches the crosswalk-keyed Madden offense ratings and the FTN charting
// feed and returns the ONE engine-ready [0,1] offense film anchor (higher = better) that
// scouting.Profile.OffenseFilm carries for each rostered QB/RB/WR/TE whose Madden record
// resolves. A player NOT in the returned map has no Madden signal (an ordinary miss — the
// Madden name+birthdate match resolves ~77% of the feed — and his film composite stays
// Data-Parity neutral).
//
// This is FILM Thread C's C-4 step 3, the K3 recipe LOCKED at the C-3 expert-panel gate:
// Madden is the UNIVERSAL offense-film BACKBONE (the equal-weight mean of the position's
// curated Madden sub-attributes, each _rating normalized /maddenRatingMax — the same K1
// machinery the IDP composite uses, sourced from each position's rubric Madden-mapping
// table). FTN charting is a BOUNDED DELTA-OVERLAY, not a co-driver: for a player above the
// FTN charting floor, his charted quality is percentile-ranked within the rostered
// above-floor population for his role, and the signed gap to his Madden backbone is
// discounted (ftnDiscount, the ~15% fidelity discount) and CLAMPED to ±ftnOverlayBound
// before being added to the backbone. Below the floor there is NO overlay — the composite
// is the pure Madden backbone.
//
// WHY THE BOUND (the panel's headline risk): FTN clears its floor at only n≈38–94, so two
// near-identical talents must not rank materially apart merely because one crossed the
// floor. The ±ftnOverlayBound clamp GUARANTEES a floor-crosser and an identical-Madden
// non-crosser can never diverge by more than ftnOverlayBound — it caps exactly that
// regime-switch discontinuity (Christopher, 2026-07-21: B=0.10).
//
// The upstream blend (this composite's SHARE of the film budget and the reserved K2
// NFLProduction seat) lives in rankings.applyScouting; this leaf owns only the raw ratings
// -> normalized composite, so the engine keeps consuming a single [0,1] FilmComposite and
// this package imports no engine/store.
//
// HARD BOUNDARY: the offense film composite applies at QB/RB/WR/TE ONLY. A defender or K is
// skipped (offenseMaddenComposite returns ok=false for a position with no curated set) —
// the composite must not bleed to any other position.
//
// ZERO-LEAK (hard constraint): RawMaddenRating carries only integer athletic/skill grades,
// and RawVeteranFilm carries only charted RATES (contested-catch rate, drop rate, …) — no
// fantasy points, projected volume, or MFL scoring column exists on either to bind.
//
// Failures: a genuine fetch failure of EITHER feed (network/HTTP/parse, or a fetch that
// resolved zero records) is surfaced loudly — a film-less league should be visible, matching
// BuildIDPFilm/BuildRAS. A player-level miss (no gsis, no Madden row, a non-offense position,
// a record with none of the curated attrs, or an FTN group below the floor) is ordinary.
func BuildOffenseFilm(
	ctx context.Context,
	client *http.Client,
	maddenURL string,
	ftnSources []veteranfilm.SeasonSource,
	recvFloor, passFloor int,
	cw crosswalk.Map,
	rosterMFLIDs []string,
	pos PositionLookup,
) (map[playerid.PlayerID]float64, error) {
	if client == nil {
		return nil, fmt.Errorf("assembly: BuildOffenseFilm requires a non-nil *http.Client")
	}
	if pos == nil {
		return nil, fmt.Errorf("assembly: BuildOffenseFilm requires a non-nil PositionLookup")
	}

	raw, err := madden.Fetch(ctx, client, maddenURL, cw.MaddenResolver())
	if err != nil {
		return nil, fmt.Errorf("assembly: fetch madden: %w", err)
	}
	ftn, err := veteranfilm.Fetch(ctx, client, ftnSources, recvFloor, passFloor)
	if err != nil {
		return nil, fmt.Errorf("assembly: fetch veteran film: %w", err)
	}

	rows := gatherOffenseRows(raw, ftn, cw, rosterMFLIDs, pos)
	ranker := newRolePercentiles(rows)

	out := make(map[playerid.PlayerID]float64, len(rows))
	for _, r := range rows {
		out[r.pid] = blendOffenseRow(r, ranker)
	}
	return out, nil
}

const (
	// ftnDiscount is the ~15% FTN fidelity discount (K3): the FTN charting proxy is
	// trusted, but thinner and less durable than Madden, so its signed nudge is scaled to
	// 0.85 of the raw gap before it is bounded and added to the backbone.
	ftnDiscount = 0.85
	// ftnOverlayBound is the LOCKED max the FTN overlay can move the Madden backbone
	// (Christopher, 2026-07-21: B=0.10). It caps the floor-crosser-vs-non-crosser
	// discontinuity the C-3 panel flagged as K3's headline risk.
	ftnOverlayBound = 0.10
)

// offenseRole marks which FTN role group a position draws its charting from: passers pull
// PasserFilm, everyone else (RB/WR/TE) pulls ReceiverFilm.
type offenseRole int

const (
	roleReceiver offenseRole = iota
	rolePasser
)

func offenseRoleOf(pos domain.Position) offenseRole {
	if pos == domain.PosQB {
		return rolePasser
	}
	return roleReceiver
}

// offenseFilmRow is one rostered offense player's assembled inputs: his Madden backbone
// (always present — a row exists only when the Madden record resolved), plus the raw FTN
// role quality when he cleared the charting floor. ftnQuality is percentile-ranked into a
// [0,1] overlay input later, once the whole population is known.
type offenseFilmRow struct {
	pid        playerid.PlayerID
	role       offenseRole
	backbone   float64
	ftnQuality float64
	hasFTN     bool
}

// gatherOffenseRows joins each rostered offense id → gsis → Madden backbone, attaching the
// player's raw FTN role quality when his role group cleared the floor. A player with no
// resolved Madden composite (non-offense position, no row, or no curated attrs) yields no
// row — his film stays Data-Parity neutral.
func gatherOffenseRows(
	raw map[string]madden.RawMaddenRating,
	ftn map[string]veteranfilm.RawVeteranFilm,
	cw crosswalk.Map,
	rosterMFLIDs []string,
	pos PositionLookup,
) []offenseFilmRow {
	rows := make([]offenseFilmRow, 0, len(rosterMFLIDs))
	seen := make(map[playerid.PlayerID]bool, len(rosterMFLIDs))
	for _, mfl := range rosterMFLIDs {
		pid, err := playerid.New(mfl)
		if err != nil {
			continue // malformed — an upstream layer should have caught this; skip
		}
		if seen[pid] {
			continue // a duplicate roster id would double-count in the percentile population
		}
		seen[pid] = true
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
			continue // no resolved position → cannot apply the offense boundary — ordinary miss
		}
		backbone, ok := offenseMaddenComposite(rc, position)
		if !ok {
			continue // non-offense position, or none of the curated attrs present — no signal
		}
		role := offenseRoleOf(position)
		quality, hasFTN := ftnRoleQuality(ftn[gsis], role)
		rows = append(rows, offenseFilmRow{
			pid: pid, role: role, backbone: backbone, ftnQuality: quality, hasFTN: hasFTN,
		})
	}
	return rows
}

// blendOffenseRow applies the K3 overlay: below the floor (no FTN) the composite is the pure
// Madden backbone; above it, the FTN percentile gap to the backbone is discounted, clamped
// to ±ftnOverlayBound, and added — the whole thing clamped to [0,1].
func blendOffenseRow(r offenseFilmRow, ranker rolePercentiles) float64 {
	if !r.hasFTN || ranker.popSize(r.role) <= 1 {
		// Below the floor, OR the only charted player in his role: with no peer
		// distribution there is no percentile information, so the FTN overlay is inert and
		// the composite is the pure Madden backbone (same regime as below-floor). This is
		// deliberately NOT a pull toward the 0.5 midpoint (GLM C-4 step-3 review, MED-3).
		return r.backbone
	}
	ftnScore := ranker.percentile(r.role, r.ftnQuality)
	delta := ftnDiscount * (ftnScore - r.backbone)
	if delta > ftnOverlayBound {
		delta = ftnOverlayBound
	} else if delta < -ftnOverlayBound {
		delta = -ftnOverlayBound
	}
	return clamp01(r.backbone + delta)
}

// offenseMaddenComposite maps one player's raw Madden ratings to the engine-ready [0,1]
// offense film backbone, or ok=false when the row yields no usable signal: a non-offense
// position (the hard boundary), or a record carrying NONE of the curated attributes.
func offenseMaddenComposite(rc madden.RawMaddenRating, pos domain.Position) (float64, bool) {
	terms, ok := offenseMaddenTerms(pos)
	if !ok {
		return 0, false // non-offense position — the offense film composite is QB/RB/WR/TE ONLY
	}
	return averageTerms(rc, terms)
}

// offenseMaddenTerms is the LOCKED K3 curation: for each offense position, the terms the
// equal-weight backbone averages (a func, not a package global — gochecknoglobals). Each
// term is a list of EA attribute names averaged together BEFORE the composite mean — the
// terms mirror each position's rubric Madden-mapping table rows (docs/scoring-engine/
// {QB,RB,WR,TE}_Rubric.md, "Madden attribute mapping"). The few rubric rows with unequal
// intra-row weights (e.g. RB "0.7·CTH + 0.3·PBLK") are collapsed to an equal average of
// their attrs — the same equal-weight principle K1 locked for the coverage term; the mean
// is robust and the intra-row split carries negligible ordering information. ok=false marks
// a non-offense position (the hard boundary).
//
// INTER-ROW REPEATS ARE DELIBERATE (Christopher, 2026-07-21, ruling on the GLM C-4 step-3
// review MED-1): some attrs appear in two rows (WR spectacularCatch/catchInTraffic in the
// "good hands" AND "high-point specialist" rows; TE runBlock/catching/agility; QB awareness),
// so they carry ~2× weight. That is intended — the rubric ROWS are distinct scoring
// archetypes, and a player elite at two archetypes that both value an attr legitimately earns
// it in both. This is the per-row-mean reading of "full union from rubric tables", NOT a
// flatten-and-dedup; do not "fix" the repeats.
//
// NOTE: the EA field names below are the suffix-stripped "_rating" keys the madden fetcher
// emits (speed is confirmed against live fixtures; the rest follow EA's camelCase
// convention). averageTerms averages only the PRESENT terms, so a key that does not match a
// live attribute degrades that one term to absent rather than corrupting the backbone —
// VERIFY at the live gate that composites are non-degenerate.
func offenseMaddenTerms(pos domain.Position) ([][]string, bool) {
	switch pos {
	case domain.PosQB:
		return [][]string{
			{"throwPower"},
			{"throwAccuracyShort", "throwAccuracyMid"},
			{"throwAccuracyDeep"},
			{"throwOnTheRun", "acceleration"},
			{"awareness", "playAction"},
			{"throwUnderPressure", "awareness"},
		}, true
	case domain.PosRB:
		return [][]string{
			{"speed", "acceleration"},
			{"trucking", "breakTackle", "strength"},
			{"jukeMove", "spinMove", "agility"},
			{"ballCarrierVision"},
			{"catching", "passBlock"},
		}, true
	case domain.PosWR:
		return [][]string{
			{"speed", "acceleration"},
			{"catching", "catchInTraffic", "spectacularCatch"},
			{"shortRouteRunning", "mediumRouteRunning", "deepRouteRunning"},
			{"release", "strength"},
			{"breakTackle", "jukeMove", "agility"},
			{"jumping", "spectacularCatch", "catchInTraffic"},
			{"trucking", "breakTackle", "strength"},
		}, true
	case domain.PosTE:
		return [][]string{
			{"speed", "acceleration"},
			{"runBlock", "passBlock", "strength"},
			{"catching", "catchInTraffic", "jumping"},
			{"shortRouteRunning", "mediumRouteRunning"},
			{"breakTackle", "agility"},
			{"runBlock", "catching", "agility"},
		}, true
	case domain.PosDT, domain.PosDE, domain.PosLB, domain.PosCB, domain.PosS,
		domain.PosK, domain.PosFlag:
		return nil, false // non-offense — the offense film composite is QB/RB/WR/TE ONLY
	default:
		return nil, false
	}
}

// ftnRoleQuality collapses one player's raw FTN charting to a single quality scalar (higher
// = better) for his role, or ok=false when his role group is absent (below the charting
// floor or never filled). Receiver quality is the equal-weight mean of contested-catch rate,
// created-reception rate, and (1 − drop rate) — drop inverted so higher is better. Passer
// quality is (1 − interception-worthy rate) — the one unambiguously negative-avoidance passer
// trait; ThrowawayRate is held by the fetcher but EXCLUDED here, its sign being ambiguous (a
// throwaway is often GOOD decision-making), pending a calibration call.
func ftnRoleQuality(rf veteranfilm.RawVeteranFilm, role offenseRole) (float64, bool) {
	switch role {
	case rolePasser:
		if rf.Passer == nil {
			return 0, false
		}
		return clamp01(1 - rf.Passer.InterceptionWorthyRate), true
	case roleReceiver:
		if rf.Receiver == nil {
			return 0, false
		}
		r := rf.Receiver
		q := (r.ContestedCatchRate + r.CreatedReceptionRate + (1 - r.DropRate)) / 3.0
		return clamp01(q), true
	default:
		return 0, false
	}
}

// rolePercentiles holds the sorted above-floor FTN quality distribution per role so a
// player's charted quality can be mapped to its percentile within his own role population.
type rolePercentiles struct {
	byRole map[offenseRole][]float64
}

// newRolePercentiles collects the sorted FTN quality values of every above-floor row, keyed
// by role — the population each overlay percentile is measured against.
//
// POPULATION = MADDEN-RESOLVED above-floor players (Christopher, 2026-07-21, ruling on the
// GLM C-4 step-3 review MED-2): a row exists only when the Madden backbone resolved, so an
// above-floor player with no Madden record is NOT in this population. That is intended for
// v1 — the percentile drives an overlay only for players who receive a composite, and the
// ±ftnOverlayBound clamp caps any bias from the narrower denominator. Revisit at the film
// calibration pass if the FTN population should span all above-floor rostered players.
func newRolePercentiles(rows []offenseFilmRow) rolePercentiles {
	byRole := map[offenseRole][]float64{}
	for _, r := range rows {
		if r.hasFTN {
			byRole[r.role] = append(byRole[r.role], r.ftnQuality)
		}
	}
	for role := range byRole {
		sort.Float64s(byRole[role])
	}
	return rolePercentiles{byRole: byRole}
}

// popSize is the above-floor charted population for a role — the peer count a percentile is
// measured against. blendOffenseRow uses it to make a singleton's overlay inert (no peers).
func (p rolePercentiles) popSize(role offenseRole) int { return len(p.byRole[role]) }

// percentile maps a quality value to its [0,1] rank within its role population using the
// midpoint (Hazen) convention: (countBelow + 0.5·countEqual) / n. A singleton population
// returns 0.5, but blendOffenseRow short-circuits singletons to the backbone before calling
// this, so that 0.5 never actually drives an overlay.
func (p rolePercentiles) percentile(role offenseRole, q float64) float64 {
	q = clamp01(q) // the Nextafter equal-count trick below is only valid for q ∈ [0,1] (LOW-1)
	vals := p.byRole[role]
	n := len(vals)
	if n <= 1 {
		return 0.5
	}
	below := sort.SearchFloat64s(vals, q)                    // count strictly < q (values are sorted)
	equal := sort.SearchFloat64s(vals, math.Nextafter(q, 2)) // index past the last == q
	equal -= below
	return (float64(below) + 0.5*float64(equal)) / float64(n)
}

// clamp01 constrains a value to [0,1] — a defensive rail for both the normalized rates and
// the overlaid composite.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// OffenseFilmGroup wraps a blended offense composite as the non-nil scouting.OffenseFilm
// group the merge step writes into Profile.OffenseFilm (present at QB/RB/WR/TE when the
// Madden record resolved). Exported so the app's thin merge loop stays a copy — mirroring
// IDPFilmGroup and every other S-Phase signal.
func OffenseFilmGroup(composite float64) *scouting.OffenseFilm {
	return &scouting.OffenseFilm{Composite: composite}
}
