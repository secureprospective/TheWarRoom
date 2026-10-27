package leaguerules

const (
	default32Lineup = "DEFAULT-32 spec §1, locked 2026-10-26"
	default32Roster = "DEFAULT-32 spec §2, locked 2026-10-26"
	default32OLine  = "DEFAULT-32 spec §3, locked 2026-10-26"
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
	r.IR.Slots = cited(12, default32Roster)
	r.IR.Eligible = cited([]NFLStatus{InjuredReserve, PUP}, default32Roster)
	r.IR.MustLeaveHealthy = cited(true, default32Roster)
	r.Scoring = Default32Scoring()
	return r
}
