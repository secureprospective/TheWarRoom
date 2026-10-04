// Package model is the measurable (plan Stage 6–7): a player's production blended with a prior
// built from pre-NFL facts, on a within-position percentile scale. It is pure: callers read the
// measure store and hand it observations.
package model

import (
	"math"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// Obs is one stored value: a player's measure in a season and week. Season 0 is a fact with no
// period; week 0 a season total.
type Obs struct {
	Player  string
	Season  int
	Week    int
	Measure string
	Value   float64
	Text    string
}

// The measures the model reads.
const (
	mPosition    = "context.nfl_position"
	mBirth       = "context.birth_date"
	mRookie      = "context.rookie_season"
	mDraftPick   = "prior.draft_pick"
	mWeekPoints  = "outcome.weekly_fantasy_points"
	mOffSnaps    = "exposure.offense_snaps"
	mDefSnaps    = "exposure.defense_snaps"
	mTeamsSnaps  = "exposure.special_teams_snaps"
	collegePrefx = "prior.college_"
)

// Measures lists everything the model reads, for the caller's query.
func Measures() []string {
	out := []string{mPosition, mBirth, mRookie, mDraftPick, mWeekPoints, mOffSnaps, mDefSnaps, mTeamsSnaps}
	for _, c := range Combine() {
		out = append(out, "prior."+c)
	}
	for _, c := range collegeMeasures() {
		out = append(out, collegePrefx+c)
	}
	return out
}

// Combine is the athletic testing the prior reads.
func Combine() []string {
	return []string{"height", "weight", "forty", "vertical", "broad_jump", "bench", "three_cone", "shuttle"}
}

func collegeMeasures() []string {
	return []string{"receiving_yards", "team_receiving_yards", "rushing_yards", "team_rushing_yards",
		"pass_attempts", "passing_yards", "fg_attempts", "fg_made",
		"total_tackles", "team_total_tackles", "sacks", "team_sacks", "tackles_for_loss",
		"team_tackles_for_loss", "passes_defended", "team_passes_defended", "interceptions", "team_interceptions"}
}

// Player is one player's facts that do not change by season.
type Player struct {
	ID        string
	Position  domain.Position // "" when the position is not one the model scores
	Birth     time.Time       // zero when unknown
	Rookie    int             // first NFL season; 0 when unknown
	DraftPick float64         // overall pick; 0 when undrafted or unknown
	Combine   map[string]float64
	College   map[int]map[string]float64 // college season → measure (without the prior.college_ prefix)
}

// AgeAt is the player's age on September 1 of season, or NaN when the birth date is unknown.
func (p Player) AgeAt(season int) float64 {
	if p.Birth.IsZero() {
		return math.NaN()
	}
	return time.Date(season, time.September, 1, 0, 0, 0, 0, time.UTC).Sub(p.Birth).Hours() / 24 / 365.25
}

// Experience is season's NFL season number, 1 for a rookie, or 0 when unknown.
func (p Player) Experience(season int) int {
	if p.Rookie == 0 || season < p.Rookie {
		return 0
	}
	return season - p.Rookie + 1
}

// Season is a player's NFL season: the weeks he took a snap or scored, and his league points.
type Season struct {
	Player string
	Year   int
	Points map[int]float64 // league fantasy points by week played; 0 when he played and scored none
}

// Games is the number of weeks played.
func (s Season) Games() int { return len(s.Points) }

// PPG is points per game played, or NaN with no games.
func (s Season) PPG() float64 {
	if len(s.Points) == 0 {
		return math.NaN()
	}
	total := 0.0
	for _, p := range s.Points {
		total += p
	}
	return total / float64(len(s.Points))
}

// Data is everything the model knows: players by id, and seasons by player and year.
type Data struct {
	Players map[string]*Player
	Seasons map[string]map[int]*Season
}

// Build assembles players and seasons from observations. A week counts as played when the player
// took a snap or scored; weeks after lastWeek[season] (the league's fantasy season) are left out.
func Build(obs []Obs, lastWeek map[int]int) Data {
	d := Data{Players: map[string]*Player{}, Seasons: map[string]map[int]*Season{}}
	player := func(id string) *Player {
		p, ok := d.Players[id]
		if !ok {
			p = &Player{ID: id, Combine: map[string]float64{}, College: map[int]map[string]float64{}}
			d.Players[id] = p
		}
		return p
	}
	for _, o := range obs {
		switch {
		case o.Season == 0:
			player(o.Player).set(o)
		case strings.HasPrefix(o.Measure, collegePrefx):
			c := player(o.Player).College
			if c[o.Season] == nil {
				c[o.Season] = map[string]float64{}
			}
			c[o.Season][strings.TrimPrefix(o.Measure, collegePrefx)] = o.Value
		case o.Week >= 1 && o.Week <= lastWeek[o.Season]:
			d.week(o)
		}
	}
	return d
}

func (p *Player) set(o Obs) {
	switch o.Measure {
	case mPosition:
		p.Position = FitPosition(o.Text)
	case mBirth:
		if t, err := time.Parse("2006-01-02", o.Text); err == nil {
			p.Birth = t
		}
	case mRookie:
		p.Rookie = int(o.Value)
	case mDraftPick:
		p.DraftPick = o.Value
	default:
		if name, ok := strings.CutPrefix(o.Measure, "prior."); ok {
			p.Combine[name] = o.Value
		}
	}
}

func (d Data) week(o Obs) {
	switch o.Measure {
	case mWeekPoints, mOffSnaps, mDefSnaps, mTeamsSnaps:
	default:
		return
	}
	if d.Seasons[o.Player] == nil {
		d.Seasons[o.Player] = map[int]*Season{}
	}
	s := d.Seasons[o.Player][o.Season]
	if s == nil {
		s = &Season{Player: o.Player, Year: o.Season, Points: map[int]float64{}}
		d.Seasons[o.Player][o.Season] = s
	}
	if o.Measure == mWeekPoints {
		s.Points[o.Week] += o.Value
	} else if _, ok := s.Points[o.Week]; !ok && o.Value > 0 {
		s.Points[o.Week] = 0
	}
}

// FitPosition maps an nflverse position to the position the model scores it at. Generic "DB",
// specialists and offensive linemen map to "".
func FitPosition(nfl string) domain.Position {
	switch nfl {
	case "QB":
		return domain.PosQB
	case "RB", "FB":
		return domain.PosRB
	case "WR":
		return domain.PosWR
	case "TE":
		return domain.PosTE
	case "K":
		return domain.PosK
	case "DT", "NT", "DL":
		return domain.PosDT
	case "DE":
		return domain.PosDE
	case "LB", "OLB", "ILB", "MLB":
		return domain.PosLB
	case "CB":
		return domain.PosCB
	case "S", "SAF", "FS", "SS":
		return domain.PosS
	}
	return ""
}

// Positions are the positions the model scores.
func Positions() []domain.Position {
	return []domain.Position{domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK,
		domain.PosDT, domain.PosDE, domain.PosLB, domain.PosCB, domain.PosS}
}
