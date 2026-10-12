package labcompile

import "slices"

type ledgerEvent struct {
	Ts             int64
	Order          int
	Kind, Pid, Fid string
}
type LedgerYear struct {
	Rostered   int      `json:"rostered"`
	Matched    int      `json:"matched"`
	Extra      int      `json:"replayedNotRostered"`
	Rate       float64  `json:"rate"`
	Changed    int      `json:"changed"`
	Explained  int      `json:"explained"`
	ChangeRate *float64 `json:"changeRate"`
}
type Ledger struct {
	Changed    int                  `json:"changed"`
	Explained  int                  `json:"explained"`
	ChangeRate *float64             `json:"changeRate"`
	SnapshotAt string               `json:"snapshotAt"`
	ByYear     map[int64]LedgerYear `json:"byYear"`
	Rostered   int                  `json:"rostered"`
	Matched    int                  `json:"matched"`
	Rate       *float64             `json:"rate"`
}

func ledgerEvents(trades []Trade, moves []Move) map[int64][]ledgerEvent {
	out := map[int64][]ledgerEvent{}
	for _, t := range trades {
		for gi, assets := range [][]Asset{t.AGave, t.BGave} {
			receiver := t.B
			if gi == 1 {
				receiver = t.A
			}
			for _, a := range assets {
				if a.Kind == "player" {
					out[t.Year] = append(out[t.Year], ledgerEvent{t.Ts, 1, "to", value(a.ID), receiver})
				}
			}
		}
	}
	for _, m := range moves {
		switch m.Kind {
		case "add":
			out[m.Year] = append(out[m.Year], ledgerEvent{m.Ts, 2, "to", m.Pid, m.Fid})
		case "drop":
			out[m.Year] = append(out[m.Year], ledgerEvent{m.Ts, 0, "drop", m.Pid, m.Fid})
		}
	}
	for y := range out {
		slices.SortFunc(out[y], compareEvent)
	}
	return out
}
func compareEvent(a, b ledgerEvent) int {
	if a.Ts < b.Ts {
		return -1
	}
	if a.Ts > b.Ts {
		return 1
	}
	if a.Order != b.Order {
		return a.Order - b.Order
	}
	for _, pair := range [][2]string{{a.Kind, b.Kind}, {a.Pid, b.Pid}, {a.Fid, b.Fid}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	return 0
}
func (l *Lab) owners(y, w int64) map[string]string {
	out := map[string]string{}
	for _, key := range l.RosterOrder {
		if key.Year == y && key.Week == w {
			for _, pid := range l.Rosters[key].Order {
				out[pid] = key.Fid
			}
		}
	}
	return out
}
func (l *Lab) ledgerCompare(starts map[yearWeek]int64, events map[int64][]ledgerEvent) map[int64]LedgerYear {
	out := map[int64]LedgerYear{}
	for y := int64(firstYear); y <= lastYear; y++ {
		weeks := []int64{}
		for key := range starts {
			if key.Year == y && len(l.owners(y, key.Week)) > 0 {
				weeks = append(weeks, key.Week)
			}
		}
		slices.Sort(weeks)
		total := LedgerYear{}
		for i := 0; i+1 < len(weeks); i++ {
			w, next := weeks[i], weeks[i+1]
			if next != w+1 {
				continue
			}
			l.ledgerInterval(y, w, next, starts, events[y], &total)
		}
		if total.Rostered > 0 {
			total.Rate = pythonRound(float64(total.Matched)/float64(total.Rostered), 4)
			total.ChangeRate = ratio(total.Explained, total.Changed)
			out[y] = total
		}
	}
	return out
}
func (l *Lab) ledgerInterval(y, w, next int64, starts map[yearWeek]int64, events []ledgerEvent, total *LedgerYear) {
	before := l.owners(y, w)
	owner := l.owners(y, w)
	lo, hi := starts[yearWeek{y, w}], starts[yearWeek{y, next}]
	for _, e := range events {
		if e.Ts < lo || e.Ts >= hi {
			continue
		}
		if e.Kind == "drop" {
			if owner[e.Pid] == e.Fid {
				delete(owner, e.Pid)
			}
		} else {
			owner[e.Pid] = e.Fid
		}
	}
	actual := l.owners(y, next)
	for pid, fid := range actual {
		total.Rostered++
		if owner[pid] == fid {
			total.Matched++
		}
		if before[pid] != fid {
			total.Changed++
			if owner[pid] == fid {
				total.Explained++
			}
		}
	}
	for pid := range owner {
		if _, ok := actual[pid]; !ok {
			total.Extra++
		}
	}
}
func ratio(n, d int) *float64 {
	if d == 0 {
		return nil
	}
	return ptr(pythonRound(float64(n)/float64(d), 4))
}
func (l *Lab) reconcile(trades []Trade, moves []Move) Ledger {
	events := ledgerEvents(trades, moves)
	start := l.ledgerCompare(l.WeekStart, events)
	end := l.ledgerCompare(l.WeekEnd, events)
	sum := func(m map[int64]LedgerYear) int {
		v := 0
		for _, r := range m {
			v += r.Matched
		}
		return v
	}
	best, at := start, "first kickoff"
	if sum(end) > sum(start) {
		best, at = end, "last kickoff"
	}
	out := Ledger{SnapshotAt: at, ByYear: best}
	for _, v := range best {
		out.Changed += v.Changed
		out.Explained += v.Explained
		out.Rostered += v.Rostered
		out.Matched += v.Matched
	}
	out.ChangeRate = ratio(out.Explained, out.Changed)
	out.Rate = ratio(out.Matched, out.Rostered)
	return out
}
