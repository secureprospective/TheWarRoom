package facts

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// move is one player joining (+1) or leaving (-1) a franchise at a time.
type move struct {
	ts   int64
	fid  string
	sign int
}

// rosterLedger: from 2018, every week-to-week roster change is a recorded move. Before 2018 MFL's
// stored weekly rosters barely change within a season, so the proof starts in 2018. Rosters are
// MFL's at the last kickoff of each week.
func (c *checker) rosterLedger() CheckResult {
	r := CheckResult{Name: "Roster changes are recorded moves (2018 on)",
		Detail: "Rosters at the last kickoff of each week, compared with the next week."}
	kickoff, rosters := c.lastKickoffs(), c.weekRosters()
	moves := c.playerMoves()
	weeks := make([]yearWeek, 0, len(rosters))
	for k := range rosters {
		weeks = append(weeks, k)
	}
	slices.SortFunc(weeks, func(a, b yearWeek) int { return cmp.Or(cmp.Compare(a.year, b.year), cmp.Compare(a.week, b.week)) })
	for _, k := range weeks {
		next := yearWeek{k.year, k.week + 1}
		t0, ok0 := kickoff[k]
		t1, ok1 := kickoff[next]
		if len(rosters[next]) == 0 || !ok0 || !ok1 {
			continue
		}
		for _, fp := range changed(rosters[k], rosters[next]) {
			r.Checked++
			sign, verb := -1, "left"
			if rosters[next][fp] {
				sign, verb = 1, "joined"
			}
			if slices.ContainsFunc(moves[fp[1]], func(m move) bool { return t0 < m.ts && m.ts <= t1 && m.fid == fp[0] && m.sign == sign }) {
				continue
			}
			c.differ(&r, fmt.Sprintf("%d/%02d/%s/%s/%s", k.year, k.week, fp[0], fp[1], verb), fmt.Sprintf(
				"%d week %d to %d: %s %s %s with no move recorded in that window.", k.year, k.week, k.week+1,
				c.player(fp[1]), verb, c.team(k.year, &fp[0])))
		}
	}
	return r
}

// lastKickoffs is each week's last NFL kickoff.
func (c *checker) lastKickoffs() map[yearWeek]int64 {
	out := map[yearWeek]int64{}
	n := c.f.NflGame
	for i, year := range n.Year {
		k := yearWeek{year, n.Week[i]}
		if t := n.Kickoff[i]; t != nil && (out[k] == 0 || *t > out[k]) {
			out[k] = *t
		}
	}
	return out
}

// weekRosters is each week's stored rosters from 2018, as franchise and player pairs.
func (c *checker) weekRosters() map[yearWeek]map[[2]string]bool {
	out := map[yearWeek]map[[2]string]bool{}
	rw := c.f.RosterWeek
	for i, year := range rw.Year {
		if rw.Week[i] == 0 || year < 2018 {
			continue
		}
		k := yearWeek{year, rw.Week[i]}
		if out[k] == nil {
			out[k] = map[[2]string]bool{}
		}
		out[k][[2]string{rw.Fid[i], rw.Pid[i]}] = true
	}
	return out
}

// changed is the symmetric difference of two weeks' rosters, ordered.
func changed(before, after map[[2]string]bool) [][2]string {
	var out [][2]string
	for fp := range before {
		if !after[fp] {
			out = append(out, fp)
		}
	}
	for fp := range after {
		if !before[fp] {
			out = append(out, fp)
		}
	}
	slices.SortFunc(out, func(a, b [2]string) int { return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1])) })
	return out
}

// playerMoves lists, per player, every recorded move through trades, free agency, waivers and roster loads.
func (c *checker) playerMoves() map[string][]move {
	types := map[int64]bool{}
	ts := map[int64]int64{}
	t := c.f.Txn
	for i, id := range t.TxnID {
		switch t.Type[i] {
		case "TRADE", "FREE_AGENT", "WAIVER", "LOAD_ROSTERS":
			types[id] = true
			if t.Ts[i] != nil {
				ts[id] = *t.Ts[i]
			}
		}
	}
	out := map[string][]move{}
	a := c.f.TxnAsset
	for i, id := range a.TxnID {
		if !types[id] || a.Kind[i] != "player" {
			continue
		}
		pid := value(a.Pid[i])
		if from := value(a.FromFid[i]); from != "" {
			out[pid] = append(out[pid], move{ts[id], from, -1})
		}
		if to := value(a.ToFid[i]); to != "" {
			out[pid] = append(out[pid], move{ts[id], to, 1})
		}
	}
	return out
}

