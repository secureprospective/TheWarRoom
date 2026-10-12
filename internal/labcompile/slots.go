package labcompile

import "slices"

type SlotCheck struct {
	Known   int `json:"teamDraftsWithKnownSlot"`
	Checked int `json:"standingsEstimateChecked"`
	Matched int `json:"standingsEstimateMatched"`
}
type PickProof struct {
	Checked int `json:"checked"`
	Matched int `json:"matched"`
}
type slotEvidence struct {
	Count map[int64]int
	Order []int64
}

func (l *Lab) estimatedSlots() map[teamYear]int64 {
	out := map[teamYear]int64{}
	for y := int64(firstYear); y <= lastYear; y++ {
		prev := l.Final[y-1]
		order := sortedKeys(prev)
		slices.SortStableFunc(order, func(a, b string) int {
			ak, bk := l.draftOrderKey(y-1, a), l.draftOrderKey(y-1, b)
			for i := range ak {
				if ak[i] != bk[i] {
					return int(ak[i] - bk[i])
				}
			}
			return 0
		})
		for i, fid := range order {
			out[teamYear{y, fid}] = int64(i + 1)
		}
	}
	return out
}
func (l *Lab) draftOrderKey(y int64, fid string) [3]int64 {
	last := int64(0)
	switch fid {
	case l.Champion[y]:
		last = 2
	case l.Finalist[y]:
		last = 1
	}
	playoff := int64(0)
	if l.Playoffs[y][fid] {
		playoff = 1
	}
	return [3]int64{last, playoff, -l.Final[y][fid]}
}
func knownSlots(trades []Trade, draft []Pick) map[teamYear]int64 {
	traded := map[pickOwner]bool{}
	moved := map[pickKey]bool{}
	for _, t := range trades {
		for _, a := range appendAssets(t) {
			if a.Kind != "pick" {
				continue
			}
			if value(a.Orig) != "" {
				traded[pickOwner{*a.Orig, value(a.Year), value(a.Round)}] = true
			} else {
				moved[pickKey{value(a.Year), value(a.Round), value(a.Slot)}] = true
			}
		}
	}
	perRound := map[pickOwner]int{}
	for _, d := range draft {
		perRound[pickOwner{d.Fid, d.Year, d.Round}]++
	}
	evidence := map[teamYear]slotEvidence{}
	for _, d := range draft {
		owner := pickOwner{d.Fid, d.Year, d.Round}
		if perRound[owner] != 1 || traded[owner] || moved[pickKey{d.Year, d.Round, d.Slot}] {
			continue
		}
		key := teamYear{d.Year, d.Fid}
		e := evidence[key]
		if e.Count == nil {
			e.Count = map[int64]int{}
		}
		if e.Count[d.Slot] == 0 {
			e.Order = append(e.Order, d.Slot)
		}
		e.Count[d.Slot]++
		evidence[key] = e
	}
	out := map[teamYear]int64{}
	for key, e := range evidence {
		best := e.Order[0]
		for _, slot := range e.Order {
			if e.Count[slot] > e.Count[best] {
				best = slot
			}
		}
		out[key] = best
	}
	return out
}
func appendAssets(t Trade) []Asset {
	out := make([]Asset, 0, len(t.AGave)+len(t.BGave))
	out = append(out, t.AGave...)
	return append(out, t.BGave...)
}
func slotChecks(known, estimated map[teamYear]int64, trades []Trade, draft []Pick) (SlotCheck, PickProof) {
	check := SlotCheck{Known: len(known)}
	for k, v := range known {
		if _, ok := estimated[k]; ok {
			check.Checked++
		}
		if estimated[k] == v {
			check.Matched++
		}
	}
	last := map[pickKey]string{}
	order := tradeOrder(trades)
	for _, i := range order {
		t := trades[i]
		for gi, gave := range [][]Asset{t.AGave, t.BGave} {
			receiver := t.B
			if gi == 1 {
				receiver = t.A
			}
			for _, a := range gave {
				if a.Kind == "pick" && value(a.Slot) != 0 {
					last[pickKey{value(a.Year), value(a.Round), *a.Slot}] = receiver
				}
			}
		}
	}
	made := map[pickKey]string{}
	for _, d := range draft {
		made[pickKey{d.Year, d.Round, d.Slot}] = d.Fid
	}
	proof := PickProof{}
	for k, fid := range last {
		if _, ok := made[k]; ok {
			proof.Checked++
		}
		if made[k] == fid {
			proof.Matched++
		}
	}
	return check, proof
}
func tradeOrder(trades []Trade) []int {
	order := make([]int, len(trades))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		if trades[a].Ts < trades[b].Ts {
			return -1
		}
		if trades[a].Ts > trades[b].Ts {
			return 1
		}
		return 0
	})
	return order
}
func sortStrings(v []string) { slices.Sort(v) }
