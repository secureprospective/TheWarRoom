package model

import "math"

// Past is one of a player's seasons as evidence: its year, the games he played and his
// percentile at his position.
type Past struct {
	Year  int
	Games float64
	Pct   float64
}

// recencyWeight is the weight of a season d seasons before the one projected: 1 for the
// season before, then the fitted weights; 0 outside three seasons.
func (p Params) recencyWeight(d int) float64 {
	switch d {
	case 1:
		return 1
	case 2, 3:
		return p.Recency[d-2]
	}
	return 0
}

// step is the arc step from season u to u+1. It is 0 when the player's age is unknown, and
// before his first NFL season: the prior already is his expected level as a rookie.
func (p Params) step(pl *Player, u int) float64 {
	age := pl.AgeAt(u)
	if math.IsNaN(age) || (pl.Rookie != 0 && u < pl.Rookie) {
		return 0
	}
	return p.ArcStep(age, pl.Experience(u))
}

// Project is the player's expected percentile in season target from his seasons before it.
// Each of the three seasons before target is moved along the arc to the season before target
// and weighted by recency and games; the weighted mean is blended with the prior by Z on the
// effective count (Σw·g)²/Σw²·g against KDynasty, and one more arc step carries the blend to
// target. With one season this is exactly the season-pair prediction the dynasty k and the arc
// were fitted on. e is the effective count.
func (p Params) Project(pl *Player, past []Past, target int) (pct, e float64) {
	var sw, swx, sw2 float64
	for _, s := range past {
		w := p.recencyWeight(target - s.Year)
		if w == 0 || s.Games <= 0 {
			continue
		}
		moved := s.Pct
		for u := s.Year; u < target-1; u++ {
			moved += p.step(pl, u)
		}
		sw += w * s.Games
		swx += w * s.Games * moved
		sw2 += w * w * s.Games
	}
	blend := p.Prior(pl)
	if sw > 0 {
		e = sw * sw / sw2
		z := p.Z(e, p.KDynasty)
		blend = z*swx/sw + (1-z)*blend
	}
	return blend + p.step(pl, target-1), e
}

// Horizon is how far ahead the dynasty measurable looks.
type Horizon struct {
	Seasons  int     // seasons counted, the one valued first
	Discount float64 // each season's weight against the one before
}

// Value is a player's two measurables for a season, on the percentile scale and in league
// points per game, with what produced them.
type Value struct {
	Now, NowPPG         float64 // expected level this season, per game he plays
	Dynasty, DynastyPPG float64 // discounted expected level over the horizon; a season off the field counts 0
	Prior               float64
	PastGames           float64 // effective games from earlier seasons
	SeasonGames         float64 // games this season
	ZPast, ZNow         float64 // weight on earlier seasons against the prior; on this season against them
	OnField             float64 // probability he plays next season
}

// fullSeason is the games a projected season is counted at: the risk of missing it is the
// survival arc's.
const fullSeason = 17.0

// Value scores pl in season from his earlier seasons and the games so far in this one. On-field-
// now blends this season's games, by k_now, with the projection from earlier seasons. Dynasty
// carries that level along the arc for h.Seasons seasons, each counted at the chance he is still
// on the field and discounted. scale turns percentiles back into points per game.
func (p Params) Value(pl *Player, past []Past, current Past, h Horizon, scale Scale) Value {
	season := current.Year
	proj, e := p.Project(pl, past, season)
	v := Value{Prior: p.Prior(pl), PastGames: e, SeasonGames: current.Games, Now: proj}
	if e > 0 {
		v.ZPast = p.Z(e, p.KDynasty)
	}
	if current.Games > 0 {
		v.ZNow = p.Z(current.Games, p.KNow)
		v.Now = v.ZNow*current.Pct + (1-v.ZNow)*proj
	}
	v.Now = unit(v.Now)
	v.NowPPG = scale.PPG(v.Now)

	onNow := p.onFieldNow(pl, past, current, proj)
	v.OnField = onNow * p.survives(pl, season, fullSeason, v.Now)
	next, _ := p.Project(pl, append(past[:len(past):len(past)], current), season+1)
	level, on := v.Now, onNow
	var weight, sumPct, sumPPG float64
	for t := range max(h.Seasons, 1) {
		switch {
		case t == 1:
			level, on = unit(next), v.OnField
		case t > 1:
			on *= p.survives(pl, season+t-1, fullSeason, level)
			level = unit(level + p.step(pl, season+t-1))
		}
		d := math.Pow(h.Discount, float64(t))
		weight += d
		sumPct += d * on * level
		sumPPG += d * on * scale.PPG(level)
	}
	v.Dynasty, v.DynastyPPG = sumPct/weight, sumPPG/weight
	return v
}

// onFieldNow is the chance he plays this season: 1 once he has; before his first NFL game, the
// chance a rookie at his draft slot becomes a regular; otherwise each season from his last one
// is survived in turn, a season he missed at 0 games.
func (p Params) onFieldNow(pl *Player, past []Past, current Past, proj float64) float64 {
	last := Past{Year: math.MinInt}
	for _, s := range past {
		if s.Games > 0 && s.Year > last.Year {
			last = s
		}
	}
	if current.Games > 0 {
		return 1
	}
	if last.Year == math.MinInt {
		return p.Debuts(pl.DraftPick)
	}
	on := p.survives(pl, last.Year, last.Games, last.Pct)
	for u := last.Year + 1; u < current.Year; u++ {
		on *= p.survives(pl, u, 0, proj)
	}
	return on
}

// survives is the chance pl plays season u+1, having played games at percentile pct in u. An
// unknown age is taken as the arc's centre.
func (p Params) survives(pl *Player, u int, games, pct float64) float64 {
	age := pl.AgeAt(u)
	if math.IsNaN(age) {
		age = ArcCenter
	}
	return p.Survives(age, pl.DraftPick, games, pct)
}

func unit(v float64) float64 { return min(max(v, 0), 1) }
