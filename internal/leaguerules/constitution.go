package leaguerules

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type LeagueRules struct {
	Teams          Value[int]
	Conferences    Value[int]
	Divisions      Value[int]
	Format         Value[Format]
	InitialRosters Value[InitialRosters]
	Alignment      Value[map[string]string]
	// DivisionConferences names the conference of each division in Alignment. A rule set stored before
	// it existed reads it as zero and omits it, so that set encodes, and digests, as it did.
	DivisionConferences Value[map[string]string] `json:",omitzero"`
	Dynasty             Value[bool]
	Carry               Value[CarryPolicy]
	Keepers             Value[int]
}

func legacyLeague() LeagueRules {
	return LeagueRules{
		Teams:               cited[int](32, "Census CM-001"),
		Conferences:         cited[int](2, "Census CM-001"),
		Divisions:           cited[int](8, "Census CM-001"),
		Format:              cited[Format](HeadToHead, "Census CM-001"),
		InitialRosters:      cited[InitialRosters](ByDraft, "Census CM-001"),
		Alignment:           Needs[map[string]string]("Census CM-002: carried at setup"),
		DivisionConferences: Needs[map[string]string]("Census CM-002: carried at setup"),
		Dynasty:             cited[bool](true, "Census CM-005"),
		Carry:               cited[CarryPolicy](CarryAll, "Census CM-005"),
		Keepers:             Needs[int]("Census CM-005: only for keeper leagues"),
	}
}

func leagueFields() []ruleField {
	return []ruleField{
		field[int]{path: "league.teams", label: "Teams", access: func(r *RuleSet) *Value[int] { return &r.League.Teams }},
		field[int]{path: "league.conferences", label: "Conferences", access: func(r *RuleSet) *Value[int] { return &r.League.Conferences }},
		field[int]{path: "league.divisions", label: "Divisions", access: func(r *RuleSet) *Value[int] { return &r.League.Divisions }},
		field[Format]{path: "league.format", label: "Competition format", access: func(r *RuleSet) *Value[Format] { return &r.League.Format }},
		field[InitialRosters]{path: "league.initialRosters", label: "First rosters", access: func(r *RuleSet) *Value[InitialRosters] { return &r.League.InitialRosters }},
		field[map[string]string]{path: "league.alignment", label: "Division alignment", access: func(r *RuleSet) *Value[map[string]string] { return &r.League.Alignment }},
		field[map[string]string]{path: "league.divisionConferences", label: "Conference of each division", access: func(r *RuleSet) *Value[map[string]string] { return &r.League.DivisionConferences }},
		field[bool]{path: "league.dynasty", label: "Dynasty league", access: func(r *RuleSet) *Value[bool] { return &r.League.Dynasty }},
		field[CarryPolicy]{path: "league.carry", label: "Roster carry policy", access: func(r *RuleSet) *Value[CarryPolicy] { return &r.League.Carry }},
		field[int]{path: "league.keepers", label: "Keepers", access: func(r *RuleSet) *Value[int] { return &r.League.Keepers }},
	}
}

type RosterRules struct {
	Limits          Value[Bounds]
	ReserveExcluded Value[bool]
	PositionBounds  Value[map[Position]Bounds]
}

func legacyRoster() RosterRules {
	return RosterRules{
		Limits:          cited[Bounds](bounds(18, 80, "fact R"), "fact R"),
		ReserveExcluded: cited[bool](true, "fact R"),
		PositionBounds:  cited[map[Position]Bounds](map[Position]Bounds{}, "fact R"),
	}
}

func rosterFields() []ruleField {
	return []ruleField{
		field[Bounds]{path: "roster.limits", label: "Active roster size", access: func(r *RuleSet) *Value[Bounds] { return &r.Roster.Limits }},
		field[bool]{path: "roster.reserveExcluded", label: "Reserves excluded from roster size", access: func(r *RuleSet) *Value[bool] { return &r.Roster.ReserveExcluded }},
		field[map[Position]Bounds]{path: "roster.positionBounds", label: "Roster position limits", access: func(r *RuleSet) *Value[map[Position]Bounds] { return &r.Roster.PositionBounds }},
	}
}

type LineupRules struct {
	Positions      Value[map[Position]Bounds]
	Groups         Value[map[LineupGroup]Bounds]
	InvalidPenalty Value[Penalty]
	MissedFlag     Value[bool]
}

