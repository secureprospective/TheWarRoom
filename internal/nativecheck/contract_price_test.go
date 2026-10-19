package nativecheck_test

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const million = domain.Money(100_000_000)

// priceInput is a QB at $10M through 2028 in the 2026 season, under the Legacy NFL preset.
func priceInput() nativecheck.Input {
	rules := leaguerules.LegacyNFL()
	rules.Season = 2026
	return nativecheck.Input{Rules: rules, Contract: nativecheck.Contract{Player: "0099", Name: "Test Player",
		Position: leaguerules.QB, PositionHeld: true, Salary: 10 * million, Expiration: 2028,
		Status: domain.CStatusUFA}}
}

func price(t *testing.T, in nativecheck.Input, action nativecheck.ContractAction) nativecheck.Pricing {
	t.Helper()
	got, err := nativecheck.PriceContract(in, action)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func wantMoney(t *testing.T, label string, got *domain.Money, want domain.Money) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %d cents", label, got, want)
	}
}

func wantRefusal(t *testing.T, got nativecheck.Pricing, fragment string) {
	t.Helper()
	if !strings.Contains(got.Refusal, fragment) {
		t.Fatalf("refusal %q does not say %q (needs %v)", got.Refusal, fragment, got.Needs)
	}
}

func wantNeeds(t *testing.T, got nativecheck.Pricing, fragment string) {
	t.Helper()
	if got.Refusal != "" || len(got.Needs) == 0 || !strings.Contains(got.Needs[0], fragment) {
		t.Fatalf("expected a ruling about %q, got refusal %q needs %v", fragment, got.Refusal, got.Needs)
	}
}

func wantDead(t *testing.T, got nativecheck.Pricing, season int, want domain.Money) {
	t.Helper()
	if got.Refusal != "" || len(got.Dead) != 1 || got.Dead[0].Season != season || got.Dead[0].Cents != want {
		t.Fatalf("dead money = %+v (refusal %q), want %d cents in %d", got.Dead, got.Refusal, want, season)
	}
}

func TestReleaseChargesTheRestructuredRateOnARestructuredContract(t *testing.T) {
	in := priceInput()
	in.Contract.Salary = 10_987_00000
	plain := price(t, in, nativecheck.ActionCut)
	// 35% of two remaining $10,987,000 salaries is $7,690,900, which snaps to $7,690,000.
	wantDead(t, plain, 2026, 7_690_00000)
	if !plain.Removed {
		t.Fatal("a released player stays on the roster")
	}
	in.Trail.Restructures = 1
	// 50% of $21,974,000 is $10,987,000, which snaps to $10,990,000.
	wantDead(t, price(t, in, nativecheck.ActionCut), 2026, 10_990_00000)
	in.Contract.Expiration = 2026
	if got := price(t, in, nativecheck.ActionCut); got.Refusal != "" || len(got.Dead) != 0 {
		t.Fatalf("a contract in its last year leaves no dead money: %+v", got)
	}
}

func TestReleaseRoundsToTheCentBeforeSnapping(t *testing.T) {
	// 35% of $71,428.56 is 2,499,999.6 cents. To the cent first that is 2,500,000, and the snap to the
	// nearest $10,000 then goes up to 3,000,000. A single division would have given 2,000,000.
	in := priceInput()
	in.Contract.Salary, in.Contract.Expiration = 7_142_856, 2027
	wantDead(t, price(t, in, nativecheck.ActionCut), 2026, 3_000_000)
}

func TestBuyoutRateFollowsYearsLeftAndTheLimit(t *testing.T) {
	in := priceInput()
	for _, tc := range []struct {
		expiration int
		want       domain.Money
	}{{2028, 6 * million}, {2029, 7_50 * million / 100}, {2030, 9 * million}} {
		in.Contract.Expiration = tc.expiration
		wantDead(t, price(t, in, nativecheck.ActionBuyout), 2026, tc.want)
	}
	in.Contract.Expiration = 2027
	wantRefusal(t, price(t, in, nativecheck.ActionBuyout), "one year left cannot be bought out")
	in.Contract.Expiration = 2031
	wantRefusal(t, price(t, in, nativecheck.ActionBuyout), "no rate for 5 years left")
	in.Contract.Expiration = 2029
	in.Trail.Used = map[nativecheck.ContractAction]int{nativecheck.ActionBuyout: 1}
	if got := price(t, in, nativecheck.ActionBuyout); got.Refusal != "" {
		t.Fatalf("the second buyout of the season is within the limit of 2: %s", got.Refusal)
	}
	in.Trail.Used[nativecheck.ActionBuyout] = 2
	wantRefusal(t, price(t, in, nativecheck.ActionBuyout), "already used its 2 buyouts this season")
}

