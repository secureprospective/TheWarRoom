package labcompile

import (
	"regexp"
)

func (l *Lab) teamRow(y int64, fid string, ap AnalysisTeam) teamRow {
	rec, ok := l.RankAfter[yearWeek{y, l.LastDone[y]}][fid]
	r1, rostered := l.Rosters[teamWeek{y, 1, fid}]
	out := teamRow{Year: y, Fid: fid, Champion: l.Champion[y] == fid, Finalist: l.Finalist[y] == fid, Name: l.Names[fid][y], AllPlay: ap.AllPlay, ExpectedWins: ap.ExpectedWins, Luck: ap.Luck}
	if y < lastYear {
		if rank, exists := l.Final[y][fid]; exists {
			out.Rank = ptr(rank)
		}
	}
	if ok {
		out.LiveRank = ptr(rec.Rank)
		out.W = ptr(rec.W)
		out.L = ptr(rec.L)
		out.T = ptr(rec.T)
		out.Pf = ptr(pythonRound(rec.PF, 1))
	}
	if len(l.Playoffs[y]) > 0 {
		out.Playoffs = ptr(l.Playoffs[y][fid])
	}
	if rostered {
		used, extended, tagged := rosterStats(r1)
		out.CapUsed = ptr(pythonRound(used, 2))
		out.Extended = ptr(extended)
		out.Tagged = ptr(tagged)
	}
	ages := l.starterAges(y, fid)
	if len(ages) > 0 {
		out.StarterAge = ptr(pythonRound(preciseMean(ages), 2))
	}
	return out
}
func rosterStats(r roster) (float64, int64, int64) {
	extended, tagged := int64(0), int64(0)
	salaries := []float64{}
	ext := regexp.MustCompile(`(?i)\bEXT|Extend`)
	tag := regexp.MustCompile(`(?i)^\s*(FT|Franchis)`)
	for _, pid := range r.Order {
		c := r.Players[pid]
		salaries = append(salaries, value(c.Salary))
		if ext.MatchString(c.Note) {
			extended++
		}
		if tag.MatchString(c.Note) {
			tagged++
		}
	}
	return pythonSum(salaries), extended, tagged
}
func (l *Lab) starterAges(y int64, fid string) []float64 {
	ages := []float64{}
	for w := int64(1); w <= l.Regular[y]; w++ {
		for _, pid := range l.Starts[teamWeek{y, w, fid}] {
			a := l.ageAt(pid, l.WeekStart[yearWeek{y, w}])
			if value(a) != 0 {
				ages = append(ages, *a)
			}
		}
	}
	return ages
}
func (l *Lab) playerRow(pid string) playerRow {
	name, ok := l.PlayerNames[pid]
	if !ok {
		name = pid
	}
	out := playerRow{ID: pid, Name: name, Pos: l.pos(pid, lastYear)}
	if born, ok := l.Birth[pid]; ok {
		out.Birth = ptr(born)
	}
	if nfl, ok := l.NFLDraft[pid]; ok {
		out.NflYear = ptr(nfl.Year)
		out.NflRound = nfl.Round
	}
	return out
}
