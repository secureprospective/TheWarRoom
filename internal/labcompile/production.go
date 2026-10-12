package labcompile

import (
	"math"
	"slices"
)

func (l *Lab) buildProduction() {
	keys := make([]yearWeek, 0, len(l.Scores))
	for key := range l.Scores {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b yearWeek) int {
		if a.Year != b.Year {
			return int(a.Year - b.Year)
		}
		return int(a.Week - b.Week)
	})
	for _, key := range keys {
		if key.Week > l.Regular[key.Year] || !l.Done[key] {
			continue
		}
		for _, pid := range l.ScoreOrder[key] {
			pts := l.Scores[key][pid]
			py := playerYear{pid, key.Year}
			if l.WeeklyPts[py] == nil {
				l.WeeklyPts[py] = map[int64]float64{}
				l.WeeklyOrder = append(l.WeeklyOrder, py)
			}
			l.WeeklyPts[py][key.Week] = pts
			l.SeasonPts[py] += pts
			if pts != 0 {
				l.SeasonGames[py]++
			}
		}
	}
	for y := int64(firstYear); y <= lastYear; y++ {
		l.buildStarterLine(y)
		l.buildSeasonRanks(y)
	}
}
func (l *Lab) buildStarterLine(y int64) {
	counts := map[string]int64{}
	weeks := int64(0)
	for w := int64(1); w <= l.Regular[y]; w++ {
		if !l.Done[yearWeek{y, w}] {
			continue
		}
		weeks++
		for _, fid := range sortedKeys(l.Names) {
			for _, pid := range l.Starts[teamWeek{y, w, fid}] {
				if p := l.pos(pid, y); p != nil {
					counts[*p]++
				}
			}
		}
	}
	l.StarterLine[y] = map[string]*int64{}
	for _, p := range positions() {
		if weeks == 0 {
			l.StarterLine[y][p] = nil
		} else {
			l.StarterLine[y][p] = ptr(int64(math.RoundToEven(float64(counts[p]) / float64(weeks))))
		}
	}
}
func (l *Lab) buildSeasonRanks(y int64) {
	byPos := map[string][]string{}
	for key, pts := range l.SeasonPts {
		if key.Year != y || pts <= 0 {
			continue
		}
		if p := l.pos(key.Pid, y); p != nil {
			byPos[*p] = append(byPos[*p], key.Pid)
		}
	}
	for _, pids := range byPos {
		slices.SortFunc(pids, func(a, b string) int {
			ap, bp := l.SeasonPts[playerYear{a, y}], l.SeasonPts[playerYear{b, y}]
			if ap > bp {
				return -1
			}
			if ap < bp {
				return 1
			}
			if a > b {
				return -1
			}
			if a < b {
				return 1
			}
			return 0
		})
		for i, pid := range pids {
			l.SeasonRank[playerYear{pid, y}] = int64(i + 1)
		}
	}
}
func (l *Lab) tierFromRank(rank *int64, pos *string, y int64) *string {
	line := value(l.StarterLine[y][value(pos)])
	if rank == nil || line == 0 {
		return nil
	}
	switch {
	case *rank <= max(int64(1), line/4):
		return ptr("top")
	case *rank <= line:
		return ptr("starter")
	case *rank <= line*2:
		return ptr("depth")
	default:
		return ptr("fringe")
	}
}
func (l *Lab) knownTier(pid string, y, ts int64) *string {
	pos := l.pos(pid, y)
	if pos == nil {
		return nil
	}
	week := l.weekAt(y, ts)
	if week >= 5 {
		done := []int64{}
		for w := int64(1); w < week && w <= l.Regular[y]; w++ {
			if l.Done[yearWeek{y, w}] {
				done = append(done, w)
			}
		}
		pts, games := productionDuring(l.WeeklyPts[playerYear{pid, y}], done)
		if games >= 3 {
			rank := l.liveRank(*pos, y, done, pts/float64(games))
			return l.tierFromRank(&rank, pos, y)
		}
	}
	prev := y - 1
	if rank, ok := l.SeasonRank[playerYear{pid, prev}]; ok {
		return l.tierFromRank(&rank, pos, prev)
	}
	if drafted, ok := l.NFLDraft[pid]; ok && drafted.Year >= prev {
		return ptr("rookie")
	}
	return ptr("none")
}
func productionDuring(weeks map[int64]float64, done []int64) (float64, int64) {
	points, games := []float64{}, int64(0)
	for _, w := range done {
		points = append(points, weeks[w])
		if weeks[w] != 0 {
			games++
		}
	}
	return pythonSum(points), games
}
func (l *Lab) liveRank(pos string, y int64, done []int64, ppg float64) int64 {
	rank := int64(1)
	for key, weeks := range l.WeeklyPts {
		if key.Year != y || value(l.pos(key.Pid, y)) != pos {
			continue
		}
		pts, games := productionDuring(weeks, done)
		if games >= 3 && pts/float64(games) > ppg {
			rank++
		}
	}
	return rank
}

type Use struct {
	Starts *int64
	Points *float64
	Weeks  int64
}

func (l *Lab) afterUse(pid string, y, ts int64, fid string) Use {
	starts, pts, seen := int64(0), 0.0, int64(0)
	w := l.weekAt(y, ts) + 1
	for seen < 17 && y <= lastYear {
		if w > l.Regular[y] {
			y++
			w = 1
			continue
		}
		if !l.Done[yearWeek{y, w}] {
			break
		}
		seen++
		if slices.Contains(l.Starts[teamWeek{y, w, fid}], pid) {
			starts++
			pts += l.WeeklyPts[playerYear{pid, y}][w]
		}
		w++
	}
	if seen == 0 {
		return Use{}
	}
	return Use{ptr(starts), ptr(pythonRound(pts, 1)), seen}
}
func (l *Lab) ageAt(pid string, ts int64) *float64 {
	b, ok := l.Birth[pid]
	if !ok || b == 0 {
		return nil
	}
	return ptr(pythonRound(float64(ts-b)/(365.25*float64(daySeconds)), 1))
}
