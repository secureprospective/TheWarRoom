package migrationtest

import (
	"fmt"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
)

// Spec is the shape of a synthetic MFL league.
type Spec struct {
	Teams int
	// Divisions lists each division's size; empty is a league without divisions.
	Divisions []int
	// Conferences is how many conferences the divisions are dealt into; 0 is none.
	Conferences int
	NoSalaries  bool
	NoContracts bool
	IDPHeavy    bool
	Superflex   bool
	NoKickers   bool
	OddScoring  bool
	NoReserves  bool
	// EmptyFranchise leaves the last franchise with no players.
	EmptyFranchise bool
	// NoEmail leaves the first owner without an email.
	NoEmail bool
	// MidSeason holds this season's draft results; otherwise the league is read in the offseason.
	MidSeason bool
	Players   int
}

// Specs are the edge leagues the move is proved on.
func Specs() map[string]Spec {
	return map[string]Spec{
		"8 teams":                  {Teams: 8, Divisions: []int{4, 4}, Conferences: 1, MidSeason: true},
		"10 teams odd divisions":   {Teams: 10, Divisions: []int{3, 3, 4}, Conferences: 1},
		"12 teams no divisions":    {Teams: 12},
		"14 teams superflex":       {Teams: 14, Divisions: []int{7, 7}, Superflex: true, MidSeason: true},
		"16 teams IDP heavy":       {Teams: 16, Divisions: []int{4, 4, 4, 4}, Conferences: 2, IDPHeavy: true},
		"20 teams cap off":         {Teams: 20, Divisions: []int{5, 5, 5, 5}, NoSalaries: true, NoContracts: true},
		"32 teams":                 {Teams: 32, Divisions: []int{4, 4, 4, 4, 4, 4, 4, 4}, Conferences: 2, MidSeason: true},
		"kickers off odd scoring":  {Teams: 10, OddScoring: true, NoKickers: true},
		"taxi and IR off":          {Teams: 12, Divisions: []int{6, 6}, NoReserves: true},
		"empty franchise no email": {Teams: 8, EmptyFranchise: true, NoEmail: true, MidSeason: true},
	}
}

// Synthetic builds an MFL league from a spec, every value written as MFL writes it.
func Synthetic(name string, s Spec) leaguemigration.Export {
	players := s.Players
	if players == 0 {
		players = 6
	}
	config := league.RawConfig{Source: "mfl:2026", CurrentSeason: 2026, KeeperType: "dynasty",
		RosterSize: "30", TaxiSquad: "4", InjuredReserve: "3", UsesSalaries: "1", SalaryCapAmount: "200",
		UsesContractYear: "1", StartWeek: "1", EndWeek: "17", LastRegularSeasonWeek: "14",
		IncludeIRWithSalary: "50", IncludeTaxiWithSalary: "0", StandingsSort: "PCT,PTS,H2H,",
		RosterLimits: []league.PositionLimit{{Name: "QB", Limit: "0-4"}, {Name: "RB", Limit: "0-0"}}}
	if s.NoSalaries {
		config.UsesSalaries, config.SalaryCapAmount = "0", ""
	}
	if s.NoReserves {
		config.TaxiSquad, config.InjuredReserve = "0", "0"
	}
	config.Starters, config.ScoringRules = lineup(s), scoring(s)
	config.Conferences, config.Divisions = alignment(s)
	out := leaguemigration.Export{LeagueID: fmt.Sprintf("9%04d", s.Teams), Name: name, Season: 2026,
		Config: config, ReadAt: ReadAt, Players: map[string]string{}}
	franchise := 0
	for d, size := range append([]int{}, s.Divisions...) {
		for range size {
			franchise++
			config.Franchises = append(config.Franchises, team(franchise, fmt.Sprintf("%02d", d)))
		}
	}
	for franchise < s.Teams {
		franchise++
		config.Franchises = append(config.Franchises, team(franchise, ""))
	}
	out.Config.Franchises = config.Franchises
	out.Rosters = rosters(s, players)
	for _, r := range out.Rosters {
		out.Players[r.Player] = "Player " + r.Player
	}
	out.Draft, out.Future = picks(s)
	for _, p := range out.Draft {
		if p.Player != nil {
			out.Players[p.Player.String()] = "Rookie " + p.Player.String()
		}
	}
	for i := 1; i <= s.Teams; i++ {
		owner := leaguemigration.Owner{Franchise: fid(i), Name: fmt.Sprintf("Owner %d", i),
			Email: fmt.Sprintf("owner%d@example.com", i)}
		if s.NoEmail && i == 1 {
			owner.Email = ""
		}
		out.Owners = append(out.Owners, owner)
	}
	out.Adjustments = []leaguemigration.Adjustment{{ID: "adj-1", Franchise: fid(2), Amount: "1.5", Description: "Dead money"},
		{ID: "adj-2", Franchise: fid(1), Amount: "-0.25", Description: "Cap credit"}}
	if s.NoSalaries {
		out.Adjustments = nil
	}
	out.Champions = []leaguemigration.Champion{{Season: 2025, Franchise: fid(3)}}
	return out
}

func fid(i int) string { return fmt.Sprintf("%04d", i) }

func team(i int, division string) league.Franchise {
	return league.Franchise{ID: fid(i), Name: fmt.Sprintf("Team %d", i), Abbrev: fmt.Sprintf("T%d", i), Division: division}
}

