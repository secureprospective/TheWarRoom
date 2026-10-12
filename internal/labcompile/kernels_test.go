package labcompile

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"math"
	"os"
	"testing"
)

type rawSolve struct {
	X       [][]float64 `json:"x"`
	Y       []float64   `json:"y"`
	Ridge   float64     `json:"ridge"`
	Free    int         `json:"free"`
	Weights []float64   `json:"weights"`
	Beta    []float64   `json:"beta"`
	Matrix  [][]float64 `json:"matrix"`
	Horizon *int        `json:"horizon"`
}
type rawFit struct {
	Indices  []int                 `json:"indices"`
	Classes  []string              `json:"classes"`
	WithCost bool                  `json:"withCost"`
	Prices   map[string]classPrice `json:"prices"`
	Meta     priceMeta             `json:"meta"`
}
type rawKernels struct {
	Solve       []rawSolve `json:"solve"`
	ForecastFit []rawSolve `json:"forecastFit"`
	Inverse     []struct {
		Matrix [][]float64 `json:"matrix"`
		Result [][]float64 `json:"result"`
	} `json:"inverse"`
	Fold []struct {
		PID    string `json:"pid"`
		Result int    `json:"result"`
	} `json:"fold"`
	Discounted []struct {
		Vector   []float64 `json:"vector"`
		Discount float64   `json:"discount"`
		Result   float64   `json:"result"`
	} `json:"discounted"`
	Round []struct {
		X      float64 `json:"x"`
		Digits int     `json:"digits"`
		Result float64 `json:"result"`
	} `json:"round"`
	LocalLinear []struct {
		Points    [][2]float64 `json:"points"`
		At        float64      `json:"at"`
		Bandwidth float64      `json:"bandwidth"`
		Result    *float64     `json:"result"`
	} `json:"localLinear"`
	ForecastX []struct {
		forecastFeatures
		Result []float64 `json:"result"`
	} `json:"forecastX"`
	ForecastCheck []struct {
		Pairs  map[string][][2]float64     `json:"pairs"`
		Result map[string]forecastAccuracy `json:"result"`
	} `json:"forecastCheck"`
	FitPrices   []rawFit      `json:"fitPrices"`
	PriceInputs []valuedSides `json:"priceInputs"`
	Interval    []struct {
		Price   float64 `json:"price"`
		SE      float64 `json:"se"`
		Ratio   float64 `json:"ratio"`
		RatioSE float64 `json:"ratioSE"`
		Result  struct {
			Value float64 `json:"value"`
			Lo    float64 `json:"lo"`
			Hi    float64 `json:"hi"`
		} `json:"result"`
	} `json:"interval"`
	BacktestRecords json.RawMessage `json:"backtestRecords"`
}

func kernels(t *testing.T) rawKernels {
	t.Helper()
	b, err := os.ReadFile("testdata/kernels.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	if len(b) >= 1<<20 {
		t.Fatal("kernel fixture exceeds 1 MiB")
	}
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(r)
	closeErr := r.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	var out rawKernels
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err := d.Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func near(t *testing.T, got, want float64) {
	t.Helper()
	if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > math.Max(1e-9, 1e-9*math.Max(math.Abs(got), math.Abs(want))) {
		t.Errorf("got %.17g, want %.17g", got, want)
	}
}

func vectorNear(t *testing.T, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length %d, want %d", len(got), len(want))
	}
	for i := range want {
		near(t, got[i], want[i])
	}
}
func matrixNear(t *testing.T, got, want [][]float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("rows %d, want %d", len(got), len(want))
	}
	for i := range want {
		vectorNear(t, got[i], want[i])
	}
}

func TestReferenceLinearKernels(t *testing.T) {
	f := kernels(t)
	for _, rows := range [][]rawSolve{f.Solve, f.ForecastFit} {
		for _, r := range rows {
			beta, a := solve(r.X, r.Y, r.Ridge, r.Free, r.Weights)
			vectorNear(t, beta, r.Beta)
			matrixNear(t, a, r.Matrix)
		}
	}
	for _, r := range f.Inverse {
		matrixNear(t, inverse(r.Matrix), r.Result)
	}
	for _, r := range f.LocalLinear {
		got := localLinear(r.Points, r.At, r.Bandwidth)
		if got == nil || r.Result == nil {
			if got != r.Result {
				t.Fatal("null mismatch")
			}
			continue
		}
		near(t, *got, *r.Result)
	}
}

