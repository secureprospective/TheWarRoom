package labcompile

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

func TestCompilerSeam(t *testing.T) {
	_, err := Compile(t.Context(), facts.Facts{}, Evidence{})
	if err == nil || !strings.Contains(err.Error(), "timestamps") {
		t.Fatalf("unexpected seam error: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Compile(ctx, facts.Facts{}, Evidence{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	in := Evidence{Built: time.Unix(1, 0), ArchiveThrough: time.Unix(1, 0)}
	if _, err := compile(t.Context(), facts.Facts{}, in, nil); err == nil || !strings.Contains(err.Error(), "analysis function") {
		t.Fatalf("nil analyse: %v", err)
	}
	if _, err := compile(ctx, facts.Facts{}, in, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel compile: %v", err)
	}
	for _, invalid := range []Evidence{{}, {Built: in.Built, ArchiveThrough: in.ArchiveThrough, Pages: PageExam{1, 2}}, {Built: in.Built, ArchiveThrough: in.ArchiveThrough, Pages: PageExam{-1, 0}}} {
		if invalid.Validate() == nil {
			t.Fatal("accepted invalid evidence")
		}
	}
}
func TestMissingFactsAndRawInputs(t *testing.T) {
	l, err := newLab(facts.Facts{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.seasons(facts.Facts{}); err == nil || !strings.Contains(err.Error(), "missing season settings") {
		t.Fatalf("missing setting: %v", err)
	}
	if _, err := newLab(facts.Facts{Player: facts.PlayerColumns{Pid: []string{"001"}, Name: []string{"A"}, Birthdate: []*string{ptr("invalid")}}}, 0); err == nil {
		t.Fatal("accepted invalid birthdate")
	}
	if (RawTradeOrder{A: "bad\x00"}).Validate() == nil {
		t.Fatal("accepted invalid trade order")
	}
	if _, err := orderAssets(facts.Facts{}, 1, "0001", "missing,", nil); err == nil {
		t.Fatal("guessed absent trade asset")
	}
	if _, err := eraOf(2012); err == nil {
		t.Fatal("guessed absent era")
	}
	if normPos(ptr("Def")) != nil || normPos(ptr("Off")) != nil {
		t.Fatal("aggregate positions must remain unknown")
	}
	if contractUntil(ptr("EXT2025")) != nil {
		t.Fatal("contract text became a year")
	}
}
func validAnalysis() RawAnalysis {
	return RawAnalysis{Market: Market{Anchor: "pick1:next", Classes: []MarketClass{}}, Checks2: AnalysisChecks{Forecast: map[string]ForecastCheck{}}}
}
func TestAnalysisValidation(t *testing.T) {
	if (RawAnalysis{}).Validate() == nil {
		t.Fatal("accepted absent analysis")
	}
	raw := validAnalysis()
	raw.PickSlots = []AnalysisPick{{Values: []*float64{ptr(1.0)}}}
	if raw.Validate() == nil {
		t.Fatal("accepted incomplete horizons")
	}
	raw = validAnalysis()
	raw.Legs = []AnalysisLeg{{Value: ptr(math.NaN())}}
	if raw.Validate() == nil {
		t.Fatal("accepted NaN")
	}
	raw = validAnalysis()
	raw.Delivered = []AnalysisDelivery{{Expected: math.Inf(1)}}
	if raw.Validate() == nil {
		t.Fatal("accepted infinite delivery")
	}
	trades := []Trade{{AGave: []Asset{{Kind: "player"}}}}
	for _, a := range []Analysis{{Legs: []AnalysisLeg{{Trade: 1}}}, {Legs: []AnalysisLeg{{Side: 2}}}, {Legs: []AnalysisLeg{{Asset: 1}}}, {Legs: []AnalysisLeg{{}, {}}}, {Deals: []AnalysisDeal{{Trade: -1}}}, {Deals: []AnalysisDeal{{}, {}}}, {Delivered: []AnalysisDelivery{{Trade: 1}}}} {
		if a.validateIndices(trades) == nil {
			t.Fatal("accepted invalid or duplicate index")
		}
	}
	if err := Analysis(validAnalysis()).validateIndices(trades); err != nil {
		t.Fatal(err)
	}
}
func TestUndefinedLabMethods(t *testing.T) {
	l, err := newLab(facts.Facts{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	l.buildCalendar(nil, nil)
	if l.weekAt(2026, 0) != 0 || l.contract("001", 2026, 0, "0001") != nil || l.contractAfter("001", 2026, 0, "0001") != nil {
		t.Fatal("absent snapshots fabricated data")
	}
	if l.capRoom(2026, 0, "0001") != nil || l.tierFromRank(ptr(int64(1)), ptr("QB"), 2026) != nil {
		t.Fatal("absent cap or starter line fabricated a result")
	}
	if use := l.afterUse("001", 2026, 0, "0001"); use.Starts != nil || use.Points != nil || use.Weeks != 0 {
		t.Fatalf("no played weeks: %+v", use)
	}
	if ledger := l.reconcile(nil, nil); ledger.Rate != nil || ledger.ChangeRate != nil || ledger.Rostered != 0 {
		t.Fatalf("empty ledger: %+v", ledger)
	}
	if l.knownTier("unknown", 2026, 0) != nil || l.pos("unknown", 2026) != nil {
		t.Fatal("unknown player got a position or tier")
	}
	if l.Marks[2026].Deadline != nil {
		t.Fatal("no prior deadlines fabricated a forecast")
	}
}
func TestPythonSumAndEasternDates(t *testing.T) {
	if pythonSum([]float64{1e16, 1, -1e16}) != 1 {
		t.Fatal("lost Python sum compensation")
	}
	cases := []struct {
		Date  string
		Epoch int64
	}{{"1988-03-21", 574923600}, {"2000-07-01", 962424000}, {"2026-03-08", 1772946000}}
	for _, c := range cases {
		d, err := time.Parse("2006-01-02", c.Date)
		if err != nil {
			t.Fatal(err)
		}
		if got := easternEpoch(d); got != c.Epoch {
			t.Errorf("%s got %d want %d", c.Date, got, c.Epoch)
		}
		if dayOf(easternEpoch(d)) != d {
			t.Fatal("date roundtrip")
		}
	}
}
func TestParityScalarRules(t *testing.T) {
	for _, c := range []struct {
		A, B, Path string
		Equal      bool
	}{{"1", "2", "count", false}, {"null", "0", "count", false}, {`"0151"`, `"151"`, "pid", false}, {"1.0", "1", "legs.salary[0]", true}, {"1.0", "1.0000000001", "legs.salary[0]", false}, {"1.0", "1.0000000001", "market.price", true}} {
		if sameJSONScalar([]byte(c.A), []byte(c.B), c.Path) != c.Equal {
			t.Fatalf("wrong scalar comparison %+v", c)
		}
	}
}
