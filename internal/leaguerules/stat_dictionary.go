package leaguerules

import (
	"slices"
	"sync"
)

// StatDefinition is one stat a league can score: every event in MFL's scoring dictionary and every
// stat the app's feeds record. A commissioner can rate any of them at any value; one no feed records
// yet still scores, it simply has nothing recorded to count.
type StatDefinition struct {
	ID    StatID
	Label string
	Group string
	// MFL is MFL's event code; empty for a stat only this app scores.
	MFL string
	// Feed says a live feed records the stat, or every stat it is counted from.
	Feed bool
	// Parts makes the stat the total of these recorded stats.
	Parts []StatID
	// Plays makes the stat a count of these stats' plays at least AtLeast long.
	Plays   []StatID
	AtLeast Decimal
	// Lengths says the stat is recorded play by play as each play's length, so a rule's range
	// applies to each play rather than to the week.
	Lengths bool
}

// NoFeedWords is how a stat no feed records yet is named beside it.
const NoFeedWords = "no live feed yet"

type statRow struct {
	mfl, id, label, group string
	feed                  bool
}

type statIndex struct {
	list  []StatDefinition
	byID  map[StatID]StatDefinition
	byMFL map[string]StatDefinition
}

//nolint:gochecknoglobals // fixed by the code, built once, read-only
var stats = sync.OnceValue(func() statIndex {
	index := statIndex{list: buildStatDictionary(), byID: map[StatID]StatDefinition{}, byMFL: map[string]StatDefinition{}}
	for _, s := range index.list {
		index.byID[s.ID] = s
		if s.MFL != "" {
			index.byMFL[s.MFL] = s
		}
	}
	return index
})

// StatDictionary lists every stat a league can score, in MFL's dictionary order then the app's own.
func StatDictionary() []StatDefinition { return slices.Clone(stats().list) }

// StatByID finds a stat's definition.
func StatByID(id StatID) (StatDefinition, bool) {
	s, ok := stats().byID[id]
	return s, ok
}

// StatByMFL finds the stat an MFL event code scores.
func StatByMFL(code string) (StatDefinition, bool) {
	s, ok := stats().byMFL[code]
	return s, ok
}

// StatWords names a stat for a GM, saying when no feed records it yet.
func StatWords(s StatDefinition) string {
	if s.Feed {
		return s.Label
	}
	return s.Label + " (" + NoFeedWords + ")"
}

func statChoices() []Choice {
	list := stats().list
	choices := make([]Choice, len(list))
	for i, s := range list {
		choices[i] = Choice{Value: string(s.ID), Label: StatWords(s)}
	}
	return choices
}

func buildStatDictionary() []StatDefinition {
	var out []StatDefinition
	for _, rows := range [][]statRow{mflOffenseStats(), mflKickingStats(), mflDefenseStats(), mflTeamStats(), appStats()} {
		for _, r := range rows {
			out = append(out, StatDefinition{ID: StatID(r.id), Label: r.label, Group: r.group, MFL: r.mfl, Feed: r.feed})
		}
	}
	for i := range out {
		derive(&out[i])
	}
	return out
}

// derive fills how a stat is counted from others, for the stats no feed records directly.
func derive(s *StatDefinition) {
	plays := map[StatID]struct {
		from    []StatID
		atLeast Decimal
	}{
		"long_pass_20": {[]StatID{"long_pass"}, 200000}, "long_pass_30": {[]StatID{"long_pass"}, 300000},
		"long_pass_40": {[]StatID{"long_pass"}, 400000}, "long_pass_50": {[]StatID{"long_pass"}, 500000},
		"long_rush_20": {[]StatID{"long_rush"}, 200000}, "long_rush_40": {[]StatID{"long_rush"}, 400000},
		"long_reception_20": {[]StatID{"long_reception"}, 200000},
		"long_reception_40": {[]StatID{"long_reception"}, 400000},
		"fg_made":           {[]StatID{"fg_made_distance"}, 0}, "fg_missed": {[]StatID{"fg_missed_distance"}, 0},
		"fg_attempt": {[]StatID{"fg_made_distance", "fg_missed_distance"}, 0},
	}
	parts := map[StatID][]StatID{
		"rushing_receiving_yards": {"rushing_yards", "receiving_yards"},
		"passing_rushing_yards":   {"passing_yards", "rushing_yards"},
		"scrimmage_yards":         {"rushing_yards", "receiving_yards"},
		"pat_missed_or_blocked":   {"pat_missed", "pat_blocked"},
		"pat_attempt":             {"pat_made", "pat_missed", "pat_blocked"},
		"punt_or_other_return_td": {"punt_return_td", "other_return_td"},
		"touches":                 {"rushing_attempt", "reception"},
	}
	if p, ok := plays[s.ID]; ok {
		s.Plays, s.AtLeast = p.from, p.atLeast
	}
	s.Parts = parts[s.ID]
	s.Lengths = slices.Contains([]StatID{"fg_made_distance", "fg_missed_distance", "passing_td_length",
		"rushing_td_length", "receiving_td_length", "pick_six_thrown_length", "punt_return_td_length",
		"kickoff_return_td_length", "defensive_fumble_recovery_td_length", "lost_fumble_return_td_length",
		"offensive_fumble_recovery_td_length", "interception_return_td_length", "blocked_fg_td_length",
		"missed_fg_return_td_length", "blocked_punt_td_length", "defensive_td_length",
		"defensive_special_teams_td_length"}, s.ID)
}

