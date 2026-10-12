package facts

import (
	"cmp"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"
)

// cent is how far two MFL scores may differ and still agree: MFL scores carry two decimals.
const cent = 0.011

// CheckResult is one proof of the facts against MFL's own figures. Every difference it finds is
// an Item; the check passes only when each item's identity is pinned in the exceptions file.
type CheckResult struct {
	Name      string      `json:"name"`
	Detail    string      `json:"detail"`
	Checked   int         `json:"checked"`
	Agreed    int         `json:"agreed"`
	Explained int         `json:"explained"`
	Review    int         `json:"review"`
	Passed    bool        `json:"passed"`
	Items     []CheckItem `json:"items"`
}

// CheckItem is one difference: its identity, what it is in plain words (names, never ids), and the
// pinned reading of it. Unpinned items fail the check.
type CheckItem struct {
	Key    string `json:"key"`
	Words  string `json:"words"`
	Class  string `json:"class"` // explained, review, or unpinned
	Reason string `json:"reason"`
}

// Check classes, as the exceptions file pins them.
const (
	ClassExplained = "explained" // a named, understood difference in how MFL records things
	ClassReview    = "review"    // listed for Christopher's review; accepted as MFL recorded it
	ClassUnpinned  = "unpinned"  // not in the exceptions file: fails the compile
)

// checker holds the facts, their views and the lookups every check shares.
type checker struct {
	f       Facts
	v       Views
	names   franchiseNames
	players map[string]int
	pinned  Exceptions
	eastern *time.Location
}

// Check runs the nine proofs of the Lab's checks.py over the compiled facts. A difference whose
// identity the exceptions do not pin fails its check; the error names every one.
func (f Facts) Check(pinned Exceptions) ([]CheckResult, error) {
	eastern, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, fmt.Errorf("check facts: %w", err)
	}
	c := checker{f: f, v: f.Views(), names: f.franchiseNames(), players: map[string]int{}, pinned: pinned, eastern: eastern}
	for i, pid := range f.Player.Pid {
		c.players[pid] = i
	}
	results := []CheckResult{c.records(), c.points(), c.potentialPoints(), c.allPlay(), c.teamScores(), c.lineupScores(),
		c.rosterLedger(), c.pickChains(), c.birthdates()}
	var unpinned []string
	for i := range results {
		results[i].settle()
		for _, it := range results[i].Items {
			if it.Class == ClassUnpinned {
				unpinned = append(unpinned, results[i].Name+": "+it.Words)
			}
		}
	}
	if len(unpinned) > 0 {
		return results, fmt.Errorf("check facts: %d differences are not pinned: %s", len(unpinned), strings.Join(unpinned, "; "))
	}
	return results, nil
}

// differ records one difference, read through the exceptions file.
func (c *checker) differ(r *CheckResult, key, words string) {
	e, ok := c.pinned.find(r.Name, key)
	it := CheckItem{Key: key, Words: words, Class: ClassUnpinned}
	if ok {
		it.Class, it.Reason = e.Class, e.Reason
	}
	r.Items = append(r.Items, it)
}

func (r *CheckResult) settle() {
	r.Passed = true
	for _, it := range r.Items {
		switch it.Class {
		case ClassExplained:
			r.Explained++
		case ClassReview:
			r.Review++
		default:
			r.Passed = false
		}
	}
	r.Agreed = r.Checked - len(r.Items)
	slices.SortFunc(r.Items, func(a, b CheckItem) int { return cmp.Compare(a.Key, b.Key) })
}

func (c *checker) team(year int64, fid *string) string {
	if name := c.names.of(year, fid); name != nil && *name != "" {
		return *name
	}
	return "a franchise not in that season's league"
}

// player is a player's name as a sentence reads it, First Last.
func (c *checker) player(pid string) string {
	i, ok := c.players[pid]
	if !ok {
		return "a player MFL no longer lists"
	}
	name := c.f.Player.Name[i]
	if last, first, ok := strings.Cut(name, ", "); ok && first != "" && !strings.Contains(first, ", ") {
		return first + " " + last
	}
	return name
}

func (c *checker) day(ts *int64) string {
	if ts == nil || *ts == 0 {
		return "an unknown day"
	}
	return time.Unix(*ts, 0).In(c.eastern).Format("Jan 2, 2006")
}

func yearFidKey(year int64, fid string) string { return fmt.Sprintf("%d/%s", year, fid) }

func record(w, l, t *int64) string {
	return fmt.Sprintf("%s-%s-%s", intText(w), intText(l), intText(t))
}