func alignment(s Spec) ([]league.Conference, []league.Division) {
	var conferences []league.Conference
	for c := range s.Conferences {
		conferences = append(conferences, league.Conference{ID: fmt.Sprintf("%02d", c), Name: fmt.Sprintf("Conference %d", c+1)})
	}
	var divisions []league.Division
	for d := range s.Divisions {
		division := league.Division{ID: fmt.Sprintf("%02d", d), Name: fmt.Sprintf("Division %d", d+1)}
		if s.Conferences > 0 {
			division.Conference = fmt.Sprintf("%02d", d%s.Conferences)
		}
		divisions = append(divisions, division)
	}
	return conferences, divisions
}

func lineup(s Spec) league.Starters {
	starters := league.Starters{Count: "10", IOPStarters: "7", IDPStarters: "3", Positions: []league.PositionLimit{
		{Name: "QB", Limit: "1"}, {Name: "RB", Limit: "2-3"}, {Name: "WR", Limit: "2-4"}, {Name: "TE", Limit: "1-2"},
		{Name: "PK", Limit: "1"}, {Name: "LB", Limit: "1-3"}, {Name: "CB", Limit: "1-2"}}}
	if s.Superflex {
		starters.Positions[0].Limit = "1-2"
	}
	if s.NoKickers {
		starters.Positions = append(starters.Positions[:4], starters.Positions[5:]...)
		starters.Count, starters.IOPStarters = "9", "6"
	}
	if s.IDPHeavy {
		starters.Count, starters.IDPStarters = "18", "11"
		starters.Positions = append(starters.Positions, league.PositionLimit{Name: "DT", Limit: "2-3"},
			league.PositionLimit{Name: "DE", Limit: "2-3"}, league.PositionLimit{Name: "S", Limit: "2-3"})
	}
	return starters
}

func scoring(s Spec) []league.PositionRuleSet {
	rules := []league.PositionRuleSet{{Positions: "QB|RB|WR|TE", Rules: []league.ScoringRule{
		{Event: "PY", Points: "*.04", Range: "-50-999"}, {Event: "#P", Points: "4", Range: "0-10"},
		{Event: "RY", Points: "*.1", Range: "-50-999"}, {Event: "CC", Points: "*1", Range: "0-99"}}}}
	if !s.NoKickers {
		rules = append(rules, league.PositionRuleSet{Positions: "PK", Rules: []league.ScoringRule{
			{Event: "EP", Points: "1", Range: "0-10"}}})
	}
	rules = append(rules, league.PositionRuleSet{Positions: "DT|DE|LB|CB|S", Rules: []league.ScoringRule{
		{Event: "TK", Points: "*1.5", Range: "0-99"}}})
	if s.OddScoring {
		rules = append(rules, league.PositionRuleSet{Positions: "QB", Rules: []league.ScoringRule{
			{Event: "IN", Points: "-2.5", Range: "0-10"}, {Event: "PY", Points: "3", Range: "300-399"},
			{Event: "PY", Points: "5", Range: "400-999"}, {Event: "ZZQ", Points: "1", Range: "0-10"}}},
			league.PositionRuleSet{Positions: "TE", Rules: []league.ScoringRule{{Event: "CC", Points: "*.5", Range: "0-99"}}})
	}
	return rules
}

func rosters(s Spec, players int) []leaguemigration.Rostered {
	statuses := []string{"ROSTER", "ROSTER", "INJURED_RESERVE", "TAXI_SQUAD"}
	if s.NoReserves {
		statuses = []string{"ROSTER"}
	}
	var out []leaguemigration.Rostered
	for f := 1; f <= s.Teams; f++ {
		if s.EmptyFranchise && f == s.Teams {
			continue
		}
		for p := range players {
			// Leading zeros are part of an MFL id: 0007 and 7 are different players.
			id := fmt.Sprintf("%04d", f*100+p)
			r := leaguemigration.Rostered{Franchise: fid(f), Player: id, Status: statuses[p%len(statuses)],
				Salary: strconv.FormatFloat(0.5+float64(p)*1.25, 'f', 2, 64), ContractYear: strconv.Itoa(2026 + p%4),
				ContractStatus: "UFA", ContractInfo: fmt.Sprintf("Signed %d", 2023+p%3)}
			if s.NoSalaries {
				r.Salary = ""
			}
			if s.NoContracts {
				r.ContractYear, r.ContractStatus, r.ContractInfo = "", "", ""
			}
			out = append(out, r)
		}
	}
	return out
}

func picks(s Spec) ([]draftreports.Pick, []draftreports.FuturePick) {
	var draft []draftreports.Pick
	if s.MidSeason {
		for slot := 1; slot <= s.Teams; slot++ {
			pick := draftreports.Pick{Round: 1, Slot: slot, Franchise: fid(s.Teams + 1 - slot)}
			if slot%3 == 0 {
				pick.Skipped = true
			} else {
				pick.Player = pid(fmt.Sprintf("%05d", 90000+slot))
			}
			draft = append(draft, pick)
		}
	}
	var future []draftreports.FuturePick
	for year := 2027; year <= 2028; year++ {
		for round := 1; round <= 2; round++ {
			for f := 1; f <= s.Teams; f++ {
				owner := f
				if f == 1 && round == 1 {
					owner = 2 // a traded pick: original franchise 1, owned by franchise 2
				}
				future = append(future, draftreports.FuturePick{Year: year, Round: round, Original: fid(f), Owner: fid(owner)})
			}
		}
	}
	return draft, future
}
