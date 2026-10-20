package normalize

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
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
	if s, err := normalizeRosterStatus("INJURED_RESERVE", id); err != nil || s != domain.RosterIR {
		t.Errorf("INJURED_RESERVE = %q, %v; want IR", s, err)
	}
	if s, err := normalizeRosterStatus("IR", id); err != nil || s != domain.RosterIR {
		t.Fatalf("IR → (%q,%v)", s, err)
	}
	if _, err := normalizeRosterStatus("BOGUS", id); err == nil {
		t.Fatal("unknown roster status should fail loud")
	}
}

// TestContract_TypesRow exercises the single-row normalize: salary→cents, the un-padded id
// canonicalized, contractStatus dirty→enum and roster status.
func TestContract_TypesRow(t *testing.T) {
	raw := rosters.RawRoster{
		FranchiseID:    "0025",
		PlayerID:       "815", // un-padded — must still canonicalize to 0815
		Salary:         "17.70",
		ContractYear:   "2026",
		ContractStatus: "UFA ",
		ContractInfo:   "see notes",
		Status:         "ROSTER",
	}
	pr, err := Contract(raw)
	if err != nil {
		t.Fatalf("Contract: %v", err)
	}
	if pr.MFLID.String() != "0815" || pr.FranchiseID != "0025" || pr.ContractInfo != "see notes" {
		t.Fatalf("identity wrong: %+v", pr)
	}
	if pr.Salary != domain.Money(1_770_000_000) || pr.ContractYear != 2026 || pr.ContractStatus != domain.CStatusUFA {
		t.Fatalf("field parse wrong: %+v", pr)
	}
	if pr.RosterStatus != domain.RosterActive {
		t.Fatalf("status wrong: %+v", pr)
	}
}
