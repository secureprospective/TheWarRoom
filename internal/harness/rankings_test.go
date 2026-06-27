package harness

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

type stubParams struct{}

func (stubParams) GetCapTiers() (params.CapTiers, error) {
	return params.CapTiers{ColdCeiling: 1.2, HotFloor: 4.8}, nil
}
func (stubParams) GetGlobal(string) (float64, error) { return 0.03, nil }

type stubCap struct{}

func (stubCap) GetSalaryCap() string { return "1000" }

func testAssembler() *composition.Assembler { return composition.New(stubParams{}, stubCap{}) }

func TestRankRookiesSortedAndScored(t *testing.T) {
	rows := RankRookies(testAssembler(), SampleRookies(), RubricRegistry{})
	if len(rows) != len(SampleRookies()) {
		t.Fatalf("expected %d rows, got %d", len(SampleRookies()), len(rows))
	}
	// Descending AdjustedScore, no errors in the sample set.
	for i := 1; i < len(rows); i++ {
		if rows[i-1].Err == "" && rows[i].Err == "" &&
			rows[i-1].Result.AdjustedScore < rows[i].Result.AdjustedScore {
			t.Fatalf("rows not sorted desc at %d: %v < %v",
				i, rows[i-1].Result.AdjustedScore, rows[i].Result.AdjustedScore)
		}
		if rows[i].Err != "" {
			t.Fatalf("unexpected error row: %s", rows[i].Err)
		}
	}
	// WR Bravo has HasRAS=false → flagged imputed.
	for _, r := range rows {
		if r.MFLID == "0302" && !r.RASImputed {
			t.Fatalf("WR Bravo should be RAS-imputed")
		}
	}
}

// TestRankRookiesSurfacesFailures: a bad spec is not dropped — it appears with Err set and
// sinks below the scored rows.
func TestRankRookiesSurfacesFailures(t *testing.T) {
	specs := []composition.PlayerSpec{
		{MFLID: "0301", Name: "WR Good", Position: domain.PosWR, BasePoints: 200, Age: 22, RAS: 9, HasRAS: true, Salary: 10},
		{MFLID: "9999", Name: "Broken", Position: domain.PosFlag, BasePoints: 100, Age: 22}, // FLAG → unscorable
	}
	rows := RankRookies(testAssembler(), specs, RubricRegistry{})
	if rows[0].Err != "" {
		t.Fatalf("scored row should sort first, got error row: %s", rows[0].Err)
	}
	last := rows[len(rows)-1]
	if last.MFLID != "9999" || last.Err == "" {
		t.Fatalf("broken spec should sink to bottom with Err set, got %+v", last)
	}
}