func TestBuyoutAveragesUnevenSalariesBeforeRounding(t *testing.T) {
	// Three remaining salaries of $10,000,000.01 each: 75% of the exact mean is 750,000,000.75 cents,
	// which is 750,000,001 cents before the snap and $7,500,000 after it.
	in := priceInput()
	in.Contract.Salary, in.Contract.Expiration = 1_000_000_001, 2029
	wantDead(t, price(t, in, nativecheck.ActionBuyout), 2026, 750_000_000)
}

func TestSwitchedOffAndUnruledActionsAreNotQuoted(t *testing.T) {
	for _, action := range nativecheck.ContractActions() {
		if action == nativecheck.ActionCut {
			continue
		}
		off := priceInput()
		switch action {
		case nativecheck.ActionTag:
			off.Rules.Contracts.Tag.Value.Enabled = leaguerules.Known(false)
		case nativecheck.ActionExtension:
			off.Rules.Contracts.Extension.Value.Enabled = leaguerules.Known(false)
		case nativecheck.ActionRestructure:
			off.Rules.Contracts.Restructure.Value.Enabled = leaguerules.Known(false)
		case nativecheck.ActionBuyout:
			off.Rules.Contracts.Buyout.Value.Enabled = leaguerules.Known(false)
		case nativecheck.ActionTender:
			off.Rules.Contracts.Tender.Value.Enabled = leaguerules.Known(false)
		case nativecheck.ActionResign:
			off.Rules.Contracts.ReSign.Value.Enabled = leaguerules.Known(false)
		case nativecheck.ActionRetire:
			off.Rules.Contracts.Retirement.Value.Enabled = leaguerules.Known(false)
		case nativecheck.ActionDeath:
			off.Rules.Contracts.Death.Value.Enabled = leaguerules.Known(false)
		case nativecheck.ActionCapRelief:
			off.Rules.Contracts.CapRelief.Value.Enabled = leaguerules.Known(false)
		case nativecheck.ActionCut:
		}
		got := price(t, off, action)
		wantRefusal(t, got, "turned off")
		if got.Salary != nil || len(got.Dead) != 0 || len(got.Schedule) != 0 {
			t.Fatalf("%s: a switched-off action still carries a price: %+v", action, got)
		}
	}
	unruled := priceInput()
	unruled.Rules.Contracts.Tag = leaguerules.Needs[leaguerules.TagRules]("test")
	wantNeeds(t, price(t, unruled, nativecheck.ActionTag), "franchise tag is on or off")
	unruled = priceInput()
	unruled.Rules.Cap.Quantum = leaguerules.Needs[leaguerules.Money]("test")
	wantNeeds(t, price(t, unruled, nativecheck.ActionCut), "money rounding increment")
}

