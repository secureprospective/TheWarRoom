package fit

import (
	"math"
	"math/rand/v2"
	"testing"

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