func TestReferenceScalarKernels(t *testing.T) {
	f := kernels(t)
	for _, r := range f.Fold {
		if got := foldOf(r.PID); got != r.Result {
			t.Errorf("fold %s: %d != %d", r.PID, got, r.Result)
		}
	}
	for _, r := range f.Discounted {
		near(t, discounted(r.Vector, r.Discount), r.Result)
	}
	for _, r := range f.Round {
		got := pythonRound(r.X, r.Digits)
		if math.Float64bits(got) != math.Float64bits(r.Result) {
			t.Errorf("round(%g,%d): %.17g != %.17g", r.X, r.Digits, got, r.Result)
		}
	}
	for _, r := range f.ForecastX {
		vectorNear(t, forecastX(r.forecastFeatures), r.Result)
	}
}

func TestReferenceForecastCheck(t *testing.T) {
	for _, record := range kernels(t).ForecastCheck {
		for key, pairs := range record.Pairs {
			got := forecastCheck(pairs)
			want, ok := record.Result[key]
			if !ok {
				if got != nil {
					t.Fatalf("%s: expected absent check", key)
				}
				continue
			}
			if got == nil || got.N != want.N {
				t.Fatalf("%s: got %+v, want %+v", key, got, want)
			}
			near(t, got.Explained, want.Explained)
		}
	}
}

func TestReferencePriceFits(t *testing.T) {
	f := kernels(t)
	if len(f.FitPrices) != 9 {
		t.Fatal("full, seven splits and cost fit required")
	}
	for _, r := range f.FitPrices {
		trades := make([]valuedSides, len(r.Indices))
		for i, index := range r.Indices {
			trades[i] = f.PriceInputs[index]
		}
		prices, meta := fitPrices(trades, r.Classes, r.WithCost)
		if len(prices) != len(r.Prices) || meta == nil {
			t.Fatal("missing fitted classes")
		}
		for c, want := range r.Prices {
			got := prices[c]
			near(t, got.Price, want.Price)
			near(t, got.SE, want.SE)
		}
		near(t, meta.PerAsset, r.Meta.PerAsset)
		near(t, meta.PerAssetSE, r.Meta.PerAssetSE)
		if meta.Trades != r.Meta.Trades {
			t.Fatal("trade count differs")
		}
		if r.WithCost {
			near(t, meta.PerCapMillion, r.Meta.PerCapMillion)
			near(t, meta.PerCapMillionSE, r.Meta.PerCapMillionSE)
		}
	}
}

func TestReferenceDeliveredIntervals(t *testing.T) {
	for _, r := range kernels(t).Interval {
		got := deliveredInterval(r.Price, r.SE, r.Ratio, r.RatioSE)
		if got == nil {
			t.Fatal("valid reference interval absent")
		}
		vectorNear(t, got[:], []float64{r.Result.Value, r.Result.Lo, r.Result.Hi})
	}
}

func TestUndefinedFitsAreAbsent(t *testing.T) {
	if beta, a := solve(nil, nil, 1, 0, nil); beta != nil || a != nil {
		t.Fatal("empty solve fabricated a fit")
	}
	if fit, meta := fitPrices(nil, []string{anchor}, false); fit != nil || meta != nil {
		t.Fatal("empty split fabricated a fit")
	}
	if got := localLinear(nil, 1, 8); got != nil {
		t.Fatal("empty smoothing fabricated a value")
	}
	pairs := make([][2]float64, 30)
	for i := range pairs {
		pairs[i] = [2]float64{0, 0.1}
	}
	if forecastCheck(pairs) != nil {
		t.Fatal("zero variance fabricated an explained share")
	}
	if deliveredInterval(1, 1e6, 1, 1) != nil {
		t.Fatal("overflow fabricated an interval")
	}
}
