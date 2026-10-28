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
	// The startup draft fills a new league's empty rosters when its first rosters are drafted. A rule set
	// stored before these existed reads them unset and omits them, so it encodes, and digests, as it did.
	// Unset, each follows its default (rookiedraft.ResolveStartup): rounds to fill the active roster, a
	// random first round, snake order, one-year contracts at the lowest minimum salary.
	StartupRounds  Value[int]                `json:",omitzero"`
	StartupOrder   Value[StartupOrderSource] `json:",omitzero"`
	StartupList    Value[[]string]           `json:",omitzero"`
	StartupPattern Value[OrderPattern]       `json:",omitzero"`
	StartupYears   Value[int]                `json:",omitzero"`
	StartupSalary  Value[Money]              `json:",omitzero"`
	// StartupRookies says whether the incoming rookie class is in the startup draft's pool. It stands
	// alone: off only leaves the class out of the pool, and never turns a rookie draft on or off.
	StartupRookies Value[bool] `json:",omitzero"`
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
		// A startup draft fills the whole active roster (fact R's maximum) with one-year contracts at the
		// lowest minimum salary (Rulebook §6), in snake order from a random first round.
		StartupRounds:  cited[int](80, startupRuling+"; fact R"),
		StartupOrder:   cited[StartupOrderSource](RandomOrder, startupRuling),
		StartupList:    cited[[]string]([]string{}, startupRuling),
		StartupPattern: cited[OrderPattern](Snake, startupRuling),
		StartupYears:   cited[int](1, startupRuling),
		StartupSalary:  cited[Money](33000000, startupRuling+"; Rulebook §6"),
		StartupRookies: cited(false, startupRookiesRuling),
	}
}

// startupRuling is Christopher's ruling R1 of 2026-10-27: a league's first rosters may be drafted, snake first.
const startupRuling = "Startup draft ruling R1, 2026-10-27"

// startupRookiesRuling is Christopher's 2026-10-27 setting: the rookie class sits out a startup draft
// unless the commissioner puts it in.
const startupRookiesRuling = "Startup draft rookies, 2026-10-27: off unless the commissioner turns it on"

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
		field[int]{path: "draft.startupRounds", label: "Startup draft rounds", access: func(r *RuleSet) *Value[int] { return &r.Draft.StartupRounds }},
		field[StartupOrderSource]{path: "draft.startupOrder", label: "Startup draft first round order", access: func(r *RuleSet) *Value[StartupOrderSource] { return &r.Draft.StartupOrder }},
		field[[]string]{path: "draft.startupList", label: "Startup draft order set by the commissioner", access: func(r *RuleSet) *Value[[]string] { return &r.Draft.StartupList }},
		field[OrderPattern]{path: "draft.startupPattern", label: "Startup draft later round pattern", access: func(r *RuleSet) *Value[OrderPattern] { return &r.Draft.StartupPattern }},
		field[int]{path: "draft.startupYears", label: "Startup draft contract years", access: func(r *RuleSet) *Value[int] { return &r.Draft.StartupYears }},
		field[Money]{path: "draft.startupSalary", label: "Startup draft salary", access: func(r *RuleSet) *Value[Money] { return &r.Draft.StartupSalary }},
		field[bool]{path: "draft.startupRookies", label: "Incoming rookie class in the startup draft pool", access: func(r *RuleSet) *Value[bool] { return &r.Draft.StartupRookies }},
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

// StartupRoundsFor is the startup draft rounds that fill the active roster: its maximum, less any O-Line
// cards, which are auctioned rather than drafted. ok is false while the maximum needs a ruling.
func StartupRoundsFor(r RuleSet) (int, bool) {
	limits := r.Roster.Limits
	if limits.State != IsKnown || limits.Value.Max.State != IsKnown {
		return 0, false
	}
	rounds := limits.Value.Max.Value
	if OLineOn(r) {
		cards := 1
		if line := r.Roster.PositionBounds.Value[OL]; line.Max.State == IsKnown {
			cards = line.Max.Value
		}
		rounds -= cards
	}
	return rounds, rounds >= 1
}

// LowestMinimumSalary is the least of the minimum salaries, the startup draft's default contract.
func LowestMinimumSalary(r RuleSet) (Money, bool) {
	lowest, found := Money(0), false
	if r.Contracts.MinimumSalaries.State != IsKnown {
		return 0, false
	}
	for _, row := range r.Contracts.MinimumSalaries.Value {
		if row.Minimum.State == IsKnown && (!found || row.Minimum.Value < lowest) {
			lowest, found = row.Minimum.Value, true
		}
	}
	return lowest, found
}
