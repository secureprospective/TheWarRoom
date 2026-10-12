package labcompile

import (
	"fmt"
	"slices"
	"time"
)

// Markers uses civil UTC midnights for Eastern calendar dates; these are not instants.
type Markers struct {
	Open, NFLDraft, DraftStart, DraftEnd, Kickoff, Deadline, RegularEnd, SeasonEnd *time.Time
	DeadlineExpected                                                               bool
}

func easternOffset(t time.Time) int64 {
	year := t.Year()
	var start, end time.Time
	if year >= 2007 {
		start = nthSunday(year, time.March, 2, 7)
		end = nthSunday(year, time.November, 1, 6)
	} else {
		start = nthSunday(year, time.April, 1, 7)
		if year < 1987 {
			start = lastSunday(year, time.April, 7)
		}
		end = lastSunday(year, time.October, 6)
		if year == 1974 {
			start = time.Date(year, time.January, 6, 7, 0, 0, 0, time.UTC)
		}
		if year == 1975 {
			start = time.Date(year, time.February, 23, 7, 0, 0, 0, time.UTC)
		}
	}
	if !t.Before(start) && t.Before(end) {
		return -4 * 3600
	}
	return -5 * 3600
}
func nthSunday(year int, month time.Month, n, hour int) time.Time {
	d := time.Date(year, month, 1, hour, 0, 0, 0, time.UTC)
	return d.AddDate(0, 0, (7-int(d.Weekday()))%7+(n-1)*7)
}
func lastSunday(year int, month time.Month, hour int) time.Time {
	d := time.Date(year, month+1, 0, hour, 0, 0, 0, time.UTC)
	return d.AddDate(0, 0, -int(d.Weekday()))
}
func dayOf(ts int64) time.Time {
	t := time.Unix(ts, 0).UTC()
	t = t.Add(time.Duration(easternOffset(t)) * time.Second)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
func easternEpoch(d time.Time) int64 {
	candidate := d.Add(5 * time.Hour)
	return d.Unix() - easternOffset(candidate)
}
func dateAt(ts int64) *time.Time {
	if ts == 0 {
		return nil
	}
	d := dayOf(ts)
	return &d
}
func nflDraftDate(y int64) time.Time {
	dates := [][2]int{{4, 25}, {5, 8}, {4, 30}, {4, 28}, {4, 27}, {4, 26}, {4, 25}, {4, 23}, {4, 29}, {4, 28}, {4, 27}, {4, 25}, {4, 24}, {4, 23}}
	d := dates[y-firstYear]
	return time.Date(int(y), time.Month(d[0]), d[1], 0, 0, 0, 0, time.UTC)
}

type draftSpan struct{ First, Last int64 }

func (l *Lab) buildCalendar(trades []Trade, spans map[int64]draftSpan) {
	for y := int64(firstYear); y <= lastYear; y++ {
		kickoff := l.WeekStart[yearWeek{y, 1}]
		end := l.WeekEnd[yearWeek{y, 17}]
		if end == 0 {
			for key, ts := range l.WeekEnd {
				if key.Year == y {
					end = max(end, ts)
				}
			}
		}
		clean := []int64{}
		for _, t := range trades {
			if t.Year == y && !t.OneSided {
				clean = append(clean, t.Ts)
			}
		}
		slices.Sort(clean)
		mk := Markers{NFLDraft: ptrDate(nflDraftDate(y)), Kickoff: dateAt(kickoff), SeasonEnd: dateAt(end), RegularEnd: dateAt(l.WeekEnd[yearWeek{y, l.Regular[y]}])}
		if len(clean) > 0 {
			mk.Open = dateAt(clean[0])
		}
		if span, ok := spans[y]; ok {
			mk.DraftStart = dateAt(span.First)
			mk.DraftEnd = dateAt(span.Last)
		}
		mk.Deadline = l.observedDeadline(y, kickoff, end, clean)
		if mk.Deadline == nil && y == lastYear {
			mk.Deadline = l.expectedDeadline(y)
			mk.DeadlineExpected = mk.Deadline != nil
		}
		l.Marks[y] = mk
	}
}
func ptrDate(d time.Time) *time.Time { return &d }
func (l *Lab) observedDeadline(y, kickoff, end int64, clean []int64) *time.Time {
	last := int64(0)
	for _, ts := range clean {
		if kickoff != 0 && ts >= kickoff && (end == 0 || ts <= end) {
			last = ts
		}
	}
	if last == 0 {
		return nil
	}
	for _, ts := range clean {
		if ts > last {
			if float64(ts-last)/float64(daySeconds) > 40 {
				return dateAt(last)
			}
			return nil
		}
	}
	if y < lastYear {
		return dateAt(last)
	}
	return nil
}
func (l *Lab) expectedDeadline(y int64) *time.Time {
	for _, prev := range []int64{y - 1, y - 2} {
		deadline := l.Marks[prev].Deadline
		if deadline == nil {
			continue
		}
		w := l.weekAt(prev, easternEpoch(*deadline))
		ts := l.WeekStart[yearWeek{y, w + 1}]
		if ts != 0 {
			return ptrDate(dayOf(ts).AddDate(0, 0, -4))
		}
		return nil
	}
	return nil
}
func (l *Lab) phase(year, ts int64) string {
	mk, d := l.Marks[year], dayOf(ts)
	if mk.SeasonEnd != nil && d.After(*mk.SeasonEnd) {
		return "winter"
	}
	if mk.Deadline != nil && d.After(*mk.Deadline) {
		return "closed"
	}
	if mk.Kickoff != nil && !d.Before(*mk.Kickoff) {
		return l.seasonPhase(year, ts, d, mk)
	}
	if mk.NFLDraft != nil && d.Before(*mk.NFLDraft) {
		return "opening"
	}
	if mk.DraftStart == nil {
		return "summer"
	}
	if d.Before(*mk.DraftStart) {
		return "between"
	}
	if mk.DraftEnd != nil && !d.After(*mk.DraftEnd) {
		return "rookieDraft"
	}
	return "summer"
}
func (l *Lab) seasonPhase(year, ts int64, d time.Time, mk Markers) string {
	if mk.Deadline != nil && mk.Deadline.Sub(d) < 14*24*time.Hour {
		return "deadline"
	}
	if l.weekAt(year, ts) <= 4 {
		return "early"
	}
	return "midseason"
}

type Situation struct {
	Rank   *int64
	Basis  *string
	Played int64
	Record *string
}

func (l *Lab) situation(year, ts int64, fid string) Situation {
	week := l.weekAt(year, ts)
	finished := int64(0)
	for w := int64(1); w <= min(week, l.Regular[year]); w++ {
		end := l.WeekEnd[yearWeek{year, w}]
		if end != 0 && end+daySeconds <= ts && l.Done[yearWeek{year, w}] {
			finished = w
		}
	}
	if end := l.Marks[year].RegularEnd; end != nil && dayOf(ts).After(*end) && l.LastDone[year] != 0 {
		finished = l.LastDone[year]
	}
	if finished >= 3 {
		t, ok := l.RankAfter[yearWeek{year, finished}][fid]
		var rank *int64
		if ok {
			rank = ptr(t.Rank)
		}
		record := fmt.Sprintf("%d-%d", t.W, t.L)
		if t.T != 0 {
			record += fmt.Sprintf("-%d", t.T)
		}
		return Situation{rank, ptr("live"), finished, &record}
	}
	if rank, ok := l.Final[year-1][fid]; ok {
		return Situation{Rank: ptr(rank), Basis: ptr("last")}
	}
	return Situation{}
}
func (l *Lab) yearBefore(year int64, basis *string, fid string) *int64 {
	back := int64(0)
	switch value(basis) {
	case "live":
		back = 1
	case "last":
		back = 2
	}
	if back == 0 {
		return nil
	}
	if rank, ok := l.Final[year-back][fid]; ok {
		return ptr(rank)
	}
	return nil
}
