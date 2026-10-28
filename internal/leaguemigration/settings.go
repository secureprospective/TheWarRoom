package leaguemigration

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// citation is where every value the import carries came from, as the rules screen shows it.
const citation = "Your MFL league"

// notHeld is the citation of a setting MFL does not hold: it waits for the commissioner's ruling.
const notHeld = "Your MFL league does not hold this setting"

// carried are the settings the app counts or carries from the starting snapshot itself (the franchises,
// their divisions and conferences), so the translator leaves them to it and the review reads them from
// the franchises instead.
func carried() map[leaguerules.FieldPath]bool {
	return map[leaguerules.FieldPath]bool{"league.teams": true, "league.divisions": true, "league.alignment": true,
		"league.divisionConferences": true, "league.sides": true}
}

// Settings carries an MFL league's settings into the app value for value. What MFL holds is set as MFL
// holds it; what it does not hold waits for a ruling and is listed; what MFL holds but the app cannot
// read is a problem, never a guess.
func Settings(config league.RawConfig) (leaguerules.RuleSet, []string, []string) {
	t := translation{}
	r := leaguerules.RuleSet{}
	t.league(&r, config)
	t.roster(&r, config)
	t.lineup(&r, config)
	t.scoring(&r, config)
	t.standings(&r, config)
	t.cap(&r, config)
	if weeks, ok := t.regularWeeks(config); ok {
		r.Schedule.RegularWeeks = known(weeks)
	}
	t.waiting(r)
	return r, t.problems, t.notes
}

type translation struct{ problems, notes []string }

func (t *translation) problem(format string, args ...any) {
	t.problems = append(t.problems, fmt.Sprintf(format, args...))
}

func known[T any](v T) leaguerules.Value[T] {
	return leaguerules.Value[T]{State: leaguerules.IsKnown, Value: v, Citation: citation}
}

func (t *translation) number(what, raw string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 {
		t.problem("MFL's %s reads %q, which is not a whole number", what, raw)
		return 0, false
	}
	return n, true
}

func (t *translation) league(r *leaguerules.RuleSet, config league.RawConfig) {
	if n := len(config.Conferences); n > 0 {
		r.League.Conferences = known(n)
	}
	switch strings.ToLower(strings.TrimSpace(config.KeeperType)) {
	case "dynasty":
		r.League.Dynasty = known(true)
	case "redraft", "none":
		r.League.Dynasty = known(false)
	}
	r.League.Pools = known(leaguerules.OnePool)
}

func (t *translation) roster(r *leaguerules.RuleSet, config league.RawConfig) {
	if strings.TrimSpace(config.RosterSize) != "" {
		if size, ok := t.number("roster size", config.RosterSize); ok {
			r.Roster.Limits = known(leaguerules.Bounds{
				Min: leaguerules.Needs[int](notHeld + ": a roster minimum"), Max: known(size)})
			// MFL counts IR and the taxi squad apart from the roster size.
			r.Roster.ReserveExcluded = known(true)
		}
	}
	if limits, ok := t.positionBounds("roster limit", config.RosterLimits, true); ok {
		r.Roster.PositionBounds = known(limits)
	}
	if strings.TrimSpace(config.InjuredReserve) != "" {
		if n, ok := t.number("IR size", config.InjuredReserve); ok {
			r.IR.Slots = known(n)
		}
	}
	if strings.TrimSpace(config.TaxiSquad) != "" {
		if n, ok := t.number("taxi squad size", config.TaxiSquad); ok {
			r.Taxi.Slots = known(n)
		}
	}
}

func (t *translation) lineup(r *leaguerules.RuleSet, config league.RawConfig) {
	if positions, ok := t.positionBounds("starting limit", config.Starters.Positions, false); ok {
		r.Lineup.Positions = known(positions)
	}
	groups := map[leaguerules.LineupGroup]leaguerules.Bounds{}
	for _, g := range []struct {
		group leaguerules.LineupGroup
		raw   string
	}{{leaguerules.Total, config.Starters.Count}, {leaguerules.Offense, config.Starters.IOPStarters},
		{leaguerules.Defense, config.Starters.IDPStarters}} {
		if strings.TrimSpace(g.raw) == "" {
			continue
		}
		if n, ok := t.number("starter count", g.raw); ok {
			groups[g.group] = leaguerules.Bounds{Min: known(n), Max: known(n)}
		}
	}
	if len(groups) > 0 {
		r.Lineup.Groups = known(groups)
	}
}

// positionBounds reads MFL's per-position limits ("1", "2-4"). In roster limits "0-0" is MFL's word for
// no limit, so the position is left out; in starters it means none may start.
func (t *translation) positionBounds(what string, limits []league.PositionLimit,
	zeroIsOpen bool) (map[leaguerules.Position]leaguerules.Bounds, bool) {
	if len(limits) == 0 {
		return nil, false
	}
	out := map[leaguerules.Position]leaguerules.Bounds{}
	whole := true
	for _, l := range limits {
		position, ok := position(l.Name)
		if !ok {
			t.problem("MFL's %s names the position %q, which the app does not hold yet", what, l.Name)
			whole = false
			continue
		}
		low, high, ok := limitRange(l.Limit)
		if !ok {
			t.problem("MFL's %s for %s reads %q, which is not a number or a range", what, l.Name, l.Limit)
			whole = false
			continue
		}
		if zeroIsOpen && low == 0 && high == 0 {
			continue
		}
		out[position] = leaguerules.Bounds{Min: known(low), Max: known(high)}
	}
	return out, whole
}

