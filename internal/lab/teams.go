package lab

import (
	"math"
	"slices"
	"strings"
)

type TeamScope struct {
	From     int      `json:"from"`
	To       int      `json:"to"`
	Tiers    []string `json:"tiers"`
	Windows  []string `json:"windows"`
	Postures []string `json:"postures"`
	Team     string   `json:"team"`
}
type SideAsset struct {
	Class string   `json:"class"`
	Value float64  `json:"value"`
	Age   *float64 `json:"age"`
}
type TeamSide struct {
	TradeID   int         `json:"tradeID"`
	Fid       string      `json:"fid"`
	Other     string      `json:"other"`
	Season    int         `json:"season"`
	Window    string      `json:"window"`
	Phase     string      `json:"phase"`
	Tier      string      `json:"tier"`
	OtherTier string      `json:"otherTier"`
	Rank      int         `json:"rank"`
	Now       float64     `json:"now"`
	Real      *float64    `json:"real"`
	Players   float64     `json:"players"`
	Picks     float64     `json:"picks"`
	PicksIn   int         `json:"picksIn"`
	PicksOut  int         `json:"picksOut"`
	AgeIn     []float64   `json:"ageIn"`
	AgeOut    []float64   `json:"ageOut"`
	Got       []SideAsset `json:"got"`
	Gave      []SideAsset `json:"gave"`
	Posture   string      `json:"posture"`
}

func TeamTiers() []string    { return []string{"contender", "hunt", "rebuild"} }
func TeamPostures() []string { return []string{"now", "later", "swap"} }

// TierOf maps a trade-day rank to its tier. Rank 0 is the decode shape of a
// missing rank (JS null), which the reference leaves unclassed; ranks 1–10
// contend, 11–22 hunt, 23 up rebuild.
func TierOf(rank int) string {
	if rank == 0 {
		return ""
	}
	if rank <= 10 {
		return "contender"
	}
	if rank < 23 {
		return "hunt"
	}
	return "rebuild"
}
func sideAssets(t Trade, fid string, got bool) ([]SideAsset, []float64, float64, float64, int) {
	assets, ages := []SideAsset{}, []float64{}
	players, picks, count := 0.0, 0.0, 0
	for _, l := range t.Legs {
		if l.XClass == nil || (got && l.To != fid) || (!got && l.From != fid) {
			continue
		}
		value := 0.0
		if l.XValue != nil {
			value = *l.XValue
		}
		assets = append(assets, SideAsset{*l.XClass, value, l.Age})
		if l.Age != nil {
			ages = append(ages, *l.Age)
		}
		if strings.HasPrefix(*l.XClass, "pick") {
			picks += value
			count++
		} else {
			players += value
		}
	}
	return assets, ages, players, picks, count
}
func TradeSides(d *Data) []TeamSide { return tradeSides(d, 2) }

// tradeSides counts a side's realised return once it has played threshold seasons;
// the mutant gate replays it with threshold 1 to prove the golden catches the flip.
func tradeSides(d *Data, threshold int) []TeamSide {
	out := []TeamSide{}
	for _, t := range d.Trades {
		// A valued trade always carries its season and window in the compiled facts;
		// one without them cannot be placed in a season scope, so it has no side here.
		if t.ANet == nil || t.Season == nil || t.Window == nil {
			continue
		}
		for i, fid := range []string{t.A, t.B} {
			other, rank, otherRank, sign := t.B, t.ARank, t.BRank, 1.0
			if i == 1 {
				other, rank, otherRank, sign = t.A, t.BRank, t.ARank, -1
			}
			got, ageIn, playerIn, pickIn, picksIn := sideAssets(t, fid, true)
			gave, ageOut, playerOut, pickOut, picksOut := sideAssets(t, fid, false)
			s := TeamSide{TradeID: t.ID, Fid: fid, Other: other, Season: *t.Season, Window: *t.Window, Phase: t.Phase, Tier: TierOf(rank), OtherTier: TierOf(otherRank), Rank: rank, Now: sign * *t.ANet, Players: playerIn - playerOut, Picks: pickIn - pickOut, PicksIn: picksIn, PicksOut: picksOut, AgeIn: ageIn, AgeOut: ageOut, Got: got, Gave: gave, Posture: "swap"}
			if t.ARealNet != nil && t.RealSeasons != nil && *t.RealSeasons >= threshold {
				s.Real = ptr(sign * *t.ARealNet)
			}
			if s.Players > 0.05 && s.Picks < -0.05 {
				s.Posture = "now"
			}
			if s.Players < -0.05 && s.Picks > 0.05 {
				s.Posture = "later"
			}
			out = append(out, s)
		}
	}
	return out
}
func SidesInScope(d *Data, s TeamScope) []TeamSide { return sidesInScope(d, s, 2) }

func sidesInScope(d *Data, s TeamScope, threshold int) []TeamSide {
	return FilterSides(tradeSides(d, threshold), func(x TeamSide) bool {
		return x.Season >= s.From && x.Season <= s.To && (s.Tiers == nil || slices.Contains(s.Tiers, x.Tier)) && (s.Windows == nil || slices.Contains(s.Windows, x.Window)) && (s.Postures == nil || slices.Contains(s.Postures, x.Posture)) && (s.Team == "" || s.Team == x.Fid)
	})
}
func FilterSides(sides []TeamSide, keep func(TeamSide) bool) []TeamSide {
	out := []TeamSide{}
	for _, s := range sides {
		if keep(s) {
			out = append(out, s)
		}
	}
	return out
}

