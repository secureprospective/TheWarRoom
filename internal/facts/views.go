package facts

import (
	"math"
	"reflect"
	"strconv"
)

// Views are the Lab's named readings (views.sql), computed from the fact columns on demand and
// never stored: what "played", "regular season", "all-play" and a team's season mean, defined once.
type Views struct {
	PlayedGame  GameColumns
	RegularWeek RegularWeekColumns
	AllPlayWeek AllPlayWeekColumns
	TeamSeason  TeamSeasonColumns
}

type RegularWeek struct {
	Year int64 `json:"year"`
	Week int64 `json:"week"`
}

type AllPlayWeek struct {
	Year int64  `json:"year"`
	Week int64  `json:"week"`
	Fid  string `json:"fid"`
	W    int64  `json:"w"`
	L    int64  `json:"l"`
	T    int64  `json:"t"`
}

// TeamSeason sets MFL's official standings (mfl_*) beside the same figures counted from the weekly
// games (reg_*, all_weeks_*), so any difference is visible. All-play is MFL's where MFL publishes
// it (2022 on), otherwise counted MFL's way.
type TeamSeason struct {
	Year           int64    `json:"year"`
	Fid            string   `json:"fid"`
	Name           *string  `json:"name"`
	DivisionName   *string  `json:"division_name"`
	ConferenceName *string  `json:"conference_name"`
	MflW           *int64   `json:"mfl_w"`
	MflL           *int64   `json:"mfl_l"`
	MflT           *int64   `json:"mfl_t"`
	MflPf          *float64 `json:"mfl_pf"`
	MflPa          *float64 `json:"mfl_pa"`
	MflPp          *float64 `json:"mfl_pp"`
	RegW           *int64   `json:"reg_w"`
	RegL           *int64   `json:"reg_l"`
	RegT           *int64   `json:"reg_t"`
	RegPf          *float64 `json:"reg_pf"`
	RegGames       *int64   `json:"reg_games"`
	AllWeeksPf     *float64 `json:"all_weeks_pf"`
	AllWeeksPp     *float64 `json:"all_weeks_pp"`
	AllPlayW       *int64   `json:"all_play_w"`
	AllPlayL       *int64   `json:"all_play_l"`
	AllPlayT       *int64   `json:"all_play_t"`
	AllPlaySource  string   `json:"all_play_source"`
}

type RegularWeekColumns struct {
	Year []int64 `json:"year"`
	Week []int64 `json:"week"`
}

type AllPlayWeekColumns struct {
	Year []int64  `json:"year"`
	Week []int64  `json:"week"`
	Fid  []string `json:"fid"`
	W    []int64  `json:"w"`
	L    []int64  `json:"l"`
	T    []int64  `json:"t"`
}

type TeamSeasonColumns struct {
	Year           []int64    `json:"year"`
	Fid            []string   `json:"fid"`
	Name           []*string  `json:"name"`
	DivisionName   []*string  `json:"division_name"`
	ConferenceName []*string  `json:"conference_name"`
	MflW           []*int64   `json:"mfl_w"`
	MflL           []*int64   `json:"mfl_l"`
	MflT           []*int64   `json:"mfl_t"`
	MflPf          []*float64 `json:"mfl_pf"`
	MflPa          []*float64 `json:"mfl_pa"`
	MflPp          []*float64 `json:"mfl_pp"`
	RegW           []*int64   `json:"reg_w"`
	RegL           []*int64   `json:"reg_l"`
	RegT           []*int64   `json:"reg_t"`
	RegPf          []*float64 `json:"reg_pf"`
	RegGames       []*int64   `json:"reg_games"`
	AllWeeksPf     []*float64 `json:"all_weeks_pf"`
	AllWeeksPp     []*float64 `json:"all_weeks_pp"`
	AllPlayW       []*int64   `json:"all_play_w"`
	AllPlayL       []*int64   `json:"all_play_l"`
	AllPlayT       []*int64   `json:"all_play_t"`
	AllPlaySource  []string   `json:"all_play_source"`
}

