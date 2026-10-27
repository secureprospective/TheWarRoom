package leaguerules

const (
	default32Citation = "DEFAULT-32 spec §4, locked 2026-10-26"
	default32Lineup   = "DEFAULT-32 spec §1, locked 2026-10-26"
	default32Roster   = "DEFAULT-32 spec §2, locked 2026-10-26"
	default32OLine    = "DEFAULT-32 spec §3, locked 2026-10-26"
)

// Default32 is the 32-Team Default league: Legacy's rules for 32 teams with the spec's lineup, roster,
// IR and scoring. Every value is a commissioner setting the league can change afterwards.
// Taxi keeps Legacy's 8 spots, 3-year limit and calendar-year measure; the spec's "years out of the
// NFL don't count" and the automatic moves (taxi expiry, IR return) have no rule field yet.
func Default32() RuleSet {
	r := LegacyNFLForTeams(32)
	// 19 starters: QB 1, RB 1, WR 1, three RB/WR/TE flex, PK 1, the O-Line card, DT/DE/LB 1 each,
	// CB 2, S 1 and five IDP flex. A position's maximum is its minimum plus its flex spots; the
	// exact offense and defense counts make the flex, with no per-position caps beyond that.
	r.Lineup.Positions = cited(map[Position]Bounds{
		QB: bounds(1, 1, default32Lineup), RB: bounds(1, 4, default32Lineup), WR: bounds(1, 4, default32Lineup),
		TE: bounds(0, 3, default32Lineup), PK: bounds(1, 1, default32Lineup), OL: bounds(1, 1, default32Lineup),
		DT: bounds(1, 6, default32Lineup), DE: bounds(1, 6, default32Lineup), LB: bounds(1, 6, default32Lineup),
		CB: bounds(2, 7, default32Lineup), S: bounds(1, 6, default32Lineup),
	}, default32Lineup)
	r.Lineup.Groups = cited(map[LineupGroup]Bounds{
		Offense: bounds(6, 6, default32Lineup), Defense: bounds(11, 11, default32Lineup), Total: bounds(19, 19, default32Lineup),
	}, default32Lineup)
	// 44 active including the O-Line card; taxi and IR sit outside it. The minimum stays Legacy's.
	r.Roster.Limits = cited(Bounds{Min: r.Roster.Limits.Value.Min, Max: cited(44, default32Roster)}, default32Roster)
	r.Roster.ReserveExcluded = cited(true, default32Roster)
	r.Roster.PositionBounds = cited(map[Position]Bounds{OL: bounds(1, 1, default32OLine)}, default32OLine)
	r.Taxi.Slots = cited(8, default32Roster)
	r.Taxi.MaximumYears = cited(3, default32Roster)
	r.Taxi.Basis = cited(NFLSeasons, default32Roster)
	r.IR.Slots = cited(12, default32Roster)
	r.IR.Eligible = cited([]NFLStatus{InjuredReserve, PUP}, default32Roster)
	r.IR.MustLeaveHealthy = cited(true, default32Roster)
	r.Scoring = Default32Scoring()
	return r
}

