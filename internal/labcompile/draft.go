package labcompile

import "slices"

func (l *Lab) starterSeasons(pid string, y int64) (int64, *string) {
	out := int64(0)
	var best *string
	tiers := []string{"top", "starter", "depth", "fringe"}
	for year := y; year < min(y+3, int64(lastYear)); year++ {
		rank, ok := l.SeasonRank[playerYear{pid, year}]
		if !ok {
			continue
		}
		tier := l.tierFromRank(&rank, l.pos(pid, year), year)
		if tier == nil {
			continue
		}
		if *tier == "top" || *tier == "starter" {
			out++
		}
		if best == nil || slices.Index(tiers, *tier) < slices.Index(tiers, *best) {
			best = tier
		}
	}
	return out, best
}
func positionGroup(pos *string) *string {
	if pos == nil {
		return nil
	}
	switch *pos {
	case "QB", "RB", "WR", "TE", "PK", "LB":
		return pos
	case "DT", "DE":
		return ptr("DL")
	case "CB", "S":
		return ptr("DB")
	}
	return nil
}
func seasonWins(vals []*float64, y int64, weeks map[int64]int64) []*float64 {
	if vals == nil {
		return make([]*float64, 5)
	}
	out := make([]*float64, len(vals))
	for j, r := range vals {
		if r == nil {
			continue
		}
		n, ok := weeks[y+int64(j)]
		if !ok {
			n = 13
		}
		out[j] = ptr(pythonRound(*r*float64(n), 3))
	}
	return out
}
func (l *Lab) draftRow(d Pick, overall int64, a analysisIndex) draftRow {
	n, best := l.starterSeasons(d.Pid, d.Year)
	totals := []float64{}
	for y := d.Year; y < min(d.Year+3, int64(lastYear)); y++ {
		totals = append(totals, l.SeasonPts[playerYear{d.Pid, y}])
	}
	pos := l.pos(d.Pid, d.Year)
	s := l.situation(d.Year, d.Ts, d.Fid)
	key := pickKey{d.Year, d.Round, d.Slot}
	out := draftRow{Year: d.Year, Round: d.Round, Slot: d.Slot, Overall: overall, Fid: d.Fid, Pid: d.Pid, Pos: pos, Ts: d.Ts, Group: positionGroup(pos), StarterSeasons: n, Best: best, SeasonsSeen: max(int64(0), min(int64(3), lastYear-d.Year)), Pts3: pythonRound(pythonSum(totals), 1), Age: l.ageAt(d.Pid, d.Ts), Rank: s.Rank, Prev: l.yearBefore(d.Year, s.Basis, d.Fid), Wins: seasonWins(a.PickSeasons[key], d.Year, a.Weeks), SlotWins: seasonWins(a.PickSlots[key], d.Year, a.Weeks)}
	if nfl, ok := l.NFLDraft[d.Pid]; ok && nfl.Year == d.Year {
		out.NflRound = nfl.Round
		out.NflPick = nfl.Pick
	}
	if d.Year+1 <= lastYear {
		_, kept := l.Rosters[teamWeek{d.Year + 1, 1, d.Fid}].Players[d.Pid]
		out.KeptYear2 = ptr(kept)
	}
	return out
}