func mflOffenseStats() []statRow {
	return []statRow{
		{"#P", "passing_td", "Passing touchdown", "Passing", true},
		{"PS", "passing_td_length", "Length of passing touchdown", "Passing", false},
		{"PY", "passing_yards", "Passing yards", "Passing", true},
		{"APY", "passing_average", "Average yards per pass completion", "Passing", false},
		{"PA", "pass_attempt", "Pass attempt", "Passing", false},
		{"PC", "pass_completion", "Pass completion", "Passing", false},
		{"PCP", "pass_completion_percentage", "Pass completion percentage", "Passing", false},
		{"INC", "pass_incompletion", "Pass incompletion", "Passing", false},
		{"IN", "interception_thrown", "Interception thrown", "Passing", true},
		{"#IT", "pick_six_thrown", "Interception thrown returned for a touchdown", "Passing", false},
		{"IT", "pick_six_thrown_length", "Length of interception thrown returned for a touchdown", "Passing", false},
		{"TSK", "times_sacked", "Quarterback sacked", "Passing", false},
		{"TSY", "times_sacked_yards", "Quarterback sacked yardage", "Passing", false},
		{"RATE", "passer_rating", "Passer rating", "Passing", false},
		{"P20", "long_pass_20", "Completion of 20 yards or more", "Passing", true},
		{"P30", "long_pass_30", "Completion of 30 yards or more", "Passing", true},
		{"P40", "long_pass_40", "Completion of 40 yards or more", "Passing", true},
		{"P50", "long_pass_50", "Completion of 50 yards or more", "Passing", true},
		{"P2", "passing_two_point", "Passing two point conversion", "Passing", true},
		{"#R", "rushing_td", "Rushing touchdown", "Rushing", true},
		{"RS", "rushing_td_length", "Length of rushing touchdown", "Rushing", false},
		{"RY", "rushing_yards", "Rushing yards", "Rushing", true},
		{"ARY", "rushing_average", "Average yards per rush", "Rushing", false},
		{"RA", "rushing_attempt", "Rushing attempt", "Rushing", true},
		{"R20", "long_rush_20", "Rush of 20 yards or more", "Rushing", true},
		{"R40", "long_rush_40", "Rush of 40 yards or more", "Rushing", true},
		{"R2", "rushing_two_point", "Rushing two point conversion", "Rushing", true},
		{"#C", "receiving_td", "Receiving touchdown", "Receiving", true},
		{"RC", "receiving_td_length", "Length of receiving touchdown", "Receiving", false},
		{"CY", "receiving_yards", "Receiving yards", "Receiving", true},
		{"ACY", "receiving_average", "Average yards per reception", "Receiving", false},
		{"CC", "reception", "Reception", "Receiving", true},
		{"TGT", "target", "Target", "Receiving", false},
		{"C20", "long_reception_20", "Reception of 20 yards or more", "Receiving", true},
		{"C40", "long_reception_40", "Reception of 40 yards or more", "Receiving", true},
		{"C2", "receiving_two_point", "Receiving two point conversion", "Receiving", true},
		{"RCY", "rushing_receiving_yards", "Rushing and receiving yards", "Total yardage", true},
		{"PRY", "passing_rushing_yards", "Passing and rushing yards", "Total yardage", true},
		{"TYS", "scrimmage_yards", "Total yards from scrimmage", "Total yardage", true},
		{"TY", "total_yards", "Total yards", "Total yardage", false},
		{"#TD", "total_touchdowns", "Total touchdowns", "Total yardage", false},
	}
}