func intText(n *int64) string {
	if n == nil {
		return "none"
	}
	return fmt.Sprint(*n)
}

func sameInts(a, b []*int64) bool {
	return slices.EqualFunc(a, b, func(x, y *int64) bool { return (x == nil && y == nil) || (x != nil && y != nil && *x == *y) })
}

// records: win-loss from the weekly games equals MFL's standings. MFL counted playoff games in the
// 2013 standings; from 2014 its standings count the regular season only.
func (c *checker) records() CheckResult {
	r := CheckResult{Name: "Win-loss records match MFL standings",
		Detail: "MFL counted playoff games in the 2013 standings; from 2014 its standings count the regular season only."}
	every := map[yearFid][3]int64{}
	g := c.v.PlayedGame
	for i, year := range g.Year {
		if g.Opp[i] == nil {
			continue
		}
		k := yearFid{year, g.Fid[i]}
		wlt := every[k]
		switch value(g.Result[i]) {
		case "W":
			wlt[0]++
		case "L":
			wlt[1]++
		case "T":
			wlt[2]++
		}
		every[k] = wlt
	}
	ts := c.v.TeamSeason
	for i, year := range ts.Year {
		e, ok := every[yearFid{year, ts.Fid[i]}]
		if !ok {
			continue
		}
		r.Checked++
		mfl := []*int64{ts.MflW[i], ts.MflL[i], ts.MflT[i]}
		if sameInts(mfl, []*int64{ts.RegW[i], ts.RegL[i], ts.RegT[i]}) {
			continue
		}
		explained := year == 2013 && sameInts(mfl, []*int64{&e[0], &e[1], &e[2]})
		words := fmt.Sprintf("%d %s: MFL standings %s, weekly games %s.", year, c.team(year, &ts.Fid[i]),
			record(ts.MflW[i], ts.MflL[i], ts.MflT[i]), record(ts.RegW[i], ts.RegL[i], ts.RegT[i]))
		if explained {
			words = fmt.Sprintf("%d %s: MFL standings %s count its playoff games (regular season %s).", year,
				c.team(year, &ts.Fid[i]), record(ts.MflW[i], ts.MflL[i], ts.MflT[i]), record(ts.RegW[i], ts.RegL[i], ts.RegT[i]))
		}
		c.differ(&r, yearFidKey(year, ts.Fid[i]), words)
	}
	return r
}

func farApart(a, b *float64) bool { return a == nil || b == nil || math.Abs(*a-*b) > cent }

func (c *checker) points() CheckResult {
	r := CheckResult{Name: "Points for match MFL standings", Detail: "MFL totals points over every week it scored, playoffs included."}
	ts := c.v.TeamSeason
	for i, year := range ts.Year {
		r.Checked++
		if farApart(ts.MflPf[i], ts.AllWeeksPf[i]) {
			c.differ(&r, yearFidKey(year, ts.Fid[i]), fmt.Sprintf("%d %s: MFL %s points, weekly games %s.", year,
				c.team(year, &ts.Fid[i]), floatText(ts.MflPf[i]), floatText(ts.AllWeeksPf[i])))
		}
	}
	return r
}

func (c *checker) potentialPoints() CheckResult {
	r := CheckResult{Name: "Potential points match MFL standings", Detail: "Where MFL publishes potential points (2022 on)."}
	ts := c.v.TeamSeason
	for i, year := range ts.Year {
		if ts.MflPp[i] == nil {
			continue
		}
		r.Checked++
		if farApart(ts.MflPp[i], ts.AllWeeksPp[i]) {
			c.differ(&r, yearFidKey(year, ts.Fid[i]), fmt.Sprintf("%d %s: MFL %s potential points, weekly games %s.",
				year, c.team(year, &ts.Fid[i]), floatText(ts.MflPp[i]), floatText(ts.AllWeeksPp[i])))
		}
	}
	return r
}

func floatText(f *float64) string {
	if f == nil {
		return "none"
	}
	return fmt.Sprintf("%.2f", *f)
}

func (c *checker) allPlay() CheckResult {
	r := CheckResult{Name: "All-play records match MFL standings",
		Detail: "Counted every week in which all teams scored, playoffs included, as MFL does."}
	counted := map[yearFid][3]int64{}
	a := c.v.AllPlayWeek
	for i, year := range a.Year {
		k := yearFid{year, a.Fid[i]}
		wlt := counted[k]
		counted[k] = [3]int64{wlt[0] + a.W[i], wlt[1] + a.L[i], wlt[2] + a.T[i]}
	}
	s := c.f.Standings
	for i, year := range s.Year {
		got, ok := counted[yearFid{year, s.Fid[i]}]
		if s.AllPlayW[i] == nil || !ok {
			continue
		}
		r.Checked++
		mfl := []*int64{s.AllPlayW[i], s.AllPlayL[i], s.AllPlayT[i]}
		if !sameInts(mfl, []*int64{&got[0], &got[1], &got[2]}) {
			c.differ(&r, yearFidKey(year, s.Fid[i]), fmt.Sprintf("%d %s: MFL all-play %s, counted %s.", year,
				c.team(year, &s.Fid[i]), record(mfl[0], mfl[1], mfl[2]), record(&got[0], &got[1], &got[2])))
		}
	}
	return r
}

