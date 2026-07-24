package pfrpassrush

// C-1 DISTRIBUTION SAMPLER (THROWAWAY — delete after the evidence sheet is captured, per the
// cmd/defsample precedent). This sets NO weight and locks NO decision: it is the evidence the
// pressure-composite recipe + the SL-021 EMA new_observation weight rest on (FILM_Calibration
// C-1). It pulls the live pfrpassrush pass-rush population + the live Madden defense ratings and
// reports, per DL/edge position: the distribution of each CANDIDATE pressure composite, and its
// Pearson correlation with the LOCKED Madden IDP composite (K1). A high |r| means the pressure
// signal double-counts Madden (little new information for the EMA to smooth); a low |r| means it
// is a genuine independent grade that earns a real seat.
//
//	TWR_C1_SAMPLE=1 go test -run TestC1_PassRushDistributions -v ./internal/ingestion/pfrpassrush/...

import (
	"context"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/madden"
)

// c1Season is the pass-rush season sampled; c24 Madden is the only live EA slug (m24 = 2023),
// so 2023 pairs the two feeds on the same football year.
const c1Season = "2023"

// candidate is one pressure-composite recipe under evaluation as the SL-021 new_observation.
// value returns the raw per-game rate (NOT yet [0,1]-normalized — normalization is part of what
// the evidence decides) or ok=false when the row cannot produce it (no games played).
type candidate struct {
	name  string
	value func(RawPassRush) (float64, bool)
}

func c1Candidates() []candidate {
	perGame := func(n float64, g int) (float64, bool) {
		if g <= 0 {
			return 0, false
		}
		return n / float64(g), true
	}
	return []candidate{
		{"pressures/g", func(r RawPassRush) (float64, bool) { return perGame(float64(r.Pressures), r.Games) }},
		{"sacks/g", func(r RawPassRush) (float64, bool) { return perGame(r.Sacks, r.Games) }},
		{"knockdowns/g", func(r RawPassRush) (float64, bool) { return perGame(float64(r.QBKnockdowns), r.Games) }},
		{"hurries/g", func(r RawPassRush) (float64, bool) { return perGame(float64(r.Hurries), r.Games) }},
		// components/g = the sack+knockdown+hurry sum per game (PFR's prss is its own aggregate;
		// this candidate avoids leaning on that single headline column).
		{"components/g", func(r RawPassRush) (float64, bool) {
			return perGame(r.Sacks+float64(r.QBKnockdowns)+float64(r.Hurries), r.Games)
		}},
	}
}

// idpBucket maps a raw PFR position label to the DL/edge bucket the pass-rush signal serves, or
// ok=false for a non-pass-rush position (CB/S/etc. — excluded from this sampler).
func idpBucket(pfrPos string) (string, bool) {
	p := strings.ToUpper(strings.TrimSpace(pfrPos))
	switch {
	case strings.Contains(p, "DT") || strings.Contains(p, "NT"):
		return "DT", true
	case strings.Contains(p, "DE") || strings.Contains(p, "EDGE"):
		return "DE", true
	case strings.Contains(p, "LB"): // OLB/ILB/MLB — the pressure side of LB
		return "LB", true
	default:
		return "", false
	}
}

// dlEdgeMaddenTerms replicates the LOCKED K1 curation for the three pass-rush buckets ONLY
// (throwaway copy of assembly.idpMaddenTerms — the sampler must not import the unexported recipe).
func dlEdgeMaddenTerms(bucket string) [][]string {
	switch bucket {
	case "DT":
		return [][]string{{"powerMoves"}, {"strength"}, {"blockShedding"}, {"finesseMoves"},
			{"acceleration"}, {"tackle"}, {"playRecognition"}}
	case "DE":
		return [][]string{{"speed"}, {"acceleration"}, {"powerMoves"}, {"strength"}, {"finesseMoves"},
			{"agility"}, {"blockShedding"}, {"tackle"}, {"pursuit"}, {"playRecognition"}}
	case "LB":
		return [][]string{{"speed"}, {"pursuit"}, {"tackle"}, {"hitPower"}, {"strength"},
			{"zoneCoverage"}, {"playRecognition"}, {"blockShedding"}, {"awareness"}, {"powerMoves"}}
	default:
		return nil
	}
}

