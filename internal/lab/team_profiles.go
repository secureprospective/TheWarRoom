package lab

import "slices"

type TierShift struct {
	Early float64 `json:"early"`
	Late  float64 `json:"late"`
	From  [2]int  `json:"from"`
	To    [2]int  `json:"to"`
	Clear bool    `json:"clear"`
}
type TierTrendRow struct {
	Season int       `json:"season"`
	N      int       `json:"n"`
	Now    TeamShare `json:"now"`
	Later  TeamShare `json:"later"`
	Net    *float64  `json:"net"`
}
type TierTrend struct {
	Tier  string                `json:"tier"`
	Rows  []TierTrendRow        `json:"rows"`
	Shift map[string]*TierShift `json:"shift"`
}
type TeamTrends struct {
	Seasons []int       `json:"seasons"`
	Tiers   []TierTrend `json:"tiers"`
}

func TierTrends(sides []TeamSide) TeamTrends {
	years := SideYears(sides)
	out := TeamTrends{Seasons: years, Tiers: []TierTrend{}}
	for _, tier := range TeamTiers() {
		mine := FilterSides(sides, func(s TeamSide) bool { return s.Tier == tier })
		trend := TierTrend{Tier: tier, Rows: []TierTrendRow{}, Shift: map[string]*TierShift{"now": nil, "later": nil}}
		for _, y := range years {
			ys := FilterSides(mine, func(s TeamSide) bool { return s.Season == y })
			b := Behaviour(ys, nil)
			trend.Rows = append(trend.Rows, TierTrendRow{y, len(ys), b.Posture["now"], b.Posture["later"], b.Now.Mean})
		}
		for _, id := range []string{"now", "later"} {
			trend.Shift[id] = tierShift(mine, years, id, 20)
		}
		out.Tiers = append(out.Tiers, trend)
	}
	return out
}

type TierPairCell struct {
	Other string   `json:"other"`
	N     int      `json:"n"`
	Now   TeamEdge `json:"now"`
	Real  TeamEdge `json:"real"`
}
type TierPair struct {
	Tier  string         `json:"tier"`
	Cells []TierPairCell `json:"cells"`
}

func TierPairs(sides []TeamSide) []TierPair {
	out := []TierPair{}
	for _, a := range TeamTiers() {
		row := TierPair{Tier: a, Cells: []TierPairCell{}}
		for _, b := range TeamTiers() {
			xs := FilterSides(sides, func(s TeamSide) bool { return s.Tier == a && s.OtherTier == b })
			row.Cells = append(row.Cells, TierPairCell{b, len(xs), teamEdge(sideNow(xs, false)), teamEdge(sideNow(xs, true))})
		}
		out = append(out, row)
	}
	return out
}

type TeamPartner struct {
	Fid string `json:"fid"`
	N   int    `json:"n"`
}
type TeamRow struct {
	Fid string `json:"fid"`
	TeamBehaviour
	Sides         []TeamSide         `json:"sides"`
	Tiers         map[string]float64 `json:"tiers"`
	Partner       *TeamPartner       `json:"partner"`
	PartnersCount int                `json:"partnersCount"`
}

