package normalize

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
)

// aggregateAudit is the cross-tab of the players DB against the two filters that
// must agree for the rosters ID-range drop to be safe: the position-code classifier
// (normalize) and the ID-range check (ingestion.IsTeamAggregateID, what rosters
// uses). realInRange is the only failure bucket — a real playable-position player
// whose id sits in the reserved [151,782] block, which the rosters fetcher would
// silently drop. The others are reported for the record.
type aggregateAudit struct {
	realInRange    []players.RawPlayer // FAILURE: real position, id in [151,782]
	realOutOfRange int                 // real position, id out of range — the norm
	aggInRange     int                 // aggregate, id in range — intended (rosters drops)
	aggOutOfRange  []players.RawPlayer // aggregate, id out of range — join backstop covers
	flagInRange    int                 // unknown code (→PosFlag) in range — admin-flagged
}

// auditAggregateFilter classifies every raw player exactly as NewLookup does and
// buckets it by (position class) × (in reserved ID range). Pure: no I/O, so it is
// unit-testable with a planted violation and reused by the live test against real
// data. A record is "real" when it maps to an engine position (not an aggregate
// code and not the PosFlag catch-all).
func auditAggregateFilter(raws []players.RawPlayer) aggregateAudit {
	posMap := newPositionMap()
	aggSet := newAggregateSet()

	var a aggregateAudit
	for _, rp := range raws {
		pos, isAgg := classifyPosition(rp.Position, posMap, aggSet)
		inRange := ingestion.IsTeamAggregateID(rp.ID)
		isReal := !isAgg && pos != domain.PosFlag

		switch {
		case isReal && inRange:
			a.realInRange = append(a.realInRange, rp)
		case isReal:
			a.realOutOfRange++
		case isAgg && inRange:
			a.aggInRange++
		case isAgg:
			a.aggOutOfRange = append(a.aggOutOfRange, rp)
		case inRange: // PosFlag (unknown code) in range
			a.flagInRange++
		}
	}
	return a
}

// TestAuditAggregateFilter_CatchesRealInRange proves the audit has teeth: a real
// QB whose id falls in [151,782] MUST land in realInRange (the bucket the live
// assertion fails on). Without this, a green live run would prove nothing — the
// check could be vacuously passing. (Technical pillar: a gate is not real until a
// planted violation is seen to fail it.)
func TestAuditAggregateFilter_CatchesRealInRange(t *testing.T) {
	raws := []players.RawPlayer{
		{ID: "0500", Name: "Planted, Real", Position: "QB", Team: "FA"}, // 500 ∈ [151,782]
	}
	a := auditAggregateFilter(raws)
	if len(a.realInRange) != 1 {
		t.Fatalf("planted real-player-in-range not caught: realInRange=%d, want 1", len(a.realInRange))
	}
	if a.realInRange[0].ID != "0500" {
		t.Errorf("caught wrong record: %q", a.realInRange[0].ID)
	}
}

// TestAuditAggregateFilter_Buckets confirms each class lands where expected and the
// danger bucket stays empty for well-formed data: a real player out of range, a
// team aggregate in range (the intended drop), and an aggregate out of range (the
// live "Coach"/"PN" case — join backstop, not a failure).
func TestAuditAggregateFilter_Buckets(t *testing.T) {
	raws := []players.RawPlayer{
		{ID: "13345", Name: "Real, Outofrange", Position: "WR", Team: "KC"}, // real, > 782
		{ID: "0200", Name: "", Position: "Def", Team: "KC"},                 // aggregate, in range
		{ID: "16147", Name: "", Position: "Coach", Team: "KC"},              // aggregate, out of range
	}
	a := auditAggregateFilter(raws)

	if len(a.realInRange) != 0 {
		t.Errorf("realInRange should be empty, got %d", len(a.realInRange))
	}
	if a.realOutOfRange != 1 {
		t.Errorf("realOutOfRange = %d, want 1", a.realOutOfRange)
	}
	if a.aggInRange != 1 {
		t.Errorf("aggInRange = %d, want 1", a.aggInRange)
	}
	if len(a.aggOutOfRange) != 1 {
		t.Errorf("aggOutOfRange = %d, want 1", len(a.aggOutOfRange))
	}
}
