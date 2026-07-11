package normalize

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
)

// TestClassifyPosition locks the raw-MFL-code → engine-position mapping, including
// the two remaps (PK→K, EDGE→DE per OQ-004), XX→FLAG, unknown→FLAG, and aggregate
// detection.
func TestClassifyPosition(t *testing.T) {
	posMap := newPositionMap()
	aggSet := newAggregateSet()

	tests := []struct {
		raw     string
		wantPos domain.Position
		wantAgg bool
	}{
		{"QB", domain.PosQB, false},
		{"S", domain.PosS, false},
		{"PK", domain.PosK, false},    // remap
		{"EDGE", domain.PosDE, false}, // OQ-004 remap
		{"DE", domain.PosDE, false},
		{"XX", domain.PosFlag, false},        // unclassified → admin
		{"Synergist", domain.PosFlag, false}, // unknown code → admin
		{"Def", "", true},                    // aggregate
		{"TMWR", "", true},                   // aggregate
		{"Coach", "", true},                  // aggregate
		{" QB ", domain.PosQB, false},        // trimmed
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			pos, agg := classifyPosition(tt.raw, posMap, aggSet)
			if pos != tt.wantPos || agg != tt.wantAgg {
				t.Fatalf("classifyPosition(%q) = (%q,%v), want (%q,%v)", tt.raw, pos, agg, tt.wantPos, tt.wantAgg)
			}
		})
	}
}

// TestNewLookup_ReservedRangeFlags is the B3 reserved-range policy (Christopher):
// a non-aggregate record carrying an id in [151,782] is FLAGGED for admin review and
// the build CONTINUES — it must not halt the whole league's normalization. Both a
// real-position player and an unknown code flag; a record after them still lands,
// proving the build kept going.
func TestNewLookup_ReservedRangeFlags(t *testing.T) {
	lk, err := NewLookup([]players.RawPlayer{
		{ID: "500", Name: "Leak, Real", Position: "QB", Team: "KCC"},     // real pos in range → flag
		{ID: "600", Name: "Phantom, Who", Position: "XX", Team: "FA"},    // unknown in range → flag
		{ID: "13593", Name: "Real, Player", Position: "WR", Team: "KCC"}, // out of range → normal
	})
	if err != nil {
		t.Fatalf("reserved-range anomaly must not halt the build: %v", err)
	}
	for _, id := range []string{"500", "600"} {
		e, ok := lk.entry(mustID(t, id))
		if !ok || e.position != domain.PosFlag {
			t.Fatalf("id %s should be admitted and flagged for review, got %+v ok=%v", id, e, ok)
		}
	}
	if e, ok := lk.entry(mustID(t, "13593")); !ok || e.position != domain.PosWR {
		t.Fatalf("build did not continue past the flagged records: %+v ok=%v", e, ok)
	}
}

// TestNewLookup_AggregateInRangeOK confirms the inverse: an ACTUAL aggregate
// ("Def") in the same id range is expected there and must NOT error.
func TestNewLookup_AggregateInRangeOK(t *testing.T) {
	lk, err := NewLookup([]players.RawPlayer{
		{ID: "500", Name: "Chiefs D", Position: "Def", Team: "KCC"},
	})
	if err != nil {
		t.Fatalf("aggregate in range should be fine: %v", err)
	}
	if e, ok := lk.entry(mustID(t, "500")); !ok || !e.isAggregate {
		t.Fatalf("aggregate not recorded as aggregate: %+v ok=%v", e, ok)
	}
}

