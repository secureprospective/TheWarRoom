package assembly

import (
	"math"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// The college signals are read from the college seasons history holds for each player (the
// prior.college_* measures the Signals load stores). Each player gets two: the college
// production share for one season, and the breakout age, his age at the first scanned season a
// share crossed the breakout line. Offense reads yardage shares, IDP the mean of each position's
// defensive shares. A player with no season, birth date or position-defined share is an ordinary
// miss and is left out, so the rubric reads him as neutral.

// The breakout lines: offense on yardage share, IDP on the lower averaged component share.
const (
	BreakoutThreshold    = 0.20
	BreakoutThresholdIDP = 0.12
)

// CollegeSignals are the college scouting signals for rostered players.
type CollegeSignals struct {
	Share    map[playerid.PlayerID]float64 // production share in the last completed college season
	Breakout map[playerid.PlayerID]float64 // age at the first scanned season over the breakout line
}

// BuildCollege reads each rostered player's share for season year and his breakout age over
// seasons from players, history's facts keyed by MFL id.
func BuildCollege(players map[string]*model.Player, year int, seasons []int, rosterMFLIDs []string,
	pos PositionLookup) CollegeSignals {
	out := CollegeSignals{Share: map[playerid.PlayerID]float64{}, Breakout: map[playerid.PlayerID]float64{}}
	for _, mfl := range rosterMFLIDs {
		pid, err := playerid.New(mfl)
		if err != nil {
			continue
		}
		p, ok := pos.Position(mfl)
		pl := players[mfl]
		if !ok || pl == nil {
			continue
		}
		if s, ok := collegeShare(pl.College[year], p); ok {
			out.Share[pid] = s
		}
		if age, ok := breakoutAge(pl, seasons, p); ok {
			out.Breakout[pid] = age
		}
	}
	return out
}

// breakoutAge is the player's age on September 1 of the first season, in ascending order, whose
// breakout share reaches the position's line.
func breakoutAge(pl *model.Player, seasons []int, pos domain.Position) (float64, bool) {
	threshold := BreakoutThreshold
	if isIDP(pos) {
		threshold = BreakoutThresholdIDP
	}
	for _, yr := range seasons {
		s, ok := breakoutShare(pl.College[yr], pos)
		if !ok {
			continue
		}
		if s < threshold {
			continue
		}
		age := pl.AgeAt(yr)
		if math.IsNaN(age) || age < 0 {
			return 0, false
		}
		return age, true
	}
	return 0, false
}

func isIDP(pos domain.Position) bool {
	switch pos {
	case domain.PosDT, domain.PosDE, domain.PosLB, domain.PosCB, domain.PosS:
		return true
	case domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK, domain.PosFlag:
	}
	return false
}

// share is a player's part of his team's total for a college measure.
func share(c map[string]float64, measure string) float64 {
	return ingestion.Share(c[measure], c["team_"+measure])
}

// collegeShare is the production share: receiving yards at WR and TE, 0.70 rushing + 0.30
// receiving yards at RB, and at IDP the mean of the position's components: CB passes defended
// and interceptions; S interceptions and tackles; LB tackles, sacks and tackles for loss; DT and
// DE tackles for loss and sacks. ok is false with no season or no share for the position.
func collegeShare(c map[string]float64, pos domain.Position) (float64, bool) {
	if c == nil {
		return 0, false
	}
	switch pos {
	case domain.PosWR, domain.PosTE:
		return share(c, "receiving_yards"), true
	case domain.PosRB:
		return 0.70*share(c, "rushing_yards") + 0.30*share(c, "receiving_yards"), true
	case domain.PosCB:
		return mean(share(c, "passes_defended"), share(c, "interceptions")), true
	case domain.PosS:
		return mean(share(c, "interceptions"), share(c, "total_tackles")), true
	case domain.PosLB:
		return mean(share(c, "total_tackles"), share(c, "sacks"), share(c, "tackles_for_loss")), true
	case domain.PosDT, domain.PosDE:
		return mean(share(c, "tackles_for_loss"), share(c, "sacks")), true
	case domain.PosQB, domain.PosK, domain.PosFlag:
	}
	return 0, false
}

// breakoutShare is the share the breakout line reads: receiving yards at WR and TE, rushing
// yards at RB, and the production share at IDP.
func breakoutShare(c map[string]float64, pos domain.Position) (float64, bool) {
	if c == nil {
		return 0, false
	}
	switch pos {
	case domain.PosWR, domain.PosTE:
		return share(c, "receiving_yards"), true
	case domain.PosRB:
		return share(c, "rushing_yards"), true
	case domain.PosDT, domain.PosDE, domain.PosLB, domain.PosCB, domain.PosS:
		return collegeShare(c, pos)
	case domain.PosQB, domain.PosK, domain.PosFlag:
	}
	return 0, false
}

func mean(vals ...float64) float64 {
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
