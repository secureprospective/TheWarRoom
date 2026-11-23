package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// tolerance is half a cent of a point, in the league's ten-thousandths.
const tolerance = leaguerules.Decimal(50)

// reader is the read-only door to the league: the serverclient agent in read mode.
type reader interface {
	Read(ctx context.Context, command string, scope serverhttp.RawWorkflowScope) (json.RawMessage, error)
}

func readAs[T any](ctx context.Context, r reader, command string, scope serverhttp.RawWorkflowScope) (T, error) {
	var out T
	data, err := r.Read(ctx, command, scope)
	if err != nil {
		return out, fmt.Errorf("read %s: %w", command, err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("the league sent an unreadable %s answer: %w", command, err)
	}
	return out, nil
}

func points(d leaguerules.Decimal) string { return fmt.Sprintf("%.4f", float64(d)/10000) }

func absDec(d leaguerules.Decimal) leaguerules.Decimal {
	if d < 0 {
		return -d
	}
	return d
}

// Check runs the four checks for one week. An error means the league could not be read (connection
// or token), which is not a finding about the league.
func Check(ctx context.Context, r reader, season, week int) (Report, error) {
	var rep Report
	week0 := serverhttp.RawWorkflowScope{Season: season, Week: week}
	results, err := readAs[commissioner.NativeResults](ctx, r, "results.read", week0)
	if err != nil {
		return rep, fmt.Errorf("can't read the week's results: %w", err)
	}
	checkArithmetic(&rep, results)
	games, err := readAs[leagueservice.NFLGames](ctx, r, "nfl.games.read", serverhttp.RawWorkflowScope{})
	if err != nil {
		return rep, fmt.Errorf("can't read the NFL games: %w", err)
	}
	checkFinality(&rep, results, games, week)
	standings, err := readAs[commissioner.NativeStandings](ctx, r, "standings.read", serverhttp.RawWorkflowScope{Season: season})
	if err != nil {
		return rep, fmt.Errorf("can't read the standings: %w", err)
	}
	if err := checkStandings(ctx, &rep, r, standings, results, season, week); err != nil {
		return rep, err
	}
	if err := checkLineupLock(ctx, &rep, r, results, games, season, week); err != nil {
		return rep, err
	}
	return rep, nil
}

// startersTotal is a team's starters' points plus its adjustment; false when a starter has no points.
func startersTotal(s commissioner.FranchiseWeekScore) (leaguerules.Decimal, leaguerules.Decimal, bool) {
	var sum leaguerules.Decimal
	for _, st := range s.Starters {
		if st.Points == nil {
			return 0, 0, false
		}
		sum += *st.Points
	}
	return sum, sum + s.Adjustment, true
}

// gameTotalProblems lists each game side whose shown points differ from its team's score.
func gameTotalProblems(res commissioner.NativeResults) []string {
	byFranchise := map[string]commissioner.FranchiseWeekScore{}
	for _, s := range res.Scores {
		byFranchise[s.Franchise] = s
	}
	var bad []string
	for _, g := range res.Games {
		for _, side := range []struct {
			id  string
			pts *leaguerules.Decimal
		}{{g.Home, g.HomePoints}, {g.Away, g.AwayPoints}} {
			s, ok := byFranchise[side.id]
			if ok && side.pts != nil && s.Points != nil && absDec(*side.pts-*s.Points) > tolerance {
				bad = append(bad, fmt.Sprintf("%s: the game shows %s but the team score is %s", s.Name, points(*side.pts), points(*s.Points)))
			}
		}
	}
	return bad
}

// checkArithmetic: each franchise's score is its starters' points plus its adjustment, and each game
// shows the same totals as the franchise scores.
func checkArithmetic(rep *Report, res commissioner.NativeResults) {
	var bad, detail []string
	checked, waiting := 0, 0
	for _, s := range res.Scores {
		if s.Points == nil {
			waiting++
			continue
		}
		sum, want, ok := startersTotal(s)
		if !ok {
			bad = append(bad, fmt.Sprintf("%s has a total but a starter has no points", s.Name))
			continue
		}
		checked++
		detail = append(detail, fmt.Sprintf("%s: starters %s + adjustment %s = %s; reported %s", s.Name, points(sum), points(s.Adjustment), points(want), points(*s.Points)))
		if absDec(want-*s.Points) > tolerance {
			bad = append(bad, fmt.Sprintf("%s reports %s but the starters and adjustments add to %s", s.Name, points(*s.Points), points(want)))
		}
	}
	bad = append(bad, gameTotalProblems(res)...)
	switch {
	case len(bad) > 0:
		rep.add("Scores add up", Red, fmt.Sprintf("%d score(s) do not add up", len(bad)), append(bad, detail...))
	case checked == 0:
		rep.add("Scores add up", Amber, "no team has a finished score yet", detail)
	case waiting > 0:
		rep.add("Scores add up", Amber, fmt.Sprintf("%d team scores add up; %d are not finished yet", checked, waiting), detail)
	default:
		rep.add("Scores add up", Green, fmt.Sprintf("all %d team scores equal their starters plus adjustments", checked), detail)
	}
}

func gamesOfWeek(g leagueservice.NFLGames, season, week int) []gameFact {
	var out []gameFact
	for _, game := range g.Board.Games {
		if (game.Week != 0 && game.Week != week) || (game.Season != 0 && game.Season != season) {
			continue
		}
		out = append(out, gameFact{game.Home.Abbreviation, game.Away.Abbreviation, game.Kickoff, game.Status == "post"})
	}
	return out
}

type gameFact struct {
	home, away string
	kickoff    time.Time
	final      bool
}

// checkFinality: every NFL game is over and the league's evidence is the official final kind.
func checkFinality(rep *Report, res commissioner.NativeResults, g leagueservice.NFLGames, week int) {
	var why []string
	games := gamesOfWeek(g, res.Season, week)
	if len(games) == 0 {
		why = append(why, "the NFL scoreboard lists no games for this week")
	}
	for _, f := range games {
		if !f.final {
			why = append(why, fmt.Sprintf("%s at %s is not final", f.away, f.home))
		}
	}
	if e := res.Evidence; e == nil {
		why = append(why, "the results carry no statistics evidence")
	} else {
		if e.Provisional {
			why = append(why, "the results still use provisional statistics")
		}
		if e.Reduced {
			why = append(why, "the results run on reduced statistics (a source is missing)")
		}
		if e.Stale {
			why = append(why, "the results use statistics that are out of date")
		}
	}
	if len(why) > 0 {
		rep.add("Official stats final", Amber, "official stats not final yet; the other checks may change", why)
		return
	}
	rep.add("Official stats final", Green, fmt.Sprintf("all %d NFL games are final and the statistics are official", len(games)), nil)
}

type tally struct {
	w, l, t int
	pf      leaguerules.Decimal
}

// tallyWeeks adds up every decided game of weeks 1..through from the league's own results.
func tallyWeeks(ctx context.Context, r reader, thisWeek commissioner.NativeResults, season, through int) (map[string]*tally, error) {
	got := map[string]*tally{}
	for w := 1; w <= through; w++ {
		res := thisWeek
		if w != thisWeek.Week {
			var err error
			res, err = readAs[commissioner.NativeResults](ctx, r, "results.read", serverhttp.RawWorkflowScope{Season: season, Week: w})
			if err != nil {
				return nil, fmt.Errorf("can't read week %d results: %w", w, err)
			}
		}
		for _, g := range res.Games {
			if (g.Status != commissioner.GameFinal && g.Status != commissioner.GameTie) || g.HomePoints == nil || g.AwayPoints == nil {
				continue
			}
			h, a := tallyOf(got, g.Home), tallyOf(got, g.Away)
			h.pf += *g.HomePoints
			a.pf += *g.AwayPoints
			switch {
			case g.Status == commissioner.GameTie:
				h.t++
				a.t++
			case g.Winner == g.Home:
				h.w++
				a.l++
			default:
				a.w++
				h.l++
			}
		}
	}
	return got, nil
}

// checkStandings recomputes every team's record and points-for from the weekly games and compares.
func checkStandings(ctx context.Context, rep *Report, r reader, st commissioner.NativeStandings, thisWeek commissioner.NativeResults, season, week int) error {
	through := st.Through
	if through < 1 {
		rep.add("Standings match games", Amber, "the standings count no weeks yet", nil)
		return nil
	}
	got, err := tallyWeeks(ctx, r, thisWeek, season, through)
	if err != nil {
		return err
	}
	var bad, detail []string
	for _, row := range st.Computed {
		t := got[row.Franchise]
		if t == nil {
			t = &tally{}
		}
		detail = append(detail, fmt.Sprintf("%s: games say %d-%d-%d, %s points; standings say %d-%d-%d, %s", row.Name, t.w, t.l, t.t, points(t.pf), row.Wins, row.Losses, row.Ties, points(row.PointsFor)))
		if t.w != row.Wins || t.l != row.Losses || t.t != row.Ties {
			bad = append(bad, fmt.Sprintf("%s: the games make it %d-%d-%d but the standings show %d-%d-%d", row.Name, t.w, t.l, t.t, row.Wins, row.Losses, row.Ties))
		}
		if absDec(t.pf-row.PointsFor) > tolerance {
			bad = append(bad, fmt.Sprintf("%s: the games add to %s points but the standings show %s", row.Name, points(t.pf), points(row.PointsFor)))
		}
	}
	switch {
	case len(bad) > 0:
		rep.add("Standings match games", Red, fmt.Sprintf("%d mismatch(es) between the games and the standings", len(bad)), append(bad, detail...))
	case through < week:
		rep.add("Standings match games", Amber, fmt.Sprintf("standings match the games, but only count through week %d", through), detail)
	default:
		rep.add("Standings match games", Green, fmt.Sprintf("records and points for %d teams match the games through week %d", len(st.Computed), through), detail)
	}
	return nil
}

func tallyOf(m map[string]*tally, id string) *tally {
	if m[id] == nil {
		m[id] = &tally{}
	}
	return m[id]
}

// lockProblems compares a team's scored starters with its lineup at one kickoff.
func lockProblems(s commissioner.FranchiseWeekScore, lineup commissioner.NativeLineup, kickoff map[string]time.Time, k time.Time) []string {
	var bad []string
	when := k.UTC().Format(time.RFC3339)
	scored := map[string]bool{}
	for _, st := range s.Starters {
		scored[st.Player] = true
	}
	atLock := map[string]commissioner.LineupPlayer{}
	for _, p := range lineup.Starters {
		atLock[p.Player] = p
	}
	for _, st := range s.Starters {
		if kk := kickoff[st.Team]; !kk.Equal(k) {
			continue
		}
		if _, in := atLock[st.Player]; !in {
			bad = append(bad, fmt.Sprintf("%s: %s was scored as a starter but was not in the lineup at kickoff (%s)", s.Name, st.Name, when))
		}
	}
	for id, p := range atLock {
		if kk := kickoff[p.Team]; !kk.Equal(k) {
			continue
		}
		if !scored[id] {
			bad = append(bad, fmt.Sprintf("%s: %s was a starter at kickoff (%s) but was not scored", s.Name, p.Name, when))
		}
	}
	return bad
}

// kickoffTimes is the distinct kickoff instants of a team's scored starters, earliest first.
func kickoffTimes(s commissioner.FranchiseWeekScore, kickoff map[string]time.Time) []time.Time {
	seen := map[time.Time]bool{}
	var times []time.Time
	for _, st := range s.Starters {
		if k, ok := kickoff[st.Team]; ok && !k.IsZero() && !seen[k] {
			seen[k] = true
			times = append(times, k)
		}
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	return times
}

// checkLineupLock: at each starter's kickoff, the franchise's lineup must hold exactly the starters that
// were scored. A starter scored who was not in the lineup then (or the reverse) is red.
func checkLineupLock(ctx context.Context, rep *Report, r reader, res commissioner.NativeResults, g leagueservice.NFLGames, season, week int) error {
	kickoff := map[string]time.Time{}
	for _, f := range gamesOfWeek(g, season, week) {
		kickoff[f.home], kickoff[f.away] = f.kickoff, f.kickoff
	}
	if len(kickoff) == 0 {
		rep.add("Lineups locked at kickoff", Unknown, "UNKNOWN: no kickoff times are available for this week", nil)
		return nil
	}
	var bad, unknown, notes []string
	checked := 0
	for _, s := range res.Scores {
		times := kickoffTimes(s, kickoff)
		denied := false
		for _, k := range times {
			lineup, err := readAs[commissioner.NativeLineup](ctx, r, "lineup.read",
				serverhttp.RawWorkflowScope{Franchise: s.Franchise, Week: week, At: k.UTC().Format(time.RFC3339)})
			if serverclient.IsRefused(err) {
				unknown = append(unknown, fmt.Sprintf("UNKNOWN: token cannot read franchise %s (%s)", s.Franchise, s.Name))
				denied = true
				break
			}
			if err != nil {
				return fmt.Errorf("can't read %s's lineup: %w", s.Name, err)
			}
			bad = append(bad, lockProblems(s, lineup, kickoff, k)...)
		}
		if !denied && len(times) > 0 {
			checked++
			notes = append(notes, fmt.Sprintf("%s: %d starters checked at %d kickoff time(s)", s.Name, len(s.Starters), len(times)))
		}
	}
	detail := append(append(bad, unknown...), notes...)
	switch {
	case len(bad) > 0:
		rep.add("Lineups locked at kickoff", Red, fmt.Sprintf("%d starter(s) differ from the lineup at kickoff", len(bad)), detail)
	case len(unknown) > 0:
		rep.add("Lineups locked at kickoff", Unknown, fmt.Sprintf("UNKNOWN: this token cannot read %d team lineup(s); %d checked and fine", len(unknown), checked), detail)
	default:
		rep.add("Lineups locked at kickoff", Green, fmt.Sprintf("%d lineups match what was locked at kickoff", checked), detail)
	}
	return nil
}
