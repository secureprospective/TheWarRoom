package leaguerules

import "time"

type BidsRules struct {
	UFA                  Value[BidMode]
	RFA                  Value[BidMode]
	YearMultipliers      Value[[]Value[Decimal]]
	MaximumActive        Value[int]
	OneYearMaximum       Value[Money]
	Tiers                Value[[]BidTier]
	MinimumIncrement     Value[Decimal]
	SnipeAfter           Value[time.Duration]
	SnipeIncrement       Value[Decimal]
	MultiYearEndWeek     Value[int]
	PlayoffOnlyAfterWeek Value[int]
	PlayoffMaximumYears  Value[int]
	PlayoffCapLifted     Value[bool]
	PointDecimals        Value[int]
}

func legacyBids() BidsRules {
	return BidsRules{
		UFA:                  cited(Open, "fact 3; Christopher 2026-10-13: Legacy UFA open"),
		RFA:                  cited(Open, "fact 3; Christopher 2026-10-13: Legacy RFA open"),
		YearMultipliers:      cited[[]Value[Decimal]]([]Value[Decimal]{cited(Decimal(10000), "Rulebook §6"), cited(Decimal(12000), "Rulebook §6"), cited(Decimal(14000), "Rulebook §6"), cited(Decimal(16000), "Rulebook §6")}, "Rulebook §6"),
		MaximumActive:        cited[int](30, "Rulebook §6"),
		OneYearMaximum:       cited[Money](1200000000, "Rulebook §6"),
		Tiers:                cited[[]BidTier](legacyBidTiers(), "Rulebook §6"),
		MinimumIncrement:     cited[Decimal](1000, "Rulebook §6"),
		SnipeAfter:           cited[time.Duration](20*time.Hour, "Rulebook §6"),
		SnipeIncrement:       cited[Decimal](10000, "Rulebook §6"),
		MultiYearEndWeek:     cited[int](13, "Rulebook §6"),
		PlayoffOnlyAfterWeek: cited[int](12, "Rulebook §6"),
		PlayoffMaximumYears:  cited[int](1, "Rulebook §6"),
		PlayoffCapLifted:     cited[bool](true, "Rulebook §6"),
		PointDecimals:        cited[int](3, "Rulebook §6"),
	}
}

func bidsFields() []ruleField {
	return []ruleField{
		field[BidMode]{path: "bids.ufa", label: "Unrestricted free agent bidding", access: func(r *RuleSet) *Value[BidMode] { return &r.Bids.UFA }},
		field[BidMode]{path: "bids.rfa", label: "Restricted free agent bidding", access: func(r *RuleSet) *Value[BidMode] { return &r.Bids.RFA }},
		field[[]Value[Decimal]]{path: "bids.yearMultipliers", label: "Bid year multipliers", access: func(r *RuleSet) *Value[[]Value[Decimal]] { return &r.Bids.YearMultipliers }},
		field[int]{path: "bids.maximumActive", label: "Maximum active bids", access: func(r *RuleSet) *Value[int] { return &r.Bids.MaximumActive }},
		field[Money]{path: "bids.oneYearMaximum", label: "One year bid limit", access: func(r *RuleSet) *Value[Money] { return &r.Bids.OneYearMaximum }},
		field[[]BidTier]{path: "bids.tiers", label: "Bid salary length limits", access: func(r *RuleSet) *Value[[]BidTier] { return &r.Bids.Tiers }},
		field[Decimal]{path: "bids.minimumIncrement", label: "Minimum bid increase", access: func(r *RuleSet) *Value[Decimal] { return &r.Bids.MinimumIncrement }},
		field[time.Duration]{path: "bids.snipeAfter", label: "Late bid threshold", access: func(r *RuleSet) *Value[time.Duration] { return &r.Bids.SnipeAfter }},
		field[Decimal]{path: "bids.snipeIncrement", label: "Late bid minimum increase", access: func(r *RuleSet) *Value[Decimal] { return &r.Bids.SnipeIncrement }},
		field[int]{path: "bids.multiYearEndWeek", label: "Last multi year bidding week", access: func(r *RuleSet) *Value[int] { return &r.Bids.MultiYearEndWeek }},
		field[int]{path: "bids.playoffOnlyAfterWeek", label: "Playoff only bidding after week", access: func(r *RuleSet) *Value[int] { return &r.Bids.PlayoffOnlyAfterWeek }},
		field[int]{path: "bids.playoffMaximumYears", label: "Playoff bid maximum years", access: func(r *RuleSet) *Value[int] { return &r.Bids.PlayoffMaximumYears }},
		field[bool]{path: "bids.playoffCapLifted", label: "Lift one year bid cap in playoffs", access: func(r *RuleSet) *Value[bool] { return &r.Bids.PlayoffCapLifted }},
		field[int]{path: "bids.pointDecimals", label: "Bid point decimal places", access: func(r *RuleSet) *Value[int] { return &r.Bids.PointDecimals }},
	}
}

type WaiversRules struct {
	Trigger  Value[WaiverTrigger]
	Duration Value[time.Duration]
	Method   Value[WaiverMethod]
}

func legacyWaivers() WaiversRules {
	return WaiversRules{
		Trigger:  cited[WaiverTrigger](OnCut, "Census CM-012"),
		Duration: cited[time.Duration](24*time.Hour, "Census CM-012"),
		Method:   cited[WaiverMethod](ReverseStandings, "fact 2"),
	}
}

func waiversFields() []ruleField {
	return []ruleField{
		field[WaiverTrigger]{path: "waivers.trigger", label: "Waiver trigger", access: func(r *RuleSet) *Value[WaiverTrigger] { return &r.Waivers.Trigger }},
		field[time.Duration]{path: "waivers.duration", label: "Waiver period", access: func(r *RuleSet) *Value[time.Duration] { return &r.Waivers.Duration }},
		field[WaiverMethod]{path: "waivers.method", label: "Waiver priority", access: func(r *RuleSet) *Value[WaiverMethod] { return &r.Waivers.Method }},
	}
}

type BidProcess uint8

const (
	UFAProcess BidProcess = iota
	RFAProcess
	UDFAProcess
)

func (r BidsRules) Mode(process BidProcess) Value[BidMode] {
	switch process {
	case UFAProcess, UDFAProcess:
		return r.UFA
	case RFAProcess:
		return r.RFA
	default:
		return Needs[BidMode]("Choose a bidding process")
	}
}
