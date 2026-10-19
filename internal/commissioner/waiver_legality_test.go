package commissioner

import (
	"math"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestNativeWaiverDeadMoneyUsesConfiguredRateAndPerCellQuantum(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	rules.Season = 2026
	rules.Contracts.DeadCap = leaguerules.Known[leaguerules.Decimal](3500)
	rules.Cap.Quantum = leaguerules.Known[leaguerules.Money](1000000)
	player := NativePlayer{Salary: 105000000, ExpirationYear: 2028}
	// $367,500 per cell rounds to $370,000, then two future cells are counted.
	amount, err := waiverDeadMoney(player, rules)
	if err != nil || amount != 74000000 {
		t.Fatalf("per-cell dead money: %d, %v", amount, err)
	}
	rules.Contracts.DeadCap = leaguerules.Known[leaguerules.Decimal](2000)
	amount, err = waiverDeadMoney(player, rules)
	if err != nil || amount != 42000000 {
		t.Fatalf("configured rate: %d, %v", amount, err)
	}
	player.Restructured = true
	rules.Contracts.RestructuredDeadCap = leaguerules.Known[leaguerules.Decimal](5000)
	amount, err = waiverDeadMoney(player, rules)
	if err != nil || amount != 106000000 {
		t.Fatalf("restructured rate: %d, %v", amount, err)
	}
	player.ExpirationYear = 2026
	amount, err = waiverDeadMoney(player, rules)
	if err != nil || amount != 0 {
		t.Fatalf("expiring contract charged: %d, %v", amount, err)
	}
	player.ExpirationYear, player.Salary = 9999, domain.Money(math.MaxInt64)
	if _, err := waiverDeadMoney(player, rules); err == nil {
		t.Fatal("dead money overflow accepted")
	}
}

func TestNativeWaiverOwnClaimCountsReliefWithoutMutatingAssets(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	rules.Season = 2026
	rules.Roster.Limits = leaguerules.Known(leaguerules.Bounds{Min: leaguerules.Known(0), Max: leaguerules.Known(2)})
	rules.Roster.PositionBounds = leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{})
	rules.Contracts.ReleaseReSignSameSeason = leaguerules.Known(true)
	rules.Contracts.ClaimClearsDeadCap = leaguerules.Known(true)
	rules.Cap.Amount = leaguerules.Known[leaguerules.Money](100000000)
	rules.Cap.Enforcement = leaguerules.Known(leaguerules.HardCap)
	seed := &RawNativeSnapshot{Players: []NativePlayer{}, Adjustments: []NativeCharge{
		{ID: "cut", Franchise: "one", Season: 2026, Amount: 70000000}}}
	w := &waiverWindow{ID: "cut", Season: 2026, Player: NativePlayer{ID: "1000", Franchise: "one", Salary: 100000000}}
	if err := waiverClaimAllows(seed, rules, w, "one", nativecheck.Facts{}, time.UTC); err != nil {
		t.Fatal(err)
	}
	if len(seed.Players) != 0 || seed.Adjustments[0].Amount != 70000000 {
		t.Fatal("claim check mutated assets")
	}
	rules.Contracts.ClaimClearsDeadCap = leaguerules.Known(false)
	if err := waiverClaimAllows(seed, rules, w, "one", nativecheck.Facts{}, time.UTC); err == nil {
		t.Fatal("claim ignored retained dead money")
	}
}
