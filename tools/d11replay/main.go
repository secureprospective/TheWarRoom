// Command d11replay walks the D11 native-projection method forward over held nflverse weeks
// scored in the Default32 rules and reports the method's replay bars. Read-only; no store.
package main

import (
	"compress/gzip"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type obs struct {
	season, week int
	pts          float64
}

type params struct {
	dir                            string
	half, prior, minS, lo, hi, cap float64
}

type res struct {
	in, n, nThin int
	se, seRaw    float64
}

type pr struct{ med, act float64 }

type replay struct {
	opp    map[string]string
	given  map[string]map[int]float64
	hist   map[string][]obs
	pos    map[string]leaguerules.Position
	byPos  map[leaguerules.Position]*res
	weekly map[string][]pr
}

func positionOf(raw string) (leaguerules.Position, bool) {
	p, ok := map[string]leaguerules.Position{"QB": "QB", "RB": "RB", "FB": "RB", "WR": "WR", "TE": "TE", "K": "PK",
		"DE": "DE", "DT": "DT", "NT": "DT", "LB": "LB", "OLB": "LB", "ILB": "LB", "MLB": "LB",
		"CB": "CB", "S": "S", "SS": "S", "FS": "S", "SAF": "S", "DB": "S"}[strings.ToUpper(raw)]
	return p, ok
}

func main() {
	var p params
	flag.StringVar(&p.dir, "dir", "", "cache dir with stats_player_week_YYYY.csv.gz")
	flag.Float64Var(&p.half, "half", 8, "recency half-life in weeks")
	flag.Float64Var(&p.prior, "prior", 1.0/3, "prior-season weight")
	flag.Float64Var(&p.minS, "min", 8, "MIN_SCORES shrink constant")
	flag.Float64Var(&p.lo, "lo", 0.10, "floor quantile")
	flag.Float64Var(&p.hi, "hi", 0.90, "ceiling quantile")
	flag.Float64Var(&p.cap, "cap", 0, "MATCHUP_CAP; 0 = no matchup factor (method step 4)")
	flag.Parse()
	r, err := load(p.dir)
	if err != nil {
		log.Fatal(err)
	}
	r.walk(p)
	r.report(p)
}

func load(dir string) (*replay, error) {
	r := &replay{hist: map[string][]obs{}, pos: map[string]leaguerules.Position{},
		byPos: map[leaguerules.Position]*res{}, weekly: map[string][]pr{},
		opp: map[string]string{}, given: map[string]map[int]float64{}}
	rules := leaguerules.Default32Scoring()
	for _, y := range []int{2023, 2024, 2025} {
		path := fmt.Sprintf("%s/stats_player_week_%d.csv.gz", dir, y)
		weeks, err := readSeason(path)
		if err != nil {
			return nil, err
		}
		if err := r.readOpponents(path); err != nil {
			return nil, err
		}
		for _, w := range weeks {
			p, ok := positionOf(w.Position)
			if w.SeasonType != "REG" || !ok {
				continue
			}
			v, _, err := fantasyscore.Score(rules, p, w.Line)
			if err != nil || v.State != leaguerules.IsKnown {
				continue
			}
			r.hist[w.GSIS] = append(r.hist[w.GSIS], obs{w.Season, w.Week, float64(v.Value) / 10000})
			r.pos[w.GSIS] = p
			if o := r.opp[fmt.Sprintf("%s|%d|%d", w.GSIS, w.Season, w.Week)]; o != "" {
				k := fmt.Sprintf("%d|%s|%s", w.Season, o, p)
				if r.given[k] == nil {
					r.given[k] = map[int]float64{}
				}
				r.given[k][w.Week] += float64(v.Value) / 10000
			}
		}
	}
	for _, h := range r.hist {
		sort.Slice(h, func(a, b int) bool { return key(h[a]) < key(h[b]) })
	}
	return r, nil
}

func readSeason(path string) ([]source.PlayerWeek, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gunzip %s: %w", path, err)
	}
	weeks, err := source.ReadWeeks(gz)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return weeks, nil
}