func legacyLineup() LineupRules {
	return LineupRules{
		Positions:      cited[map[Position]Bounds](legacyLineupBounds(), "Census CM-004 Q-A1"),
		Groups:         cited[map[LineupGroup]Bounds](legacyGroups(), "Census CM-004 Q-A1"),
		InvalidPenalty: cited[Penalty](ZeroTopStarter, "Rulebook §3"),
		MissedFlag:     cited[bool](true, "Census CM-004"),
	}
}

func lineupFields() []ruleField {
	return []ruleField{
		field[map[Position]Bounds]{path: "lineup.positions", label: "Starting position limits", access: func(r *RuleSet) *Value[map[Position]Bounds] { return &r.Lineup.Positions }},
		field[map[LineupGroup]Bounds]{path: "lineup.groups", label: "Starting group limits", access: func(r *RuleSet) *Value[map[LineupGroup]Bounds] { return &r.Lineup.Groups }},
		field[Penalty]{path: "lineup.invalidPenalty", label: "Invalid lineup penalty", access: func(r *RuleSet) *Value[Penalty] { return &r.Lineup.InvalidPenalty }},
		field[bool]{path: "lineup.missedFlag", label: "Flag missed lineups", access: func(r *RuleSet) *Value[bool] { return &r.Lineup.MissedFlag }},
	}
}

type StandingsRules struct {
	Criteria Value[[]Criterion]
}

func legacyStandings() StandingsRules {
	return StandingsRules{
		Criteria: cited[[]Criterion]([]Criterion{WinPercentage, HeadToHeadRecord, DivisionPercentage, ConferencePercentage, AllPlayPercentage, Points, PowerRank}, "Census CM-007"),
	}
}

func standingsFields() []ruleField {
	return []ruleField{
		field[[]Criterion]{path: "standings.criteria", label: "Standings tie breakers", access: func(r *RuleSet) *Value[[]Criterion] { return &r.Standings.Criteria }},
	}
}

type HouseRules struct {
	Texts Value[map[string]string]
}

func legacyHouse() HouseRules {
	return HouseRules{
		Texts: cited[map[string]string](map[string]string{"legacy-reference": "docs/league-rules/Official_Rulebook.md v2026 (read-only reference)"}, "Census CM-017"),
	}
}

func houseFields() []ruleField {
	return []ruleField{
		field[map[string]string]{path: "house.texts", label: "House rules", access: func(r *RuleSet) *Value[map[string]string] { return &r.House.Texts }},
	}
}

type IRRules struct {
	Slots            Value[int]
	Eligible         Value[[]NFLStatus]
	MustLeaveHealthy Value[bool]
	AutoReturn       AutoMoveValue `json:",omitzero"`
}

func legacyIR() IRRules {
	return IRRules{
		Slots:            cited[int](12, "fact 4a; Census CM-010"),
		Eligible:         cited[[]NFLStatus]([]NFLStatus{InjuredReserve, PUP, NFI, Out, Suspended}, "fact 4a"),
		MustLeaveHealthy: cited[bool](true, "fact 4a"),
		AutoReturn:       legacyAutoMove(),
	}
}

func irFields() []ruleField {
	return []ruleField{
		field[bool]{path: "ir.autoReturn", label: "Automatically return ineligible IR players", access: func(r *RuleSet) *Value[bool] { return (*Value[bool])(&r.IR.AutoReturn) }},
		field[int]{path: "ir.slots", label: "IR spots", access: func(r *RuleSet) *Value[int] { return &r.IR.Slots }},
		field[[]NFLStatus]{path: "ir.eligible", label: "IR eligible designations", access: func(r *RuleSet) *Value[[]NFLStatus] { return &r.IR.Eligible }},
		field[bool]{path: "ir.mustLeaveHealthy", label: "Leave IR when healthy", access: func(r *RuleSet) *Value[bool] { return &r.IR.MustLeaveHealthy }},
	}
}

type TaxiRules struct {
	Slots        Value[int]
	MaximumYears Value[int]
	Basis        Value[ExperienceBasis]
	AutoPromote  AutoMoveValue `json:",omitzero"`
}

func legacyTaxi() TaxiRules {
	return TaxiRules{
		AutoPromote:  legacyAutoMove(),
		Slots:        cited[int](8, "Rulebook §2"),
		MaximumYears: cited[int](3, "fact 4b"),
		Basis:        cited[ExperienceBasis](CalendarYears, "fact 4b"),
	}
}

