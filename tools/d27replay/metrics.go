package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/teamneeds"
)

func say(format string, a ...any) { _, _ = fmt.Fprintf(os.Stdout, format+"\n", a...) }

// tally is a pass rate.
type tally struct{ hit, n int }

func (t *tally) add(ok bool) {
	t.n++
	if ok {
		t.hit++
	}
}
func (t tally) String() string {
	if t.n == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%d/%d = %.0f%% (sign p=%.3f)", t.hit, t.n, 100*float64(t.hit)/float64(t.n), signP(t.hit, t.n))
}

// signP is the one-sided binomial chance of k or more of n at a coin's odds.
func signP(k, n int) float64 {
	p := 0.0
	for i := k; i <= n; i++ {
		lg, _ := math.Lgamma(float64(n + 1))
		a, _ := math.Lgamma(float64(i + 1))
		b, _ := math.Lgamma(float64(n - i + 1))
		p += math.Exp(lg - a - b - float64(n)*math.Ln2)
	}
	return p
}

func (r *replay) run(ctx context.Context) error {
	var surplus, sold, bought tally
	var critical, baseline tally
	skipped, judged := 0, 0
	for _, s := range r.seasons {
		for _, t := range s.trades {
			if (t.aPicks > 0) == (t.bPicks > 0) {
				continue // the bar reads trades where exactly one side paid picks
			}
			ros, ok := s.before(t)
			if !ok {
				skipped++
				continue
			}
			y, w := asOf(t)
			vals, err := r.valuesAt(ctx, s, y, w)
			if err != nil {
				return err
			}
			lg := league(ros, vals, s.pos)
			// seller: received the picks, so sent the players.
			seller, sGave, payer, pGave := t.a, t.aGave, t.b, t.bGave
			if t.aPicks > 0 {
				seller, sGave, payer, pGave = t.b, t.bGave, t.a, t.aGave
			}
			if len(sGave) == 0 {
				continue
			}
			judged++
			surplus.add(sideSurplus(lg.Teams[seller], sGave, vals, s.pos) > sideSurplus(lg.Teams[payer], pGave, vals, s.pos))
			for _, p := range sGave {
				sold.add(!lg.Teams[seller].InBest(p))
			}
			for _, p := range pGave {
				bought.add(!lg.Teams[payer].InBest(p))
			}
		}
		if err := r.needs(ctx, s, &critical, &baseline); err != nil {
			return err
		}
	}
	say("")
	say("D27 replay, league 14432, Best %d, model DynastyPPG", bestN)
	say("Bar 1 (>= 70%%): one-sided pick trades judged %d (%d skipped: export and ledger disagree)", judged, skipped)
	say("  pick receiver had more surplus at the positions it sold than the payer had at its own: %s", surplus)
	say("  players sold for picks that sat outside the seller's Best %d: %s", bestN, sold)
	say("  (contrast) players the pick payer also sent that sat outside its Best %d: %s", bestN, bought)
	say("Bar 2 (>= 50%%): critical needs at week 8 addressed by week 13 with a Best-%d-grade player: %s", bestN, critical)
	say("  (baseline) same rate at positions with no need: %s", baseline)
	return nil
}

// sideSurplus is the franchise's surplus value summed over the positions of the players it sent.
func sideSurplus(t teamneeds.Team, gave []string, vals map[string]value, pos func(string) string) float64 {
	seen := map[string]bool{}
	sum := 0.0
	for _, p := range gave {
		position := vals[p].pos
		if position == "" {
			position = pos(p)
		}
		if !seen[position] {
			seen[position] = true
			sum += t.SurplusAt(position)
		}
	}
	return sum
}

// needs grades every franchise at week 8 and asks whether each critical need was met by week
// 13: a player at that position on the week-13 roster, not on week 8's, valued at week 8 at
// least the franchise's 21st-best value (he would make its Best 21).
func (r *replay) needs(ctx context.Context, s *season, critical, baseline *tally) error {
	vals, err := r.valuesAt(ctx, s, s.year, 7)
	if err != nil {
		return err
	}
	at8, at13 := s.rosters[8], s.rosters[13]
	lg := league(at8, vals, s.pos)
	say("week-8 template %d: %v", s.year, lg.Template)
	anyAdd := tally{}
	defer func() {
		say("  (diagnostic, not the bar) %d critical needs met by ANY added player at the position: %s", s.year, anyAdd)
	}()
	for fid, t := range lg.Teams {
		if len(t.Best) < bestN {
			continue
		}
		met, added := additions(at8[fid], at13[fid], vals, s.pos, vals[t.Best[bestN-1]].ppg)
		crit := map[string]bool{}
		for _, n := range t.Needs {
			if n.Grade == teamneeds.Critical {
				crit[n.Position] = true
				critical.add(met[n.Position])
				anyAdd.add(added[n.Position])
			}
		}
		for pos := range lg.Template {
			if !crit[pos] && !needed(t, pos) {
				baseline.add(met[pos])
			}
		}
	}
	return nil
}

// additions are the positions of players on the later roster and not the earlier: met counts
// only those valued at least bar, added every one.
func additions(before, after []string, vals map[string]value, pos func(string) string, bar float64) (met, added map[string]bool) {
	met, added = map[string]bool{}, map[string]bool{}
	for _, p := range after {
		if slices.Contains(before, p) {
			continue
		}
		position := vals[p].pos
		if position == "" {
			position = pos(p)
		}
		added[position] = true
		if vals[p].valid && vals[p].ppg >= bar {
			met[position] = true
		}
	}
	return met, added
}

func needed(t teamneeds.Team, pos string) bool {
	for _, n := range t.Needs {
		if n.Position == pos {
			return true
		}
	}
	return false
}
