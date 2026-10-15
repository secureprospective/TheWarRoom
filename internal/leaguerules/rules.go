package leaguerules

type RuleSet struct {
	LeagueID  string
	Season    int
	League    LeagueRules
	Roster    RosterRules
	Lineup    LineupRules
	Scoring   ScoringRules
	Standings StandingsRules
	Cap       CapRules
	Contracts ContractsRules
	IR        IRRules
	Taxi      TaxiRules
	Waivers   WaiversRules
	Bids      BidsRules
	Trades    TradesRules
	Schedule  ScheduleRules
	Playoffs  PlayoffsRules
	House     HouseRules
	Calendar  CalendarRules
	Draft     DraftRules
	Notify    NotifyRules
}

func LegacyNFL() RuleSet {
	return RuleSet{
		League:    legacyLeague(),
		Roster:    legacyRoster(),
		Lineup:    legacyLineup(),
		Scoring:   legacyScoring(),
		Standings: legacyStandings(),
		Cap:       legacyCap(),
		Contracts: legacyContracts(),
		IR:        legacyIR(),
		Taxi:      legacyTaxi(),
		Waivers:   legacyWaivers(),
		Bids:      legacyBids(),
		Trades:    legacyTrades(),
		Schedule:  legacySchedule(),
		Playoffs:  legacyPlayoffs(),
		House:     legacyHouse(),
		Calendar:  legacyCalendar(),
		Draft:     legacyDraft(),
		Notify:    legacyNotify(),
	}
}

func ruleFields() []ruleField {
	var result []ruleField
	result = append(result, leagueFields()...)
	result = append(result, rosterFields()...)
	result = append(result, lineupFields()...)
	result = append(result, scoringFields()...)
	result = append(result, standingsFields()...)
	result = append(result, capFields()...)
	result = append(result, contractsFields()...)
	result = append(result, irFields()...)
	result = append(result, taxiFields()...)
	result = append(result, waiversFields()...)
	result = append(result, bidsFields()...)
	result = append(result, tradesFields()...)
	result = append(result, scheduleFields()...)
	result = append(result, playoffsFields()...)
	result = append(result, houseFields()...)
	result = append(result, calendarFields()...)
	result = append(result, draftFields()...)
	result = append(result, notifyFields()...)
	return result
}