// readOpponents keeps each player-week's opponent; source.ReadWeeks does not carry it.
func (r *replay) readOpponents(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gunzip %s: %w", path, err)
	}
	rows, err := csv.NewReader(gz).ReadAll()
	if err != nil || len(rows) == 0 {
		return fmt.Errorf("csv %s: %w", path, err)
	}
	col := map[string]int{}
	for i, h := range rows[0] {
		col[h] = i
	}
	for _, row := range rows[1:] {
		r.opp[row[col["player_id"]]+"|"+row[col["season"]]+"|"+row[col["week"]]] = row[col["opponent_team"]]
	}
	return nil
}

// allowed is a defense's average points allowed to the position over its last 8 games this
// season before week; ok needs at least 3 games.
func (r *replay) allowed(pos leaguerules.Position, team string, season, week int) (float64, bool) {
	g := r.given[fmt.Sprintf("%d|%s|%s", season, team, pos)]
	var sum float64
	n := 0
	for w := week - 1; w >= 1 && n < 8; w-- {
		if v, ok := g[w]; ok {
			sum += v
			n++
		}
	}
	return sum / float64(max(n, 1)), n >= 3
}

// matchup is the method step-4 multiplier: the opponent's rank in points allowed to the
// position, mapped linearly to [1-cap, 1+cap]. Too little history gives 1.
func (r *replay) matchup(p params, pos leaguerules.Position, opp string, season, week int) float64 {
	mine, ok := r.allowed(pos, opp, season, week)
	if p.cap == 0 || !ok {
		return 1
	}
	below, total := 0, 0
	prefix, suffix := fmt.Sprintf("%d|", season), "|"+string(pos)
	for k := range r.given {
		if !strings.HasPrefix(k, prefix) || !strings.HasSuffix(k, suffix) {
			continue
		}
		if a, ok := r.allowed(pos, strings.TrimSuffix(strings.TrimPrefix(k, prefix), suffix), season, week); ok {
			total++
			if a < mine {
				below++
			}
		}
	}
	if total < 2 {
		return 1
	}
	return 1 + p.cap*(2*float64(below)/float64(total-1)-1)
}

// samples are every player's and every position's weighted weeks strictly before (season, week).
func (r *replay) samples(p params, season, week int) (map[string][]ws, map[leaguerules.Position][]ws) {
	player := map[string][]ws{}
	position := map[leaguerules.Position][]ws{}
	for id, h := range r.hist {
		for _, o := range h {
			if key(o) >= season*100+week {
				break
			}
			if w := weight(o, season, week, p.half, p.prior); w > 0 {
				player[id] = append(player[id], ws{o.pts, w})
				position[r.pos[id]] = append(position[r.pos[id]], ws{o.pts, w})
			}
		}
	}
	return player, position
}

func (r *replay) walk(p params) {
	for _, season := range []int{2024, 2025} {
		for week := 2; week <= 18; week++ {
			player, position := r.samples(p, season, week)
			for id, h := range r.hist {
				if act, ok := scored(h, season, week); ok && len(position[r.pos[id]]) > 0 {
					mult := r.matchup(p, r.pos[id], r.opp[fmt.Sprintf("%s|%d|%d", id, season, week)], season, week)
					r.score(p, id, player[id], position[r.pos[id]], act, mult, fmt.Sprintf("%d|%d|%s", season, week, r.pos[id]))
				}
			}
		}
	}
}

func scored(h []obs, season, week int) (float64, bool) {
	for _, o := range h {
		if o.season == season && o.week == week {
			return o.pts, true
		}
	}
	return 0, false
}

func (r *replay) score(p params, id string, ps, pos []ws, act, mult float64, k string) {
	games := float64(len(ps))
	mix := mixture(ps, pos, games/(games+p.minS))
	f, m, c := quant(mix, p.lo), quant(mix, 0.5)*mult, quant(mix, p.hi)*mult
	s := r.byPos[r.pos[id]]
	if s == nil {
		s = &res{}
		r.byPos[r.pos[id]] = s
	}
	s.n++
	if act >= f && act <= c {
		s.in++
	}
	if games > 0 && games < p.minS {
		s.nThin++
		s.se += (m - act) * (m - act)
		rm := quant(ps, 0.5)
		s.seRaw += (rm - act) * (rm - act)
	}
	r.weekly[k] = append(r.weekly[k], pr{m, act})
}

