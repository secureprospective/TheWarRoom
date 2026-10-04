package model

import (
	"math"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// PriorFeatures names the prior's inputs, in the order PriorInputs returns them: draft capital
// (log of the overall pick), age entering the league, the combine, and college production.
func PriorFeatures() []string {
	return append(append([]string{"draft", "entry_age"}, Combine()...), "college")
}

// PriorInputs returns the player's prior inputs in PriorFeatures order and whether each is known.
// An undrafted player's draft input is unknown, so the prior's missing term carries undrafted.
func (p *Player) PriorInputs() (x []float64, known []bool) {
	n := len(PriorFeatures())
	x, known = make([]float64, n), make([]bool, n)
	set := func(i int, v float64, ok bool) {
		if ok && !math.IsNaN(v) && !math.IsInf(v, 0) {
			x[i], known[i] = v, true
		}
	}
	set(0, math.Log(p.DraftPick), p.DraftPick >= 1)
	set(1, p.AgeAt(p.Rookie), p.Rookie != 0)
	for i, c := range Combine() {
		v, ok := p.Combine[c]
		set(2+i, v, ok)
	}
	share, ok := p.collegeProduction()
	set(n-1, share, ok)
	return x, known
}

// collegeProduction is the player's last college season's production for his position: his share
// of the team's yards at RB, WR and TE, yards per attempt at QB, field goal rate at K, and the
// mean of his defensive shares elsewhere.
func (p *Player) collegeProduction() (float64, bool) {
	last := 0
	for yr := range p.College {
		if yr > last && (p.Rookie == 0 || yr < p.Rookie) {
			last = yr
		}
	}
	c := p.College[last]
	if c == nil {
		return 0, false
	}
	share := func(own string) float64 { return ratio(c[own], c["team_"+own]) }
	switch p.Position {
	case domain.PosWR, domain.PosTE:
		return known(share("receiving_yards"))
	case domain.PosRB:
		return known(ratio(c["receiving_yards"]+c["rushing_yards"], c["team_receiving_yards"]+c["team_rushing_yards"]))
	case domain.PosQB:
		return known(ratio(c["passing_yards"], c["pass_attempts"]))
	case domain.PosK:
		return known(ratio(c["fg_made"], c["fg_attempts"]))
	case domain.PosDT, domain.PosDE:
		return known(mean(share("tackles_for_loss"), share("sacks")))
	case domain.PosLB:
		return known(mean(share("total_tackles"), share("sacks"), share("tackles_for_loss")))
	case domain.PosCB:
		return known(mean(share("passes_defended"), share("interceptions")))
	case domain.PosS:
		return known(mean(share("interceptions"), share("total_tackles")))
	case domain.PosFlag:
	}
	return 0, false
}

// ratio is a/b, or NaN when b is not positive.
func ratio(a, b float64) float64 {
	if b <= 0 {
		return math.NaN()
	}
	return a / b
}

// mean is the mean of vals; NaN if any is NaN.
func mean(vals ...float64) float64 {
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func known(v float64) (float64, bool) {
	return v, !math.IsNaN(v) && !math.IsInf(v, 0)
}
