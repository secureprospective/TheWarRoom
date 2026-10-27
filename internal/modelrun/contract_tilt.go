package modelrun

import (
	"math"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Contract tilt: in a 32-team dynasty a cheap, long deal is worth more than the same player on
// an expensive one. Each rostered player's going rate is the median salary of the rostered
// players nearest him in dynasty value, all positions together; his value moves by how far
// under or over it he is paid, counted fully at three or more years left, and never by more
// than the cap (dynasty.contract_tilt).
const (
	tiltNeighbours = 25   // players either side whose salaries set the going rate
	tiltStrength   = 0.10 // tilt per unit of (going rate − salary) / going rate, before the cap
	tiltFullYears  = 3    // years left at which the tilt counts in full
)

// contractTilt applies the tilt to scores in place and returns each tilted player's tilt.
// Players without a positive salary (free agents, unknown contracts) are left as valued and do
// not set going rates.
func contractTilt(scores []history.ModelScore, roster map[string]state.PlayerState, season int, maxTilt float64,
	scale func(pos domain.Position) func(ppg float64) float64) map[string]float64 {
	if maxTilt <= 0 {
		return nil
	}
	var paid []int
	for i, sc := range scores {
		if p, ok := roster[sc.MFLID]; ok && p.Salary > 0 {
			paid = append(paid, i)
		}
	}
	slices.SortStableFunc(paid, func(a, b int) int {
		return compareDesc(scores[a].DynastyPPG, scores[b].DynastyPPG)
	})
	salaries := make([]float64, len(paid))
	for k, i := range paid {
		salaries[k] = float64(roster[scores[i].MFLID].Salary)
	}
	tilts := map[string]float64{}
	for k, i := range paid {
		sc := &scores[i]
		p := roster[sc.MFLID]
		rate := median(salaries[max(0, k-tiltNeighbours):min(len(salaries), k+tiltNeighbours+1)])
		years := min(float64(max(p.ExpirationYear-season+1, 1)), tiltFullYears) / tiltFullYears
		tilt := math.Max(-maxTilt, math.Min(maxTilt, tiltStrength*(rate-float64(p.Salary))/rate*years))
		v := &sc.Value
		v.DynastyPPG *= 1 + tilt
		if pct := scale(domain.Position(sc.Position)); pct != nil {
			v.Dynasty = pct(v.DynastyPPG)
		}
		tilts[sc.MFLID] = tilt
	}
	return tilts
}

func compareDesc(a, b float64) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	}
	return 0
}

func median(xs []float64) float64 {
	s := slices.Clone(xs)
	slices.Sort(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}
