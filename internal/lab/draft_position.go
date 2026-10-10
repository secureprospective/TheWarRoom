package lab

import (
	"math"
	"slices"
)

type DraftPosition struct {
	Group   string       `json:"group"`
	N       int          `json:"n"`
	All     []JudgedPick `json:"all"`
	Overall *DraftRatio  `json:"overall"`
	Hits    *DraftHits   `json:"hits"`
	Kept    *float64     `json:"kept"`
	Where   []DraftCell  `json:"where"`
	When    []DraftCell  `json:"when"`
	NFL     []DraftCell  `json:"nfl"`
	Age     []DraftCell  `json:"age"`
	Curve   []*float64   `json:"curve"`
}

func draftSelect(picks []JudgedPick, keep func(JudgedPick) bool) []JudgedPick {
	out := []JudgedPick{}
	for _, p := range picks {
		if keep(p) {
			out = append(out, p)
		}
	}
	return out
}
func judgedCount(picks []JudgedPick) int {
	n := 0
	for _, p := range picks {
		if p.Judged {
			n++
		}
	}
	return n
}
func draftCell(picks []JudgedPick, curve []*float64) DraftCell {
	r := VsSlot(picks)
	n := judgedCount(picks)
	c := ratioCell(r, n)
	sum := 0.0
	for _, p := range picks {
		if p.Judged {
			sum += float64(p.Overall)
		}
	}
	if n > 0 {
		at := int(math.Floor(sum/float64(n) + 0.5))
		c.At = ptr(at)
		if r != nil {
			value := 0.0
			if curve[at] != nil {
				value = *curve[at] * r.Ratio
			}
			c.Like = PlaysLike(curve, ptr(value))
		}
	}
	return c
}
func markExtremes(cells []DraftCell) []DraftCell {
	known := []int{}
	for i, c := range cells {
		if c.Ratio != nil && c.Hi != nil {
			known = append(known, i)
		}
	}
	if len(known) < 2 {
		return cells
	}
	top, bottom := known[0], known[0]
	for _, i := range known {
		if *cells[i].Ratio > *cells[top].Ratio {
			top = i
		}
		if *cells[i].Ratio < *cells[bottom].Ratio {
			bottom = i
		}
	}
	se := func(i int) float64 { return (*cells[i].Hi - *cells[i].Lo) / (2 * Z) }
	best, worst := true, true
	for _, i := range known {
		if i != top && *cells[top].Ratio-*cells[i].Ratio <= Z*math.Hypot(se(top), se(i)) {
			best = false
		}
		if i != bottom && *cells[i].Ratio-*cells[bottom].Ratio <= Z*math.Hypot(se(bottom), se(i)) {
			worst = false
		}
	}
	cells[top].Best = best
	cells[bottom].Worst = worst
	return cells
}
func ageBand(p JudgedPick) string {
	if p.Age == nil {
		return ""
	}
	if *p.Age < 22 {
		return "a21"
	}
	if *p.Age < 23 {
		return "a22"
	}
	return "a23"
}
func PositionDraft(picks []JudgedPick, group string) DraftPosition {
	mine := draftSelect(picks, func(p JudgedPick) bool { return *p.Group == group })
	curve := SlotCurve(picks)
	out := DraftPosition{Group: group, N: judgedCount(mine), All: mine, Overall: VsSlot(mine), Hits: HitRate(mine, picks), Where: []DraftCell{}, When: markExtremes(BySeason(mine)), NFL: []DraftCell{}, Age: []DraftCell{}, Curve: curve}
	kept, total := 0, 0
	for _, p := range mine {
		if p.KeptYear2 != nil {
			total++
			if *p.KeptYear2 {
				kept++
			}
		}
	}
	if total > 0 {
		out.Kept = ptr(float64(kept) / float64(total))
	}
	for _, seg := range DraftSegments() {
		c := draftCell(draftSelect(mine, func(p JudgedPick) bool { return p.Segment == seg }), curve)
		c.Segment = seg
		out.Where = append(out.Where, c)
	}
	for _, nfl := range DraftNFLRounds() {
		c := draftCell(draftSelect(mine, func(p JudgedPick) bool { return p.NFL == nfl }), curve)
		c.NFL = nfl
		out.NFL = append(out.NFL, c)
	}
	for _, age := range []string{"a21", "a22", "a23"} {
		c := draftCell(draftSelect(mine, func(p JudgedPick) bool { return ageBand(p) == age }), curve)
		c.Age = age
		out.Age = append(out.Age, c)
	}
	out.Where = markExtremes(out.Where)
	out.NFL = markExtremes(out.NFL)
	out.Age = markExtremes(out.Age)
	return out
}

