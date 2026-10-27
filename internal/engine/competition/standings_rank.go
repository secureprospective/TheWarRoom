package competition

import (
	"fmt"
	"sort"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type RankTeam struct{ ID, Name, Division, Conference string }

type RankGame struct {
	Home, Away             string
	HomePoints, AwayPoints leaguerules.Decimal
}

// Tally is a win-loss-tie record.
type Tally struct{ W, L, T int }

func (t Tally) games() int { return t.W + t.L + t.T }

func (t Tally) Words() string {
	if t.T == 0 {
		return fmt.Sprintf("%d-%d", t.W, t.L)
	}
	return fmt.Sprintf("%d-%d-%d", t.W, t.L, t.T)
}

// RankInput is everything the tie-breakers read. Only decided games are in it.
type RankInput struct {
	Teams    []RankTeam
	Games    []RankGame
	AllPlay  map[string]Tally
	Stands   map[string]Tally
	Points   map[string]leaguerules.Decimal
	Criteria []leaguerules.Criterion
}

// Ranking is the order the criteria produce and the plain line behind every tie they broke.
type Ranking struct {
	Order   []string
	Records map[string]Tally
	Why     map[string][]string
	Skipped []string
	// Level lists each group of teams every tie-breaker left Level, which the order only keeps in league order.
	Level [][]string
}

// rankKey is a comparable value: a fraction, so a record compares exactly.
type rankKey struct{ num, den int64 }

func (a rankKey) beats(b rankKey) bool { return a.num*b.den > b.num*a.den }

// evaluation is one criterion read over one tied group: either why it does not apply, or a key and
// words for every team.
type evaluation struct {
	skip  string
	keys  map[string]rankKey
	words map[string]string
}

type ranker struct {
	in      RankInput
	teams   map[string]RankTeam
	overall map[string]Tally
	within  map[string]map[string]Tally
	out     Ranking
}

// criterionLabels are the plain names of the tie-breakers an owner reads.
func criterionLabels() map[leaguerules.Criterion]string {
	return map[leaguerules.Criterion]string{
		leaguerules.WinPercentage: "winning percentage", leaguerules.HeadToHeadRecord: "head-to-head record",
		leaguerules.DivisionPercentage: "division record", leaguerules.ConferencePercentage: "conference record",
		leaguerules.AllPlayPercentage: "all-play record", leaguerules.Points: "points scored",
		leaguerules.PowerRank: "power rank",
	}
}

func CriterionLabel(c leaguerules.Criterion) string {
	if label, ok := criterionLabels()[c]; ok {
		return label
	}
	return string(c)
}

func RankStandings(in RankInput) Ranking {
	r := &ranker{in: in, teams: map[string]RankTeam{}, overall: map[string]Tally{},
		within: map[string]map[string]Tally{"division": {}, "conference": {}},
		out:    Ranking{Why: map[string][]string{}, Skipped: []string{}}}
	ids := make([]string, 0, len(in.Teams))
	for _, t := range in.Teams {
		r.teams[t.ID] = t
		ids = append(ids, t.ID)
	}
	r.count()
	r.out.Records = r.overall
	r.out.Order = r.rank(ids, 0)
	return r.out
}

// groupings are the shared groups a record can be counted within.
func groupings() map[string]func(RankTeam) string {
	return map[string]func(RankTeam) string{"division": func(t RankTeam) string { return t.Division },
		"conference": func(t RankTeam) string { return t.Conference }}
}

// count tallies each decided game for both sides: overall, and within a division or conference they share.
// An adjusted record replaces only the overall record.
func (r *ranker) count() {
	for _, g := range r.in.Games {
		for _, side := range [][2]string{{g.Home, g.Away}, {g.Away, g.Home}} {
			mine, theirs := g.HomePoints, g.AwayPoints
			if side[0] == g.Away {
				mine, theirs = theirs, mine
			}
			result := ResultOf(mine, theirs)
			r.overall[side[0]] = result.Add(r.overall[side[0]])
			for kind, attr := range groupings() {
				if a := attr(r.teams[side[0]]); a != "" && a == attr(r.teams[side[1]]) {
					r.within[kind][side[0]] = result.Add(r.within[kind][side[0]])
				}
			}
		}
	}
	for id, t := range r.in.Stands {
		r.overall[id] = t
	}
}

func ResultOf(mine, theirs leaguerules.Decimal) Tally {
	switch {
	case mine > theirs:
		return Tally{W: 1}
	case mine < theirs:
		return Tally{L: 1}
	}
	return Tally{T: 1}
}

func (t Tally) Add(o Tally) Tally { return Tally{t.W + o.W, t.L + o.L, t.T + o.T} }

// A missing record skips the criterion rather than inventing a losing percentage.
func (r *ranker) percentage(group []string, records map[string]Tally) evaluation {
	ev := evaluation{keys: map[string]rankKey{}, words: map[string]string{}}
	for _, id := range group {
		t := records[id]
		if t.games() == 0 {
			return evaluation{skip: r.teams[id].Name + " has no games yet"}
		}
		ev.keys[id], ev.words[id] = rankKey{int64(2*t.W + t.T), int64(2 * t.games())}, t.Words()
	}
	return ev
}

// headToHead applies only when every tied team has met every other, then reads their games with each other.
func (r *ranker) headToHead(group []string) evaluation {
	in := map[string]bool{}
	for _, id := range group {
		in[id] = true
	}
	records, met := map[string]Tally{}, map[[2]string]bool{}
	for _, g := range r.in.Games {
		if !in[g.Home] || !in[g.Away] {
			continue
		}
		met[[2]string{min(g.Home, g.Away), max(g.Home, g.Away)}] = true
		records[g.Home] = ResultOf(g.HomePoints, g.AwayPoints).Add(records[g.Home])
		records[g.Away] = ResultOf(g.AwayPoints, g.HomePoints).Add(records[g.Away])
	}
	for i, a := range group {
		for _, b := range group[i+1:] {
			if !met[[2]string{min(a, b), max(a, b)}] {
				return evaluation{skip: "not every tied team has played every other"}
			}
		}
	}
	return r.percentage(group, records)
}

// shared reads a division or conference record, which only compares teams that share one.
func (r *ranker) shared(group []string, kind string, attr func(RankTeam) string) evaluation {
	first := attr(r.teams[group[0]])
	for _, id := range group {
		switch value := attr(r.teams[id]); {
		case value == "":
			return evaluation{skip: fmt.Sprintf("the league has not said which %s %s is in", kind, r.teams[id].Name)}
		case value != first:
			return evaluation{skip: fmt.Sprintf("they are not all in the same %s", kind)}
		}
	}
	return r.percentage(group, r.within[kind])
}

func (r *ranker) byPoints(group []string) evaluation {
	ev := evaluation{keys: map[string]rankKey{}, words: map[string]string{}}
	for _, id := range group {
		points, ok := r.in.Points[id]
		if !ok {
			return evaluation{skip: r.teams[id].Name + " has no complete weekly score yet"}
		}
		ev.keys[id], ev.words[id] = rankKey{int64(points), 1}, PointsWords(points)+" points scored"
	}
	return ev
}

func (r *ranker) evaluate(c leaguerules.Criterion, group []string) evaluation {
	switch c {
	case leaguerules.WinPercentage:
		return r.percentage(group, r.overall)
	case leaguerules.HeadToHeadRecord:
		return r.headToHead(group)
	case leaguerules.DivisionPercentage:
		return r.shared(group, "division", func(t RankTeam) string { return t.Division })
	case leaguerules.ConferencePercentage:
		return r.shared(group, "conference", func(t RankTeam) string { return t.Conference })
	case leaguerules.AllPlayPercentage:
		return r.percentage(group, r.in.AllPlay)
	case leaguerules.Points:
		return r.byPoints(group)
	case leaguerules.PowerRank:
		return evaluation{skip: "no power rank is held for this league yet"}
	}
	return evaluation{skip: "the app does not count this tie-breaker"}
}

// trail is what a tied group has met so far: criteria that left it level, and ones that did not apply.
type trail struct{ level, skipped []string }

// rank orders a group that is level so far. A criterion that splits it ranks the parts, and each part
// that is still tied starts again from the first criterion.
func (r *ranker) rank(group []string, depth int) []string {
	var seen trail
	for _, c := range r.in.Criteria {
		ev := r.evaluate(c, group)
		if ev.skip != "" {
			line := fmt.Sprintf("%s did not apply: %s", CriterionLabel(c), ev.skip)
			seen.skipped = append(seen.skipped, line)
			if depth == 0 {
				r.out.Skipped = append(r.out.Skipped, Capital(line)+".")
			}
			continue
		}
		tiers := splitTiers(group, ev.keys)
		if len(tiers) == 1 {
			seen.level = append(seen.level, CriterionLabel(c))
			continue
		}
		r.note(group, strings.Join(append(seen.levelLines(group, r), seen.skippedLines()...), " ")+
			fmt.Sprintf(" %s put %s.", Capital(CriterionLabel(c)), r.tiersText(tiers, ev)))
		ordered := []string{}
		for _, tier := range tiers {
			if len(tier) > 1 {
				tier = r.rank(tier, depth+1)
			}
			ordered = append(ordered, tier...)
		}
		return ordered
	}
	if len(group) > 1 {
		r.out.Level = append(r.out.Level, append([]string{}, group...))
		r.note(group, strings.Join(seen.skippedLines(), " ")+fmt.Sprintf(
			" %s remain level on the available tie-breakers, so they are listed in the league's franchise order.",
			Capital(r.namesOf(group))))
	}
	return group
}

func (s trail) levelLines(group []string, r *ranker) []string {
	if len(s.level) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("%s were level on %s.", Capital(r.namesOf(group)), JoinAnd(s.level))}
}

