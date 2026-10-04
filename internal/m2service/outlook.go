package m2service

import (
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
)

// Outlook is a franchise's context columns. They are shown beside the board and never enter the
// power score: on Legacy NFL's 2021–2025 seasons none of them added to what the roster and the
// results already predicted (docs/modules/M2_Power_Ranking_Factors.md).
type Outlook struct {
	// CapRoom is the cap less what MFL charges against it; DeadCap the salary adjustments within
	// that charge, in $M.
	CapRoom, DeadCap float64
	HasCap           bool
	// Luck is head-to-head wins above what the franchise's all-play rate would have earned.
	Luck    float64
	HasLuck bool
	// ProjW and ProjL are the projected regular-season record: wins so far plus each remaining
	// game's win chance.
	ProjW, ProjL float64
	HasProj      bool
}

// OutlookInputs is what the context columns read. Now is every player's on-field-now value,
// whichever view is ranked: the projected record is about this season's games. Schedule is nil
// when it could not be read, and SeasonWeeks is the last regular-season week.
type OutlookInputs struct {
	Standings   []leaguestandings.RawStanding
	Now         []PlayerValue
	Schedule    []leagueschedule.RawScheduleWeek
	SeasonWeeks int
	DeadCap     map[string]domain.Money
}

// Outlooks returns each franchise's context columns. A column whose inputs are missing is left
// unset rather than failing the board.
func (s *Service) Outlooks(in OutlookInputs) (map[string]Outlook, error) {
	out := map[string]Outlook{}
	parsed := map[string]parsedStanding{}
	for _, st := range in.Standings {
		ps, err := parseStanding(st)
		if err != nil {
			return nil, err
		}
		parsed[st.FranchiseID] = ps
		out[st.FranchiseID] = s.capAndLuck(st.FranchiseID, ps, in.DeadCap)
	}
	for fid, w := range s.projectedWins(parsed, in) {
		o, ps := out[fid], parsed[fid]
		played := ps.h2hW + ps.h2hL + ps.h2hT
		won := float64(ps.h2hW) + 0.5*float64(ps.h2hT)
		o.ProjW, o.ProjL, o.HasProj = won+w.wins, float64(played+w.games)-won-w.wins, true
		out[fid] = o
	}
	return out, nil
}

// capAndLuck fills the columns that need only the standings, the cap and the mirror.
func (s *Service) capAndLuck(fid string, ps parsedStanding, deadCap map[string]domain.Money) Outlook {
	var o Outlook
	if limit, err := domain.ParseMoneyMillions(s.rb.GetSalaryCap()); err == nil {
		if used, ok := s.state.CapUsed(fid); ok {
			o.CapRoom, o.DeadCap, o.HasCap = (limit - used).Millions(), deadCap[fid].Millions(), true
		}
	}
	if games := ps.allPlayW + ps.allPlayL + ps.allPlayT; games > 0 {
		played := float64(ps.h2hW + ps.h2hL + ps.h2hT)
		o.Luck, o.HasLuck = float64(ps.h2hW)+0.5*float64(ps.h2hT)-ps.allPlayWinPct*played, true
	}
	return o
}

// remaining is a franchise's expected wins and the count of its games left.
type remaining struct {
	wins  float64
	games int
}

// projectedWins is each franchise's expected wins in the regular-season games not yet in the
// standings, from its lineup's on-field-now points blended with its points per game so far.
// None without a schedule or a season length.
func (s *Service) projectedWins(parsed map[string]parsedStanding, in OutlookInputs) map[string]remaining {
	if in.Schedule == nil || in.SeasonWeeks <= 0 {
		return nil
	}
	played := 0
	for _, ps := range parsed {
		played = max(played, ps.h2hW+ps.h2hL+ps.h2hT)
	}
	lineup := s.counter(AggLineup)
	byFranchise := s.byFranchise(in.Now)
	expected := map[string]float64{}
	for fid, ps := range parsed {
		value, _ := lineup.count(byFranchise[fid])
		expected[fid] = powerrankings.ExpectedPoints(value, ps.pf, ps.h2hW+ps.h2hL+ps.h2hT)
	}
	var games []powerrankings.Game
	for _, w := range in.Schedule {
		week, err := strconv.Atoi(strings.TrimSpace(w.Week))
		if err != nil || week <= played || week > in.SeasonWeeks {
			continue
		}
		for _, m := range w.Matchups {
			games = append(games, powerrankings.Game{Home: m.Franchises[0].FranchiseID, Away: m.Franchises[1].FranchiseID})
		}
	}
	wins := powerrankings.ExpectedWins(games, expected)
	out := map[string]remaining{}
	for fid := range expected {
		r := remaining{wins: wins[fid]}
		for _, g := range games {
			if g.Home == fid || g.Away == fid {
				r.games++
			}
		}
		out[fid] = r
	}
	return out
}