// teamScores: a team's score equals its starters' scores. MFL added a 3-point home bonus in the
// 2014-2015 regular season and in playoff games.
func (c *checker) teamScores() CheckResult {
	r := CheckResult{Name: "Team scores equal their starters",
		Detail: "MFL added a 3-point home bonus in the 2014-2015 regular season and in playoff games."}
	starters := map[[3]string]*nullableSum{}
	l := c.f.Lineup
	for i, year := range l.Year {
		if l.Started[i] != 1 {
			continue
		}
		k := [3]string{fmt.Sprint(year), fmt.Sprint(l.Week[i]), l.Fid[i]}
		if starters[k] == nil {
			starters[k] = &nullableSum{}
		}
		starters[k].add(l.Score[i])
	}
	g := c.v.PlayedGame
	for i, year := range g.Year {
		s, ok := starters[[3]string{fmt.Sprint(year), fmt.Sprint(g.Week[i]), g.Fid[i]}]
		if !ok || !s.any {
			continue
		}
		r.Checked++
		st := round2(s.s.total())
		diff := math.Round((*g.Score[i]-st)*100) / 100
		if math.Abs(diff) <= cent {
			continue
		}
		c.differ(&r, fmt.Sprintf("%d/%02d/%s", year, g.Week[i], g.Fid[i]), fmt.Sprintf(
			"%d week %d, %s: team score %.2f, its starters sum to %.2f (%+.2f).", year, g.Week[i],
			c.team(year, &g.Fid[i]), *g.Score[i], st, diff))
	}
	return r
}

func (c *checker) lineupScores() CheckResult {
	r := CheckResult{Name: "Lineup scores equal MFL player scores"}
	type weekPlayer struct {
		year, week int64
		pid        string
	}
	scores := map[weekPlayer]*float64{}
	pw := c.f.PlayerWeek
	for i, year := range pw.Year {
		scores[weekPlayer{year, pw.Week[i], pw.Pid[i]}] = pw.Score[i]
	}
	l := c.f.Lineup
	for i, year := range l.Year {
		score, ok := scores[weekPlayer{year, l.Week[i], l.Pid[i]}]
		if l.Score[i] == nil || !ok {
			continue
		}
		r.Checked++
		if farApart(l.Score[i], score) {
			c.differ(&r, fmt.Sprintf("%d/%02d/%s/%s", year, l.Week[i], l.Fid[i], l.Pid[i]), fmt.Sprintf(
				"%d week %d, %s for %s: lineup %s, player scores %s.", year, l.Week[i], c.player(l.Pid[i]),
				c.team(year, &l.Fid[i]), floatText(l.Score[i]), floatText(score)))
		}
	}
	return r
}

// birthdates: every rostered or lineup player has MFL's birthdate.
func (c *checker) birthdates() CheckResult {
	r := CheckResult{Name: "Rostered players have an MFL birthdate", Detail: "MFL lists no birthdate for these players."}
	rostered := map[string]bool{}
	for _, pid := range c.f.RosterWeek.Pid {
		rostered[pid] = true
	}
	for _, pid := range c.f.Lineup.Pid {
		rostered[pid] = true
	}
	p := c.f.Player
	for i, pid := range p.Pid {
		if !rostered[pid] {
			continue
		}
		r.Checked++
		if p.Birthdate[i] == nil {
			c.differ(&r, pid, fmt.Sprintf("%s (%s)", c.player(pid), orNone(p.Position[i])))
		}
	}
	return r
}

func orNone(s *string) string {
	if s == nil || *s == "" {
		return "no position"
	}
	return *s
}

var pickTraded = regexp.MustCompile(`Pick traded from ([^.\n]+)\.`)

// nullableSum is SQL's SUM over a column with nulls: nulls are skipped, and all-null is null.
type nullableSum struct {
	s   sum
	any bool
}

func (n *nullableSum) add(x *float64) {
	if x != nil {
		n.s.add(*x)
		n.any = true
	}
}
