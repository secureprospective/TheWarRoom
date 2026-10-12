package labcompile

import (
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

func (l *Lab) readWeeks(f facts.Facts) {
	l.readKickoffs(f)
	for i, y := range f.PlayerWeek.Year {
		pts := f.PlayerWeek.Score[i]
		if pts == nil {
			continue
		}
		key := yearWeek{y, f.PlayerWeek.Week[i]}
		pid := f.PlayerWeek.Pid[i]
		if l.Scores[key] == nil {
			l.Scores[key] = map[string]float64{}
		}
		l.Scores[key][pid] = *pts
		l.ScoreOrder[key] = append(l.ScoreOrder[key], pid)
	}
	l.readRosters(f)
	l.readResults(f)
	for i, y := range f.Lineup.Year {
		if f.Lineup.Started[i] != 1 {
			continue
		}
		key := teamWeek{y, f.Lineup.Week[i], f.Lineup.Fid[i]}
		l.Starts[key] = append(l.Starts[key], f.Lineup.Pid[i])
	}
	for i, y := range f.CapCharge.Year {
		amount, ts := f.CapCharge.Amount[i], f.CapCharge.Ts[i]
		if amount != nil && ts != nil && *amount != 0 && *ts != 0 {
			key := teamYear{y, value(f.CapCharge.Fid[i])}
			l.Dead[key] = append(l.Dead[key], charge{*ts, *amount})
		}
	}
}
func (l *Lab) readKickoffs(f facts.Facts) {
	for i, y := range f.NflGame.Year {
		ts := f.NflGame.Kickoff[i]
		if ts == nil || *ts == 0 {
			continue
		}
		key := yearWeek{y, f.NflGame.Week[i]}
		if old, ok := l.WeekStart[key]; !ok || *ts < old {
			l.WeekStart[key] = *ts
		}
		if old, ok := l.WeekEnd[key]; !ok || *ts > old {
			l.WeekEnd[key] = *ts
		}
	}
}
func (l *Lab) readRosters(f facts.Facts) {
	for i, y := range f.RosterWeek.Year {
		w := f.RosterWeek.Week[i]
		if w <= 0 {
			continue
		}
		key := teamWeek{y, w, f.RosterWeek.Fid[i]}
		r, exists := l.Rosters[key]
		if !exists {
			r = roster{Players: map[string]Contract{}}
			l.RosterOrder = append(l.RosterOrder, key)
		}
		pid := f.RosterWeek.Pid[i]
		if _, exists := r.Players[pid]; !exists {
			r.Order = append(r.Order, pid)
		}
		r.Players[pid] = Contract{Salary: f.RosterWeek.Salary[i], Until: contractUntil(f.RosterWeek.ContractYear[i]), Status: nonempty(f.RosterWeek.ContractStatus[i]), Note: strings.TrimSpace(value(f.RosterWeek.ContractInfo[i])), Slot: f.RosterWeek.Status[i]}
		l.Rosters[key] = r
	}
}
func contractUntil(p *string) *int64 {
	if p == nil || *p == "" {
		return nil
	}
	for _, c := range *p {
		if c < '0' || c > '9' {
			return nil
		}
	}
	n, err := strconv.ParseInt(*p, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}
func nonempty(p *string) *string {
	if value(p) == "" {
		return nil
	}
	return p
}
func (l *Lab) readResults(f facts.Facts) {
	scores := map[teamWeek]*float64{}
	for i, y := range f.Game.Year {
		scores[teamWeek{y, f.Game.Week[i], f.Game.Fid[i]}] = f.Game.Score[i]
	}
	v := f.Views()
	unfinished := map[yearWeek]bool{}
	for i, y := range v.RegularWeek.Year {
		key := yearWeek{y, v.RegularWeek.Week[i]}
		l.Done[key] = true
		l.Regular[y] = max(l.Regular[y], key.Week)
	}
	for i, y := range f.Game.Year {
		if f.Game.Score[i] == nil {
			unfinished[yearWeek{y, f.Game.Week[i]}] = true
		}
	}
	for key := range unfinished {
		delete(l.Done, key)
	}
	for i, y := range v.PlayedGame.Year {
		res := value(v.PlayedGame.Result[i])
		if res != "W" && res != "L" && res != "T" {
			continue
		}
		key := teamWeek{y, v.PlayedGame.Week[i], v.PlayedGame.Fid[i]}
		l.ResultOrder = append(l.ResultOrder, key)
		l.Results[key] = gameResult{Result: res, PF: value(v.PlayedGame.Score[i]), PA: scores[teamWeek{y, key.Week, value(v.PlayedGame.Opp[i])}], Opp: v.PlayedGame.Opp[i]}
	}
}

func (l *Lab) weekAt(year, ts int64) int64 {
	week := int64(0)
	for key, start := range l.WeekStart {
		if key.Year == year && start <= ts {
			week = max(week, key.Week)
		}
	}
	return week
}
func (l *Lab) buildStandings() {
	fids := sortedKeys(l.Names)
	for y := int64(firstYear); y <= lastYear; y++ {
		tally := map[string]standing{}
		for w := int64(1); w <= l.Regular[y]; w++ {
			if !l.Done[yearWeek{y, w}] {
				continue
			}
			l.LastDone[y] = w
			for _, fid := range fids {
				if r, ok := l.Results[teamWeek{y, w, fid}]; ok {
					t := tally[fid]
					switch r.Result {
					case "W":
						t.W++
					case "L":
						t.L++
					case "T":
						t.T++
					}
					t.PF += r.PF
					tally[fid] = t
				}
			}
			order := sortedKeys(tally)
			slices.SortStableFunc(order, func(a, b string) int { return compareStanding(tally[a], tally[b]) })
			ranks := map[string]standing{}
			for i, fid := range order {
				t := tally[fid]
				t.Rank = int64(i + 1)
				ranks[fid] = t
			}
			l.RankAfter[yearWeek{y, w}] = ranks
		}
		last := l.LastDone[y]
		if last > 0 && (last == l.Regular[y] || y < lastYear) {
			l.Final[y] = map[string]int64{}
			for fid, t := range l.RankAfter[yearWeek{y, last}] {
				l.Final[y][fid] = t.Rank
			}
		}
	}
}
func compareStanding(a, b standing) int {
	ar := (float64(a.W) + 0.5*float64(a.T)) / float64(max(int64(1), a.W+a.L+a.T))
	br := (float64(b.W) + 0.5*float64(b.T)) / float64(max(int64(1), b.W+b.L+b.T))
	if ar > br {
		return -1
	}
	if ar < br {
		return 1
	}
	if a.PF > b.PF {
		return -1
	}
	if a.PF < b.PF {
		return 1
	}
	return 0
}

type playoffKey struct {
	Year    int64
	Bracket string
}
type playoffFinal struct {
	Week       int64
	Game       string
	Home, Away string
	HP, AP     *float64
}

func (l *Lab) buildPlayoffs(f facts.Facts) {
	finals := map[playoffKey]playoffFinal{}
	for i, y := range f.PlayoffGame.Year {
		l.readPlayoffGame(f, i, y, finals)
	}
	for i, y := range f.PlayoffBracket.Year {
		if !strings.Contains(strings.ToLower(value(f.PlayoffBracket.Name[i])), "bowl") {
			continue
		}
		v := finals[playoffKey{y, f.PlayoffBracket.BracketID[i]}]
		if v.HP == nil || v.AP == nil || (*v.HP == 0 && *v.AP == 0) {
			continue
		}
		l.Champion[y], l.Finalist[y] = v.Away, v.Home
		if *v.HP > *v.AP {
			l.Champion[y], l.Finalist[y] = v.Home, v.Away
		}
	}
}

func (l *Lab) readPlayoffGame(f facts.Facts, i int, y int64, finals map[playoffKey]playoffFinal) {
	fid := f.PlayoffGame.Fid[i]
	if fid == nil {
		return
	}
	if l.Playoffs[y] == nil {
		l.Playoffs[y] = map[string]bool{}
	}
	l.Playoffs[y][*fid] = true
	key := playoffKey{y, f.PlayoffGame.BracketID[i]}
	week, game := value(f.PlayoffGame.Week[i]), value(f.PlayoffGame.GameID[i])
	final, ok := finals[key]
	if !ok || week > final.Week || (week == final.Week && game > final.Game) {
		final = playoffFinal{Week: week, Game: game}
	}
	if week == final.Week && game == final.Game {
		switch value(f.PlayoffGame.Side[i]) {
		case "home":
			final.Home = *fid
			final.HP = f.PlayoffGame.Points[i]
		case "away":
			final.Away = *fid
			final.AP = f.PlayoffGame.Points[i]
		}
	}
	finals[key] = final
}
