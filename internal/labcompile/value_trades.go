package labcompile

import "slices"

type draftPayKey struct{ Year, Round int64 }
type tradeValueItem struct {
	valuedAsset
	Asset  Asset
	Valued bool
	Real   []*float64
}
type tradeValue struct {
	Index  int
	Season int64
	Done   int
	Window string
	Sides  [2][]tradeValueItem
}

func rookieSalaries(l *Lab, draft []Pick) map[draftPayKey]float64 {
	seen := map[draftPayKey][]float64{}
	for _, d := range draft {
		for week := int64(1); week <= 17; week++ {
			c, ok := l.Rosters[teamWeek{d.Year, week, d.Fid}].Players[d.Pid]
			if !ok {
				continue
			}
			if c.Salary != nil {
				key := draftPayKey{d.Year, d.Round}
				seen[key] = append(seen[key], *c.Salary)
			}
			break
		}
	}
	out := map[draftPayKey]float64{}
	for key, values := range seen {
		slices.Sort(values)
		n := len(values)
		if n%2 == 1 {
			out[key] = values[n/2]
		} else {
			out[key] = (values[n/2-1] + values[n/2]) / 2
		}
	}
	return out
}
func contractCost(v *valuer, a Asset, season int64, done int, giver, receiver string, ts, year int64, pay map[draftPayKey]float64) []float64 {
	if a.Kind == "pick" {
		return rookieCost(a, season, pay)
	}
	c := v.wins.lab.contract(value(a.ID), year, ts, giver)
	if c == nil {
		c = v.wins.lab.contractAfter(value(a.ID), year, ts, receiver)
	}
	if c == nil || c.Salary == nil || value(c.Until) == 0 {
		return nil
	}
	out := make([]float64, horizon)
	for k := range out {
		if season+int64(k) > *c.Until {
			continue
		}
		share := 1.0
		if k == 0 && done > 0 {
			weeks := v.weeks(season)
			if weeks == 0 {
				return nil
			}
			share -= float64(done) / float64(weeks)
		}
		out[k] = *c.Salary * share
	}
	return out
}
func rookieCost(a Asset, season int64, pay map[draftPayKey]float64) []float64 {
	salary, ok := pay[draftPayKey{value(a.Year), value(a.Round)}]
	if !ok || salary == 0 {
		last := int64(0)
		for key := range pay {
			last = max(last, key.Year)
		}
		salary, ok = pay[draftPayKey{last, value(a.Round)}]
	}
	if !ok {
		return nil
	}
	term := int64(3)
	if value(a.Round) == 1 {
		term = 5
	}
	out := make([]float64, horizon)
	for k := range out {
		age := season + int64(k) - value(a.Year)
		if age >= 0 && age < term {
			out[k] = salary
		}
	}
	return out
}
func tradeWindow(phase string) string {
	switch phase {
	case "opening", "between", "winter", "closed":
		return "spring"
	case "rookieDraft", "summer":
		return "summer"
	default:
		return "season"
	}
}
func valueTrades(l *Lab, trades []Trade, v *valuer, pay map[draftPayKey]float64) []tradeValue {
	indices := make([]int, len(trades))
	for i := range indices {
		indices[i] = i
	}
	slices.SortStableFunc(indices, func(a, b int) int {
		if trades[a].Ts < trades[b].Ts {
			return -1
		}
		if trades[a].Ts > trades[b].Ts {
			return 1
		}
		return 0
	})
	out := []tradeValue{}
	for _, index := range indices {
		t := trades[index]
		if t.Year < 2017 || t.OneSided || !valueableKinds(t) {
			continue
		}
		season, done := v.frame(t.Year, t.Ts)
		row := tradeValue{Index: index, Season: season, Done: done, Window: tradeWindow(l.phase(t.Year, t.Ts))}
		row.Sides[0] = valueSide(t.AGave, t.A, t.B, t, v, season, done, pay)
		row.Sides[1] = valueSide(t.BGave, t.B, t.A, t, v, season, done, pay)
		out = append(out, row)
	}
	return out
}
func valueableKinds(t Trade) bool {
	for _, side := range [][]Asset{t.AGave, t.BGave} {
		for _, a := range side {
			if a.Kind != "player" && a.Kind != "pick" {
				return false
			}
		}
	}
	return true
}
func valueSide(side []Asset, giver, receiver string, t Trade, v *valuer, season int64, done int, pay map[draftPayKey]float64) []tradeValueItem {
	out := make([]tradeValueItem, 0, len(side))
	for _, a := range side {
		var forecast []float64
		var actual []*float64
		if a.Kind == "player" {
			forecast = v.player(value(a.ID), season, done)
			actual = v.realizedPlayer(value(a.ID), season, done)
		} else {
			forecast = v.pick(a, season)
			actual = v.realizedPick(a, season)
		}
		valued := forecast != nil
		if forecast == nil {
			forecast = make([]float64, horizon)
		}
		cost := contractCost(v, a, season, done, giver, receiver, t.Ts, t.Year, pay)
		out = append(out, tradeValueItem{valuedAsset: valuedAsset{forecast, assetClass(a, v, season, t.Ts), cost}, Asset: a, Valued: valued, Real: actual})
	}
	return out
}
func commonSeasons(v tradeValue) []int {
	out := []int{}
	for k := range horizon {
		if allRealAt(v, k) {
			out = append(out, k)
		}
	}
	return out
}
func allRealAt(v tradeValue, k int) bool {
	for _, side := range v.Sides {
		for _, item := range side {
			if item.Real == nil || item.Real[k] == nil {
				return false
			}
		}
	}
	return true
}

func fillValueLegs(a *Analysis, rows []tradeValue, prices map[string]MarketClass, perAsset float64) {
	a.Legs = []AnalysisLeg{}
	a.Deals = []AnalysisDeal{}
	for _, row := range rows {
		common := commonSeasons(row)
		sums := [2][2]float64{}
		for side, items := range row.Sides {
			for index, it := range items {
				x := discounted(it.Value, discount)
				var actual *float64
				if len(common) > 0 {
					r := realizedCommon(it, common)
					actual = ptr(pythonRound(r, 3))
					sums[side][1] += r
				}
				a.Legs = append(a.Legs, AnalysisLeg{Trade: row.Index, Side: side, Asset: index, Value: ptr(pythonRound(x, 3)), Class: ptr(it.Class), Price: ptr(pythonRound(x*prices[it.Class].Price+perAsset, 3)), Real: actual, RealSeasons: ptr(int64(len(common)))})
				sums[side][0] += x
			}
		}
		var net *float64
		if len(common) > 0 {
			net = ptr(pythonRound(sums[1][1]-sums[0][1], 3))
		}
		a.Deals = append(a.Deals, AnalysisDeal{Trade: row.Index, ANet: ptr(pythonRound(sums[1][0]-sums[0][0], 3)), ARealNet: net, RealSeasons: ptr(int64(len(common))), Window: ptr(row.Window), Season: ptr(row.Season)})
	}
}