func taxiFields() []ruleField {
	return []ruleField{
		field[bool]{path: "taxi.autoPromote", label: "Automatically promote ineligible taxi players", access: func(r *RuleSet) *Value[bool] { return (*Value[bool])(&r.Taxi.AutoPromote) }},
		field[int]{path: "taxi.slots", label: "Taxi spots", access: func(r *RuleSet) *Value[int] { return &r.Taxi.Slots }},
		field[int]{path: "taxi.maximumYears", label: "Taxi experience limit", access: func(r *RuleSet) *Value[int] { return &r.Taxi.MaximumYears }},
		field[ExperienceBasis]{path: "taxi.basis", label: "Taxi experience measure", access: func(r *RuleSet) *Value[ExperienceBasis] { return &r.Taxi.Basis }},
	}
}

// LegacyNFLForTeams adapts the template to a team count without guessing a smaller league's playoffs.
func LegacyNFLForTeams(count int) RuleSet {
	r := LegacyNFL()
	r.League.Teams = cited(count, "Legacy template: chosen team count")
	if count == 32 {
		return r
	}
	// Legacy divisions hold four teams; a smaller conference keeps that size with fewer divisions.
	perConference := max(1, count/2/4)
	r.League.Divisions = cited(2*perConference, "Legacy template: divisions of about four teams")
	// The playoff field is competitive: the app does not scale Legacy's seven per conference.
	r.Playoffs.PerConference = Needs[int]("Choose how many teams per conference make the playoffs in this smaller league.")
	r.Playoffs.DivisionWinners = Needs[int]("Choose the playoff field first, then how many division winners qualify.")
	r.Playoffs.WildCards = Needs[int]("Choose the playoff field first, then how many wild cards qualify.")
	r.Playoffs.ByeSeed = Needs[int]("Choose first-round byes for this smaller playoff field.")
	r.Playoffs.Graph = Needs[[]BracketNode]("Choose a bracket for this smaller playoff field.")
	r.Playoffs.FinalWeek = Needs[int]("Choose the final week after setting the smaller bracket.")
	if count < 4 {
		r.Trades.Review = Needs[ReviewRoute]("Fewer than four teams: choose who reviews trades.")
		r.Trades.DOTSeats = Needs[int]("Choose how many people review trades.")
	}
	r.Schedule.Pattern = Needs[SchedulePattern]("Choose how often teams meet in this smaller league.")
	r.Trades.MaximumParties = cited(min(3, count), "Legacy template: bounded by team count")
	return r
}

// NotifyRules are what the league tells its owners inside the app. Nothing is delivered outside it.
type NotifyRules struct {
	Transactions Value[bool]
	Reminders    Value[bool]
}

func legacyNotify() NotifyRules {
	return NotifyRules{
		Transactions: cited[bool](true, "Census CM-051"),
		Reminders:    cited[bool](true, "Census CM-051"),
	}
}

func notifyFields() []ruleField {
	return []ruleField{
		field[bool]{path: "notify.transactions", label: "Tell the league about transactions", access: func(r *RuleSet) *Value[bool] { return &r.Notify.Transactions }},
		field[bool]{path: "notify.reminders", label: "Remind owners of deadlines", access: func(r *RuleSet) *Value[bool] { return &r.Notify.Reminders }},
	}
}

// AutoMoveValue omits only the unchanged Legacy off default from rule-set JSON. Seeds and
// commissioner edits still encode explicit Value[bool] settings, including their citations.
type AutoMoveValue Value[bool]

func legacyAutoMove() AutoMoveValue {
	return AutoMoveValue(cited(false, "Automatic reserve moves off in Legacy"))
}

func (v AutoMoveValue) IsZero() bool { return v == legacyAutoMove() }

// UnmarshalJSON restores the omitted Legacy default; other values retain their recorded citation.
func (r *IRRules) UnmarshalJSON(body []byte) error {
	type plain IRRules
	value := plain{AutoReturn: legacyAutoMove()}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("read IR rules: %w", err)
	}
	*r = IRRules(value)
	return nil
}

func (r *TaxiRules) UnmarshalJSON(body []byte) error {
	type plain TaxiRules
	value := plain{AutoPromote: legacyAutoMove()}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("read taxi rules: %w", err)
	}
	*r = TaxiRules(value)
	return nil
}