func position(raw string) (leaguerules.Position, bool) {
	p := leaguerules.Position(strings.TrimSpace(raw))
	switch p {
	case leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.TE, leaguerules.PK, leaguerules.DT,
		leaguerules.DE, leaguerules.LB, leaguerules.CB, leaguerules.S:
		return p, true
	case leaguerules.OL:
		return p, false
	}
	return p, false
}

func limitRange(raw string) (int, int, bool) {
	low, high, ranged := strings.Cut(strings.TrimSpace(raw), "-")
	a, err := strconv.Atoi(low)
	if err != nil || a < 0 {
		return 0, 0, false
	}
	if !ranged {
		return a, a, true
	}
	b, err := strconv.Atoi(high)
	if err != nil || b < a {
		return 0, 0, false
	}
	return a, b, true
}

func (t *translation) scoring(r *leaguerules.RuleSet, config league.RawConfig) {
	blocks := make([]leaguerules.MFLRuleBlock, 0, len(config.ScoringRules))
	for _, b := range config.ScoringRules {
		block := leaguerules.MFLRuleBlock{Positions: b.Positions}
		for _, rule := range b.Rules {
			block.Rules = append(block.Rules, leaguerules.MFLRule{Event: rule.Event, Points: rule.Points, Range: rule.Range})
			if _, ok := leaguerules.StatByMFL(strings.TrimSpace(rule.Event)); !ok {
				t.notes = append(t.notes, fmt.Sprintf("MFL scores the event %q for %s, which the app does not read "+
					"yet: the rule is kept exactly as MFL holds it, and scores nothing until the app reads that event",
					rule.Event, b.Positions))
			}
		}
		blocks = append(blocks, block)
	}
	scoring, err := leaguerules.FromMFLScoring(blocks, citation)
	if err != nil {
		t.problem("MFL's scoring could not be read: %v", err)
		return
	}
	r.Scoring = scoring
}

func (t *translation) standings(r *leaguerules.RuleSet, config league.RawConfig) {
	codes := map[string]leaguerules.Criterion{"PCT": leaguerules.WinPercentage, "H2H": leaguerules.HeadToHeadRecord,
		"DIVPCT": leaguerules.DivisionPercentage, "CONFPCT": leaguerules.ConferencePercentage,
		"ALL_PLAY_PCT": leaguerules.AllPlayPercentage, "PTS": leaguerules.Points, "PWR": leaguerules.PowerRank}
	var criteria []leaguerules.Criterion
	for _, code := range strings.Split(config.StandingsSort, ",") {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		c, ok := codes[code]
		if !ok {
			t.notes = append(t.notes, fmt.Sprintf("MFL breaks standings ties by %q, which the app does not hold "+
				"yet: the tie breakers wait for your ruling", code))
			return
		}
		criteria = append(criteria, c)
	}
	if len(criteria) > 0 {
		r.Standings.Criteria = known(criteria)
	}
}

func (t *translation) cap(r *leaguerules.RuleSet, config league.RawConfig) {
	if strings.TrimSpace(config.UsesSalaries) != "1" {
		t.notes = append(t.notes, "Your MFL league runs without salaries: the salary cap waits for your ruling")
		return
	}
	amount, err := domain.ParseMoneyMillions(config.SalaryCapAmount)
	if err != nil {
		t.problem("MFL's salary cap reads %q, which is not an amount", config.SalaryCapAmount)
	} else {
		r.Cap.Amount = known(leaguerules.Money(amount))
	}
	for _, share := range []struct {
		what  string
		raw   string
		value *leaguerules.Value[leaguerules.Decimal]
	}{{"IR salary share", config.IncludeIRWithSalary, &r.Cap.IRShare},
		{"taxi salary share", config.IncludeTaxiWithSalary, &r.Cap.TaxiShare}} {
		if strings.TrimSpace(share.raw) == "" {
			continue
		}
		if n, ok := t.number(share.what, share.raw); ok && n <= 100 {
			*share.value = known(leaguerules.Decimal(n * 100))
		} else if ok {
			t.problem("MFL's %s reads %q, which is more than 100 percent", share.what, share.raw)
		}
	}
}

func (t *translation) regularWeeks(config league.RawConfig) (int, bool) {
	if strings.TrimSpace(config.StartWeek) == "" || strings.TrimSpace(config.LastRegularSeasonWeek) == "" {
		return 0, false
	}
	start, ok := t.number("first week", config.StartWeek)
	last, ok2 := t.number("last regular season week", config.LastRegularSeasonWeek)
	if !ok || !ok2 {
		return 0, false
	}
	if last < start {
		t.problem("MFL's regular season ends in week %d, before it starts in week %d", last, start)
		return 0, false
	}
	return last - start + 1, true
}

// waiting lists the settings MFL does not hold, by their labels on the rules screen.
func (t *translation) waiting(r leaguerules.RuleSet) {
	skip := carried()
	var labels []string
	for _, c := range leaguerules.SeedChanges(r, "", 0, epoch) {
		if skip[c.Field] || leaguerules.DescribeField(c.Field, c.New).State == "Set" {
			continue
		}
		if f, ok := leaguerules.Field(c.Field); ok {
			labels = append(labels, f.Label)
		}
	}
	if len(labels) > 0 {
		t.notes = append(t.notes, fmt.Sprintf("Your MFL league does not hold %d settings the app runs; "+
			"each waits for your ruling on the rules screen: %s", len(labels), strings.Join(labels, ", ")))
	}
}
