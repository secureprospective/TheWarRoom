package facts

import (
	"fmt"
	"strings"
)

// seasonWeeks are the weeks the archive holds weekly files for.
const seasonWeeks = 17

func weeklyFile(year int64, name string, week int) string {
	return fmt.Sprintf("%d/weekly/%s_W%02d.json", year, name, week)
}

// results reads weeklyResults into game and lineup rows. Teams MFL scored without a head-to-head
// game become games with no opponent; MFL's BYE stand-in and its 2018 standalone copies are skipped
// and counted in load_note.
func (c *compiler) results(year int64) error {
	src := seasonFile(year, "weeklyResults_WYTD")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	var dupes, byes int64
	for _, wk := range d.get("allWeeklyResults", "weeklyResults").rows() {
		week := wk.get("week").whole()
		if week == nil {
			return fmt.Errorf("%s: week without a number", src)
		}
		sides, copies, err := weekSides(wk, src)
		if err != nil {
			return err
		}
		dupes += copies
		for _, teams := range sides {
			for i, f := range teams {
				fid, _ := keyID(f, "id", src) // weekSides proved every id
				if fid == "BYE" {
					byes++
					continue
				}
				var opp *string
				if len(teams) == 2 {
					if other, _ := keyID(teams[1-i], "id", src); other != "BYE" {
						opp = ptr(other)
					}
				}
				c.facts.Game = append(c.facts.Game, Game{
					Year: year, Week: *week, Fid: fid, Opp: opp, IsHome: f.get("isHome").whole(), Score: f.get("score").num(),
					Result: nonempty(f.get("result").str()), OptPts: f.get("opt_pts").num(), Src: src,
				})
				if err := c.lineup(year, *week, fid, f, src); err != nil {
					return err
				}
			}
		}
	}
	c.addNote("results", src, "standalone copy of a team already in a game that week: identical score, skipped", dupes)
	c.addNote("results", src, "MFL's BYE stand-in team: not a franchise, skipped", byes)
	return nil
}

// weekSides returns each head-to-head game's teams, then each team scored alone that week. A
// standalone copy of a team already in a game must carry the same score; it is skipped and counted.
func weekSides(wk RawValue, src string) ([][]RawValue, int64, error) {
	var sides [][]RawValue
	scores := map[string]*float64{}
	for _, m := range wk.get("matchup").rows() {
		teams := m.get("franchise").rows()
		for _, f := range teams {
			id, err := keyID(f, "id", src)
			if err != nil {
				return nil, 0, err
			}
			if _, seen := scores[id]; !seen {
				scores[id] = f.get("score").num()
			}
		}
		sides = append(sides, teams)
	}
	var copies int64
	for _, f := range wk.get("franchise").rows() {
		id, err := keyID(f, "id", src)
		if err != nil {
			return nil, 0, err
		}
		if score, inGame := scores[id]; inGame {
			if !sameNumber(score, f.get("score").num()) {
				return nil, 0, fmt.Errorf("%s: standalone copy of %s differs from its game", src, id)
			}
			copies++
			continue
		}
		sides = append(sides, []RawValue{f})
	}
	return sides, copies, nil
}

// lineup records every player a franchise started, benched or listed that week.
func (c *compiler) lineup(year, week int64, fid string, f RawValue, src string) error {
	started := idSet(f.get("starters").str())
	pids := idSet(f.get("nonstarters").str())
	for pid := range started {
		pids[pid] = true
	}
	listed := map[string]RawValue{}
	for _, p := range f.get("player").rows() {
		pid, err := keyID(p, "id", src)
		if err != nil {
			return err
		}
		listed[pid] = p
		pids[pid] = true
	}
	for pid := range pids {
		p := listed[pid]
		status := value(p.get("status").str())
		var start int64
		if status == "starter" || (status == "" && started[pid]) {
			start = 1
		}
		c.facts.Lineup = append(c.facts.Lineup, Lineup{
			Year: year, Week: week, Fid: fid, Pid: pid, Started: start, ShouldStart: p.get("shouldStart").whole(),
			Score: p.get("score").num(), Src: src,
		})
	}
	return nil
}

