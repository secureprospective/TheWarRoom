package labcompile

import "time"

type valuer struct {
	wins             *winsModel
	forecast         *forecastModel
	picks            *picksModel
	known, estimated map[teamYear]int64
	playedThrough    int64
}

func newValuer(w *winsModel, f *forecastModel, p *picksModel, known, estimated map[teamYear]int64, playedThrough int64) *valuer {
	if playedThrough == 0 {
		playedThrough = f.last
	}
	return &valuer{w, f, p, known, estimated, playedThrough}
}
func (v *valuer) frame(year, ts int64) (int64, int) {
	if end := v.wins.lab.Marks[year].RegularEnd; end != nil && dayOf(ts).After(*end) {
		return year + 1, 0
	}
	done := 0
	for _, week := range v.wins.played[year] {
		if end := v.wins.lab.WeekEnd[yearWeek{year, week}]; end != 0 && end+daySeconds <= ts {
			done++
		}
	}
	return year, done
}
func (v *valuer) weeks(season int64) int64 {
	if season <= v.forecast.last {
		return v.wins.lab.Regular[season]
	}
	return 13
}
func (v *valuer) player(pid string, season int64, done int) []float64 {
	rates := v.forecast.rates(pid, season-1)
	if rates == nil {
		return nil
	}
	if done > 0 {
		played := v.wins.played[season]
		seen := played[:min(done, len(played))]
		rates[0] = (float64(done)*v.wins.rate(pid, season, seen) + 6*rates[0]) / float64(done+6)
	}
	out := make([]float64, horizon)
	for k, r := range rates {
		weeks := v.weeks(season + int64(k))
		if k == 0 {
			weeks -= int64(done)
		}
		out[k] = r * float64(max(0, weeks))
	}
	return out
}
func (v *valuer) pickSlot(asset Asset) int64 {
	if slot := value(asset.Slot); slot != 0 {
		return slot
	}
	return v.known[teamYear{value(asset.Year), value(asset.Orig)}]
}
func (v *valuer) pick(asset Asset, season int64) []float64 {
	slot := v.pickSlot(asset)
	made := v.picks.overall[pickKey{value(asset.Year), value(asset.Round), slot}]
	rates := v.picks.rates(value(asset.Round), made.Overall)
	if rates == nil {
		return nil
	}
	out := shiftRates(rates, int(value(asset.Year)-season))
	for k := range out {
		out[k] *= float64(v.weeks(season + int64(k)))
	}
	return out
}
func (v *valuer) realizedPlayer(pid string, season int64, done int) []*float64 {
	out := make([]*float64, horizon)
	for k := range out {
		year := season + int64(k)
		if year > v.playedThrough {
			continue
		}
		weeks := v.wins.played[year]
		if k == 0 {
			weeks = weeks[min(done, len(weeks)):]
		}
		values := make([]float64, len(weeks))
		for i, week := range weeks {
			values[i] = v.wins.week[playerYear{pid, year}][week]
		}
		out[k] = ptr(pythonSum(values))
	}
	return out
}
func (v *valuer) realizedPick(asset Asset, season int64) []*float64 {
	slot := v.pickSlot(asset)
	made, ok := v.picks.overall[pickKey{value(asset.Year), value(asset.Round), slot}]
	if slot == 0 || !ok {
		return nil
	}
	return v.realizedPlayer(made.Pid, season, 0)
}
func nextDraft(l *Lab, ts int64) int64 {
	date := dayOf(ts)
	year := int64(date.Year())
	end := l.Marks[year].DraftEnd
	if end == nil {
		end = ptrDate(time.Date(date.Year(), time.June, 1, 0, 0, 0, 0, time.UTC))
	}
	if !date.After(*end) {
		return year
	}
	return year + 1
}
func assetClass(a Asset, v *valuer, season, ts int64) string {
	if a.Kind == "pick" {
		round := "3"
		switch value(a.Round) {
		case 1:
			round = "1"
		case 2:
			round = "2"
		}
		when := "later"
		if value(a.Year) == nextDraft(v.wins.lab, ts) {
			when = "next"
		}
		return "pick" + round + ":" + when
	}
	pid := value(a.ID)
	pos := v.wins.lab.pos(pid, season)
	if pos == nil {
		pos = v.wins.lab.pos(pid, season-1)
	}
	age := playerAge(v.wins.lab, pid, season)
	if pos == nil || age == nil || ageBand(*age) == "" {
		return "player:unknown"
	}
	return value(positionGroup(pos)) + ":" + ageBand(*age)
}
