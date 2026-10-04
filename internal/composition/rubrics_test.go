package composition

import (
	"errors"
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// editedParams serves the shipped defaults with some per-position values changed.
type editedParams struct {
	fakeParams
	edits map[string]float64 // "key@POS"
}

func (e editedParams) GetPosition(key, position string) (float64, error) {
	if v, ok := e.edits[key+"@"+position]; ok {
		return v, nil
	}
	return e.fakeParams.GetPosition(key, position)
}

func rubricsWith(t *testing.T, edits map[string]float64) map[domain.Position]engine.Layer4 {
	t.Helper()
	p, c := goodStores()
	reg, err := New(editedParams{p, edits}, c).Rubrics()
	if err != nil {
		t.Fatalf("Rubrics: %v", err)
	}
	return reg
}

func eliteFilm() engine.Layer4Input {
	return engine.Layer4Input{Scouting: engine.ScoutingInput{FilmComposite: 1, HasFilm: true}}
}

func TestRubricsAtDefaultsAreTheShippedTable(t *testing.T) {
	reg := rubricsWith(t, nil)
	for pos, s := range l4.Defaults() {
		if got, want := reg[pos].Apply(eliteFilm()), l4.New(s).Apply(eliteFilm()); got != want {
			t.Errorf("%s: %+v, want %+v", pos, got, want)
		}
	}
}

func TestRubricsReadEditedSettings(t *testing.T) {
	reg := rubricsWith(t, map[string]float64{"l4.film.cap@WR": 0.10})
	if got := reg[domain.PosWR].Apply(eliteFilm()).FilmEffective; got < 1.09 || got > 1.10 {
		t.Errorf("WR film with cap 0.10 = %v, want about 1.0999", got)
	}
	if got := reg[domain.PosTE].Apply(eliteFilm()).FilmEffective; got > 1.05 {
		t.Errorf("TE film %v moved with a WR edit", got)
	}
}

// Doubling one breakout weight rescales all four, so a neutral profile stays neutral.
func TestRubricsRescaleEditedWeights(t *testing.T) {
	reg := rubricsWith(t, map[string]float64{"l4.breakout.weight.college_share@QB": 0.60})
	neutral := engine.Layer4Input{Player: engine.PlayerInput{Age: 32}}
	if got := reg[domain.PosQB].Apply(neutral).Combined; math.Abs(got-1) > 1e-12 {
		t.Errorf("neutral QB = %v after a weight edit, want 1", got)
	}
}

func TestRubricsFailOnAMissingSetting(t *testing.T) {
	p, c := goodStores()
	sentinel := errors.New("no such param")
	if _, err := New(missingParams{p, sentinel}, c).Rubrics(); !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want the missing param", err)
	}
}

type missingParams struct {
	fakeParams
	err error
}

func (m missingParams) GetPosition(string, string) (float64, error) { return 0, m.err }

// A kicker's film is composition's 0.60/0.40 blend of its two components.
func TestKickerFilmBlend(t *testing.T) {
	blend := func(m, n float64) float64 { return 0.60*m + 0.40*n } // at run time, as composition rounds
	for _, c := range []struct {
		madden, production *float64
		want               float64
		has                bool
	}{
		{ptr(0.9), ptr(0.8), blend(0.9, 0.8), true},
		{ptr(0.9), nil, blend(0.9, 0.5), true},
		{nil, nil, 0, false},
	} {
		s := PlayerSpec{Position: domain.PosK}
		if c.madden != nil {
			s.MaddenFilm, s.HasMaddenFilm = *c.madden, true
		}
		if c.production != nil {
			s.NFLProduction, s.HasNFLProduction = *c.production, true
		}
		p, cp := goodStores()
		sc := New(p, cp).scouting(s)
		if sc.HasFilm != c.has || sc.FilmComposite != c.want {
			t.Errorf("film = %v/%v, want %v/%v", sc.FilmComposite, sc.HasFilm, c.want, c.has)
		}
	}
}

func ptr(v float64) *float64 { return &v }

func TestDefaultSetHoldsEveryRubricSetting(t *testing.T) {
	set := params.DefaultSet()
	for pos, s := range l4.Defaults() {
		for _, k := range l4.Knobs() {
			if _, err := set.GetPosition(k.Key, string(pos)); (err == nil) != k.AppliesTo(s) {
				t.Errorf("%s at %s: stored = %v, applies = %v", k.Key, pos, err == nil, k.AppliesTo(s))
			}
		}
	}
}