func mflKickingStats() []statRow {
	return []statRow{
		{"#F", "fg_made", "Field goal made", "Kicking", true},
		{"FG", "fg_made_distance", "Field goal made, by distance", "Kicking", true},
		{"#M", "fg_missed", "Field goal missed", "Kicking", true},
		{"MG", "fg_missed_distance", "Field goal missed or blocked, by distance", "Kicking", true},
		{"#A", "fg_attempt", "Field goal attempted", "Kicking", true},
		{"FGP", "fg_percentage", "Field goal percentage", "Kicking", false},
		{"HBF", "fg_blocked", "Field goal had blocked", "Kicking", true},
		{"EP", "pat_made", "Extra point made", "Kicking", true},
		{"EM", "pat_missed_or_blocked", "Extra point missed or blocked", "Kicking", true},
		{"EA", "pat_attempt", "Extra point attempted", "Kicking", true},
		{"EPP", "pat_percentage", "Extra point percentage", "Kicking", false},
		{"HBE", "pat_blocked", "Extra point blocked", "Kicking", true},
		{"#N", "punt", "Punt", "Punting", false},
		{"PNY", "punt_yards", "Punt yards", "Punting", false},
		{"ANY", "punt_average", "Average yards per punt", "Punting", false},
		{"PI", "punt_inside_20", "Punt inside the 20", "Punting", false},
		{"HBP", "punt_had_blocked", "Punt had blocked", "Punting", false},
		{"#UT", "punt_or_other_return_td", "Punt or other return touchdown", "Returns", true},
		{"PR", "punt_return_td_length", "Length of punt return touchdown", "Returns", false},
		{"#U", "punt_return", "Punt return", "Returns", false},
		{"UY", "punt_return_yards", "Punt return yards", "Returns", true},
		{"AUY", "punt_return_average", "Average yards per punt return", "Returns", false},
		{"U40", "long_punt_return_40", "Punt return of 40 yards or more", "Returns", false},
		{"#KT", "kickoff_return_td", "Kickoff return touchdown", "Returns", true},
		{"KO", "kickoff_return_td_length", "Length of kickoff return touchdown", "Returns", false},
		{"#K", "kickoff_return", "Kickoff return", "Returns", false},
		{"KY", "kickoff_return_yards", "Kickoff return yards", "Returns", true},
		{"AKY", "kickoff_return_average", "Average yards per kickoff return", "Returns", false},
		{"K40", "long_kickoff_return_40", "Kickoff return of 40 yards or more", "Returns", false},
	}
}

func mflDefenseStats() []statRow {
	return []statRow{
		{"FU", "fumbles_total", "All fumbles", "Fumbles", true},
		{"FUO", "fumble_offense", "Fumble on offense", "Fumbles", false},
		{"FUS", "fumble_special_teams", "Fumble on special teams", "Fumbles", false},
		{"FUD", "fumble_defense", "Fumble on defense", "Fumbles", false},
		{"FL", "fumble_lost", "Fumbles lost", "Fumbles", true},
		{"FLO", "fumble_lost_offense", "Fumble lost on offense", "Fumbles", false},
		{"FLS", "fumble_lost_special_teams", "Fumble lost on special teams", "Fumbles", false},
		{"FLD", "fumble_lost_defense", "Fumble lost on defense", "Fumbles", false},
		{"#DR", "fumble_recovery_tds", "Fumble recovery touchdown", "Fumbles", true},
		{"DR", "defensive_fumble_recovery_td_length", "Length of defensive fumble recovery touchdown", "Fumbles", false},
		{"FC", "opponent_fumble_recovery", "Opponent fumble recovery", "Fumbles", true},
		{"FCD", "fumble_recovery_defense", "Fumble recovery on defense", "Fumbles", false},
		{"FCS", "fumble_recovery_special_teams", "Fumble recovery on special teams", "Fumbles", false},
		{"FCY", "opponent_fumble_recovery_yards", "Opponent fumble recovery yards", "Fumbles", true},
		{"#FT", "lost_fumble_return_td", "Fumble lost returned for a touchdown", "Fumbles", false},
		{"FT", "lost_fumble_return_td_length", "Length of fumble lost returned for a touchdown", "Fumbles", false},
		{"#FR", "offensive_fumble_recovery_td", "Offensive fumble recovery touchdown", "Fumbles", false},
		{"FR", "offensive_fumble_recovery_td_length", "Length of offensive fumble recovery touchdown", "Fumbles", false},
		{"OFC", "own_fumble_recovery", "Own fumble recovery", "Fumbles", true},
		{"OFCY", "own_fumble_recovery_yards", "Own fumble recovery yards", "Fumbles", false},
		{"FF", "forced_fumble", "Forced fumble", "Fumbles", true},
		{"#IR", "interception_return_td", "Interception return touchdown", "Defense", true},
		{"IR", "interception_return_td_length", "Length of interception return touchdown", "Defense", false},
		{"IC", "interception_caught", "Interception caught", "Defense", true},
		{"ICY", "interception_return_yards", "Interception return yards", "Defense", true},
		{"PD", "pass_defensed", "Pass defensed", "Defense", true},
		{"#BF", "blocked_fg_td", "Blocked field goal touchdown", "Defense", true},
		{"BF", "blocked_fg_td_length", "Length of blocked field goal touchdown", "Defense", false},
		{"BLF", "blocked_fg", "Blocked field goal", "Defense", true},
		{"#MF", "other_return_td", "Missed field goal return touchdown", "Returns", true},
		{"MF", "missed_fg_return_td_length", "Length of missed field goal return touchdown", "Returns", false},
		{"#BP", "blocked_punt_td", "Blocked punt touchdown", "Defense", true},
		{"BP", "blocked_punt_td_length", "Length of blocked punt touchdown", "Defense", false},
		{"BLP", "blocked_punt", "Blocked punt", "Defense", true},
		{"BLE", "blocked_extra_point", "Blocked extra point", "Defense", true},
		{"TKD", "defensive_tackle", "Defensive tackle", "Defense", false},
		{"ASD", "defensive_assist", "Defensive assisted tackle", "Defense", false},
		{"TK", "tackle", "Tackle", "Defense", true},
		{"AS", "assist", "Assisted tackle", "Defense", true},
		{"AS2", "assist_half", "Assisted tackles divided by two", "Defense", false},
		{"SK", "sack", "Sack", "Defense", true},
		{"SKY", "defensive_sack_yards", "Sack yards", "Defense", true},
		{"QH", "qb_hit", "Quarterback hit", "Defense", true},
		{"TKL", "tackle_for_loss", "Tackle for loss", "Defense", true},
		{"SF", "safety", "Safety", "Defense", true},
	}
}

