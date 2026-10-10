package lab

import (
	"math"
	"slices"
)

type DraftScopeFilter struct {
	From     int      `json:"from"`
	To       int      `json:"to"`
	Seasons  int      `json:"seasons"`
	Segments []string `json:"segments"`
	NFL      []string `json:"nfl"`
}
type JudgedPick struct {
	DraftPick
	Segment  string   `json:"segment"`
	NFL      string   `json:"nfl"`
	Judged   bool     `json:"judged"`
	Got      *float64 `json:"got"`
	Expected *float64 `json:"expected"`
}

func DraftSegments() []string  { return []string{"r1a", "r1b", "r2a", "r2b", "r3", "r4", "r5"} }
func DraftNFLRounds() []string { return []string{"n1", "n2", "n3", "n4", "nx"} }
func SegmentOf(p DraftPick) string {
	switch {
	case p.Round == 1 && p.Slot <= 10:
		return "r1a"
	case p.Round == 1:
		return "r1b"
	case p.Round == 2 && p.Slot <= 16:
		return "r2a"
	case p.Round == 2:
		return "r2b"
	case p.Round == 3:
		return "r3"
	case p.Round == 4:
		return "r4"
	case p.Round >= 5:
		return "r5"
	default:
		return ""
	}
}
func NFLOf(p DraftPick) string {
	if p.NflRound == nil {
		return "nx"
	}
	switch *p.NflRound {
	case 1:
		return "n1"
	case 2:
		return "n2"
	case 3:
		return "n3"
	default:
		if *p.NflRound >= 4 {
			return "n4"
		}
		return ""
	}
}
func DraftScope(d *Data, s DraftScopeFilter) []JudgedPick {
	out := []JudgedPick{}
	for _, p := range d.Draft {
		seg, nfl := SegmentOf(p), NFLOf(p)
		if p.Year < s.From || p.Year > s.To || p.Group == nil || *p.Group == "" || (s.Segments != nil && !slices.Contains(s.Segments, seg)) || (s.NFL != nil && !slices.Contains(s.NFL, nfl)) {
			continue
		}
		x := JudgedPick{DraftPick: p, Segment: seg, NFL: nfl, Judged: p.Year+s.Seasons-1 <= d.LastFinished && len(p.Wins) >= s.Seasons}
		got, expected := 0.0, 0.0
		for i := range min(s.Seasons, len(p.Wins)) {
			if p.Wins[i] == nil {
				x.Judged = false
			} else {
				got += *p.Wins[i]
			}
		}
		for i := range min(s.Seasons, len(p.SlotWins)) {
			expected += p.SlotWins[i]
		}
		if x.Judged {
			x.Got = ptr(got)
			x.Expected = ptr(expected)
		}
		out = append(out, x)
	}
	return out
}

type DraftRatio struct {
	Delivered
	Lo   float64 `json:"lo"`
	Hi   float64 `json:"hi"`
	Tone string  `json:"tone"`
}

func draftTone(lo, hi float64) string {
	if lo > 1 {
		return "up"
	}
	if hi < 1 {
		return "down"
	}
	return "even"
}
func VsSlot(picks []JudgedPick) *DraftRatio {
	records := []Delivery{}
	for _, p := range picks {
		if p.Judged {
			records = append(records, Delivery{Expected: *p.Expected, Delivered: *p.Got})
		}
	}
	r := DeliveredRatio(records)
	if r == nil {
		return nil
	}
	lo, hi := r.Ratio-Z*r.SE, r.Ratio+Z*r.SE
	return &DraftRatio{*r, lo, hi, draftTone(lo, hi)}
}

type DraftCell struct {
	Segment string   `json:"segment,omitempty"`
	Season  int      `json:"season,omitempty"`
	NFL     string   `json:"nfl,omitempty"`
	Age     string   `json:"age,omitempty"`
	N       int      `json:"n"`
	Ratio   *float64 `json:"ratio"`
	Lo      *float64 `json:"lo,omitempty"`
	Hi      *float64 `json:"hi,omitempty"`
	Tone    string   `json:"tone,omitempty"`
	At      *int     `json:"at,omitempty"`
	Like    *int     `json:"like,omitempty"`
	Best    bool     `json:"best,omitempty"`
	Worst   bool     `json:"worst,omitempty"`
}

