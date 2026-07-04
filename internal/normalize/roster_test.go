package normalize

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func mustID(t *testing.T, raw string) playerid.PlayerID {
	t.Helper()
	id, err := playerid.New(raw)
	if err != nil {
		t.Fatalf("playerid.New(%q): %v", raw, err)
	}
	return id
}

// TestNormalizeContractStatus is the locked dirty-value TABLE: every contractStatus
// value confirmed in live MFL data (docs/data-layer/MFL_API_Reference.md) mapped to
// its expected clean enum. This is the reviewed-deliverable artifact — the next
// model sees every known case and its outcome, including the deliberate FLAGs.
func TestNormalizeContractStatus(t *testing.T) {
	tests := []struct {
		raw  string
		want domain.ContractStatus
	}{
		// Clean values.
		{"UFA", domain.CStatusUFA},
		{"RFA", domain.CStatusRFA},
		{"FT1", domain.CStatusFT1},
		{"FT2", domain.CStatusFT2},
		// Trailing-whitespace variants.
		{"UFA ", domain.CStatusUFA},
		{"UFA  ", domain.CStatusUFA},
		// Parenthetical variants — prefix match keeps the franchise tag.
		{"FT1 (2026)", domain.CStatusFT1},
		{"FT2 (2026)", domain.CStatusFT2},
		// Combined value that LEADS with a known tag → that tag.
		{"FT1+ EXT2 (2024)", domain.CStatusFT1},
		// Confirmed typo (exact and, defensively, suffixed).
		{"YFA", domain.CStatusUFA},
		{"YFA (2024)", domain.CStatusUFA},
		// EXT-led values are NOT a known contract tag → admin review.
		{"EXT (2024)", domain.CStatusFlag},
		{"EXT (2024) ", domain.CStatusFlag},
		{"EXT + FT1", domain.CStatusFlag},
		{"EXT2", domain.CStatusFlag},
		// Empty → admin review (not silently a valid status).
		{"", domain.CStatusFlag},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := normalizeContractStatus(tt.raw); got != tt.want {
				t.Fatalf("normalizeContractStatus(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseSalary(t *testing.T) {
	id := mustID(t, "0531")
	tests := []struct {
		raw     string
		want    domain.Money // exact cents ($1M = 1e8 cents)
		wantErr bool
	}{
		{"7", 700_000_000, false},
		{"1.30", 130_000_000, false},
		{"17.70", 1_770_000_000, false},
		{"", 0, false}, // empty is legitimately $0
		{"  2.5 ", 250_000_000, false},
		{"free", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := parseSalary(tt.raw, id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSalary(%q) err = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("parseSalary(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestNormalizeRosterStatus(t *testing.T) {
	id := mustID(t, "0531")
	if s, err := normalizeRosterStatus("ROSTER", id); err != nil || s != domain.RosterActive {
		t.Fatalf("ROSTER → (%q,%v)", s, err)
	}
	if s, err := normalizeRosterStatus("TAXI_SQUAD", id); err != nil || s != domain.RosterTaxi {
		t.Fatalf("TAXI_SQUAD → (%q,%v)", s, err)
	}
	if _, err := normalizeRosterStatus("IR", id); err == nil {
		t.Fatal("unknown roster status should fail loud")
	}
}

// testLookup is a small players DB for the join tests.
func testLookup(t *testing.T) Lookup {
	t.Helper()
	// ids 815/850 are above the aggregate range [151,782]; 815 keeps a leading
	// zero so the join can prove canonicalization on a sub-1000 id.
	lk, err := NewLookup([]players.RawPlayer{
		{ID: "0815", Name: "Kelce, Travis", Position: "TE", Team: "KCC"},
		{ID: "15283", Name: "Moore, Rondale", Position: "XX", Team: "MIN"}, // → PosFlag
		{ID: "0850", Name: "Aggregate D", Position: "Def", Team: "KCC"},    // aggregate
	})
	if err != nil {
		t.Fatalf("testLookup: %v", err)
	}
	return lk
}

// TestPlayer_Join exercises the full single-row normalize: salary→float,
// contractStatus dirty→enum, roster status, and the name/position join.
func TestPlayer_Join(t *testing.T) {
	lk := testLookup(t)
	raw := rosters.RawRoster{
		FranchiseID:    "0025",
		PlayerID:       "815", // un-padded — must still join 0815
		Salary:         "17.70",
		ContractYear:   "2026",
		ContractStatus: "UFA ",
		ContractInfo:   "see notes",
		Status:         "ROSTER",
	}
	pr, err := Player(raw, lk)
	if err != nil {
		t.Fatalf("Player: %v", err)
	}
	if pr.MFLID.String() != "0815" || pr.Name != "Kelce, Travis" || pr.Position != domain.PosTE {
		t.Fatalf("join wrong: %+v", pr)
	}
	if pr.Salary != domain.Money(1_770_000_000) || pr.ContractYear != 2026 || pr.ContractStatus != domain.CStatusUFA {
		t.Fatalf("field parse wrong: %+v", pr)
	}
	if pr.RosterStatus != domain.RosterActive || pr.NeedsAdminReview() {
		t.Fatalf("status/flag wrong: %+v", pr)
	}
}

// TestPlayer_FlagPosition: an XX player joins but is flagged for admin review, not
// dropped or guessed.
func TestPlayer_FlagPosition(t *testing.T) {
	lk := testLookup(t)
	pr, err := Player(rosters.RawRoster{FranchiseID: "0025", PlayerID: "15283", Status: "ROSTER"}, lk)
	if err != nil {
		t.Fatalf("Player: %v", err)
	}
	if pr.Position != domain.PosFlag || !pr.NeedsAdminReview() {
		t.Fatalf("XX should flag for review: %+v", pr)
	}
}

func TestPlayer_UnknownAndAggregate(t *testing.T) {
	lk := testLookup(t)
	// Unknown player id (not in the players DB).
	if _, err := Player(rosters.RawRoster{FranchiseID: "0025", PlayerID: "99999", Status: "ROSTER"}, lk); err == nil {
		t.Fatal("unknown player should fail loud")
	}
	// A roster row pointing at an aggregate (defensive: should never happen, but
	// must not produce a half-typed record).
	if _, err := Player(rosters.RawRoster{FranchiseID: "0025", PlayerID: "850", Status: "ROSTER"}, lk); err == nil {
		t.Fatal("aggregate reference should fail loud")
	}
}

// TestRosters_GroupAndOrder confirms grouping by franchise and deterministic order.
func TestRosters_GroupAndOrder(t *testing.T) {
	lk := testLookup(t)
	raws := []rosters.RawRoster{
		{FranchiseID: "0025", PlayerID: "815", Status: "ROSTER"},
		{FranchiseID: "0001", PlayerID: "815", Status: "TAXI_SQUAD"},
		{FranchiseID: "0025", PlayerID: "15283", Status: "ROSTER"},
	}
	got, err := Rosters(raws, lk)
	if err != nil {
		t.Fatalf("Rosters: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d franchises, want 2", len(got))
	}
	if got[0].FranchiseID != "0001" || got[1].FranchiseID != "0025" {
		t.Fatalf("not sorted by franchise: %+v", got)
	}
	if len(got[1].Players) != 2 {
		t.Fatalf("franchise 0025 should have 2 players, got %d", len(got[1].Players))
	}
	// Intra-roster determinism (B3 review): players sorted by canonical id, so
	// "0815" (Kelce) precedes "15283" (Moore) regardless of MFL feed order.
	if got[1].Players[0].MFLID.String() != "0815" || got[1].Players[1].MFLID.String() != "15283" {
		t.Fatalf("franchise 0025 players not in deterministic id order: %+v", got[1].Players)
	}
}
