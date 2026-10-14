package leaguerules

import "time"

type TradesRules struct {
	Review             Value[ReviewRoute]
	DOTSeats           Value[int]
	DecideAt           Value[int]
	FutureYears        Value[int]
	Rounds             Value[Bounds]
	RosterGate         Value[bool]
	OfferExpiry        Value[time.Duration]
	MaximumParties     Value[int]
	CompensatoryPicks  Value[bool]
	CapSpace           Value[bool]
	AcceptedWithdrawal Value[bool]
}

func legacyTrades() TradesRules {
	return TradesRules{
		Review:             cited[ReviewRoute](DOTReview, "Rulebook §14"),
		DOTSeats:           Needs[int]("fact 7a: actual seats chosen by league"),
		DecideAt:           cited[int](3, "fact 7a"),
		FutureYears:        cited[int](2, "Rulebook §14"),
		Rounds:             cited[Bounds](bounds(1, 5, "Census CM-014"), "Census CM-014"),
		RosterGate:         cited[bool](true, "Census CM-014"),
		OfferExpiry:        cited[time.Duration](7*24*time.Hour, "Census CM-014"),
		MaximumParties:     cited[int](3, "ruling 4"),
		CompensatoryPicks:  cited[bool](false, "Rulebook §14"),
		CapSpace:           cited[bool](false, "Math §11"),
		AcceptedWithdrawal: cited[bool](false, "Rulebook §14"),
	}
}

func tradesFields() []ruleField {
	return []ruleField{
		field[ReviewRoute]{path: "trades.review", label: "Trade review", access: func(r *RuleSet) *Value[ReviewRoute] { return &r.Trades.Review }},
		field[int]{path: "trades.dotSeats", label: "Trade review seats", access: func(r *RuleSet) *Value[int] { return &r.Trades.DOTSeats }},
		field[int]{path: "trades.decideAt", label: "Votes to approve or veto", access: func(r *RuleSet) *Value[int] { return &r.Trades.DecideAt }},
		field[int]{path: "trades.futureYears", label: "Future pick trading years", access: func(r *RuleSet) *Value[int] { return &r.Trades.FutureYears }},
		field[Bounds]{path: "trades.rounds", label: "Tradable pick rounds", access: func(r *RuleSet) *Value[Bounds] { return &r.Trades.Rounds }},
		field[bool]{path: "trades.rosterGate", label: "Refuse invalid rosters", access: func(r *RuleSet) *Value[bool] { return &r.Trades.RosterGate }},
		field[time.Duration]{path: "trades.offerExpiry", label: "Trade offer lifetime", access: func(r *RuleSet) *Value[time.Duration] { return &r.Trades.OfferExpiry }},
		field[int]{path: "trades.maximumParties", label: "Maximum trade parties", access: func(r *RuleSet) *Value[int] { return &r.Trades.MaximumParties }},
		field[bool]{path: "trades.compensatoryPicks", label: "Trade compensatory picks", access: func(r *RuleSet) *Value[bool] { return &r.Trades.CompensatoryPicks }},
		field[bool]{path: "trades.capSpace", label: "Trade salary cap space", access: func(r *RuleSet) *Value[bool] { return &r.Trades.CapSpace }},
		field[bool]{path: "trades.acceptedWithdrawal", label: "Withdraw accepted trades", access: func(r *RuleSet) *Value[bool] { return &r.Trades.AcceptedWithdrawal }},
	}
}
