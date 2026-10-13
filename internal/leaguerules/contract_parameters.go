package leaguerules

import "time"

type ExperienceSalary struct {
	MaximumUnlimited Value[bool]
	Years            Bounds
	Minimum          Value[Money]
}
type MoneyTier struct {
	At     Value[Money]
	Amount Value[Money]
}
type BidTier struct {
	Salary         BoundsMoney
	MinimumYears   Value[int]
	LowerInclusive Value[bool]
}
type BoundsMoney struct {
	MaximumUnlimited Value[bool]
	Min              Value[Money]
	Max              Value[Money]
}
type TagRules struct {
	Enabled                 Value[bool]
	AnnualLimit             Value[int]
	TopPlayers              Value[int]
	FloorMultiplier         Value[Decimal]
	ConsecutiveLimit        Value[int]
	SecondMultiplier        Value[Decimal]
	ExtensionMultiplier     Value[Decimal]
	ExcludePlayoffContracts Value[bool]
}
type ExtensionRules struct {
	Enabled                Value[bool]
	AnnualLimit            Value[int]
	MinimumRemaining       Value[int]
	AddedYears             Value[int]
	TotalYears             Value[int]
	Multiplier             Value[Decimal]
	Floors                 Value[map[Position]Value[Money]]
	RepeatBeforeFreeAgency Value[bool]
	UFAEligible            Value[bool]
}
type RestructureRules struct {
	Enabled           Value[bool]
	AnnualLimit       Value[int]
	PerContract       Value[int]
	ExtraPerExtension Value[int]
	Tiers             Value[[]MoneyTier]
	ConserveTotal     Value[bool]
	ExistingYearsOnly Value[bool]
}
type BuyoutRules struct {
	Enabled            Value[bool]
	AnnualLimit        Value[int]
	OffseasonOnly      Value[bool]
	OneYearAllowed     Value[bool]
	Rates              Value[map[int]Value[Decimal]]
	RebidNextOffseason Value[bool]
}
type Tender struct {
	Price  Value[Money]
	Rounds Value[[]int]
}
type TenderRules struct {
	Enabled               Value[bool]
	Prices                Value[[]Tender]
	PriorSalaryMultiplier Value[Decimal]
	MatchWindow           Value[time.Duration]
	CompensatoryAllowed   Value[bool]
}
type ReSignRules struct {
	Enabled  Value[bool]
	Premiums Value[map[int]Value[Decimal]]
}
type SpecialRules struct {
	Enabled Value[bool]
	Rate    Value[Decimal]
}
type ReliefRules struct {
	Enabled Value[bool]
	Reasons Value[[]string]
}

