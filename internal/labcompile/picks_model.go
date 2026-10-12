package labcompile

import "slices"

type valuedPick struct {
	Pick
	Overall int
	Seasons []*float64
}
type picksModel struct {
	picks, unseen []valuedPick
	rounds        map[int64][]int
	curve         [][]float64
	overall       map[pickKey]valuedPick
}

func newPicks(w *winsModel, draft []Pick, last int64) *picksModel {
	p := &picksModel{rounds: map[int64][]int{}, overall: map[pickKey]valuedPick{}}
	order := map[int64]int{}
	draft = slices.Clone(draft)
	slices.SortFunc(draft, func(a, b Pick) int {
		if a.Year != b.Year {
			return int(a.Year - b.Year)
		}
		if a.Round != b.Round {
			return int(a.Round - b.Round)
		}
		return int(a.Slot - b.Slot)
	})
	maxOverall := 0
	for _, d := range draft {
		order[d.Year]++
		maxOverall = max(maxOverall, order[d.Year])
		seasons := make([]*float64, horizon)
		for k := range horizon {
			if d.Year+int64(k) <= last {
				seasons[k] = ptr(w.rate(d.Pid, d.Year+int64(k), nil))
			}
		}
		vp := valuedPick{Pick: d, Overall: order[d.Year], Seasons: seasons}
		p.overall[pickKey{d.Year, d.Round, d.Slot}] = vp
		if d.Year > last+1 {
			p.unseen = append(p.unseen, vp)
		} else {
			p.picks = append(p.picks, vp)
			p.rounds[d.Round] = append(p.rounds[d.Round], vp.Overall)
		}
	}
	for overall := 1; overall <= maxOverall; overall++ {
		seasons := make([]*float64, horizon)
		for k := range seasons {
			seasons[k] = p.smooth(overall, k, nil)
		}
		p.curve = append(p.curve, carryRates(seasons))
	}
	return p
}
func (p *picksModel) smooth(overall, season int, excludeYear *int64) *float64 {
	points := [][2]float64{}
	for _, pick := range p.picks {
		if pick.Seasons[season] != nil && (excludeYear == nil || pick.Year != *excludeYear) {
			points = append(points, [2]float64{float64(pick.Overall), *pick.Seasons[season]})
		}
	}
	return localLinear(points, float64(overall), 8)
}
func (p *picksModel) rates(round int64, overall int) []float64 {
	if len(p.curve) == 0 || len(p.rounds) == 0 {
		return nil
	}
	if overall > 0 {
		return slices.Clone(p.curve[min(overall, len(p.curve))-1])
	}
	slots := p.rounds[round]
	if len(slots) == 0 {
		last := int64(0)
		for r := range p.rounds {
			last = max(last, r)
		}
		slots = p.rounds[last]
	}
	out := make([]float64, horizon)
	for k := range out {
		values := make([]float64, len(slots))
		for i, o := range slots {
			values[i] = p.curve[o-1][k]
		}
		out[k] = pythonSum(values) / float64(len(slots))
	}
	return out
}
func (p *picksModel) expectations() ([]AnalysisPick, []AnalysisPick) {
	slots, seasons := []AnalysisPick{}, []AnalysisPick{}
	for _, pick := range p.picks {
		expected := make([]*float64, horizon)
		for k := range expected {
			expected[k] = p.smooth(pick.Overall, k, &pick.Year)
		}
		slots = append(slots, AnalysisPick{pick.Year, pick.Round, pick.Slot, expected})
		seasons = append(seasons, AnalysisPick{pick.Year, pick.Round, pick.Slot, pick.Seasons})
	}
	return slots, seasons
}
