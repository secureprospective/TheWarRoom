package leaguerules

type ScheduleRules struct {
	RegularWeeks Value[int]
	Pattern      Value[SchedulePattern]
}

func legacySchedule() ScheduleRules {
	return ScheduleRules{
		RegularWeeks: cited[int](13, "Rulebook §5"),
		Pattern:      cited[SchedulePattern](DivisionWeighted, "Census CM-015"),
	}
}

func scheduleFields() []ruleField {
	return []ruleField{
		field[int]{path: "schedule.regularWeeks", label: "Regular season weeks", access: func(r *RuleSet) *Value[int] { return &r.Schedule.RegularWeeks }},
		field[SchedulePattern]{path: "schedule.pattern", label: "Schedule pattern", access: func(r *RuleSet) *Value[SchedulePattern] { return &r.Schedule.Pattern }},
	}
}

type PlayoffsRules struct {
	PerConference   Value[int]
	DivisionWinners Value[int]
	WildCards       Value[int]
	ByeSeed         Value[int]
	Seeding         Value[SeedPolicy]
	Reseed          Value[bool]
	StartWeek       Value[int]
	FinalWeek       Value[int]
	HomeBonus       Value[Decimal]
	Graph           Value[[]BracketNode]
}

func legacyPlayoffs() PlayoffsRules {
	return PlayoffsRules{
		PerConference:   cited[int](7, "fact 11a"),
		DivisionWinners: cited[int](4, "fact 11a"),
		WildCards:       cited[int](3, "fact 11a"),
		ByeSeed:         cited[int](1, "fact 11a"),
		Seeding:         cited[SeedPolicy](DivisionWinnersFirst, "fact 11b"),
		Reseed:          cited[bool](true, "fact 11b"),
		StartWeek:       cited[int](14, "Census CM-016"),
		FinalWeek:       cited[int](17, "Census CM-016"),
		HomeBonus:       cited[Decimal](30000, "Rulebook §5"),
		Graph:           cited[[]BracketNode](legacyBracket(), "fact 11a-b"),
	}
}

func playoffsFields() []ruleField {
	return []ruleField{
		field[int]{path: "playoffs.perConference", label: "Playoff teams per conference", access: func(r *RuleSet) *Value[int] { return &r.Playoffs.PerConference }},
		field[int]{path: "playoffs.divisionWinners", label: "Division winners per conference", access: func(r *RuleSet) *Value[int] { return &r.Playoffs.DivisionWinners }},
		field[int]{path: "playoffs.wildCards", label: "Wild cards per conference", access: func(r *RuleSet) *Value[int] { return &r.Playoffs.WildCards }},
		field[int]{path: "playoffs.byeSeed", label: "First round bye seed", access: func(r *RuleSet) *Value[int] { return &r.Playoffs.ByeSeed }},
		field[SeedPolicy]{path: "playoffs.seeding", label: "Playoff seeding", access: func(r *RuleSet) *Value[SeedPolicy] { return &r.Playoffs.Seeding }},
		field[bool]{path: "playoffs.reseed", label: "Reseed playoff rounds", access: func(r *RuleSet) *Value[bool] { return &r.Playoffs.Reseed }},
		field[int]{path: "playoffs.startWeek", label: "First playoff week", access: func(r *RuleSet) *Value[int] { return &r.Playoffs.StartWeek }},
		field[int]{path: "playoffs.finalWeek", label: "Final playoff week", access: func(r *RuleSet) *Value[int] { return &r.Playoffs.FinalWeek }},
		field[Decimal]{path: "playoffs.homeBonus", label: "Home field points", access: func(r *RuleSet) *Value[Decimal] { return &r.Playoffs.HomeBonus }},
		field[[]BracketNode]{path: "playoffs.graph", label: "Playoff bracket", access: func(r *RuleSet) *Value[[]BracketNode] { return &r.Playoffs.Graph }},
	}
}

type BracketNode struct {
	ID         Value[string]
	Week       Value[int]
	HomeSeed   Value[int]
	AwaySeed   Value[int]
	HomeWinner Value[string]
	AwayWinner Value[string]
}

func legacyBracket() []BracketNode {
	rows := []struct {
		id                     string
		week, home, away       int
		homeWinner, awayWinner string
	}{
		{"wildcard_2", 14, 2, 7, "", ""}, {"wildcard_3", 14, 3, 6, "", ""}, {"wildcard_4", 14, 4, 5, "", ""},
		{"division_1", 15, 1, 0, "", "lowest remaining seed"},
		{"division_2", 15, 0, 0, "highest remaining seed", "other remaining seed"},
		{"conference", 16, 0, 0, "division winner", "division winner"},
		{"championship", 17, 0, 0, "conference winner", "conference winner"},
	}
	result := make([]BracketNode, len(rows))
	for i, row := range rows {
		result[i] = BracketNode{ID: cited(row.id, "fact 11a-b"), Week: cited(row.week, "Census CM-016"),
			HomeSeed: Needs[int]("fact 11b: reseeded winner"), AwaySeed: Needs[int]("fact 11b: reseeded winner"),
			HomeWinner: cited(row.homeWinner, "fact 11b"), AwayWinner: cited(row.awayWinner, "fact 11b")}
		if row.home != 0 {
			result[i].HomeSeed = cited(row.home, "fact 11a-b")
		}
		if row.away != 0 {
			result[i].AwaySeed = cited(row.away, "fact 11a-b")
		}
	}
	return result
}
