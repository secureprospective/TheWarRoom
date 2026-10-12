package labcompile

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

type Season struct {
	Year              int64             `json:"year"`
	Era               string            `json:"era"`
	Cap               *float64          `json:"cap"`
	RosterSize        *float64          `json:"rosterSize"`
	Taxi              *float64          `json:"taxi"`
	RegularWeeks      int64             `json:"regularWeeks"`
	ContractsRecorded float64           `json:"contractsRecorded"`
	LastFinishedWeek  int64             `json:"lastFinishedWeek"`
	PlayoffTeams      *int64            `json:"playoffTeams"`
	WeeksDone         []int64           `json:"weeksDone"`
	WeekStart         map[int64]int64   `json:"weekStart"`
	StarterLine       map[string]*int64 `json:"starterLine"`
	Champion          *string           `json:"champion"`
	Finalist          *string           `json:"finalist"`
	Marks             CalendarOutput    `json:"marks"`
}
type CalendarOutput struct {
	Open             *string `json:"open"`
	NFLDraft         *string `json:"nflDraft"`
	DraftStart       *string `json:"draftStart"`
	DraftEnd         *string `json:"draftEnd"`
	Kickoff          *string `json:"kickoff"`
	Deadline         *string `json:"deadline"`
	DeadlineExpected bool    `json:"deadlineExpected"`
	RegularEnd       *string `json:"regularEnd"`
	SeasonEnd        *string `json:"seasonEnd"`
}
type RawSettings struct {
	RosterSize *string `json:"rosterSize"`
	TaxiSquad  *string `json:"taxiSquad"`
}

func (r RawSettings) Validate() error {
	for _, v := range []*string{r.RosterSize, r.TaxiSquad} {
		if v != nil && strings.ContainsRune(*v, '\x00') {
			return fmt.Errorf("season settings contain NUL")
		}
	}
	return nil
}
func num(p *string) *float64 {
	if p == nil {
		return nil
	}
	s := strings.ReplaceAll(strings.ReplaceAll(*p, "$", ""), ",", "")
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return nil
	}
	return &n
}
func (l *Lab) seasons(f facts.Facts) ([]Season, error) {
	settings := map[int64]RawSettings{}
	for i, y := range f.Season.Year {
		var raw RawSettings
		if err := json.Unmarshal([]byte(f.Season.Raw[i]), &raw); err != nil {
			return nil, fmt.Errorf("season %d settings: %w", y, err)
		}
		if err := raw.Validate(); err != nil {
			return nil, fmt.Errorf("season %d settings: %w", y, err)
		}
		settings[y] = raw
	}
	out := make([]Season, 0, lastYear-firstYear+1)
	for y := int64(firstYear); y <= lastYear; y++ {
		st, ok := settings[y]
		if !ok {
			return nil, fmt.Errorf("missing season settings for %d", y)
		}
		era, err := eraOf(y)
		if err != nil {
			return nil, err
		}
		s := Season{Year: y, Era: era, Cap: l.Cap[y], RosterSize: num(st.RosterSize), Taxi: num(st.TaxiSquad), RegularWeeks: l.Regular[y], ContractsRecorded: l.Coverage[y], LastFinishedWeek: l.LastDone[y], WeeksDone: []int64{}, WeekStart: map[int64]int64{}, StarterLine: l.StarterLine[y], Champion: nonempty(ptr(l.Champion[y])), Finalist: nonempty(ptr(l.Finalist[y])), Marks: calendarOutput(l.Marks[y])}
		if n := len(l.Playoffs[y]); n > 0 {
			s.PlayoffTeams = ptr(int64(n))
		}
		for w := int64(1); w <= l.Regular[y]; w++ {
			if l.Done[yearWeek{y, w}] {
				s.WeeksDone = append(s.WeeksDone, w)
			}
		}
		for key, ts := range l.WeekStart {
			if key.Year == y {
				s.WeekStart[key.Week] = ts
			}
		}
		out = append(out, s)
	}
	return out, nil
}
func calendarOutput(m Markers) CalendarOutput {
	return CalendarOutput{iso(m.Open), iso(m.NFLDraft), iso(m.DraftStart), iso(m.DraftEnd), iso(m.Kickoff), iso(m.Deadline), m.DeadlineExpected, iso(m.RegularEnd), iso(m.SeasonEnd)}
}

type Franchise struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Short       string   `json:"short"`
	FormerNames []string `json:"formerNames"`
	Division    *string  `json:"division"`
}

func (l *Lab) franchises(f facts.Facts) ([]Franchise, error) {
	divisions := map[teamYear]*string{}
	for i, y := range f.FranchiseSeason.Year {
		divisions[teamYear{y, f.FranchiseSeason.Fid[i]}] = f.FranchiseSeason.Division[i]
	}
	out := []Franchise{}
	for _, fid := range sortedKeys(l.Names) {
		names := l.Names[fid]
		years := sortedKeys(names)
		latest := value(names[years[len(years)-1]])
		parts := strings.Fields(latest)
		if len(parts) == 0 {
			return nil, fmt.Errorf("franchise %s has no latest name", fid)
		}
		former := map[string]bool{}
		for _, name := range names {
			if value(name) != latest {
				former[value(name)] = true
			}
		}
		rows := []string{}
		for name := range former {
			rows = append(rows, name)
		}
		sortStrings(rows)
		out = append(out, Franchise{fid, latest, parts[len(parts)-1], rows, divisions[teamYear{years[len(years)-1], fid}]})
	}
	return out, nil
}

type Era struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Years   [2]int64 `json:"years"`
	Summary string   `json:"summary"`
}

func eras() []Era {
	return []Era{
		{"founding", "Founding years", [2]int64{2013, 2014}, "No salary cap or contracts yet. 53-man rosters and waiver claims. 2013 is only partly recorded (the league joined MFL mid-season); 12-week regular season and 12-team playoffs from 2014."},
		{"cap", "Cap arrives", [2]int64{2015, 2016}, "$100 cap and contracts begin. Rookie draft held off MFL, so traded picks are not recorded. Cap space could still be traded."},
		{"draft", "Rookie draft on MFL", [2]int64{2017, 2020}, "Rookie draft moves onto MFL and picks are recorded in trades. Free-agent adds replace waivers. Cap trading abolished. Rosters grow to 60, then 75. 12-team playoffs, 14 from 2020."},
		{"taxi", "Taxi squad, 13 weeks", [2]int64{2021, 2023}, "Practice squad of 8 arrives. Regular season becomes 13 weeks with 14-team playoffs. Cap $105, then $110. Defensive linemen score tackles."},
		{"current", "Current rules", [2]int64{2024, 2026}, "48-man active roster. Cap rises $5 a year ($115 to $125). New defensive-back scoring. Rookie draft in July. No trading from the deadline until March."},
	}
}