func legacyMinimums() []ExperienceSalary {
	prices := []Money{33000000, 38000000, 43000000, 48000000, 53000000, 58000000, 63000000}
	years := []Bounds{bounds(0, 0, "Rulebook §6"), bounds(1, 1, "Rulebook §6"), bounds(2, 2, "Rulebook §6"),
		bounds(3, 3, "Rulebook §6"), bounds(4, 6, "Rulebook §6"), bounds(7, 9, "Rulebook §6"),
		{Min: cited(10, "Rulebook §6"), Max: Needs[int]("Rulebook §6: no upper bound")}}
	result := make([]ExperienceSalary, len(prices))
	for i, price := range prices {
		result[i] = ExperienceSalary{Years: years[i], Minimum: cited(price, "Rulebook §6"), MaximumUnlimited: cited(i == len(prices)-1, "Rulebook §6")}
	}
	return result
}
func legacyTag() TagRules {
	return TagRules{Enabled: cited(true, "Rulebook §9"), AnnualLimit: cited(1, "Rulebook §9"),
		TopPlayers: cited(5, "Rulebook §9"), FloorMultiplier: cited(Decimal(12000), "Rulebook §9"),
		ConsecutiveLimit: cited(2, "Rulebook §9"), SecondMultiplier: cited(Decimal(12000), "Rulebook §9"),
		ExtensionMultiplier: cited(Decimal(12000), "Rulebook §9"), ExcludePlayoffContracts: cited(true, "Rulebook §9")}
}
func legacyExtension() ExtensionRules {
	floors := map[Position]Money{QB: 1500000000, WR: 1000000000, RB: 800000000, TE: 800000000,
		LB: 800000000, DE: 700000000, S: 500000000, DT: 400000000, CB: 300000000, PK: 300000000}
	result := make(map[Position]Value[Money], len(floors))
	for pos, amount := range floors {
		result[pos] = cited(amount, "Rulebook §10")
	}
	return ExtensionRules{Enabled: cited(true, "Rulebook §10"), AnnualLimit: cited(1, "Rulebook §10"),
		MinimumRemaining: cited(1, "Rulebook §10"), AddedYears: cited(3, "Rulebook §10"), TotalYears: cited(6, "Rulebook §10"),
		Multiplier: cited(Decimal(15000), "Rulebook §10"), Floors: cited(result, "Rulebook §10"),
		RepeatBeforeFreeAgency: cited(false, "Rulebook §10"), UFAEligible: cited(false, "Rulebook §10")}
}
func legacyRestructure() RestructureRules {
	return RestructureRules{Enabled: cited(true, "Rulebook §11"), AnnualLimit: cited(1, "Rulebook §11"),
		PerContract: cited(1, "Rulebook §11"), ExtraPerExtension: cited(1, "Rulebook §11"),
		Tiers: cited([]MoneyTier{
			{At: cited(Money(300000000), "Rulebook §11"), Amount: cited(Money(100000000), "Rulebook §11")},
			{At: cited(Money(600000000), "Rulebook §11"), Amount: cited(Money(200000000), "Rulebook §11")},
			{At: cited(Money(1200000000), "Rulebook §11"), Amount: cited(Money(300000000), "Rulebook §11")},
		}, "Rulebook §11"), ConserveTotal: cited(true, "Math §7"), ExistingYearsOnly: cited(true, "Math §7")}
}
func legacyBuyout() BuyoutRules {
	return BuyoutRules{Enabled: cited(true, "Rulebook §12"), AnnualLimit: cited(2, "Rulebook §12"),
		OffseasonOnly: cited(true, "Rulebook §12"), OneYearAllowed: cited(false, "Math §14 ruling 3"),
		Rates: cited(map[int]Value[Decimal]{2: cited(Decimal(6000), "Rulebook §12"),
			3: cited(Decimal(7500), "Rulebook §12"), 4: cited(Decimal(9000), "Rulebook §12")}, "Rulebook §12"),
		RebidNextOffseason: cited(true, "Rulebook §12")}
}
func legacyTender() TenderRules {
	prices := []Money{60000000, 100000000, 150000000, 250000000, 500000000, 650000000}
	rounds := [][]int{{5}, {4}, {3}, {2}, {1}, {1, 2}}
	tenders := make([]Tender, len(prices))
	for i, price := range prices {
		tenders[i] = Tender{Price: cited(price, "Rulebook §7"), Rounds: cited(rounds[i], "Rulebook §7")}
	}
	return TenderRules{Enabled: cited(true, "Rulebook §7"), Prices: cited(tenders, "Rulebook §7"),
		PriorSalaryMultiplier: cited(Decimal(11000), "Rulebook §7"), MatchWindow: cited(24*time.Hour, "Rulebook §7"),
		CompensatoryAllowed: cited(false, "Rulebook §14")}
}
func legacyReSign() ReSignRules {
	return ReSignRules{Enabled: cited(true, "Rulebook §7"), Premiums: cited(map[int]Value[Decimal]{
		1: cited(Decimal(0), "Rulebook §7"), 2: cited(Decimal(1500), "Rulebook §7"),
		3: cited(Decimal(3000), "Rulebook §7"), 4: cited(Decimal(4500), "Rulebook §7")}, "Rulebook §7")}
}
func special(rate Decimal, citation string) SpecialRules {
	return SpecialRules{Enabled: cited(true, citation), Rate: cited(rate, citation)}
}
func legacyRelief() ReliefRules {
	return ReliefRules{Enabled: cited(true, "Rulebook §13"), Reasons: cited([]string{
		"Career-ending injury", "Recurring injury", "Behavioral suspension"}, "Rulebook §13")}
}
func legacyBidTiers() []BidTier {
	return []BidTier{
		{Salary: BoundsMoney{MaximumUnlimited: cited(false, "Rulebook §6"), Min: cited(Money(1200000000), "Rulebook §6"), Max: cited(Money(2400000000), "Rulebook §6")},
			MinimumYears: cited(2, "Rulebook §6"), LowerInclusive: cited(false, "Rulebook §6: $12M one year allowed")},
		{Salary: BoundsMoney{MaximumUnlimited: cited(true, "Rulebook §6"), Min: cited(Money(2400000000), "Rulebook §6"), Max: Needs[Money]("Rulebook §6: no ceiling")},
			MinimumYears: cited(3, "Rulebook §6"), LowerInclusive: cited(false, "Rulebook §6")},
	}
}