// Default32Scoring is the 32-team default as commissioner scoring settings: Legacy's rule list plus
// the K7 base, the continuous chunk scale, distance kicking and Team O-Line Set B. Every number is
// a setting the commissioner can change.
func Default32Scoring() ScoringRules {
	catalogue := map[StatID]Stat{}
	ranges := map[StatID]StatRange{}
	for _, row := range default32Rows() {
		stat := scoringStatCited(row.label, row.group, row.points, default32Citation)
		if len(row.positions) != 0 {
			stat.Positions = cited(row.positions, default32Citation)
		}
		stat.Once = cited(row.once, default32Citation)
		catalogue[row.id] = stat
		if row.ranged {
			ranges[row.id] = StatRange{Min: row.min * 10000, Max: row.max * 10000}
		}
	}
	lost := scoringStatCited("Fumbles lost", "Fumbles", -20000, default32Citation)
	lost.Used = cited(false, default32Citation)
	catalogue["fumble_lost"] = lost
	// EM is one rule over missed and blocked extra points; the parts are counted only through it.
	// #UT likewise is one rule over punt returns and other returns (such as a missed field goal).
	for id, label := range map[StatID]string{"pat_missed": "Extra point missed", "pat_blocked": "Extra point blocked",
		"punt_return_td": "Punt return touchdown", "other_return_td": "Other return touchdown"} {
		part := scoringStatCited(label, "Kicking", 0, default32Citation)
		part.Used = cited(false, default32Citation)
		catalogue[id] = part
	}
	r := ScoringRules{
		Catalogue:       cited(catalogue, default32Citation),
		RoundStatPoints: cited(true, default32Citation),
		Extras:          cited(default32Extras(), default32Citation),
		Ranges:          cited(ranges, default32Citation),
		Scales:          cited(default32Scales(), default32Citation),
		Totals: cited(map[StatID][]StatID{"touches": {"rushing_attempt", "reception"},
			"pat_missed_or_blocked":   {"pat_missed", "pat_blocked"},
			"punt_or_other_return_td": {"punt_return_td", "other_return_td"}}, default32Citation),
		LineUnit: cited(default32LineUnit(), default32Citation),
	}
	return r
}

type default32Row struct {
	id           StatID
	label, group string
	points       Decimal
	ranged       bool
	min, max     Decimal
	once         bool
	positions    []Position
}

func default32Rows() []default32Row {
	idp := []Position{DT, DE, LB, CB, S}
	return []default32Row{
		{"passing_yards", "Passing yards", "Passing", 500, true, -50, 999, false, nil},
		{"passing_td", "Passing touchdown", "Passing", 50000, true, 0, 20, false, nil},
		{"interception_thrown", "Interception thrown", "Passing", -20000, true, 0, 10, false, nil},
		{"long_pass", "Completion of 40 yards or more", "Passing", 10000, true, 0, 99, false, nil},
		{"passing_two_point", "Passing two point conversion", "Passing", 20000, true, 0, 10, false, nil},
		{"rushing_yards", "Rushing yards", "Rushing", 1000, true, -50, 999, false, nil},
		{"rushing_attempt", "Rushing attempt", "Rushing", 1500, true, 0, 99, false, nil},
		{"rushing_td", "Rushing touchdown", "Rushing", 60000, true, 0, 10, false, nil},
		{"rushing_two_point", "Rushing two point conversion", "Rushing", 20000, true, 0, 10, false, nil},
		{"receiving_yards", "Receiving yards", "Receiving", 1000, true, -50, 999, false, nil},
		{"reception", "Reception", "Receiving", 10000, true, 0, 99, false, nil},
		{"receiving_td", "Receiving touchdown", "Receiving", 60000, true, 0, 10, false, nil},
		{"receiving_two_point", "Receiving two point conversion", "Receiving", 20000, true, 0, 10, false, nil},
		{"chunk_play", "Rush or catch of 20 yards or more", "Big plays", 10000, true, 0, 999, false, nil},
		{"touches", "20 or more touches", "Big plays", 30000, true, 20, 99, true, nil},
		{"kickoff_return_yards", "Kickoff return yards", "Returns", 500, true, -50, 999, false, nil},
		{"punt_return_yards", "Punt return yards", "Returns", 500, true, -50, 999, false, nil},
		{"kickoff_return_td", "Kickoff return touchdown", "Returns", 100000, true, 0, 10, false, nil},
		{"punt_or_other_return_td", "Punt or other return touchdown", "Returns", 60000, true, 0, 10, false, nil},
		{"fumbles_total", "All fumbles", "Fumbles", -20000, true, 0, 10, false, nil},
		{"own_fumble_recovery", "Own fumble recovery", "Fumbles", 20000, true, 1, 10, true, nil},
		{"fg_made_distance", "Field goal made, by distance", "Kicking", 10000, true, 0, 999, false, nil},
		{"fg_missed_distance", "Field goal missed or blocked, by distance", "Kicking", 10000, true, -999, 0, false, nil},
		{"pat_made", "Extra point made", "Kicking", 10000, true, 0, 10, false, nil},
		{"pat_missed_or_blocked", "Extra point missed or blocked", "Kicking", -10000, true, 0, 10, false, nil},
		{"tackle", "Tackle", "Defense", 20000, true, 0, 99, false, nil},
		{"assist", "Assisted tackle", "Defense", 12500, true, 0, 99, false, nil},
		{"sack", "Sack", "Defense", 45000, true, 0, 25, false, nil},
		{"tackle_for_loss", "Tackle for loss", "Defense", 25000, true, 0, 25, false, nil},
		{"qb_hit", "Quarterback hit", "Defense", 10000, true, 0, 10, false, nil},
		{"forced_fumble", "Forced fumble", "Fumbles", 40000, true, 0, 10, false, nil},
		{"opponent_fumble_recovery", "Opponent fumble recovery", "Fumbles", 30000, true, 0, 10, false, nil},
		{"opponent_fumble_recovery_yards", "Opponent fumble recovery yards", "Fumbles", 1000, true, -50, 999, false, nil},
		{"interception_caught", "Interception caught", "Defense", 50000, true, 0, 10, false, nil},
		{"interception_return_yards", "Interception return yards", "Defense", 1000, true, -50, 999, false, nil},
		{"pass_defensed", "Pass defensed", "Defense", 25000, true, 0, 99, false, nil},
		{"safety", "Safety", "Defense", 100000, true, 0, 10, false, nil},
		{"blocked_fg", "Blocked field goal", "Defense", 70000, true, 0, 10, false, nil},
		{"blocked_punt", "Blocked punt", "Defense", 70000, true, 0, 10, false, nil},
		{"blocked_extra_point", "Blocked extra point", "Defense", 70000, true, 0, 10, false, nil},
		{"fumble_recovery_tds", "Fumble recovery touchdown", "Fumbles", 60000, true, 0, 10, false, nil},
		{"interception_return_td", "Interception return touchdown", "Defense", 60000, true, 0, 10, false, nil},
		{"blocked_fg_td", "Blocked field goal touchdown", "Defense", 70000, true, 0, 10, false, idp},
		{"blocked_punt_td", "Blocked punt touchdown", "Defense", 70000, true, 0, 10, false, idp},
	}
}

