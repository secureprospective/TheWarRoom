package leaguerules

type ScoringRules struct {
	Catalogue Value[map[StatID]Stat]
}

func legacyScoring() ScoringRules {
	return ScoringRules{
		Catalogue: cited[map[StatID]Stat](legacyCatalogue(), "Rulebook §4; Math §13"),
	}
}

func scoringFields() []ruleField {
	return []ruleField{
		field[map[StatID]Stat]{path: "scoring.catalogue", label: "Scoring values", access: func(r *RuleSet) *Value[map[StatID]Stat] { return &r.Scoring.Catalogue }},
	}
}

type Bonus struct {
	At     Value[Decimal]
	Points Value[Decimal]
}
type Stat struct {
	Label     Value[string]
	Group     Value[string]
	Used      Value[bool]
	Points    Value[Decimal]
	Bonuses   Value[[]Bonus]
	Positions Value[[]Position]
}

func legacyCatalogue() map[StatID]Stat {
	rows := []struct {
		id           StatID
		label, group string
		points       Decimal
	}{
		{"passing_td", "Passing touchdown", "Passing", 50000},
		{"passing_yards", "Passing yards", "Passing", 500},
		{"interception_thrown", "Interception thrown", "Passing", -20000},
		{"passing_two_point", "Passing two point conversion", "Passing", 20000},
		{"rushing_td", "Rushing touchdown", "Rushing", 60000},
		{"rushing_yards", "Rushing yards", "Rushing", 1000},
		{"rushing_attempt", "Rushing attempt", "Rushing", 1500},
		{"rushing_two_point", "Rushing two point conversion", "Rushing", 20000},
		{"receiving_td", "Receiving touchdown", "Receiving", 60000},
		{"receiving_yards", "Receiving yards", "Receiving", 1000},
		{"reception", "Reception", "Receiving", 10000},
		{"receiving_two_point", "Receiving two point conversion", "Receiving", 20000},
		{"fg_made_0_39", "Field goal made from 0 to 39 yards", "Kicking", 30000},
		{"fg_made_40_49", "Field goal made from 40 to 49 yards", "Kicking", 40000},
		{"fg_made_50_59", "Field goal made from 50 to 59 yards", "Kicking", 50000},
		{"fg_made_60_69", "Field goal made from 60 to 69 yards", "Kicking", 60000},
		{"fg_missed_0_29", "Field goal missed from 0 to 29 yards", "Kicking", -30000},
		{"fg_missed_30_99", "Field goal missed from 30 yards or longer", "Kicking", -10000},
		{"punt_return_td", "Punt return touchdown", "Returns", 60000},
		{"punt_return_yards", "Punt return yards", "Returns", 250},
		{"kickoff_return_td", "Kickoff return touchdown", "Returns", 100000},
		{"kickoff_return_yards", "Kickoff return yards", "Returns", 250},
		{"fumble_lost", "Fumble lost", "Fumbles", -20000},
		{"own_fumble_recovery", "Own fumble recovery", "Fumbles", 20000},
		{"opponent_fumble_recovery", "Opponent fumble recovery", "Fumbles", 30000},
		{"opponent_fumble_recovery_yards", "Opponent fumble recovery yards", "Fumbles", 250},
		{"forced_fumble", "Forced fumble", "Fumbles", 40000},
		{"interception_return_td", "Interception return touchdown", "Defense", 60000},
		{"interception_caught", "Interception caught", "Defense", 50000},
		{"interception_return_yards", "Interception return yards", "Defense", 250},
		{"pass_defensed", "Pass defensed", "Defense", 25000},
		{"qb_hit", "Quarterback hit", "Defense", 10000},
		{"blocked_fg", "Blocked field goal", "Defense", 70000},
		{"blocked_punt", "Blocked punt", "Defense", 70000},
		{"blocked_extra_point", "Blocked extra point", "Defense", 70000},
		{"tackle", "Tackle", "Defense", 15000},
		{"tackle_line", "Defensive line tackle", "Defense", 25000},
		{"assist", "Assisted tackle", "Defense", 10000},
		{"assist_line", "Defensive line assisted tackle", "Defense", 15000},
		{"sack", "Sack", "Defense", 45000},
		{"tackle_for_loss", "Tackle for loss", "Defense", 25000},
		{"safety", "Safety", "Defense", 100000},
	}
	result := make(map[StatID]Stat, len(rows)+3)
	for _, row := range rows {
		result[row.id] = scoringStat(row.label, row.group, row.points)
	}
	installLongPlays(result)
	installPositionSplit(result)
	return result
}
func scoringStat(label, group string, points Decimal) Stat {
	return Stat{Label: cited(label, "Rulebook §4"), Group: cited(group, "Rulebook §4"),
		Used: cited(true, "Rulebook §4"), Points: cited(points, "Rulebook §4"),
		Bonuses: cited([]Bonus{}, "Rulebook §4"), Positions: cited([]Position{}, "Rulebook §4")}
}
func installLongPlays(catalogue map[StatID]Stat) {
	rows := []struct {
		id           StatID
		label, group string
		thresholds   []Decimal
	}{
		{"long_pass", "Long pass", "Passing", []Decimal{400000}},
		{"long_rush", "Long rush", "Rushing", []Decimal{200000, 400000}},
		{"long_reception", "Long reception", "Receiving", []Decimal{200000, 400000}},
	}
	for _, row := range rows {
		stat := scoringStat(row.label, row.group, 0)
		bonuses := make([]Bonus, len(row.thresholds))
		for i, at := range row.thresholds {
			bonuses[i] = Bonus{At: cited(at, "Rulebook §4"), Points: cited(Decimal(10000), "Rulebook §4; Math §13: stacking")}
		}
		stat.Bonuses = cited(bonuses, "Rulebook §4; Math §13: stacking")
		catalogue[row.id] = stat
	}
}
func installPositionSplit(catalogue map[StatID]Stat) {
	for _, id := range []StatID{"tackle_line", "assist_line"} {
		stat := catalogue[id]
		stat.Positions = cited([]Position{DT, DE}, "Math §13: true-position base rate")
		catalogue[id] = stat
	}
	for _, id := range []StatID{"tackle", "assist"} {
		stat := catalogue[id]
		stat.Positions = cited([]Position{LB, CB, S, QB, RB, WR, TE, PK}, "Math §13: all except DT/DE")
		catalogue[id] = stat
	}
	// Rulebook §4 gives -3 / -1; the league's decoded scoring sets the 30-yard split (OQ-002).
	for _, id := range []StatID{"fg_missed_0_29", "fg_missed_30_99"} {
		stat := catalogue[id]
		stat.Label = cited(stat.Label.Value, "Rulebook §4; MFL_Scoring_Rules_Decode OQ-002")
		catalogue[id] = stat
	}
}

// UnusedStat keeps a recorded stat available without treating it as zero-point scoring.
func UnusedStat(label, group string) Stat {
	return Stat{Label: Known(label), Group: Known(group), Used: Known(false),
		Points: Needs[Decimal]("No scoring value chosen"), Bonuses: Known([]Bonus{}), Positions: Known([]Position{})}
}