type yearWeek struct{ year, week int64 }
type yearFid struct {
	year int64
	fid  string
}

// Views computes the named readings from the compiled columns.
func (f Facts) Views() Views {
	played := f.playedGames()
	regular := f.regularWeeks()
	allPlay := allPlayWeeks(played)
	var v Views
	fill(&v.PlayedGame, played)
	fill(&v.RegularWeek, regular)
	fill(&v.AllPlayWeek, allPlay)
	fill(&v.TeamSeason, f.teamSeasons(played, regular, allPlay))
	return v
}

// playedGames: a game counts as played once MFL has a score for it. MFL fills weeks not yet played
// with a blank score and a "T" result; those are not ties.
func (f Facts) playedGames() []Game {
	g := f.Game
	var out []Game
	for i := range g.Year {
		if g.Score[i] != nil {
			out = append(out, Game{Year: g.Year[i], Week: g.Week[i], Fid: g.Fid[i], Opp: g.Opp[i], IsHome: g.IsHome[i],
				Score: g.Score[i], Result: g.Result[i], OptPts: g.OptPts[i], Src: g.Src[i]})
		}
	}
	return out
}

// regularWeeks: the weeks in which every franchise had an opponent. MFL's own last-regular-week
// setting is not reliable for past seasons (it reads 16 for 2014).
func (f Facts) regularWeeks() []RegularWeek {
	teams := map[int64]int{}
	for _, year := range f.FranchiseSeason.Year {
		teams[year]++
	}
	withOpponent := map[yearWeek]int{}
	var weeks []yearWeek
	for i, year := range f.Game.Year {
		key := yearWeek{year, f.Game.Week[i]}
		if _, seen := withOpponent[key]; !seen {
			weeks = append(weeks, key)
			withOpponent[key] = 0
		}
		if f.Game.Opp[i] != nil {
			withOpponent[key]++
		}
	}
	var out []RegularWeek
	for _, key := range weeks {
		if withOpponent[key] == teams[key.year] {
			out = append(out, RegularWeek{Year: key.year, Week: key.week})
		}
	}
	return out
}

// allPlayWeeks scores each team each week against every other team that scored that week. MFL
// counts every week in which teams scored, playoff weeks included.
func allPlayWeeks(played []Game) []AllPlayWeek {
	byWeek := map[yearWeek][]Game{}
	var weeks []yearWeek
	for _, g := range played {
		key := yearWeek{g.Year, g.Week}
		if _, seen := byWeek[key]; !seen {
			weeks = append(weeks, key)
		}
		byWeek[key] = append(byWeek[key], g)
	}
	var out []AllPlayWeek
	for _, key := range weeks {
		games := byWeek[key]
		for _, a := range games {
			row := AllPlayWeek{Year: a.Year, Week: a.Week, Fid: a.Fid}
			for _, b := range games {
				switch {
				case b.Fid == a.Fid:
					continue
				case *a.Score > *b.Score:
					row.W++
				case *a.Score < *b.Score:
					row.L++
				default:
					row.T++
				}
			}
			if len(games) > 1 {
				out = append(out, row)
			}
		}
	}
	return out
}

// seasonTally accumulates one franchise's season from its games.
type seasonTally struct {
	w, l, t, games int64
	pf, pp         sum
	ppSeen         bool
}

// seasonTallies counts each franchise's regular-season record and points, and its points and best
// possible points over every week MFL scored.
func seasonTallies(played []Game, regular []RegularWeek) (reg, tot map[yearFid]*seasonTally) {
	isRegular := map[yearWeek]bool{}
	for _, r := range regular {
		isRegular[yearWeek{r.Year, r.Week}] = true
	}
	reg, tot = map[yearFid]*seasonTally{}, map[yearFid]*seasonTally{}
	for _, g := range played {
		key := yearFid{g.Year, g.Fid}
		t := tally(tot, key)
		t.pf.add(*g.Score)
		if g.OptPts != nil {
			t.pp.add(*g.OptPts)
			t.ppSeen = true
		}
		if !isRegular[yearWeek{g.Year, g.Week}] {
			continue
		}
		r := tally(reg, key)
		r.games++
		r.pf.add(*g.Score)
		switch value(g.Result) {
		case "W":
			r.w++
		case "L":
			r.l++
		case "T":
			r.t++
		}
	}
	return reg, tot
}

