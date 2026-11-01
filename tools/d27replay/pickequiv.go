package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/pickvalue"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// teams is league 14432's size; bands are fifths of a round, which is how owners word a pick.
const (
	teams      = 32
	bandsRound = 5
	rounds     = 5
)

// pickModel is Pick Value fitted as of a trade: the model run's values and records for the
// trade's season, and only the league drafts held then (before the season's draft for an
// offseason trade, through it in season).
func (r *replay) pickModel(key string, season int, offseason bool) (pickvalue.Model, error) {
	mkey := fmt.Sprintf("%s|%d|%t", key, season, offseason)
	if m, ok := r.picks[mkey]; ok {
		return m, nil
	}
	all, err := pickvalue.History()
	if err != nil {
		return pickvalue.Model{}, fmt.Errorf("pick history: %w", err)
	}
	held := all[:0:0]
	for _, p := range all {
		if p.Year < season || (!offseason && p.Year == season) {
			held = append(held, p)
		}
	}
	discount, err := params.DefaultSet().GetGlobal(params.KeyPickValueDiscount)
	if err != nil {
		return pickvalue.Model{}, fmt.Errorf("discount: %w", err)
	}
	m, err := pickvalue.Fit(held, r.players[key], season, discount)
	if err != nil {
		return pickvalue.Model{}, fmt.Errorf("fit %s: %w", mkey, err)
	}
	r.picks[mkey] = m
	return m, nil
}

// pickPPG values one MFL pick string: DP_r_s is this draft's round r+1 slot s+1 (both 0-based);
// FP_franchise_year_round a future round with its slot unknown.
func pickPPG(m pickvalue.Model, id string, season int) (float64, bool) {
	f := strings.Split(id, "_")
	var year, round, slot int
	var err1, err2 error
	switch {
	case f[0] == "DP" && len(f) == 3:
		year = season
		round, err1 = strconv.Atoi(f[1])
		slot, err2 = strconv.Atoi(f[2])
		round, slot = round+1, slot+1
	case f[0] == "FP" && len(f) == 4:
		year, err1 = strconv.Atoi(f[2])
		round, err2 = strconv.Atoi(f[3])
	default:
		return 0, false
	}
	if err1 != nil || err2 != nil || year < season {
		return 0, false
	}
	v, err := m.Pick(year, round, slot, teams)
	if err != nil || v.Unavailable {
		return 0, false
	}
	return v.PPG, true
}

// band is the fifth-of-a-round of this draft whose pick is closest in value to v: 0 is the top
// fifth of round 1. A value above the first pick is band 0; below the last, the last band.
func band(m pickvalue.Model, v float64) int {
	best, bestGap := 0, math.Inf(1)
	for n := 1; n <= rounds*teams; n++ {
		if gap := math.Abs(m.Selection(n, 0).PPG - v); gap < bestGap {
			best, bestGap = n, gap
		}
	}
	return (best - 1) * bandsRound / teams
}

// sumPPG adds the model values of players; ok is false when one was not valued.
func sumPPG(ids []string, vals map[string]value) (float64, bool) {
	sum := 0.0
	for _, p := range ids {
		v, ok := vals[p]
		if !ok || !v.valid {
			return 0, false
		}
		sum += v.ppg
	}
	return sum, true
}