// maddenComposite averages the present curated terms to [0,1] (the K1 recipe), ok=false when none
// are present.
func c1MaddenComposite(rc madden.RawMaddenRating, bucket string) (float64, bool) {
	var sum float64
	var n int
	for _, term := range dlEdgeMaddenTerms(bucket) {
		var ts float64
		var tc int
		for _, a := range term {
			if v, ok := rc.Attributes[a]; ok {
				ts += math.Min(1, math.Max(0, float64(v)/99.0))
				tc++
			}
		}
		if tc > 0 {
			sum += ts / float64(tc)
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

func pctl(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return math.NaN()
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func pearson(xs, ys []float64) float64 {
	n := float64(len(xs))
	if n < 2 {
		return math.NaN()
	}
	var sx, sy, sxx, syy, sxy float64
	for i := range xs {
		sx += xs[i]
		sy += ys[i]
		sxx += xs[i] * xs[i]
		syy += ys[i] * ys[i]
		sxy += xs[i] * ys[i]
	}
	den := math.Sqrt((n*sxx - sx*sx) * (n*syy - sy*sy))
	if den == 0 {
		return math.NaN()
	}
	return (n*sxy - sx*sy) / den
}

func TestC1_PassRushDistributions(t *testing.T) {
	if os.Getenv("TWR_C1_SAMPLE") != "1" {
		t.Skip("C-1 sampler: set TWR_C1_SAMPLE=1 to run (makes real network calls)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 120 * time.Second}

	cw, err := crosswalk.Fetch(ctx, client, crosswalk.SourceURL)
	if err != nil {
		t.Fatalf("crosswalk: %v", err)
	}
	rush, err := Fetch(ctx, client, SourceURL, c1Season, cw.PFRMap())
	if err != nil {
		t.Fatalf("pfrpassrush: %v", err)
	}
	mad, err := madden.Fetch(ctx, client, madden.RatingsURL, cw.MaddenResolver())
	if err != nil {
		t.Fatalf("madden: %v", err)
	}
	t.Logf("population: %d pass-rush records (%s), %d madden records (m24)", len(rush), c1Season, len(mad))

	reportSampler(t, rush, mad)
}

// reportSampler emits the per-bucket distribution + Madden-correlation sheet (the C-1 deliverable).
func reportSampler(t *testing.T, rush map[string]RawPassRush, mad map[string]madden.RawMaddenRating) {
	t.Helper()
	for _, bucket := range []string{"DT", "DE", "LB"} {
		perCand := map[string][]float64{}
		// perCandPaired[c] and maddenByCand[c] are appended IN LOCKSTEP — only when a player has
		// BOTH a Madden composite AND a valid candidate value — so pearson() sees index-aligned
		// arrays (a Games==0 player who lacks a per-game candidate must not desync the Madden arm).
		perCandPaired := map[string][]float64{}
		maddenByCand := map[string][]float64{}
		var nBucket, nPaired int
		for gsis, r := range rush {
			b, ok := idpBucket(r.Position)
			if !ok || b != bucket {
				continue
			}
			nBucket++
			mc, hasM := 0.0, false
			if rc, ok := mad[gsis]; ok {
				mc, hasM = c1MaddenComposite(rc, bucket)
			}
			if hasM {
				nPaired++
			}
			for _, c := range c1Candidates() {
				v, ok := c.value(r)
				if !ok {
					continue
				}
				perCand[c.name] = append(perCand[c.name], v)
				if hasM {
					perCandPaired[c.name] = append(perCandPaired[c.name], v)
					maddenByCand[c.name] = append(maddenByCand[c.name], mc)
				}
			}
		}
		t.Logf("=== %s (n=%d, paired-with-madden=%d) ===", bucket, nBucket, nPaired)
		for _, c := range c1Candidates() {
			vals := append([]float64(nil), perCand[c.name]...)
			sort.Float64s(vals)
			r := pearson(perCandPaired[c.name], maddenByCand[c.name])
			t.Logf("  %-14s n=%3d  p50=%.3f p75=%.3f p90=%.3f p95=%.3f max=%.3f  r(madden)=%+.3f",
				c.name, len(vals), pctl(vals, .50), pctl(vals, .75), pctl(vals, .90),
				pctl(vals, .95), pctl(vals, 1.0), r)
		}
	}
}