// pickHop is one trade of a pick.
type pickHop struct {
	ts       int64
	from, to string
}

// pickChains: the trade log says who held each pick last; MFL's draft results say who used it. The
// original owner comes from MFL's "Pick traded from" note, else the team that used it.
func (c *checker) pickChains() CheckResult {
	r := CheckResult{Name: "Draft picks were used by their last owner (2017 on)",
		Detail: `Original owner from MFL's "Pick traded from" note, else the team that used it.`}
	byName := map[string]map[string]bool{}
	fs := c.f.FranchiseSeason
	for i, name := range fs.Name {
		if name == nil {
			continue
		}
		if byName[value(name)] == nil {
			byName[value(name)] = map[string]bool{}
		}
		byName[value(name)][fs.Fid[i]] = true
	}
	future, current := c.pickTrades()
	d := c.f.DraftPick
	for i, year := range d.Year {
		if year < 2017 {
			continue
		}
		fid := value(d.Fid[i])
		orig := fid
		if chain := pickTraded.FindStringSubmatch(value(d.Comments[i])); chain != nil {
			if first := byName[strings.TrimSpace(chain[1])]; len(first) == 1 {
				for only := range first {
					orig = only
				}
			}
		}
		var hist []pickHop
		for _, h := range append(slices.Clone(future[pickKey{year, d.Round[i], orig}]), current[pickSlot{year, d.Round[i], d.Pick[i]}]...) {
			if d.Ts[i] == nil || *d.Ts[i] == 0 || h.ts <= *d.Ts[i] {
				hist = append(hist, h)
			}
		}
		slices.SortFunc(hist, func(a, b pickHop) int {
			return cmp.Or(cmp.Compare(a.ts, b.ts), cmp.Compare(a.from, b.from), cmp.Compare(a.to, b.to))
		})
		r.Checked++
		holder := orig
		if len(hist) > 0 {
			holder = hist[len(hist)-1].to
		}
		if holder == fid {
			continue
		}
		last := hist[len(hist)-1]
		c.differ(&r, fmt.Sprintf("%d/%02d/%02d", year, d.Round[i], d.Pick[i]), fmt.Sprintf(
			"%d pick %d.%02d: used by %s, but the trade log last sent it to %s on %s, and the draft shows no \"traded\" note.",
			year, d.Round[i], d.Pick[i], c.team(year, &fid), c.team(year, &holder), c.day(&last.ts)))
	}
	return r
}

// pickTrades indexes every traded pick: future picks by original owner, year and round; current
// picks by year, round and slot.
func (c *checker) pickTrades() (map[pickKey][]pickHop, map[pickSlot][]pickHop) {
	ts := map[int64]int64{}
	t := c.f.Txn
	for i, id := range t.TxnID {
		if t.Type[i] == "TRADE" && t.Ts[i] != nil {
			ts[id] = *t.Ts[i]
		} else if t.Type[i] == "TRADE" {
			ts[id] = 0
		}
	}
	future, current := map[pickKey][]pickHop{}, map[pickSlot][]pickHop{}
	a := c.f.TxnAsset
	for i, id := range a.TxnID {
		when, trade := ts[id]
		if !trade || a.Kind[i] != "pick" || a.PickYear[i] == nil || a.PickRound[i] == nil {
			continue
		}
		hop := pickHop{when, value(a.FromFid[i]), value(a.ToFid[i])}
		if orig := value(a.PickOrig[i]); orig != "" {
			k := pickKey{*a.PickYear[i], *a.PickRound[i], orig}
			future[k] = append(future[k], hop)
		} else if a.PickSlot[i] != nil {
			k := pickSlot{*a.PickYear[i], *a.PickRound[i], *a.PickSlot[i]}
			current[k] = append(current[k], hop)
		}
	}
	return future, current
}
