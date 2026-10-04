package model

import (
	"math"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// PriorFeatures names a position's prior inputs, in the order PriorInputs returns them: draft
// capital (log of the overall pick), age entering the league, the combine, and college production.
// College production is one number at most positions; DT reads each defensive share on its own
// and DE adds his best college season and how many he played (each beat the single number on
// every holdout season 2023–2025, positions as the league lists them).
func PriorFeatures(pos domain.Position) []string {
	base := append([]string{"draft", "entry_age"}, Combine()...)
	switch pos {
	case domain.PosDT:
		return append(base, dtShares()...)
	case domain.PosDE:
		return append(base, "college", "college_best", "college_seasons")
	case domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK, domain.PosLB, domain.PosCB,
		domain.PosS, domain.PosFlag:
	}
	return append(base, "college")
}

// dtShares are DT's college inputs: his share of his team's each, in his last college season.
func dtShares() []string {
	return []string{"college_sacks", "college_tackles_for_loss", "college_total_tackles",
		"college_passes_defended", "college_interceptions", "college_qb_hurries"}
}

// PriorInputs returns the player's prior inputs in PriorFeatures order and whether each is known.
// An undrafted player's draft input is unknown, so the prior's missing term carries undrafted.
func (p *Player) PriorInputs() (x []float64, known []bool) {
	names := PriorFeatures(p.Position)
	x, known = make([]float64, len(names)), make([]bool, len(names))
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
	college := 2 + len(Combine())
	if p.Position == domain.PosDT {
		c := p.College[p.lastCollegeSeason()]
		for i, name := range dtShares() {
			stat := strings.TrimPrefix(name, "college_")
			set(college+i, ratio(c[stat], c["team_"+stat]), c != nil)
		}
		return x, known
	}
	share, ok := p.collegeProduction()
	set(college, share, ok)
	if p.Position == domain.PosDE {
		best, n := p.bestPassRush()
		set(college+1, best, n > 0)
		set(college+2, float64(n), n > 0)
	}
	return x, known
}

// lastCollegeSeason is the latest college season before the player's first NFL season, 0 if none.
func (p *Player) lastCollegeSeason() int {
	last := 0
	for yr := range p.College {
		if yr > last && (p.Rookie == 0 || yr < p.Rookie) {
			last = yr
		}
	}
	return last
}

// bestPassRush is the mean of a DE's best college share of his team's sacks and best share of
// its tackles for loss, each over every college season before his first NFL one, and how many
// such seasons there were.
func (p *Player) bestPassRush() (best float64, seasons int) {
	sacks, tfl := math.NaN(), math.NaN()
	for yr, c := range p.College {
		if p.Rookie != 0 && yr >= p.Rookie {
			continue
		}
		seasons++
		sacks = maxKnown(sacks, ratio(c["sacks"], c["team_sacks"]))
		tfl = maxKnown(tfl, ratio(c["tackles_for_loss"], c["team_tackles_for_loss"]))
	}
	return mean(sacks, tfl), seasons
}

// maxKnown is the larger of a and b, ignoring a NaN.
func maxKnown(a, b float64) float64 {
	switch {
	case math.IsNaN(a):
		return b
	case math.IsNaN(b):
		return a
	}
	return math.Max(a, b)
}

// collegeProduction is the player's last college season's production for his position: his share
// of the team's yards at RB, WR and TE, yards per attempt at QB, field goal rate at K, and the
// mean of his defensive shares elsewhere.
func (p *Player) collegeProduction() (float64, bool) {
	c := p.College[p.lastCollegeSeason()]
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
