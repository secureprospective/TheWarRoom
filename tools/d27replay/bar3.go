package main

import "fmt"

// bar3 is D27 bar 3: the model's equivalence label for what the pick receiver sent lands within
// one fifth-of-a-round band of what it received (the picks plus any players the payer added),
// both valued as of the trade.
type bar3 struct {
	within, pure, exact, over, top tally
	skipped                        int
	gaps                           map[int]int
}

func (r *replay) equivalence(eq *bar3, t trade, key string, sent, back, paid []string, vals map[string]value) error {
	m, err := r.pickModel(key, t.year, t.week <= 1)
	if err != nil {
		return err
	}
	vSent, ok1 := sumPPG(sent, vals)
	vBack, ok2 := sumPPG(back, vals)
	vPicks, ok3 := 0.0, true
	for _, id := range paid {
		v, ok := pickPPG(m, id, t.year)
		vPicks, ok3 = vPicks+v, ok3 && ok
	}
	if !ok1 || !ok2 || !ok3 {
		eq.skipped++
		return nil
	}
	bs, br := band(m, vSent), band(m, vPicks+vBack)
	gap := bs - br
	if eq.gaps == nil {
		eq.gaps = map[int]int{}
	}
	eq.gaps[max(min(gap, 5), -5)]++
	eq.within.add(gap >= -1 && gap <= 1)
	eq.exact.add(gap == 0)
	eq.top.add(bs == 0 && br == 0)
	eq.over.add(vSent > vPicks+vBack)
	if len(back) == 0 {
		eq.pure.add(gap >= -1 && gap <= 1)
	}
	return nil
}

func (eq bar3) report() {
	say("Bar 3 (>= 60%%): sent vs received equivalence labels within one fifth-of-a-round band: %s (%d skipped: an unvalued player or pick)", eq.within, eq.skipped)
	say("  same band exactly: %s; picks-only trades (payer sent no players): %s", eq.exact, eq.pure)
	say("  both sides at or above the 1.01 pick (band saturates, so a match says nothing): %s", eq.top)
	say("  model valued what the pick receiver sent above what it got: %s", eq.over)
	line := ""
	for g := -5; g <= 5; g++ {
		line += fmt.Sprintf(" %+d:%d", g, eq.gaps[g])
	}
	say("  band gap (sent minus received; negative = sent rated a higher pick, ±5 = 5 or more):%s", line)
}
