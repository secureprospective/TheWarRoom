package labcompile

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

const firstYear, lastYear, daySeconds = 2013, 2026, int64(86400)

type yearWeek struct{ Year, Week int64 }
type teamWeek struct {
	Year, Week int64
	Fid        string
}
type playerYear struct {
	Pid  string
	Year int64
}
type teamYear struct {
	Year int64
	Fid  string
}
type nflDraft struct {
	Year        int64
	Round, Pick *int64
}
type standing struct {
	Rank    int64
	W, L, T int64
	PF      float64
}
type gameResult struct {
	Result string
	PF     float64
	PA     *float64
	Opp    *string
}
type charge struct {
	Ts     int64
	Amount float64
}

// Contract is a roster snapshot, not the season-wide contract directory.
type Contract struct {
	Salary *float64 `json:"salary"`
	Until  *int64   `json:"until"`
	Status *string  `json:"status"`
	Note   string   `json:"note"`
	Slot   *string  `json:"slot"`
}
type roster struct {
	Players map[string]Contract
	Order   []string
}

// Lab is the reference compiler's state, derived solely from checked facts.
type Lab struct {
	Capture            int64
	Names              map[string]map[int64]*string
	PlayerNames        map[string]string
	Positions          map[string]map[int64]string
	Birth              map[string]int64
	NFLDraft           map[string]nflDraft
	WeekStart, WeekEnd map[yearWeek]int64
	Regular, LastDone  map[int64]int64
	Done               map[yearWeek]bool
	Scores             map[yearWeek]map[string]float64
	ScoreOrder         map[yearWeek][]string
	ResultOrder        []teamWeek
	Results            map[teamWeek]gameResult
	Starts             map[teamWeek][]string
	Rosters            map[teamWeek]roster
	RosterOrder        []teamWeek
	Dead               map[teamYear][]charge
	Cap                map[int64]*float64
	Coverage           map[int64]float64
	RankAfter          map[yearWeek]map[string]standing
	Final              map[int64]map[string]int64
	Playoffs           map[int64]map[string]bool
	Champion, Finalist map[int64]string
	Marks              map[int64]Markers
	SeasonPts          map[playerYear]float64
	SeasonGames        map[playerYear]int64
	WeeklyOrder        []playerYear
	WeeklyPts          map[playerYear]map[int64]float64
	SeasonRank         map[playerYear]int64
	StarterLine        map[int64]map[string]*int64
}

func newLab(f facts.Facts, capture int64) (*Lab, error) {
	l := &Lab{Capture: capture, Names: map[string]map[int64]*string{}, PlayerNames: map[string]string{}, Positions: map[string]map[int64]string{}, Birth: map[string]int64{}, NFLDraft: map[string]nflDraft{}, WeekStart: map[yearWeek]int64{}, WeekEnd: map[yearWeek]int64{}, Regular: map[int64]int64{}, LastDone: map[int64]int64{}, Done: map[yearWeek]bool{}, Scores: map[yearWeek]map[string]float64{}, ScoreOrder: map[yearWeek][]string{}, Results: map[teamWeek]gameResult{}, Starts: map[teamWeek][]string{}, Rosters: map[teamWeek]roster{}, Dead: map[teamYear][]charge{}, Cap: map[int64]*float64{}, Coverage: map[int64]float64{}, RankAfter: map[yearWeek]map[string]standing{}, Final: map[int64]map[string]int64{}, Playoffs: map[int64]map[string]bool{}, Champion: map[int64]string{}, Finalist: map[int64]string{}, Marks: map[int64]Markers{}, SeasonPts: map[playerYear]float64{}, SeasonGames: map[playerYear]int64{}, WeeklyPts: map[playerYear]map[int64]float64{}, SeasonRank: map[playerYear]int64{}, StarterLine: map[int64]map[string]*int64{}}
	if err := l.readReference(f); err != nil {
		return nil, err
	}
	l.readWeeks(f)
	l.buildStandings()
	l.buildPlayoffs(f)
	l.buildProduction()
	for y := int64(firstYear); y <= lastYear; y++ {
		l.Coverage[y] = l.contractCoverage(y)
	}
	return l, nil
}

func (l *Lab) readReference(f facts.Facts) error {
	for i, pid := range f.Player.Pid {
		l.PlayerNames[pid] = f.Player.Name[i]
		if born := f.Player.Birthdate[i]; born != nil && *born != "" {
			date, err := time.Parse("2006-01-02", *born)
			if err != nil {
				return fmt.Errorf("player %s birthdate: %w", pid, err)
			}
			l.Birth[pid] = easternEpoch(date)
		}
		if y := f.Player.DraftYear[i]; y != nil && *y != 0 {
			l.NFLDraft[pid] = nflDraft{*y, f.Player.DraftRound[i], f.Player.DraftPick[i]}
		}
	}
	for i, fid := range f.FranchiseSeason.Fid {
		if l.Names[fid] == nil {
			l.Names[fid] = map[int64]*string{}
		}
		l.Names[fid][f.FranchiseSeason.Year[i]] = f.FranchiseSeason.Name[i]
	}
	for i, y := range f.Season.Year {
		l.Cap[y] = f.Season.SalaryCap[i]
	}
	for i, pid := range f.PlayerSeason.Pid {
		pos := normPos(f.PlayerSeason.Position[i])
		if pos == nil {
			continue
		}
		if l.Positions[pid] == nil {
			l.Positions[pid] = map[int64]string{}
		}
		l.Positions[pid][f.PlayerSeason.Year[i]] = *pos
	}
	return nil
}

func (l *Lab) pos(pid string, year int64) *string {
	byYear := l.Positions[pid]
	if len(byYear) == 0 {
		return nil
	}
	if p, ok := byYear[year]; ok {
		return &p
	}
	years := sortedKeys(byYear)
	near := years[0]
	for _, y := range years {
		if abs64(y-year) < abs64(near-year) {
			near = y
		}
	}
	return ptr(byYear[near])
}

func ptr[T ~string | ~int64 | ~float64 | ~bool](v T) *T { return &v }
func value[T ~string | ~int64 | ~float64 | ~bool](p *T) T {
	if p != nil {
		return *p
	}
	var zero T
	return zero
}
func sortedKeys[K ~string | ~int64, V string | *string | int64 | standing | map[int64]*string](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
func positions() []string { return []string{"QB", "RB", "WR", "TE", "PK", "DT", "DE", "LB", "CB", "S"} }
func normPos(p *string) *string {
	if p == nil {
		return nil
	}
	if *p == "K" {
		return ptr("PK")
	}
	if slices.Contains(positions(), *p) {
		return p
	}
	return nil
}
func eraOf(y int64) (string, error) {
	switch {
	case y >= 2013 && y <= 2014:
		return "founding", nil
	case y <= 2016 && y >= 2015:
		return "cap", nil
	case y <= 2020 && y >= 2017:
		return "draft", nil
	case y <= 2023 && y >= 2021:
		return "taxi", nil
	case y <= 2026 && y >= 2024:
		return "current", nil
	default:
		return "", fmt.Errorf("unknown era year %d", y)
	}
}