func default32Extras() map[StatID][]PositionExtra {
	return map[StatID][]PositionExtra{
		"reception":           {{Positions: []Position{TE}, Points: 2000, Ranged: true, Range: StatRange{Min: 0, Max: 9990000}}},
		"tackle":              {{Positions: []Position{DT, DE}, Points: 10000}, {Positions: []Position{CB, S}, Points: 5000}},
		"assist":              {{Positions: []Position{DT, DE}, Points: 5000}},
		"interception_caught": {{Positions: []Position{CB, S}, Points: 10000}},
		"pass_defensed":       {{Positions: []Position{CB, S}, Points: 5000}},
	}
}

func default32Scales() map[StatID]DistanceScale {
	return map[StatID]DistanceScale{
		"long_pass":          {From: 400000, Base: 10000, Pivot: 400000},
		"chunk_play":         {From: 200000, Base: 10000, Pivot: 200000, Above: 250},
		"fg_made_distance":   {Base: 30000, Pivot: 400000, Above: 1000},
		"fg_missed_distance": {Pivot: 600000, Below: -1000},
	}
}

func default32LineUnit() LineUnit {
	return LineUnit{Used: true, NoSack: 20000, Sacks: "line_sacks", Clamped: true, Clamp: 150000,
		Weights: map[StatID]Decimal{
			"line_rb_yards_before_contact": 400, "line_rushes_10_plus": 10000, "line_rushing_tds": 10000,
			"line_short_conversions": 10000, "line_tackled_for_loss": -5000, "line_sacks": -10000,
			"line_qb_hits": -2500,
		},
		Labels: map[StatID]string{
			"line_rb_yards_before_contact": "Running back yards before contact",
			"line_rushes_10_plus":          "Rushes of 10 yards or more",
			"line_rushing_tds":             "Rushing touchdowns",
			"line_short_conversions":       "Third or fourth down rushes converted from 2 yards or less",
			"line_tackled_for_loss":        "Rushes tackled for a loss",
			"line_sacks":                   "Sacks allowed",
			"line_qb_hits":                 "Quarterback hits allowed",
		}}
}