// idSet reads MFL's comma-separated id list.
func idSet(s *string) map[string]bool {
	out := map[string]bool{}
	for _, id := range strings.Split(value(s), ",") {
		if id = strings.TrimSpace(id); id != "" {
			out[id] = true
		}
	}
	return out
}

// playerWeeks reads each week's player scores and projections, injury report and NFL schedule.
func (c *compiler) playerWeeks(year int64) error {
	for week := 1; week <= seasonWeeks; week++ {
		scores, err := c.weekScores(weeklyFile(year, "playerScores", week), "playerScores")
		if err != nil {
			return err
		}
		projected, err := c.weekScores(weeklyFile(year, "projectedScores", week), "projectedScores")
		if err != nil {
			return err
		}
		for pid := range projected {
			if _, exists := scores[pid]; !exists {
				scores[pid] = nil
			}
		}
		for pid, score := range scores {
			c.facts.PlayerWeek = append(c.facts.PlayerWeek, PlayerWeek{
				Year: year, Week: int64(week), Pid: pid, Score: score, Projected: projected[pid],
			})
		}
		if err := c.injuries(year, week); err != nil {
			return err
		}
		if err := c.nflSchedule(year, week); err != nil {
			return err
		}
	}
	return nil
}

// weekScores maps player id to score; a later row for the same player replaces an earlier one.
func (c *compiler) weekScores(src, root string) (map[string]*float64, error) {
	d, err := c.read(src)
	if err != nil {
		return nil, err
	}
	out := map[string]*float64{}
	for _, s := range d.get(root, "playerScore").rows() {
		pid, err := keyID(s, "id", src)
		if err != nil {
			return nil, err
		}
		out[pid] = s.get("score").num()
	}
	return out, nil
}

func (c *compiler) injuries(year int64, week int) error {
	src := weeklyFile(year, "injuries", week)
	d, err := c.read(src)
	if err != nil {
		return err
	}
	byPlayer := map[string]RawValue{}
	for _, x := range d.get("injuries", "injury").rows() {
		pid, err := keyID(x, "id", src)
		if err != nil {
			return err
		}
		byPlayer[pid] = x
	}
	for pid, x := range byPlayer {
		c.facts.InjuryWeek = append(c.facts.InjuryWeek, InjuryWeek{
			Year: year, Week: int64(week), Pid: pid, Status: x.get("status").str(), Details: nonempty(x.get("details").str()),
			ExpReturn: nonempty(x.get("exp_return").str()), Src: src,
		})
	}
	return nil
}

func (c *compiler) nflSchedule(year int64, week int) error {
	src := weeklyFile(year, "nflSchedule", week)
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, m := range d.get("nflSchedule", "matchup").rows() {
		teams := m.get("team").rows()
		for k, t := range teams {
			team, err := keyID(t, "id", src)
			if err != nil {
				return err
			}
			var opp *string
			if len(teams) == 2 {
				other, err := keyID(teams[1-k], "id", src)
				if err != nil {
					return err
				}
				opp = ptr(other)
			}
			c.facts.NflGame = append(c.facts.NflGame, NflGame{
				Year: year, Week: int64(week), Team: team, Opp: opp, IsHome: t.get("isHome").whole(),
				Score: t.get("score").whole(), Spread: t.get("spread").num(), Kickoff: m.get("kickoff").whole(), Src: src,
			})
		}
	}
	return nil
}

// rosters reads MFL's stored weekly rosters and its end-of-season roster (week 0).
func (c *compiler) rosters(year int64) error {
	for week := 1; week <= seasonWeeks; week++ {
		if err := c.rosterWeek(year, int64(week), weeklyFile(year, "rosters", week)); err != nil {
			return err
		}
	}
	return c.rosterWeek(year, 0, seasonFile(year, "rosters"))
}