type DraftMapRow struct {
	Group   string         `json:"group"`
	Overall *DraftRatio    `json:"overall"`
	N       int            `json:"n"`
	When    []DraftCell    `json:"when"`
	Where   []DraftMapCell `json:"where"`
}
type DraftMapCell struct {
	Segment string      `json:"segment"`
	Reading *DraftRatio `json:"reading"`
	N       int         `json:"n"`
}

func DraftMap(picks []JudgedPick, groups []string) []DraftMapRow {
	out := []DraftMapRow{}
	for _, g := range groups {
		mine := draftSelect(picks, func(p JudgedPick) bool { return *p.Group == g })
		r := DraftMapRow{Group: g, Overall: VsSlot(mine), N: judgedCount(mine), When: BySeason(mine), Where: []DraftMapCell{}}
		for _, seg := range DraftSegments() {
			xs := draftSelect(mine, func(p JudgedPick) bool { return p.Segment == seg })
			r.Where = append(r.Where, DraftMapCell{seg, VsSlot(xs), judgedCount(xs)})
		}
		out = append(out, r)
	}
	return out
}

type DraftBuyReading struct {
	Rows     []MarketRow `json:"rows"`
	Cheapest *string     `json:"cheapest"`
	Via      *string     `json:"via"`
}

func DraftBuy(m Market, picks []JudgedPick, group string) DraftBuyReading {
	out := DraftBuyReading{Rows: []MarketRow{}}
	for _, kind := range []string{group + ":young", group + ":prime", group + ":older", "pick1:next", "pick2:next", "pick3:next"} {
		for _, r := range m.Rows {
			if r.Kind == kind && r.Cost != nil {
				out.Rows = append(out.Rows, r)
			}
		}
	}
	out.Cheapest = draftCheapestAge(out.Rows, group)
	order := draftRoundOrder(picks)
	if len(order) > 0 {
		for _, r := range out.Rows {
			if r.Kind == order[0] {
				out.Via = ptr(r.Kind)
			}
		}
	}
	if out.Via == nil {
		for _, r := range out.Rows {
			if len(r.Kind) >= 4 && r.Kind[:4] == "pick" {
				out.Via = ptr(r.Kind)
				break
			}
		}
	}
	return out
}
func draftBuyCost(rows []MarketRow, kind string) float64 {
	for _, r := range rows {
		if r.Kind == kind {
			return r.Cost.Value
		}
	}
	return 0
}

func draftRoundOrder(picks []JudgedPick) []string {
	counts := make(map[string]int)
	order := []string{}
	for _, p := range picks {
		if !p.Judged {
			continue
		}
		k := []string{"pick1:next", "pick2:next", "pick3:next"}[min(3, p.Round)-1]
		if counts[k] == 0 {
			order = append(order, k)
		}
		counts[k]++
	}
	slices.SortStableFunc(order, func(a, b string) int { return counts[b] - counts[a] })
	return order
}

func draftCheapestAge(rows []MarketRow, group string) *string {
	var cheapest *string
	for _, r := range rows {
		if AssetOf(r.Kind) == group && (cheapest == nil || draftBuyCost(rows, *cheapest) > r.Cost.Value) {
			cheapest = ptr(r.Kind)
		}
	}
	return cheapest
}
