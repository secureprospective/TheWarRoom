package leaguerules

import "time"

type DraftRules struct {
	Mode         Value[DraftMode]
	Clock        Value[time.Duration]
	OnExpire     Value[ExpiryAction]
	Timeouts     Value[TimeoutPolicy]
	Rounds       Value[int]
	MaximumPicks Value[int]
	ForfeitExtra Value[bool]
	Pricing      Value[map[int][]SlotPrice]
	Lengths      Value[map[int]Value[int]]
	RFAAfter     Value[map[int]Value[bool]]
	OrderSource  Value[OrderSource]
	Order        Value[map[int][]string]
	LaterPattern Value[OrderPattern]
	TieRule      Value[TieRule]
}

func legacyDraft() DraftRules {
	return DraftRules{
		Mode:         cited[DraftMode](SlowDraft, "Census CM-040"),
		Clock:        cited[time.Duration](6*time.Hour, "Census CM-040"),
		OnExpire:     cited[ExpiryAction](SkipPick, "Census CM-040"),
		Timeouts:     cited[TimeoutPolicy](TimeoutPolicy{Unlimited: cited(true, "Census CM-040"), Count: Needs[int]("Census CM-040: unlimited")}, "Census CM-040"),
		Rounds:       cited[int](5, "Rulebook §15"),
		MaximumPicks: cited[int](15, "Rulebook §15"),
		ForfeitExtra: cited[bool](true, "Rulebook §15"),
		Pricing:      cited[map[int][]SlotPrice](legacyPricing(), "Rulebook §15"),
		Lengths:      cited[map[int]Value[int]](legacyLengths(), "Rulebook §15"),
		RFAAfter:     cited[map[int]Value[bool]](legacyRFAAfter(), "Rulebook §15"),
		OrderSource:  cited[OrderSource](WorstStandingsFirst, "Census CM-041"),
		Order:        cited[map[int][]string](map[int][]string{}, "Census CM-041"),
		LaterPattern: Needs[OrderPattern]("Census CM-041 Q-E1"),
		TieRule:      Needs[TieRule]("Census CM-041 Q-E1"),
	}
}

func draftFields() []ruleField {
	return []ruleField{
		field[DraftMode]{path: "draft.mode", label: "Draft pace", access: func(r *RuleSet) *Value[DraftMode] { return &r.Draft.Mode }},
		field[time.Duration]{path: "draft.clock", label: "Draft pick clock", access: func(r *RuleSet) *Value[time.Duration] { return &r.Draft.Clock }},
		field[ExpiryAction]{path: "draft.onExpire", label: "Expired draft clock", access: func(r *RuleSet) *Value[ExpiryAction] { return &r.Draft.OnExpire }},
		field[TimeoutPolicy]{path: "draft.timeouts", label: "Draft timeouts", access: func(r *RuleSet) *Value[TimeoutPolicy] { return &r.Draft.Timeouts }},
		field[int]{path: "draft.rounds", label: "Draft rounds", access: func(r *RuleSet) *Value[int] { return &r.Draft.Rounds }},
		field[int]{path: "draft.maximumPicks", label: "Maximum rookie picks per team", access: func(r *RuleSet) *Value[int] { return &r.Draft.MaximumPicks }},
		field[bool]{path: "draft.forfeitExtra", label: "Forfeit extra rookie picks", access: func(r *RuleSet) *Value[bool] { return &r.Draft.ForfeitExtra }},
		field[map[int][]SlotPrice]{path: "draft.pricing", label: "Rookie draft salaries", access: func(r *RuleSet) *Value[map[int][]SlotPrice] { return &r.Draft.Pricing }},
		field[map[int]Value[int]]{path: "draft.lengths", label: "Rookie contract years", access: func(r *RuleSet) *Value[map[int]Value[int]] { return &r.Draft.Lengths }},
		field[map[int]Value[bool]]{path: "draft.rfaAfter", label: "Rookie restricted free agency", access: func(r *RuleSet) *Value[map[int]Value[bool]] { return &r.Draft.RFAAfter }},
		field[OrderSource]{path: "draft.orderSource", label: "First round draft order", access: func(r *RuleSet) *Value[OrderSource] { return &r.Draft.OrderSource }},
		field[map[int][]string]{path: "draft.order", label: "Draft order overrides", access: func(r *RuleSet) *Value[map[int][]string] { return &r.Draft.Order }},
		field[OrderPattern]{path: "draft.laterPattern", label: "Later round draft pattern", access: func(r *RuleSet) *Value[OrderPattern] { return &r.Draft.LaterPattern }},
		field[TieRule]{path: "draft.tieRule", label: "Draft order tie breaker", access: func(r *RuleSet) *Value[TieRule] { return &r.Draft.TieRule }},
	}
}

type TimeoutPolicy struct {
	Unlimited Value[bool]
	Count     Value[int]
}
type SlotPrice struct {
	Slots  Bounds
	Salary Value[Money]
}

func legacyPricing() map[int][]SlotPrice {
	rows := []struct {
		round, first, last int
		salary             Money
	}{
		{1, 1, 10, 300000000}, {1, 11, 15, 240000000}, {1, 16, 20, 200000000}, {1, 21, 32, 150000000},
		{2, 1, 12, 142000000}, {2, 13, 24, 125000000}, {2, 25, 32, 108000000},
		{3, 1, 16, 83000000}, {3, 17, 32, 67000000}, {4, 1, 32, 60000000}, {5, 1, 32, 50000000},
	}
	result := make(map[int][]SlotPrice)
	for _, row := range rows {
		result[row.round] = append(result[row.round], SlotPrice{
			Slots: bounds(row.first, row.last, "Rulebook §15"), Salary: cited(row.salary, "Rulebook §15")})
	}
	return result
}
func legacyLengths() map[int]Value[int] {
	return map[int]Value[int]{1: cited(5, "Rulebook §15"), 2: cited(3, "Rulebook §15"),
		3: cited(3, "Rulebook §15"), 4: cited(3, "Rulebook §15"), 5: cited(3, "Rulebook §15")}
}
func legacyRFAAfter() map[int]Value[bool] {
	return map[int]Value[bool]{1: cited(false, "Rulebook §15"), 2: cited(true, "Rulebook §15"),
		3: cited(true, "Rulebook §15"), 4: cited(true, "Rulebook §15"), 5: cited(true, "Rulebook §15")}
}
