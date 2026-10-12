package facts

import (
	"fmt"
	"strings"
)

func (c *compiler) league(year int64) error {
	src := seasonFile(year, "league")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	lg := d.get("league")
	if lg.Object == nil {
		return nil
	}
	id, err := requiredID(lg, "id", src)
	if err != nil {
		return err
	}
	c.facts.Season = append(c.facts.Season, Season{
		Year: year, LeagueID: id, Name: lg.get("name").str(), StartWeek: lg.get("startWeek").whole(),
		EndWeek: lg.get("endWeek").whole(), LastRegularWeek: lg.get("lastRegularSeasonWeek").whole(),
		SalaryCap: lg.get("salaryCapAmount").num(), RosterSize: lg.get("rosterSize").whole(),
		TaxiSquad: ptr(lg.get("taxiSquad").dump()), InjuredReserve: ptr(lg.get("injuredReserve").dump()),
		Starters: ptr(lg.get("starters").dump()), Raw: lg.dump(), Src: src,
	})
	divs := indexRows(lg.get("divisions", "division").rows())
	confs := indexRows(lg.get("conferences", "conference").rows())
	for _, f := range lg.get("franchises", "franchise").rows() {
		fid, err := requiredID(f, "id", src)
		if err != nil {
			return err
		}
		div := divs[value(f.get("division").str())]
		conf := confs[value(div.get("conference").str())]
		c.facts.FranchiseSeason = append(c.facts.FranchiseSeason, FranchiseSeason{
			Year: year, Fid: fid, Name: f.get("name").str(), Abbrev: f.get("abbrev").str(), Division: f.get("division").str(),
			DivisionName: div.get("name").str(), Conference: div.get("conference").str(), ConferenceName: conf.get("name").str(), Src: src,
		})
	}
	return nil
}

func indexRows(rows []RawValue) map[string]RawValue {
	out := make(map[string]RawValue, len(rows))
	for _, row := range rows {
		out[value(row.get("id").str())] = row
	}
	return out
}

func (c *compiler) standings(year int64) error {
	src := seasonFile(year, "leagueStandings")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, f := range d.get("leagueStandings", "franchise").rows() {
		fid, err := requiredID(f, "id", src)
		if err != nil {
			return err
		}
		ap := allPlay(f.get("all_play_wlt"))
		c.facts.Standings = append(c.facts.Standings, Standings{
			Year: year, Fid: fid, H2hW: f.get("h2hw").whole(), H2hL: f.get("h2hl").whole(), H2hT: f.get("h2ht").whole(),
			Pf: f.get("pf").num(), Pa: f.get("pa").num(), Pp: f.get("pp").num(), Eff: f.get("eff").num(),
			AllPlayW: ap[0], AllPlayL: ap[1], AllPlayT: ap[2], Raw: f.dump(), Src: src,
		})
	}
	return nil
}

func allPlay(r RawValue) [3]*int64 {
	var out [3]*int64
	parts := strings.Split(value(r.str()), "-")
	if len(parts) == 3 {
		for i, part := range parts {
			out[i] = (RawValue{Scalar: &part}).whole()
		}
	}
	return out
}

func (c *compiler) rules(year int64) error {
	src := seasonFile(year, "rules")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, group := range d.get("rules", "positionRules").rows() {
		positions, err := requiredID(group, "positions", src)
		if err != nil {
			return err
		}
		for _, r := range group.get("rule").rows() {
			event := r.get("event").text()
			if event == nil {
				return fmt.Errorf("%s: missing scoring event", src)
			}
			c.facts.ScoringRule = append(c.facts.ScoringRule, ScoringRule{
				Year: year, Positions: positions, Event: *event, Range: r.get("range").text(), Points: r.get("points").text(), Src: src,
			})
		}
	}
	return nil
}

func (c *compiler) ruleCodes() error {
	src := "allRules.json"
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, r := range d.get("allRules", "rule").rows() {
		c.facts.RuleCode = append(c.facts.RuleCode, RuleCode{
			Code: value(r.get("abbreviation").text()), Short: r.get("shortDescription").text(), Description: r.get("detailedDescription").text(), Src: src,
		})
	}
	return nil
}

func (c *compiler) calendar(year int64) error {
	src := seasonFile(year, "calendar")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, e := range d.get("calendar", "event").rows() {
		id, err := requiredID(e, "id", src)
		if err != nil {
			return err
		}
		c.facts.CalendarEvent = append(c.facts.CalendarEvent, CalendarEvent{
			Year: year, EventID: id, Type: e.get("type").str(), Title: nonempty(e.get("title").str()),
			StartTs: e.get("start_time").whole(), EndTs: e.get("end_time").whole(), Src: src,
		})
	}
	return nil
}