func (s trail) skippedLines() []string {
	lines := make([]string, len(s.skipped))
	for i, line := range s.skipped {
		lines[i] = Capital(line) + "."
	}
	return lines
}

func (r *ranker) note(group []string, line string) {
	line = strings.TrimSpace(line)
	for _, id := range group {
		r.out.Why[id] = append(r.out.Why[id], line)
	}
}

// splitTiers orders a group best first and groups equal keys, keeping league order within a tier.
func splitTiers(group []string, keys map[string]rankKey) [][]string {
	ordered := append([]string{}, group...)
	sort.SliceStable(ordered, func(i, j int) bool { return keys[ordered[i]].beats(keys[ordered[j]]) })
	tiers := [][]string{}
	for _, id := range ordered {
		if n := len(tiers); n > 0 && !keys[tiers[n-1][0]].beats(keys[id]) {
			tiers[n-1] = append(tiers[n-1], id)
			continue
		}
		tiers = append(tiers, []string{id})
	}
	return tiers
}

func (r *ranker) tiersText(tiers [][]string, ev evaluation) string {
	parts := make([]string, len(tiers))
	for i, tier := range tiers {
		parts[i] = fmt.Sprintf("%s (%s)", r.namesOf(tier), ev.words[tier[0]])
	}
	return strings.Join(parts, " over ")
}

// namesOf names a few teams and counts a large group.
func (r *ranker) namesOf(group []string) string {
	if len(group) > 6 {
		return fmt.Sprintf("%d teams", len(group))
	}
	names := make([]string, len(group))
	for i, id := range group {
		names[i] = r.teams[id].Name
	}
	return JoinAnd(names)
}

func JoinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

func Capital(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