func TestTagPriceIsTheTopAverageWithTheFloorAndTheSecondTag(t *testing.T) {
	in := priceInput()
	in.Contract.Expiration = 2026
	in.Market = []domain.Money{30 * million, 28 * million, 26 * million, 24 * million, 22 * million, 5 * million}
	got := price(t, in, nativecheck.ActionTag)
	wantMoney(t, "tag price", got.Salary, 26*million)
	if got.Years != 1 || len(got.Schedule) != 1 || got.Schedule[0].Season != 2027 {
		t.Fatalf("a tag is a one-year deal starting next season: %+v", got)
	}
	in.Contract.Salary = 30 * million
	wantMoney(t, "floor", price(t, in, nativecheck.ActionTag).Salary, 36*million)
	in.Contract.Salary = 10 * million
	in.Trail.ConsecutiveTags, in.Trail.FirstTag = 1, 26*million
	wantMoney(t, "second tag", price(t, in, nativecheck.ActionTag).Salary, 31_20*million/100)
	in.Trail.ConsecutiveTags = 2
	wantRefusal(t, price(t, in, nativecheck.ActionTag), "tagged 2 years in a row")
	in.Trail = nativecheck.Trail{Used: map[nativecheck.ContractAction]int{nativecheck.ActionTag: 1}}
	wantRefusal(t, price(t, in, nativecheck.ActionTag), "already used its 1 tag this season")
	in.Trail = nativecheck.Trail{}
	in.Contract.Expiration = 2027
	wantRefusal(t, price(t, in, nativecheck.ActionTag), "last year of his contract")
	in.Contract.Expiration = 2026
	in.Market = in.Market[:3]
	wantRefusal(t, price(t, in, nativecheck.ActionTag), "needs the top 5")
}

func TestTagRoundsToCentsBeforeSnapping(t *testing.T) {
	// The top five average 2,600,499,999.6 cents: to the cent first that is 2,600,500,000, and the snap to
	// the nearest $10,000 then goes up. Snapping the unrounded mean would have gone down.
	in := priceInput()
	in.Contract.Expiration = 2026
	base := domain.Money(2_600_499_999)
	in.Market = []domain.Money{base, base, base, base, base + 3}
	wantMoney(t, "tag price", price(t, in, nativecheck.ActionTag).Salary, 2_601_000_000)
}

func TestExtensionPriceFloorsAndLimits(t *testing.T) {
	in := priceInput()
	got := price(t, in, nativecheck.ActionExtension)
	wantMoney(t, "QB extension", got.Salary, 15*million)
	if got.Years != 3 || len(got.Schedule) != 6 || got.Schedule[5].Season != 2031 || got.Schedule[5].Cents != 15*million {
		t.Fatalf("an extension adds the league's three years after 2028: %+v", got)
	}
	in.Contract.Position, in.Contract.Salary = leaguerules.WR, 4*million
	wantMoney(t, "WR floor", price(t, in, nativecheck.ActionExtension).Salary, 10*million)
	in.Contract.Salary = 12 * million
	wantMoney(t, "150 percent", price(t, in, nativecheck.ActionExtension).Salary, 18*million)
	in.Ask.Years = 4
	wantRefusal(t, price(t, in, nativecheck.ActionExtension), "adds at most 3 years")
	in.Ask.Years, in.Contract.Expiration = 2, 2030
	wantRefusal(t, price(t, in, nativecheck.ActionExtension), "7 contract years")
	in.Ask.Years, in.Trail.Extensions = 0, 1
	wantRefusal(t, price(t, in, nativecheck.ActionExtension), "must reach free agency")
	in.Trail, in.Contract.Expiration = nativecheck.Trail{}, 2026
	wantRefusal(t, price(t, in, nativecheck.ActionExtension), "unrestricted free agent")
	in.Contract.Status, in.Contract.Salary = domain.CStatusFT1, 20*million
	tagged := price(t, in, nativecheck.ActionExtension)
	wantMoney(t, "tagged extension", tagged.Salary, 24*million)
}