func teamIDs(sides []TeamSide, other bool) []string {
	ids := []string{}
	for _, s := range sides {
		id := s.Fid
		if other {
			id = s.Other
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids
}
func TeamRows(sides, league []TeamSide) []TeamRow {
	out := []TeamRow{}
	for _, id := range teamIDs(sides, false) {
		mine := FilterSides(sides, func(s TeamSide) bool { return s.Fid == id })
		r := TeamRow{Fid: id, TeamBehaviour: Behaviour(mine, league), Sides: mine, Tiers: make(map[string]float64)}
		for _, tier := range TeamTiers() {
			r.Tiers[tier] = float64(len(FilterSides(mine, func(s TeamSide) bool { return s.Tier == tier }))) / float64(len(mine))
		}
		partners := teamIDs(mine, true)
		r.PartnersCount = len(partners)
		for _, other := range partners {
			n := len(FilterSides(mine, func(s TeamSide) bool { return s.Other == other }))
			if r.Partner == nil || n > r.Partner.N {
				r.Partner = &TeamPartner{other, n}
			}
		}
		out = append(out, r)
	}
	return out
}

type TeamSeasonCell struct {
	Season  int        `json:"season"`
	N       int        `json:"n"`
	Now     int        `json:"now"`
	Later   int        `json:"later"`
	Swap    int        `json:"swap"`
	Mostly  *string    `json:"mostly"`
	Against int        `json:"against"`
	Net     *float64   `json:"net"`
	Finish  *int       `json:"finish"`
	Sides   []TeamSide `json:"sides"`
}

func (s TeamSeasonCell) Flagged() bool {
	return s.Against >= 2 && float64(s.Against)/float64(s.N) >= 0.3
}

type TeamKind struct {
	Kind string  `json:"kind"`
	InN  int     `json:"inN"`
	OutN int     `json:"outN"`
	InW  float64 `json:"inW"`
	OutW float64 `json:"outW"`
}
type TeamPartnerEdge struct {
	TeamPartner
	Net TeamEdge `json:"net"`
}
type TeamAsTier struct {
	Tier string `json:"tier"`
	TeamEdge
	Posture map[string]TeamShare `json:"posture"`
}
type TeamDetail struct {
	Fid       string            `json:"fid"`
	Sides     []TeamSide        `json:"sides"`
	Behaviour TeamBehaviour     `json:"behaviour"`
	BySeason  []TeamSeasonCell  `json:"bySeason"`
	Kinds     []TeamKind        `json:"kinds"`
	Partners  []TeamPartnerEdge `json:"partners"`
	AsTier    []TeamAsTier      `json:"asTier"`
}

func teamSeason(d *Data, mine []TeamSide, fid string, y int) TeamSeasonCell {
	ys := FilterSides(mine, func(s TeamSide) bool { return s.Season == y })
	c := TeamSeasonCell{Season: y, N: len(ys), Sides: ys}
	counts := make(map[string]int)
	net := 0.0
	for _, s := range ys {
		if s.Posture == "now" {
			c.Now++
		}
		if s.Posture == "later" {
			c.Later++
		}
		if (s.Tier == "rebuild" && s.Posture == "now") || (s.Tier == "contender" && s.Posture == "later") {
			c.Against++
		}
		counts[s.Tier]++
		net += s.Now
	}
	c.Swap = c.N - c.Now - c.Later
	best := 0
	for _, tier := range TeamTiers() {
		if counts[tier] > best {
			c.Mostly = ptr(tier)
			best = counts[tier]
		}
	}
	if c.N > 0 {
		c.Net = ptr(net)
	}
	for _, s := range d.TeamSeasons {
		if s.Fid == fid && s.Year == y {
			c.Finish = s.Rank
		}
	}
	return c
}
func teamKinds(mine []TeamSide) []TeamKind {
	out := []TeamKind{}
	for _, s := range mine {
		for i, assets := range [][]SideAsset{s.Got, s.Gave} {
			for _, a := range assets {
				kind := GroupOf(a.Class)
				j := slices.IndexFunc(out, func(k TeamKind) bool { return k.Kind == kind })
				if j < 0 {
					out = append(out, TeamKind{Kind: kind})
					j = len(out) - 1
				}
				if i == 0 {
					out[j].InN++
					out[j].InW += a.Value
				} else {
					out[j].OutN++
					out[j].OutW += a.Value
				}
			}
		}
	}
	out = slices.DeleteFunc(out, func(k TeamKind) bool { return k.Kind == "player:unknown" })
	slices.SortStableFunc(out, func(a, b TeamKind) int { return compareFloat(b.InW-b.OutW, a.InW-a.OutW) })
	return out
}
func TeamProfile(d *Data, sides, league []TeamSide, fid string) TeamDetail {
	mine := FilterSides(sides, func(s TeamSide) bool { return s.Fid == fid })
	p := TeamDetail{Fid: fid, Sides: mine, Behaviour: Behaviour(mine, league), BySeason: []TeamSeasonCell{}, Kinds: teamKinds(mine), Partners: []TeamPartnerEdge{}, AsTier: []TeamAsTier{}}
	for _, y := range SideYears(league) {
		p.BySeason = append(p.BySeason, teamSeason(d, mine, fid, y))
	}
	for _, other := range teamIDs(mine, true) {
		xs := FilterSides(mine, func(s TeamSide) bool { return s.Other == other })
		p.Partners = append(p.Partners, TeamPartnerEdge{TeamPartner{other, len(xs)}, teamEdge(sideNow(xs, false))})
	}
	slices.SortStableFunc(p.Partners, func(a, b TeamPartnerEdge) int { return b.N - a.N })
	for _, tier := range TeamTiers() {
		xs := FilterSides(mine, func(s TeamSide) bool { return s.Tier == tier })
		b := Behaviour(xs, league)
		p.AsTier = append(p.AsTier, TeamAsTier{tier, b.Now, b.Posture})
	}
	return p
}
