package nativecheck_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func capRules() leaguerules.CapRules { return leaguerules.LegacyNFL().Cap }

func capFigure(t *testing.T, label string, got *domain.Money, want domain.Money) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %d cents", label, got, want)
	}
}

func TestNativeCapCountsReservesAndSignedCharges(t *testing.T) {
	players := []nativecheck.CapPlayer{{Salary: 10_987_00000, Status: domain.RosterActive},
		{Salary: 5_555_55555, Status: domain.RosterIR}, {Salary: 3_333_33333, Status: domain.RosterTaxi}}
	state, err := nativecheck.CheckCap(capRules(), players, []domain.Money{2_500_00000, -1_200_00000}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	capFigure(t, "active salary", state.ActiveCents, 10_990_00000)
	capFigure(t, "IR salary", state.IRCents, 5_560_00000)
	capFigure(t, "taxi salary", state.TaxiCents, 3_330_00000)
	capFigure(t, "full contracted salary", state.SalaryCents, 19_880_00000)
	capFigure(t, "signed adjustments and dead money", state.AdjustmentCents, 1_300_00000)
	capFigure(t, "committed salary", state.CommittedCents, 21_180_00000)
	capFigure(t, "cap room", state.RoomCents, 103_820_00000)
	if state.Words != "Within the salary cap; $103,820,000.00 cap room" {
		t.Fatal(state.Words)
	}
}

func TestNativeCapRoundsEachCountedSalaryAfterPercentageMath(t *testing.T) {
	rules := capRules()
	rules.IRShare = leaguerules.Known(leaguerules.Decimal(3500))
	// Percentage math rounds 499999.5 cents up to 500000 cents, then to $10k, not zero.
	rules.TaxiShare = leaguerules.Known(leaguerules.Decimal(5000))
	players := []nativecheck.CapPlayer{{Salary: 999999, Status: domain.RosterTaxi},
		{Salary: 1_428_571, Status: domain.RosterIR}, {Salary: 1_428_571, Status: domain.RosterIR}}
	state, err := nativecheck.CheckCap(rules, players, []domain.Money{-500000}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	capFigure(t, "taxi rounds cents before snapping", state.TaxiCents, 1000000)
	capFigure(t, "each IR charge snaps separately", state.IRCents, 2000000)
	capFigure(t, "negative half snaps away from zero", state.AdjustmentCents, -1000000)
	capFigure(t, "counted total", state.CommittedCents, 2000000)
	rules.Quantum = leaguerules.Known(leaguerules.Money(100))
	state, err = nativecheck.CheckCap(rules, players, nil, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	capFigure(t, "commissioner money increment", state.IRCents, 1000000)
}

func TestNativeCapMoveEnforcementAndConditionalDrop(t *testing.T) {
	deadline := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	for _, mode := range []leaguerules.CapMode{leaguerules.HardCap, leaguerules.SoftCap} {
		rules := capRules()
		rules.Enforcement = leaguerules.Known(mode)
		state, err := nativecheck.CheckCap(rules, []nativecheck.CapPlayer{{Salary: 126_000_00000, Status: domain.RosterActive}}, nil, time.UTC)

		if err != nil {
			t.Fatal(err)
		}
		if state.Words != "Over the salary cap by $1,000,000.00" {
			t.Fatal(state.Words)
		}
		allowed, words := nativecheck.CapAllows(state, 0, time.UTC)
		if allowed != (mode == leaguerules.SoftCap) {
			t.Fatalf("%s over-cap move: %v %s", mode, allowed, words)
		}
		if mode == leaguerules.SoftCap && !strings.Contains(words, "deadline needs a ruling") {
			t.Fatal(words)
		}
		if allowed, words := nativecheck.CapAllows(state, -1_000_00000, time.UTC); !allowed ||
			words != "The move ends within the salary cap" {
			t.Fatalf("%s conditional drop: %v %s", mode, allowed, words)
		}
		rules.ComplianceDeadline = leaguerules.Known(deadline)
		state, err = nativecheck.CheckCap(rules, []nativecheck.CapPlayer{{Salary: 126_000_00000, Status: domain.RosterActive}}, nil, time.UTC)

		if err != nil {
			t.Fatal(err)
		}
		if mode == leaguerules.SoftCap && state.Enforcement !=
			"Over the salary cap: flagged; comply by 20 Oct 2026 at 12:00 AM UTC" {
			t.Fatal(state.Enforcement)
		}
	}
}

func TestNativeCapMoveUsesNetCountedDeltaAndRounding(t *testing.T) {
	state, err := nativecheck.CheckCap(capRules(), []nativecheck.CapPlayer{{Salary: 125_000_00000, Status: domain.RosterActive}}, nil, time.UTC)

	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		delta domain.Money
		allow bool
	}{{0, true}, {499999, true}, {500000, false}, {-500000, true}, {math.MaxInt64, false}} {
		if allow, words := nativecheck.CapAllows(state, tc.delta, time.UTC); allow != tc.allow {
			t.Fatalf("net delta %d: %v %s", tc.delta, allow, words)
		}
	}
}

func TestNativeCapNeedsRulesRatherThanInventingZero(t *testing.T) {
	for _, field := range []string{"cap", "ir", "taxi", "quantum", "enforcement"} {
		rules := capRules()
		switch field {
		case "cap":
			rules.Amount = leaguerules.Needs[leaguerules.Money]("")
		case "ir":
			rules.IRShare = leaguerules.Needs[leaguerules.Decimal]("")
		case "taxi":
			rules.TaxiShare = leaguerules.Needs[leaguerules.Decimal]("")
		case "quantum":
			rules.Quantum = leaguerules.Needs[leaguerules.Money]("")
		case "enforcement":
			rules.Enforcement = leaguerules.Needs[leaguerules.CapMode]("")
		}
		state, err := nativecheck.CheckCap(rules, nil, nil, time.UTC)
		if err != nil {
			t.Fatal(err)
		}
		if allow, words := nativecheck.CapAllows(state, 0, time.UTC); allow || !strings.Contains(words, "Needs a ruling") {
			t.Fatalf("%s: %v %s", field, allow, words)
		}
		if field != "enforcement" && state.RoomCents != nil {
			t.Fatalf("%s invented cap room: %+v", field, state)
		}
	}
}

func TestNativeCapSnapsTheCapAndMatchesTheSharedLegacySnap(t *testing.T) {
	for _, amount := range []domain.Money{-1500001, -1500000, -500000, -499999, 0,
		499999, 500000, 1500000, 1500001, 12500500000} {
		got, err := nativecheck.SnapCapMoney(amount, 1000000)
		if err != nil || got != domain.RoundToNearest10k(amount) {
			t.Fatalf("signed legacy snap %d = %d, %v", amount, got, err)
		}
	}
	rules := capRules()
	rules.Amount = leaguerules.Known(leaguerules.Money(12500500000))
	state, err := nativecheck.CheckCap(rules, nil, nil, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	capFigure(t, "the cap itself is snapped", state.CapCents, 12501000000)
	capFigure(t, "empty franchise has the whole snapped cap", state.RoomCents, 12501000000)
}

func TestNativeCapRejectsOverflowAndInvalidContracts(t *testing.T) {
	for _, players := range [][]nativecheck.CapPlayer{
		{{Salary: -1, Status: domain.RosterActive}},
		{{Salary: 100, Status: "unknown"}},
		{{Salary: math.MaxInt64, Status: domain.RosterActive}},
		{{Salary: 5_000_000_000_000_000_000, Status: domain.RosterActive},
			{Salary: 5_000_000_000_000_000_000, Status: domain.RosterActive}},
	} {
		if _, err := nativecheck.CheckCap(capRules(), players, nil, time.UTC); err == nil {
			t.Fatalf("invalid or overflowing salary accepted: %+v", players)
		}
	}
	if _, err := nativecheck.CheckCap(capRules(), nil, []domain.Money{math.MinInt64}, time.UTC); err == nil {
		t.Fatal("overflowing signed charge accepted")
	}
}