func TestRestructureTiersSpreadAndLimits(t *testing.T) {
	in := priceInput()
	in.Contract.Salary = 7 * million
	got := price(t, in, nativecheck.ActionRestructure)
	want := []domain.Money{5 * million, 8 * million, 8 * million}
	if got.Refusal != "" || len(got.Schedule) != 3 {
		t.Fatalf("restructure not priced: %+v", got)
	}
	for i, cell := range got.Schedule {
		if cell.Cents != want[i] || cell.Season != 2026+i {
			t.Fatalf("schedule %+v, want %v", got.Schedule, want)
		}
	}
	in.Ask.Move = 3 * million
	wantRefusal(t, price(t, in, nativecheck.ActionRestructure), "At most $2,000,000.00 can move")
	in.Ask.Move, in.Contract.Salary = 0, 2*million
	wantRefusal(t, price(t, in, nativecheck.ActionRestructure), "yearly salary of at least $3,000,000.00")
	in.Contract.Salary, in.Trail.Restructures = 13*million, 1
	wantRefusal(t, price(t, in, nativecheck.ActionRestructure), "already been restructured 1 time")
	in.Trail.Extensions = 1
	big := price(t, in, nativecheck.ActionRestructure)
	if big.Refusal != "" || big.Schedule[0].Cents != 10*million {
		t.Fatalf("an extension unlocks one more restructure, moving $3M at $13M: %+v", big)
	}
	in.Trail.Restructures = 0
	in.Trail.Used = map[nativecheck.ContractAction]int{nativecheck.ActionRestructure: 1}
	wantRefusal(t, price(t, in, nativecheck.ActionRestructure), "already used its 1 restructure this season")
	in.Trail.Used = nil
	in.Contract.Expiration = 2026
	wantRefusal(t, price(t, in, nativecheck.ActionRestructure), "none left")
}

func TestTenderAndResignPricing(t *testing.T) {
	in := priceInput()
	in.Contract.Status, in.Contract.Expiration, in.Contract.Salary = domain.CStatusRFA, 2026, million/2
	in.Ask.Level = 1
	wantMoney(t, "level 1 tender", price(t, in, nativecheck.ActionTender).Salary, 6*million/10)
	in.Contract.Salary = 1 * million
	wantMoney(t, "110 percent of last salary", price(t, in, nativecheck.ActionTender).Salary, 11*million/10)
	in.Contract.Salary, in.Ask.Years = million/2, 3
	got := price(t, in, nativecheck.ActionResign)
	wantMoney(t, "three-year re-sign", got.Salary, 78*million/100)
	if got.Years != 3 || len(got.Schedule) != 3 || got.Schedule[2].Season != 2029 {
		t.Fatalf("a re-sign runs from next season: %+v", got)
	}
	in.Ask.Years = 5
	wantRefusal(t, price(t, in, nativecheck.ActionResign), "maximum contract of 4 years")
	in.Ask.Years, in.Ask.Level = 3, 9
	wantRefusal(t, price(t, in, nativecheck.ActionResign), "tender level from 1 to 6")
	in.Ask.Level, in.Contract.Status = 1, domain.CStatusUFA
	wantRefusal(t, price(t, in, nativecheck.ActionTender), "restricted free agent")
}

func TestRetirementDeathAndCapRelief(t *testing.T) {
	in := priceInput()
	wantDead(t, price(t, in, nativecheck.ActionRetire), 2026, 6*million)
	death := price(t, in, nativecheck.ActionDeath)
	if death.Refusal != "" || len(death.Dead) != 0 || !death.Removed {
		t.Fatalf("a death leaves no cap charge: %+v", death)
	}
	in.Ask.Reason, in.Ask.Amount = "career-ending injury", 3*million+12345
	relief := price(t, in, nativecheck.ActionCapRelief)
	if relief.Refusal != "" || relief.Relief != 3*million || relief.Removed {
		t.Fatalf("cap relief %+v", relief)
	}
	in.Ask.Reason = "bad vibes"
	wantRefusal(t, price(t, in, nativecheck.ActionCapRelief), "Cap relief is for one of")
	in.Ask.Reason, in.Ask.Amount = "Recurring injury", 11*million
	wantRefusal(t, price(t, in, nativecheck.ActionCapRelief), "cannot be more than")
}

func TestOnlyTheCatalogueSetsAPrice(t *testing.T) {
	// A restructure's moving limits are the league's own: change the tier and the answer changes.
	in := priceInput()
	in.Contract.Salary = 7 * million
	in.Rules.Contracts.Restructure.Value.Tiers = leaguerules.Known([]leaguerules.MoneyTier{
		{At: leaguerules.Known(leaguerules.Money(5 * million)), Amount: leaguerules.Known(leaguerules.Money(million / 2))}})
	if got := price(t, in, nativecheck.ActionRestructure); got.Schedule[0].Cents != 65*million/10 {
		t.Fatalf("restructure ignored the league's tier: %+v", got.Schedule)
	}
}
