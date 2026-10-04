package modelrun

import (
	"math"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// The case set (plan Stage 7): each case values made-up players with the shipped fitted params
// and checks what a GM would expect. Season 2026, before its first game.

const season = 2026

func shipped(t *testing.T) (map[domain.Position]model.Params, model.Horizon) {
	t.Helper()
	p, err := newPass(model.Data{}, Spec{Season: season, Params: params.DefaultSet()})
	if err != nil {
		t.Fatal(err)
	}
	return p.params, p.horizon
}

// player is born on September 1 of the year that makes him age in 2026 and entered the league in
// rookie.
func player(pos domain.Position, age, rookie int, pick float64) *model.Player {
	return &model.Player{Position: pos, Birth: time.Date(season-age, 9, 1, 0, 0, 0, 0, time.UTC), Rookie: rookie,
		DraftPick: pick}
}

// seasons is a run of full seasons ending in 2025 at the given percentiles, oldest first.
func seasons(pcts ...float64) []model.Past {
	out := make([]model.Past, len(pcts))
	for i, p := range pcts {
		out[i] = model.Past{Year: season - len(pcts) + i, Games: 16, Pct: p}
	}
	return out
}

func value(t *testing.T, pl *model.Player, past []model.Past) model.Value {
	t.Helper()
	ps, h := shipped(t)
	scale := model.NewScale([]float64{2, 4, 6, 8, 10, 12, 14, 16, 18, 20})
	v := ps[pl.Position].Value(pl, past, model.Past{Year: season}, h, scale)
	t.Logf("%s age %.0f: now %.3f (%.1f ppg) dynasty %.3f prior %.3f Zpast %.2f plays next %.2f", pl.Position,
		pl.AgeAt(season), v.Now, v.NowPPG, v.Dynasty, v.Prior, v.ZPast, v.OnField)
	return v
}

func TestCaseRookiesByDraftRoundAreValuedAndOrdered(t *testing.T) {
	first := value(t, player(domain.PosWR, 22, season, 5), nil)
	third := value(t, player(domain.PosWR, 22, season, 70), nil)
	undrafted := value(t, player(domain.PosWR, 22, season, 0), nil)
	if undrafted.Now <= 0 || undrafted.Dynasty <= 0 {
		t.Errorf("a rookie must never be zero: %+v", undrafted)
	}
	if !(first.Now > third.Now && third.Now > undrafted.Now) {
		t.Errorf("rookies by draft slot: %v, %v, %v", first.Now, third.Now, undrafted.Now)
	}
}

func TestCaseYearTwoJump(t *testing.T) {
	second := value(t, player(domain.PosWR, 23, season-1, 40), seasons(0.5))
	third := value(t, player(domain.PosWR, 23, season-2, 40), seasons(0.5))
	if second.Now <= third.Now {
		t.Errorf("a second-year WR (%v) must get the year-2 jump over a third-year one (%v)", second.Now, third.Now)
	}
}

func TestCasePeakVeteran(t *testing.T) {
	v := value(t, player(domain.PosWR, 27, season-5, 20), seasons(0.9, 0.92, 0.92))
	if v.Now < 0.8 || v.Dynasty >= v.Now {
		t.Errorf("a peak WR: now %v should stay high and dynasty %v sit below it", v.Now, v.Dynasty)
	}
}

func TestCaseArcsDifferAt31(t *testing.T) {
	rb := value(t, player(domain.PosRB, 31, season-9, 30), seasons(0.8, 0.8, 0.8))
	qb := value(t, player(domain.PosQB, 31, season-9, 30), seasons(0.8, 0.8, 0.8))
	if rb.Dynasty/rb.Now > qb.Dynasty/qb.Now-0.05 {
		t.Errorf("an age-31 RB (dynasty/now %v) must fade faster than an age-31 QB (%v)",
			rb.Dynasty/rb.Now, qb.Dynasty/qb.Now)
	}
}

func TestCaseInjuredStarterKeepsDiscountedEvidence(t *testing.T) {
	healthy := value(t, player(domain.PosWR, 26, season-4, 15), seasons(0.88, 0.9, 0.9))
	past := seasons(0.88, 0.9, 0.95)
	past[2].Games = 3
	injured := value(t, player(domain.PosWR, 26, season-4, 15), past)
	if injured.Now < healthy.Now-0.1 || injured.PastGames >= healthy.PastGames {
		t.Errorf("injured now %v (evidence %v) vs healthy %v (%v): his earlier seasons must carry him, with less weight",
			injured.Now, injured.PastGames, healthy.Now, healthy.PastGames)
	}
}

// The athleticism × age term was tested in Stage 6 and rejected (plan R6-9), so an athletic
// late-career lineman ages like any other; only his prior differs.
func TestCaseAthleticLateCareerLinemanAgesLikeAnyOther(t *testing.T) {
	fast := player(domain.PosDT, 31, season-9, 40)
	fast.Combine = map[string]float64{"forty": 4.75, "vertical": 34, "broad_jump": 118, "bench": 35}
	slow := player(domain.PosDT, 31, season-9, 40)
	slow.Combine = map[string]float64{"forty": 5.25, "vertical": 26, "broad_jump": 100, "bench": 22}
	a, b := value(t, fast, seasons(0.7, 0.7, 0.7)), value(t, slow, seasons(0.7, 0.7, 0.7))
	viaPrior := (1 - a.ZPast) * (a.Prior - b.Prior)
	if math.Abs((a.Now-b.Now)-viaPrior) > 1e-9 || math.Abs(a.Dynasty-b.Dynasty) > 0.05 {
		t.Errorf("athletic %+v vs not %+v: with no cushion only the prior may separate them", a, b)
	}
}

func TestCaseBackupWithASmallEfficientSampleIsShrunk(t *testing.T) {
	v := value(t, player(domain.PosWR, 25, season-3, 150), []model.Past{{Year: season - 1, Games: 3, Pct: 0.98}})
	if v.Now > 0.75 || v.Now <= v.Prior {
		t.Errorf("3 games at the 98th percentile: now %v must be pulled well back toward his prior %v", v.Now, v.Prior)
	}
}