// TestNewLookup_CanonicalizesAndDerivesRookie confirms id canonicalization (the
// roster join must hit regardless of leading-zero form) and rookie derivation.
func TestNewLookup_CanonicalizesAndDerivesRookie(t *testing.T) {
	// id 99 is below the aggregate range [151,782], so a real QB there is fine.
	lk, err := NewLookup([]players.RawPlayer{
		{ID: "0099", Name: "Young, Bryce", Position: "QB", Team: "CAR"},
		{ID: "14999", Name: "Rookie, Some", Position: "WR", Team: "FA", Status: "R"},
	})
	if err != nil {
		t.Fatalf("NewLookup: %v", err)
	}
	// "99" and "0099" canonicalize to the same key.
	if _, ok := lk.entry(mustID(t, "99")); !ok {
		t.Fatal("canonical id lookup missed: 99 should match 0099")
	}
	rookie, ok := lk.entry(mustID(t, "14999"))
	if !ok || !rookie.isRookie {
		t.Fatalf("rookie flag not derived from status==R: %+v ok=%v", rookie, ok)
	}
}

// TestFacts locks the M1 orchestrator's read surface: identity + birthdate
// resolution, canonical-id matching, and the aggregate/unknown → ok=false rule.
func TestFacts(t *testing.T) {
	lk, err := NewLookup([]players.RawPlayer{
		{ID: "0099", Name: "Young, Bryce", Position: "QB", Team: "CAR", Birthdate: "996642000", DraftYear: "2023"},
		{ID: "0816", Name: "Gosnell, Created", Position: "WR", Team: "FA", DraftYear: "0"}, // commissioner-created: no birthdate, "0" undrafted sentinel
		{ID: "0200", Name: "", Position: "TMQB", Team: "FA"},                               // aggregate
	})
	if err != nil {
		t.Fatalf("NewLookup: %v", err)
	}

	f, ok := lk.Facts("99") // canonical form matches "0099"
	if !ok || f.Name != "Young, Bryce" || !f.HasBirthdate || f.Birthdate != 996642000 {
		t.Fatalf("Facts(99) = %+v ok=%v, want Young w/ birthdate 996642000", f, ok)
	}
	if !f.HasDraftYear || f.DraftYear != 2023 {
		t.Fatalf("Facts(99) draft year = %d has=%v, want 2023 present", f.DraftYear, f.HasDraftYear)
	}
	created, ok := lk.Facts("0816")
	if !ok || created.HasBirthdate {
		t.Fatalf("Facts(0816) = %+v ok=%v, want present with HasBirthdate=false", created, ok)
	}
	if created.HasDraftYear { // "0" is the undrafted sentinel → dropped at typing
		t.Fatalf("Facts(0816) draft year = %d has=%v, want HasDraftYear=false (sentinel \"0\")", created.DraftYear, created.HasDraftYear)
	}
	if _, ok := lk.Facts("0200"); ok {
		t.Fatal("Facts(0200) should be ok=false for a team aggregate")
	}
	if _, ok := lk.Facts("55555"); ok {
		t.Fatal("Facts(55555) should be ok=false for an unknown id")
	}
}

// TestNewLookup_MalformedBirthdateFailsLoud proves a corrupt DETAILS=1 birthdate
// cannot ride into the typed lookup (the fetcher validates too — defense in depth).
func TestNewLookup_MalformedBirthdateFailsLoud(t *testing.T) {
	_, err := NewLookup([]players.RawPlayer{
		{ID: "0099", Name: "Young, Bryce", Position: "QB", Team: "CAR", Birthdate: "1979-08-03"},
	})
	if err == nil {
		t.Fatal("NewLookup should error on a non-epoch birthdate, got nil")
	}
}

// TestNewLookup_MalformedDraftYearFailsLoud proves a corrupt DETAILS=1 draft year cannot ride into
// the typed lookup (the fetcher validates too — defense in depth).
func TestNewLookup_MalformedDraftYearFailsLoud(t *testing.T) {
	_, err := NewLookup([]players.RawPlayer{
		{ID: "0099", Name: "Young, Bryce", Position: "QB", Team: "CAR", DraftYear: "twenty-twenty"},
	})
	if err == nil {
		t.Fatal("NewLookup should error on a non-numeric draft year, got nil")
	}
}