// rosterWeek keeps one row per franchise and player. MFL lists a player on IR or taxi twice, once
// as ROSTER and once with the status; the specific status is the fact.
func (c *compiler) rosterWeek(year, week int64, src string) error {
	d, err := c.read(src)
	if err != nil {
		return err
	}
	rows := map[[2]string]RosterWeek{}
	var doubled int64
	for _, f := range d.get("rosters", "franchise").rows() {
		fid, err := keyID(f, "id", src)
		if err != nil {
			return err
		}
		for _, p := range f.get("player").rows() {
			pid, err := keyID(p, "id", src)
			if err != nil {
				return err
			}
			key := [2]string{fid, pid}
			if _, seen := rows[key]; seen {
				doubled++
				if value(p.get("status").str()) == "ROSTER" {
					continue
				}
			}
			rows[key] = RosterWeek{
				Year: year, Week: week, Fid: fid, Pid: pid, Status: p.get("status").str(), Salary: p.get("salary").num(),
				ContractYear: trimmed(p.get("contractYear")), ContractStatus: trimmed(p.get("contractStatus")),
				ContractInfo: trimmed(p.get("contractInfo")), Src: src,
			}
		}
	}
	for _, row := range rows {
		c.facts.RosterWeek = append(c.facts.RosterWeek, row)
	}
	c.addNote("rosters", src, "player listed twice on one roster (ROSTER and IR or taxi): kept once, with the specific status", doubled)
	return nil
}

// trimmed is a text field with surrounding space removed; blank is null.
func trimmed(r RawValue) *string {
	if s := r.str(); s != nil {
		return nonempty(ptr(strings.TrimSpace(*s)))
	}
	return nil
}

func (c *compiler) pointsAllowed(year int64) error {
	src := seasonFile(year, "pointsAllowed")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, t := range d.get("pointsAllowed", "team").rows() {
		team, err := keyID(t, "id", src)
		if err != nil {
			return err
		}
		for _, p := range t.get("position").rows() {
			position, err := keyID(p, "name", src)
			if err != nil {
				return err
			}
			c.facts.PointsAllowed = append(c.facts.PointsAllowed, PointsAllowed{
				Year: year, NflTeam: team, Position: position, Points: p.get("points").num(), Src: src,
			})
		}
	}
	return nil
}

// playoffs reads the season's brackets, then each bracket's games, one row per side MFL filled.
func (c *compiler) playoffs(year int64) error {
	src := seasonFile(year, "playoffBrackets")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, b := range d.get("playoffBrackets", "playoffBracket").rows() {
		id, err := keyID(b, "id", src)
		if err != nil {
			return err
		}
		c.facts.PlayoffBracket = append(c.facts.PlayoffBracket, PlayoffBracket{
			Year: year, BracketID: id, Name: b.get("name").str(), WinnerTitle: b.get("bracketWinnerTitle").str(),
			StartWeek: b.get("startWeek").whole(), Teams: b.get("teamsInvolved").whole(), Src: src,
		})
		if err := c.playoffGames(year, id); err != nil {
			return err
		}
	}
	return nil
}

func (c *compiler) playoffGames(year int64, bracket string) error {
	src := seasonFile(year, "playoffBracket_"+bracket)
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, round := range d.get("playoffBracket", "playoffRound").rows() {
		for _, g := range round.get("playoffGame").rows() {
			for _, side := range []string{"home", "away"} {
				t := g.get(side)
				fid := nonempty(t.get("franchise_id").str())
				if fid == nil {
					continue
				}
				c.facts.PlayoffGame = append(c.facts.PlayoffGame, PlayoffGame{
					Year: year, BracketID: bracket, Week: round.get("week").whole(), GameID: g.get("game_id").str(), Fid: fid,
					Side: ptr(side), Points: t.get("points").num(), Seed: t.get("seed").str(), Src: src,
				})
			}
		}
	}
	return nil
}

func (c *compiler) addNote(step, src, note string, n int64) {
	if n > 0 {
		c.facts.LoadNote = append(c.facts.LoadNote, LoadNote{Step: step, Src: ptr(src), Note: note, N: n})
	}
}

// season compiles one season's facts after the league itself, in the Lab loader's order: games and
// lineups, player weeks, rosters, contracts and cap, points allowed, the draft, picks owned, trade
// bait, transactions (numbered across seasons, so seasons run in order), playoffs and MFL-wide values.
func (c *compiler) season(year int64) error {
	steps := []func(int64) error{c.results, c.playerWeeks, c.rosters, c.contracts, c.pointsAllowed, c.draft, c.picksOwned,
		c.tradeBait, c.transactions, c.playoffs, c.mflWide}
	for _, step := range steps {
		if err := step(year); err != nil {
			return err
		}
	}
	return nil
}

func sameNumber(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