func (r *replay) report(p params) {
	out := os.Stdout
	var tin, tn int
	_, _ = fmt.Fprintf(out, "half=%.0f prior=%.2f min=%.0f q=[%.2f,%.2f] cap=%.2f\n", p.half, p.prior, p.minS, p.lo, p.hi, p.cap)
	_, _ = fmt.Fprintln(out, "pos   n      coverage  spearman  thinN  shrunkRMSE rawRMSE")
	for _, pos := range []leaguerules.Position{"QB", "RB", "WR", "TE", "PK", "DT", "DE", "LB", "CB", "S"} {
		s := r.byPos[pos]
		if s == nil {
			continue
		}
		tin += s.in
		tn += s.n
		var rhoSum float64
		var rhoN int
		for k, v := range r.weekly {
			if strings.HasSuffix(k, "|"+string(pos)) && len(v) >= 10 {
				rhoSum += spearman(v)
				rhoN++
			}
		}
		thin := float64(max(s.nThin, 1))
		_, _ = fmt.Fprintf(out, "%-4s %6d  %6.1f%%  %8.3f  %5d  %9.2f  %7.2f\n", pos, s.n, 100*float64(s.in)/float64(s.n),
			rhoSum/float64(rhoN), s.nThin, math.Sqrt(s.se/thin), math.Sqrt(s.seRaw/thin))
	}
	_, _ = fmt.Fprintf(out, "ALL  %6d  %6.1f%%\n", tn, 100*float64(tin)/float64(tn))
}

type ws struct{ v, w float64 }

func key(o obs) int { return o.season*100 + o.week }

func weight(o obs, season, week int, half, prior float64) float64 {
	if o.season == season {
		return math.Pow(0.5, float64(week-o.week)/half)
	}
	if season-o.season <= 2 {
		return prior
	}
	return 0
}

// mixture scales the player's sample to share of the total weight, the position's to the rest.
func mixture(player, position []ws, share float64) []ws {
	sum := func(s []ws) float64 {
		t := 0.0
		for _, x := range s {
			t += x.w
		}
		return t
	}
	out := make([]ws, 0, len(player)+len(position))
	if sp := sum(player); sp > 0 && share > 0 {
		for _, x := range player {
			out = append(out, ws{x.v, x.w / sp * share})
		}
	} else {
		share = 0
	}
	sq := sum(position)
	for _, x := range position {
		out = append(out, ws{x.v, x.w / sq * (1 - share)})
	}
	return out
}

func quant(s []ws, q float64) float64 {
	c := append([]ws(nil), s...)
	sort.Slice(c, func(a, b int) bool { return c[a].v < c[b].v })
	var tot float64
	for _, x := range c {
		tot += x.w
	}
	acc := 0.0
	for _, x := range c {
		acc += x.w
		if acc >= q*tot {
			return x.v
		}
	}
	return c[len(c)-1].v
}

func spearman(v []pr) float64 {
	n := len(v)
	a, b := make([]float64, n), make([]float64, n)
	for i, x := range v {
		a[i], b[i] = x.med, x.act
	}
	ra, rb := rank(a), rank(b)
	var ma, mb float64
	for i := range ra {
		ma += ra[i]
		mb += rb[i]
	}
	ma /= float64(n)
	mb /= float64(n)
	var num, da, db float64
	for i := range ra {
		num += (ra[i] - ma) * (rb[i] - mb)
		da += (ra[i] - ma) * (ra[i] - ma)
		db += (rb[i] - mb) * (rb[i] - mb)
	}
	return num / math.Sqrt(da*db)
}

func rank(x []float64) []float64 {
	idx := make([]int, len(x))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return x[idx[a]] < x[idx[b]] })
	r := make([]float64, len(x))
	for i := 0; i < len(idx); {
		j := i
		for j+1 < len(idx) && x[idx[j+1]] == x[idx[i]] {
			j++
		}
		for k := i; k <= j; k++ {
			r[idx[k]] = float64(i+j) / 2
		}
		i = j + 1
	}
	return r
}
