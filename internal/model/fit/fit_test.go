package fit

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
)

func TestRidgeRecoversCoefficients(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a fixed test sample
	var x [][]float64
	var y, w []float64
	for range 500 {
		a, b := r.NormFloat64(), r.NormFloat64()
		x, y, w = append(x, []float64{1, a, b}), append(y, 0.5+2*a-1*b+0.01*r.NormFloat64()), append(w, 1)
	}
	beta, err := ridge(x, y, w, 1e-6)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []float64{0.5, 2, -1} {
		if math.Abs(beta[i]-want) > 0.01 {
			t.Errorf("beta[%d] = %v, want %v", i, beta[i], want)
		}
	}
}

func TestLogisticRecoversCoefficients(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4)) //nolint:gosec // a fixed test sample
	var x [][]float64
	var y []float64
	for range 20000 {
		a := r.NormFloat64()
		v := 0.0
		if r.Float64() < sigmoid(-0.5+1.5*a) {
			v = 1
		}
		x, y = append(x, []float64{1, a}), append(y, v)
	}
	beta, err := logistic(x, y, 1e-6)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(beta[0]+0.5) > 0.06 || math.Abs(beta[1]-1.5) > 0.08 {
		t.Errorf("beta = %v, want [-0.5 1.5]", beta)
	}
}

// Synthetic players whose weekly points are true level plus noise: the fitted k must come back
// near σ²/τ² and shrinking must beat the raw odd-week mean at predicting the even weeks.
func TestKNowRecoversNoiseOverSpread(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 6)) //nolint:gosec // a fixed test sample
	const tau, sigma = 2.0, 6.0      // k = 36/4 = 9 games
	d := model.Data{Players: map[string]*model.Player{}, Seasons: map[string]map[int]*model.Season{}}
	for i := range 3000 {
		id := string(rune('a'+i%26)) + string(rune('A'+i/26%26)) + string(rune('0'+i/676))
		d.Players[id] = &model.Player{ID: id, Position: domain.PosWR}
		d.Seasons[id] = map[int]*model.Season{}
		for _, year := range []int{2023, 2024} {
			level := 10 + tau*r.NormFloat64()
			s := &model.Season{Player: id, Year: year, Points: map[int]float64{}}
			for w := 1; w <= 2+r.IntN(15); w++ {
				s.Points[w] = level + sigma*r.NormFloat64()
			}
			d.Seasons[id][year] = s
		}
	}
	sm := collect(d, d.Scales(), domain.PosWR, 2023, 2024)
	var rep Report
	k := fitKNow(sm, 2024, &rep)
	if k < 7 || k > 11 {
		t.Errorf("k = %v, want about 9", k)
	}
	if !(rep.SplitShrunk < rep.SplitRaw) {
		t.Errorf("shrunk %v should beat raw %v", rep.SplitShrunk, rep.SplitRaw)
	}
}

// The fit is reproducible: the same data gives the same params on every run, whatever order Go
// walks its maps in.
func TestRunIsDeterministic(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 8)) //nolint:gosec // a fixed test sample
	d := model.Data{Players: map[string]*model.Player{}, Seasons: map[string]map[int]*model.Season{}}
	for i := range 400 {
		id := fmt.Sprintf("p%03d", i)
		d.Players[id] = &model.Player{ID: id, Position: domain.PosWR, Rookie: 2019 + i%5, DraftPick: float64(1 + i%250),
			Birth: time.Date(1996+i%6, 3, 1, 0, 0, 0, 0, time.UTC), Combine: map[string]float64{"forty": 4.3 + r.Float64()/2}}
		d.Seasons[id] = map[int]*model.Season{}
		for year := 2021; year <= 2025; year++ {
			s := &model.Season{Player: id, Year: year, Points: map[int]float64{}}
			for w := 1; w <= 1+r.IntN(17); w++ {
				s.Points[w] = 0.1 * float64(r.IntN(300))
			}
			d.Seasons[id][year] = s
		}
	}
	first := fmt.Sprintf("%+v", Run(d, 2021, 2025))
	for range 5 {
		if got := fmt.Sprintf("%+v", Run(d, 2021, 2025)); got != first {
			t.Fatal("two runs on the same data differ")
		}
	}
}

func TestSpearmanRanksWithTies(t *testing.T) {
	if got := spearman([][2]float64{{1, 10}, {2, 20}, {3, 30}, {4, 25}}); math.Abs(got-0.8) > 1e-12 {
		t.Errorf("spearman = %v, want 0.8", got)
	}
	if got := spearman([][2]float64{{1, 5}, {1, 6}, {1, 7}}); !math.IsNaN(got) {
		t.Errorf("a column with no spread has no correlation, got %v", got)
	}
}