func mflTeamStats() []statRow {
	return []statRow{
		{"TPA", "team_points_allowed", "Total points allowed", "Team", false},
		{"STPA", "team_special_teams_points_allowed", "Special teams points against", "Team", false},
		{"OPA", "team_offensive_points_allowed", "Offensive points against", "Team", false},
		{"TYA", "team_yards_allowed", "Total net yards allowed", "Team", false},
		{"TYP", "team_passing_yards_allowed", "Passing yards allowed", "Team", false},
		{"TYR", "team_rushing_yards_allowed", "Rushing yards allowed", "Team", false},
		{"STTA", "team_special_teams_tds_allowed", "Special teams touchdowns allowed", "Team", false},
		{"OTA", "team_offensive_tds_allowed", "Offensive touchdowns allowed", "Team", false},
		{"#D", "def_tds", "Defensive touchdown", "Defense", true},
		{"DD", "defensive_td_length", "Length of defensive touchdown", "Defense", false},
		{"D2", "defensive_conversion_return", "Defensive conversion return", "Defense", false},
		{"SF1", "one_point_safety", "Safety for one point", "Defense", false},
		{"M2", "team_two_point_made", "Successful two point conversion", "Team", false},
		{"F2", "team_two_point_failed", "Failed two point conversion", "Team", false},
		{"#T", "defensive_special_teams_td", "Defensive or special teams touchdown", "Defense", false},
		{"TT", "defensive_special_teams_td_length", "Length of defensive or special teams touchdown", "Defense", false},
		{"F", "penalty", "Penalty", "Penalties", true},
		{"FY", "penalty_yards", "Penalty yards", "Penalties", true},
		{"TIME", "team_time_of_possession", "Time of possession in minutes", "Team", false},
		{"TPS", "team_points_scored", "Total points scored", "Team", false},
		{"DIF", "point_differential", "Point differential", "Team", false},
		{"OPS", "team_offensive_points", "Offensive points only", "Team", false},
		{"FD", "first_down", "First down", "First downs", false},
		{"1P", "passing_first_down", "Passing first down", "First downs", false},
		{"1R", "rushing_first_down", "Rushing first down", "First downs", false},
		{"1C", "receiving_first_down", "Receiving first down", "First downs", false},
		{"1F", "team_penalty_first_down", "Penalty first down", "Team", false},
		{"3D", "team_third_down_conversion", "Team third down conversion", "Team", false},
		{"3F", "team_third_down_failed", "Team third down failed", "Team", false},
		{"4D", "team_fourth_down_conversion", "Team fourth down conversion", "Team", false},
		{"4F", "team_fourth_down_failed", "Team fourth down failed", "Team", false},
		{"W", "game_won", "Game won", "Team", false},
		{"L", "game_lost", "Game lost", "Team", false},
		{"T", "game_tied", "Game tied", "Team", false},
		{"APYAG", "team_passing_yards_allowed_per_game", "Average passing yards allowed per game", "Team", false},
		{"APYAR", "team_passing_defense_rank", "Passing defense rank", "Team", false},
		{"ARYAG", "team_rushing_yards_allowed_per_game", "Average rushing yards allowed per game", "Team", false},
		{"ARYAR", "team_rushing_defense_rank", "Rushing defense rank", "Team", false},
	}
}