type TeamEdge struct {
	N     int      `json:"n"`
	Mean  *float64 `json:"mean"`
	Lo    *float64 `json:"lo"`
	Hi    *float64 `json:"hi"`
	Total *float64 `json:"total,omitempty"`
	Tone  string   `json:"tone"`
}

func teamEdge(values []float64) TeamEdge {
	m := MeanVar(values)
	e := TeamEdge{N: m.N, Mean: m.Mean, Tone: "even"}
	if m.N < 2 {
		return e
	}
	half := 1.96 * math.Sqrt(*m.Var/float64(m.N))
	e.Lo, e.Hi, e.Total = ptr(*m.Mean-half), ptr(*m.Mean+half), ptr(*m.Mean*float64(m.N))
	if *e.Lo > 0 {
		e.Tone = "up"
	} else if *e.Hi < 0 {
		e.Tone = "down"
	}
	return e
}

type TeamShare struct {
	K    int      `json:"k"`
	N    int      `json:"n"`
	P    *float64 `json:"p"`
	Lo   *float64 `json:"lo,omitempty"`
	Hi   *float64 `json:"hi,omitempty"`
	Base *float64 `json:"base,omitempty"`
	Tone string   `json:"tone,omitempty"`
}

func teamShare(k, n int, base *float64) TeamShare {
	s := TeamShare{K: k, N: n}
	if n == 0 {
		return s
	}
	p := float64(k) / float64(n)
	half := 1.96 * math.Sqrt(math.Max(p*(1-p), 1e-9)/float64(n))
	s.P, s.Lo, s.Hi, s.Base, s.Tone = ptr(p), ptr(p-half), ptr(p+half), base, "even"
	if base != nil {
		if *s.Lo > *base {
			s.Tone = "up"
		} else if *s.Hi < *base {
			s.Tone = "down"
		}
	}
	return s
}

type TeamBehaviour struct {
	N         int                  `json:"n"`
	PerSeason float64              `json:"perSeason"`
	Posture   map[string]TeamShare `json:"posture"`
	Partners  map[string]TeamShare `json:"partners"`
	Windows   map[string]TeamShare `json:"windows"`
	Now       TeamEdge             `json:"now"`
	Real      TeamEdge             `json:"real"`
	Picks     TeamEdge             `json:"picks"`
	Age       TeamEdge             `json:"age"`
}

func teamShares(sides, league []TeamSide, ids []string, field func(TeamSide) string, baseline bool) map[string]TeamShare {
	out := make(map[string]TeamShare)
	for _, id := range ids {
		count := len(FilterSides(sides, func(s TeamSide) bool { return field(s) == id }))
		var base *float64
		if baseline && len(league) > 0 {
			base = ptr(float64(len(FilterSides(league, func(s TeamSide) bool { return field(s) == id }))) / float64(len(league)))
		}
		out[id] = teamShare(count, len(sides), base)
	}
	return out
}

// A nil league means the same population: the reference suppresses baseline comparisons there.
func Behaviour(sides, league []TeamSide) TeamBehaviour {
	compare := league != nil
	if !compare {
		league = sides
	}
	now, delivered, picks, ages := []float64{}, []float64{}, []float64{}, []float64{}
	years := make(map[int]bool)
	for _, s := range sides {
		years[s.Season] = true
		now = append(now, s.Now)
		picks = append(picks, float64(s.PicksIn-s.PicksOut))
		if s.Real != nil {
			delivered = append(delivered, *s.Real)
		}
		if len(s.AgeIn) > 0 && len(s.AgeOut) > 0 {
			ages = append(ages, average(s.AgeIn)-average(s.AgeOut))
		}
	}
	return TeamBehaviour{N: len(sides), PerSeason: float64(len(sides)) / float64(max(1, len(years))), Posture: teamShares(sides, league, TeamPostures(), func(s TeamSide) string { return s.Posture }, compare), Partners: teamShares(sides, nil, TeamTiers(), func(s TeamSide) string { return s.OtherTier }, false), Windows: teamShares(sides, league, []string{"spring", "summer", "season"}, func(s TeamSide) string { return s.Window }, compare), Now: teamEdge(now), Real: teamEdge(delivered), Picks: teamEdge(picks), Age: teamEdge(ages)}
}
func average(xs []float64) float64 {
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}
func SideYears(sides []TeamSide) []int {
	out := []int{}
	for _, s := range sides {
		if !slices.Contains(out, s.Season) {
			out = append(out, s.Season)
		}
	}
	slices.Sort(out)
	return out
}
func sideNow(sides []TeamSide, delivered bool) []float64 {
	out := []float64{}
	for _, s := range sides {
		if !delivered {
			out = append(out, s.Now)
		} else if s.Real != nil {
			out = append(out, *s.Real)
		}
	}
	return out
}
func tierShift(mine []TeamSide, years []int, id string, threshold int) *TierShift {
	if len(years) == 0 {
		return nil
	}
	mid := years[(len(years)-1)/2]
	a := FilterSides(mine, func(s TeamSide) bool { return s.Season <= mid })
	b := FilterSides(mine, func(s TeamSide) bool { return s.Season > mid })
	if len(a) < threshold || len(b) < threshold {
		return nil
	}
	pa := float64(len(FilterSides(a, func(s TeamSide) bool { return s.Posture == id }))) / float64(len(a))
	pb := float64(len(FilterSides(b, func(s TeamSide) bool { return s.Posture == id }))) / float64(len(b))
	se := math.Sqrt(pa*(1-pa)/float64(len(a)) + pb*(1-pb)/float64(len(b)))
	return &TierShift{pa, pb, [2]int{years[0], mid}, [2]int{mid + 1, years[len(years)-1]}, math.Abs(pb-pa) > 1.96*se}
}
func compareFloat(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
