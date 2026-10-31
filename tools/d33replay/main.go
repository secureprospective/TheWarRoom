// Command d33replay checks the D33 lineup delta against a real league's trades
// (MODEL_METHODS_DRAFT §6). It reads MFL exports saved in -dir: league_Y.json,
// players_Y.json, trades_Y.json and wr_Y_W.json (weeklyResults). It never writes.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/lineupdelta"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
	"github.com/secureprospective/TheWarRoom/internal/projection"
)

const weeks = 17

// kickoff is each season's week-1 Thursday; a week's moves close the Tuesday before it.
var kickoff = map[int]string{2023: "2023-09-07", 2024: "2024-09-05", 2025: "2025-09-04"} //nolint:gochecknoglobals // replay constants

type team struct {
	roster   []string
	starters map[string]bool
	scores   map[string]float64 // only players with a game
	opt      float64
}

type season struct {
	rules powerrankings.LineupRules
	pos   map[string]string
	week  [weeks + 1]map[string]team
}

type trade struct {
	year, week int
	a, b       string
	aGave      []string
	bGave      []string
}

type replay struct {
	seasons map[int]*season
	trades  []trade
	last    int // last regular-season week
}

func main() {
	dir := flag.String("dir", ".", "directory of saved MFL exports")
	last := flag.Int("last", 13, "last regular-season week")
	flag.Parse()
	r := &replay{seasons: map[int]*season{}, last: *last}
	for y := range kickoff {
		if err := r.load(*dir, y); err != nil {
			log.Fatal(err)
		}
	}
	r.report(r.walk())
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func ids(csv string) []string {
	var out []string
	for _, s := range strings.Split(csv, ",") {
		if s = strings.TrimSpace(s); s != "" && !strings.Contains(s, "_") {
			out = append(out, s)
		}
	}
	return out
}

func (r *replay) load(dir string, y int) error {
	s := &season{pos: map[string]string{}}
	if err := s.loadRules(fmt.Sprintf("%s/league_%d.json", dir, y)); err != nil {
		return err
	}
	var players struct {
		Players struct {
			Player []struct{ ID, Position string } `json:"player"`
		} `json:"players"`
	}
	if err := readJSON(fmt.Sprintf("%s/players_%d.json", dir, y), &players); err != nil {
		return err
	}
	for _, p := range players.Players.Player {
		s.pos[p.ID] = p.Position
	}
	for w := 1; w <= weeks; w++ {
		if err := s.loadWeek(fmt.Sprintf("%s/wr_%d_%d.json", dir, y, w), w); err != nil {
			return err
		}
	}
	r.seasons[y] = s
	return r.loadTrades(fmt.Sprintf("%s/trades_%d.json", dir, y), y)
}

func (s *season) loadRules(path string) error {
	var lg struct {
		League struct {
			Starters struct {
				Count       string                         `json:"count"`
				IDPStarters string                         `json:"idp_starters"`
				Position    []struct{ Name, Limit string } `json:"position"`
			} `json:"starters"`
		} `json:"league"`
	}
	if err := readJSON(path, &lg); err != nil {
		return err
	}
	st := lg.League.Starters
	s.rules.Total, _ = strconv.Atoi(st.Count)
	s.rules.Defense, _ = strconv.Atoi(st.IDPStarters)
	for _, p := range st.Position {
		a, b, ranged := strings.Cut(p.Limit, "-")
		lo, _ := strconv.Atoi(a)
		hi := lo
		if ranged {
			hi, _ = strconv.Atoi(b)
		}
		s.rules.Slots = append(s.rules.Slots, powerrankings.Slot{Position: p.Name, Min: lo, Max: hi})
	}
	return nil
}

type mflFranchise struct {
	ID, Starters string
	Opt          string                       `json:"opt_pts"`
	Player       []struct{ ID, Score string } `json:"player"`
}

// many decodes MFL's list, which it writes as a bare object when it has one entry.
func many[T any](raw json.RawMessage) []T {
	var list []T
	if json.Unmarshal(raw, &list) == nil {
		return list
	}
	var one T
	if json.Unmarshal(raw, &one) == nil {
		return []T{one}
	}
	return nil
}

func (s *season) loadWeek(path string, w int) error {
	var wr struct {
		WeeklyResults struct {
			Matchup json.RawMessage `json:"matchup"`
		} `json:"weeklyResults"`
	}
	if err := readJSON(path, &wr); err != nil {
		return err
	}
	s.week[w] = map[string]team{}
	for _, m := range many[struct {
		Franchise json.RawMessage `json:"franchise"`
	}](wr.WeeklyResults.Matchup) {
		for _, f := range many[mflFranchise](m.Franchise) {
			t := team{starters: map[string]bool{}, scores: map[string]float64{}}
			t.opt, _ = strconv.ParseFloat(f.Opt, 64)
			for _, id := range ids(f.Starters) {
				t.starters[id] = true
			}
			for _, p := range f.Player {
				t.roster = append(t.roster, p.ID)
				if v, err := strconv.ParseFloat(p.Score, 64); err == nil {
					t.scores[p.ID] = v
				}
			}
			s.week[w][f.ID] = t
		}
	}
	return nil
}

func (r *replay) loadTrades(path string, y int) error {
	var tr struct {
		Transactions struct {
			Transaction []struct {
				Type, Franchise, Franchise2, Timestamp string
				Gave1                                  string `json:"franchise1_gave_up"`
				Gave2                                  string `json:"franchise2_gave_up"`
			} `json:"transaction"`
		} `json:"transactions"`
	}
	if err := readJSON(path, &tr); err != nil {
		return err
	}
	start, err := time.Parse("2006-01-02", kickoff[y])
	if err != nil {
		return fmt.Errorf("kickoff %d: %w", y, err)
	}
	firstClose := start.Add(-48 * time.Hour)
	for _, t := range tr.Transactions.Transaction {
		ts, err := strconv.ParseInt(t.Timestamp, 10, 64)
		if t.Type != "TRADE" || err != nil {
			continue
		}
		at := time.Unix(ts, 0)
		if at.Before(firstClose) {
			continue
		}
		week := int(at.Sub(firstClose).Hours()/(24*7)) + 1 // the week window the trade falls in
		r.trades = append(r.trades, trade{year: y, week: week, a: t.Franchise, b: t.Franchise2,
			aGave: ids(t.Gave1), bGave: ids(t.Gave2)})
	}
	return nil
}

// result is one side of one trade.
type result struct {
	predicted, thisWeek    float64 // model: rest-of-season and this-week change, points per week
	hindsight              float64 // the same rosters' best lineups at the next 4 weeks' actual scores
	optChange              float64 // league's own optimal points, 4 weeks after minus 4 before
	benched, benchedOut    int     // model "benched" players, and how many did not start next week
	wasStarting, lostStart int     // of those, the ones the owner started before the trade, and lost their start
}

// history is every player's league-scored weeks before (y, w), and each position's pool.
func (r *replay) history(y, w int) (map[string][]projection.WeekScore, map[string][]projection.WeekScore) {
	own, byPos := map[string][]projection.WeekScore{}, map[string][]projection.WeekScore{}
	for sy, s := range r.seasons {
		for sw := 1; sw <= weeks; sw++ {
			if sy > y || sy == y && sw >= w {
				continue
			}
			seen := map[string]bool{}
			for _, t := range s.week[sw] {
				for id, v := range t.scores {
					if seen[id] {
						continue
					}
					seen[id] = true
					ws := projection.WeekScore{Season: sy, Week: sw, Points: v}
					own[id] = append(own[id], ws)
					byPos[s.pos[id]] = append(byPos[s.pos[id]], ws)
				}
			}
		}
	}
	return own, byPos
}

// values is each player's D11 median as of (y, w), as the app computes it.
func (r *replay) values(y, w int) map[string]float64 {
	p := projection.DefaultParams()
	own, byPos := r.history(y, w)
	pos := map[string]projection.Shape{}
	for k, v := range byPos {
		pos[k] = projection.NewShape(v, y, w, p)
	}
	out := map[string]float64{}
	s := r.seasons[y]
	for id, v := range own {
		if rg, ok := projection.Project(projection.NewShape(v, y, w, p), pos[s.pos[id]], p); ok {
			out[id] = rg.Median
		}
	}
	return out
}

func (s *season) players(list []string, val map[string]float64, plays map[string]float64) []lineupdelta.Player {
	out := make([]lineupdelta.Player, 0, len(list))
	for _, id := range list {
		v, ok := val[id]
		_, played := plays[id]
		out = append(out, lineupdelta.Player{ID: id, Position: s.pos[id], PPG: v, Projected: ok,
			Plays: plays == nil || played})
	}
	return out
}

func (r *replay) walk() []result {
	var out []result
	cache := map[[2]int]map[string]float64{}
	for _, t := range r.trades {
		if t.week < 5 || t.week+4 > r.last || len(t.aGave)+len(t.bGave) == 0 {
			continue
		}
		key := [2]int{t.year, t.week}
		if cache[key] == nil {
			cache[key] = r.values(t.year, t.week)
		}
		for _, side := range [][4]any{{t.a, t.b, t.aGave, t.bGave}, {t.b, t.a, t.bGave, t.aGave}} {
			if res, ok := r.side(t, side[0].(string), side[1].(string), side[2].([]string), side[3].([]string),
				cache[key]); ok {
				out = append(out, res)
			}
		}
	}
	return out
}

func (r *replay) side(t trade, me, them string, give, get []string, val map[string]float64) (result, bool) {
	s := r.seasons[t.year]
	pre, ok := s.week[t.week-1][me]
	if !ok || !holds(pre.roster, give) || !holds(s.week[t.week-1][them].roster, get) {
		return result{}, false
	}
	next := s.week[t.week+1][me]
	model := lineupdelta.Compute(me, s.players(pre.roster, val, nil), give, s.players(get, val, nil), s.rules)
	week := lineupdelta.Compute(me, s.players(pre.roster, val, next.scores), give,
		s.players(get, val, next.scores), s.rules)
	res := result{predicted: model.PerWeek, thisWeek: week.PerWeek}
	for w := t.week + 1; w <= t.week+4; w++ {
		sc := map[string]float64{}
		for _, tm := range s.week[w] {
			for id, v := range tm.scores {
				sc[id] = v
			}
		}
		h := lineupdelta.Compute(me, s.players(pre.roster, sc, nil), give, s.players(get, sc, nil), s.rules)
		res.hindsight += h.PerWeek / 4
		res.optChange += (s.week[w][me].opt - s.week[w-5][me].opt) / 4
	}
	for _, c := range model.Changes {
		if c.Kind == "benched" {
			res.benched++
			if !next.starters[c.Player] {
				res.benchedOut++
			}
			if pre.starters[c.Player] {
				res.wasStarting++
				if !next.starters[c.Player] {
					res.lostStart++
				}
			}
		}
	}
	return res, true
}

func holds(roster, want []string) bool {
	have := map[string]bool{}
	for _, id := range roster {
		have[id] = true
	}
	for _, id := range want {
		if !have[id] {
			return false
		}
	}
	return true
}

// fit is Pearson r and the least-squares slope of y on x.
func fit(x, y []float64) (r, slope float64) {
	n := float64(len(x))
	var sx, sy, sxx, syy, sxy float64
	for i := range x {
		sx, sy = sx+x[i], sy+y[i]
		sxx, syy, sxy = sxx+x[i]*x[i], syy+y[i]*y[i], sxy+x[i]*y[i]
	}
	cov, vx, vy := sxy-sx*sy/n, sxx-sx*sx/n, syy-sy*sy/n
	return cov / math.Sqrt(vx*vy), cov / vx
}

// signP is the one-sided binomial chance of k or more of n at one half.
func signP(k, n int) float64 {
	var p float64
	for i := k; i <= n; i++ {
		lg, _ := math.Lgamma(float64(n + 1))
		la, _ := math.Lgamma(float64(i + 1))
		lb, _ := math.Lgamma(float64(n - i + 1))
		p += math.Exp(lg - la - lb - float64(n)*math.Ln2)
	}
	return p
}

func say(format string, a ...any) { _, _ = fmt.Fprintf(os.Stdout, format+"\n", a...) }

func (r *replay) report(res []result) {
	var px, hy, oy []float64
	up, upH, upO, benched, out, agree, both, was, lost := 0, 0, 0, 0, 0, 0, 0, 0, 0
	for _, x := range res {
		if x.predicted == 0 && x.hindsight == 0 {
			continue
		}
		px, hy, oy = append(px, x.predicted), append(hy, x.hindsight), append(oy, x.optChange)
		if x.predicted >= 2 {
			up++
			if x.hindsight > 0 {
				upH++
			}
			if x.optChange > 0 {
				upO++
			}
		}
		benched, out = benched+x.benched, out+x.benchedOut
		was, lost = was+x.wasStarting, lost+x.lostStart
		if x.predicted != 0 && x.thisWeek != 0 {
			both++
			if (x.predicted > 0) == (x.thisWeek > 0) {
				agree++
			}
		}
	}
	say("trade sides judged: %d (lineup changed: %d)", len(res), len(px))
	say("(i) predicted >= +2.0: %d; trade's own effect up %d (%.0f%%, p=%.4f); league optimal up %d (%.0f%%, p=%.4f)",
		up, upH, pct(upH, up), signP(upH, up), upO, pct(upO, up), signP(upO, up))
	rh, sh := fit(px, hy)
	ro, so := fit(px, oy)
	say("(ii) vs trade's own effect: r=%.3f slope=%.2f | vs league optimal change: r=%.3f slope=%.2f", rh, sh, ro, so)
	say("(iii) model-benched starters not starting next week: %d of %d (%.0f%%)", out, benched, pct(out, benched))
	say("(iii) of those the owner was starting before the trade: %d of %d lost the start (%.0f%%)", lost, was, pct(lost, was))
	say("(iv) this-week and rest-of-season agree in sign: %d of %d (%.0f%%)", agree, both, pct(agree, both))
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}