// appStats are the stats this app records or scores that MFL's dictionary does not name.
func appStats() []statRow {
	return []statRow{
		{"", "long_pass", "Pass, by distance", "Passing", true},
		{"", "long_rush", "Rush, by distance", "Rushing", true},
		{"", "long_reception", "Reception, by distance", "Receiving", true},
		{"", "chunk_play", "Rush or catch, by distance", "Big plays", true},
		{"", "touches", "Touches", "Big plays", true},
		{"", "fg_made_0_39", "Field goal made from 0 to 39 yards", "Kicking", true},
		{"", "fg_made_40_49", "Field goal made from 40 to 49 yards", "Kicking", true},
		{"", "fg_made_50_59", "Field goal made from 50 to 59 yards", "Kicking", true},
		{"", "fg_made_60_99", "Field goal made from 60 to 99 yards", "Kicking", true},
		{"", "fg_missed_0_29", "Field goal missed from 0 to 29 yards", "Kicking", true},
		{"", "fg_missed_30_99", "Field goal missed from 30 yards or longer", "Kicking", true},
		{"", "pat_missed", "Extra point missed", "Kicking", true},
		{"", "punt_return_td", "Punt return touchdown", "Returns", true},
		{"", "tackle_line", "Defensive line tackle", "Defense", true},
		{"", "assist_line", "Defensive line assisted tackle", "Defense", true},
		{"", "tackle_secondary", "Tackle (cornerbacks and safeties)", "Defense", true},
		{"", "pass_defensed_secondary", "Pass defensed (cornerbacks and safeties)", "Defense", true},
		{"", "interception_caught_secondary", "Interception caught (cornerbacks and safeties)", "Defense", true},
		{"", "rushing_tfl", "Rush tackled for loss", "Rushing", true},
		{"", "third_down_conversion", "Third down conversion", "Conversions", true},
		{"", "fourth_down_conversion", "Fourth down conversion", "Conversions", true},
		{"", "third_down_passing_conversion", "Third down passing conversion", "Conversions", true},
		{"", "fourth_down_passing_conversion", "Fourth down passing conversion", "Conversions", true},
		{"", "red_zone_carry", "Red zone carry", "Rushing", true},
		{"", "red_zone_target", "Red zone target", "Receiving", true},
		{"", "red_zone_pass_attempt", "Red zone pass attempt", "Passing", true},
		{"", "rushing_yards_after_contact", "Rushing yards after contact", "Rushing", true},
		{"", "receiving_yards_after_catch", "Receiving yards after catch", "Receiving", true},
		{"", "receiving_air_yards", "Receiving air yards", "Receiving", true},
		{"", "passing_air_yards", "Passing air yards", "Passing", true},
		{"", "rushing_broken_tackle", "Broken tackle", "Rushing", true},
		{"", "receiving_drop", "Drop", "Receiving", true},
		{"", "special_teams_solo_tackle", "Special teams solo tackle", "Special teams", true},
		{"", "offensive_snap", "Offensive snap", "Snaps", true},
		{"", "defensive_snap", "Defensive snap", "Snaps", true},
		{"", "special_teams_snap", "Special teams snap", "Snaps", true},
		{"", "line_rb_yards_before_contact", "Running back yards before contact", "Offensive line", true},
		{"", "line_rushes_10_plus", "Rushes of 10 yards or more", "Offensive line", true},
		{"", "line_rushing_tds", "Rushing touchdowns", "Offensive line", true},
		{"", "line_short_conversions", "Third or fourth down rushes converted from 2 yards or less",
			"Offensive line", true},
		{"", "line_tackled_for_loss", "Rushes tackled for a loss", "Offensive line", true},
		{"", "line_sacks", "Sacks allowed", "Offensive line", true},
		{"", "line_qb_hits", "Quarterback hits allowed", "Offensive line", true},
	}
}