func ratioCell(r *DraftRatio, n int) DraftCell {
	c := DraftCell{N: n}
	if r != nil {
		c.Ratio = ptr(r.Ratio)
		c.Lo = ptr(r.Lo)
		c.Hi = ptr(r.Hi)
		c.Tone = r.Tone
	}
	return c
}
func BySeason(picks []JudgedPick) []DraftCell {
	out := []DraftCell{}
	for i := range 5 {
		records := []Delivery{}
		for _, p := range picks {
			if i < len(p.Wins) && p.Wins[i] != nil && i < len(p.SlotWins) {
				records = append(records, Delivery{Expected: p.SlotWins[i], Delivered: *p.Wins[i]})
			}
		}
		c := DraftCell{Season: i + 1, N: len(records)}
		if r := DeliveredRatio(records); r != nil {
			lo, hi := r.Ratio-Z*r.SE, r.Ratio+Z*r.SE
			c.Ratio = ptr(r.Ratio)
			c.Lo = ptr(lo)
			c.Hi = ptr(hi)
			c.Tone = draftTone(lo, hi)
		}
		out = append(out, c)
	}
	return out
}
func SlotCurve(picks []JudgedPick) []*float64 {
	maximum := 1
	for _, p := range picks {
		if p.Judged {
			maximum = max(maximum, p.Overall)
		}
	}
	sums, counts := make([]float64, maximum+1), make([]int, maximum+1)
	for _, p := range picks {
		if p.Judged {
			sums[p.Overall] += *p.Expected
			counts[p.Overall]++
		}
	}
	curve := make([]*float64, maximum+1)
	for o := 1; o <= maximum; o++ {
		s, n := 0.0, 0
		for k := max(1, o-4); k <= min(maximum, o+4); k++ {
			s += sums[k]
			n += counts[k]
		}
		if n > 0 {
			curve[o] = ptr(s / float64(n))
		}
	}
	for o := 2; o <= maximum; o++ {
		if curve[o] == nil || (curve[o-1] != nil && *curve[o] > *curve[o-1]) {
			curve[o] = curve[o-1]
		}
	}
	return curve
}
func PlaysLike(curve []*float64, value *float64) *int {
	if value == nil {
		return nil
	}
	for o := 1; o < len(curve); o++ {
		if curve[o] != nil && *curve[o] <= *value {
			return ptr(o)
		}
	}
	return ptr(len(curve) - 1)
}

type DraftHits struct {
	N        int     `json:"n"`
	Rate     float64 `json:"rate"`
	Baseline float64 `json:"baseline"`
	Tone     string  `json:"tone"`
}

func isHit(p JudgedPick) bool { return p.Best != nil && (*p.Best == "top" || *p.Best == "starter") }
func HitRate(picks, all []JudgedPick) *DraftHits {
	return hitRate(picks, all, 8)
}
func hitRate(picks, all []JudgedPick, threshold int) *DraftHits {
	base := make(map[string]float64)
	for _, seg := range DraftSegments() {
		n, hits := 0, 0
		for _, p := range all {
			if p.Judged && p.Segment == seg {
				n++
				if isHit(p) {
					hits++
				}
			}
		}
		if n > 0 {
			base[seg] = float64(hits) / float64(n)
		}
	}
	n, hits, expected, v := 0, 0, 0.0, 0.0
	for _, p := range picks {
		b, ok := base[p.Segment]
		if !p.Judged || !ok {
			continue
		}
		n++
		if isHit(p) {
			hits++
		}
		expected += b
		v += b * (1 - b)
	}
	if n < threshold {
		return nil
	}
	tone := "even"
	if float64(hits)-expected > Z*math.Sqrt(v) {
		tone = "up"
	} else if expected-float64(hits) > Z*math.Sqrt(v) {
		tone = "down"
	}
	return &DraftHits{n, float64(hits) / float64(n), expected / float64(n), tone}
}