func (f Facts) teamSeasons(played []Game, regular []RegularWeek, allPlay []AllPlayWeek) []TeamSeason {
	reg, tot := seasonTallies(played, regular)
	ap := map[yearFid]*seasonTally{}
	for _, a := range allPlay {
		t := tally(ap, yearFid{a.Year, a.Fid})
		t.w, t.l, t.t = t.w+a.W, t.l+a.L, t.t+a.T
	}
	standings := map[yearFid]int{}
	for i, year := range f.Standings.Year {
		standings[yearFid{year, f.Standings.Fid[i]}] = i
	}
	fs := f.FranchiseSeason
	out := make([]TeamSeason, 0, len(fs.Year))
	for i, year := range fs.Year {
		key := yearFid{year, fs.Fid[i]}
		row := TeamSeason{Year: year, Fid: fs.Fid[i], Name: fs.Name[i], DivisionName: fs.DivisionName[i],
			ConferenceName: fs.ConferenceName[i], AllPlaySource: "counted"}
		if s, ok := standings[key]; ok {
			f.mflFigures(&row, s)
		}
		if r := reg[key]; r != nil {
			row.RegW, row.RegL, row.RegT, row.RegGames = ptr(r.w), ptr(r.l), ptr(r.t), ptr(r.games)
			row.RegPf = ptr(round2(r.pf.total()))
		}
		if t := tot[key]; t != nil {
			row.AllWeeksPf = ptr(round2(t.pf.total()))
			if t.ppSeen {
				row.AllWeeksPp = ptr(round2(t.pp.total()))
			}
		}
		if a := ap[key]; a != nil {
			row.AllPlayW, row.AllPlayL, row.AllPlayT = coalesce(row.AllPlayW, a.w), coalesce(row.AllPlayL, a.l), coalesce(row.AllPlayT, a.t)
		}
		out = append(out, row)
	}
	return out
}

// mflFigures copies MFL's standings row; MFL's own all-play, column by column, wins over the count.
func (f Facts) mflFigures(row *TeamSeason, i int) {
	s := f.Standings
	row.MflW, row.MflL, row.MflT = s.H2hW[i], s.H2hL[i], s.H2hT[i]
	row.MflPf, row.MflPa, row.MflPp = s.Pf[i], s.Pa[i], s.Pp[i]
	row.AllPlayW, row.AllPlayL, row.AllPlayT = s.AllPlayW[i], s.AllPlayL[i], s.AllPlayT[i]
	if s.AllPlayW[i] != nil {
		row.AllPlaySource = "MFL"
	}
}

func coalesce(held *int64, counted int64) *int64 {
	if held != nil {
		return held
	}
	return ptr(counted)
}

func tally(m map[yearFid]*seasonTally, key yearFid) *seasonTally {
	t := m[key]
	if t == nil {
		t = &seasonTally{}
		m[key] = t
	}
	return t
}

// sum adds floats the way SQLite's SUM does (Kahan-Babuska-Neumaier), so a rounded season total
// lands on the same cent as the Lab's.
type sum struct{ s, c float64 }

func (k *sum) add(x float64) {
	t := k.s + x
	if math.Abs(k.s) >= math.Abs(x) {
		k.c += (k.s - t) + x
	} else {
		k.c += (x - t) + k.s
	}
	k.s = t
}

func (k sum) total() float64 { return k.s + k.c }

// round2 is SQLite's ROUND(x, 2): the value printed to two decimals and read back.
func round2(x float64) float64 {
	r, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', 2, 64), 64)
	return r
}

// fill sorts rows by every column, as the facts are, and turns them into columns.
func fill[R any, C any](columns *C, rows []R) {
	sortRelation(reflect.ValueOf(rows))
	setColumns(reflect.ValueOf(rows), reflect.ValueOf(columns).Elem())
}
