package fit

import (
	"errors"
	"math"
)

var errSingular = errors.New("fit: system is singular")

// ridge solves weighted least squares with an L2 penalty lambda on every coefficient but the
// first (the intercept column of ones).
func ridge(x [][]float64, y, w []float64, lambda float64) ([]float64, error) {
	p := len(x[0])
	a := make([][]float64, p)
	b := make([]float64, p)
	for i := range a {
		a[i] = make([]float64, p)
	}
	for r, row := range x {
		for i := range p {
			b[i] += w[r] * row[i] * y[r]
			for j := range p {
				a[i][j] += w[r] * row[i] * row[j]
			}
		}
	}
	for i := 1; i < p; i++ {
		a[i][i] += lambda
	}
	return cholesky(a, b)
}

// cholesky solves a·x = b for a symmetric positive-definite a.
func cholesky(a [][]float64, b []float64) ([]float64, error) {
	n := len(a)
	l := make([][]float64, n)
	for i := range l {
		l[i] = make([]float64, n)
		for j := 0; j <= i; j++ {
			sum := a[i][j]
			for k := range j {
				sum -= l[i][k] * l[j][k]
			}
			if i == j {
				if sum <= 1e-12 {
					return nil, errSingular
				}
				l[i][i] = math.Sqrt(sum)
			} else {
				l[i][j] = sum / l[j][j]
			}
		}
	}
	z := make([]float64, n)
	for i := range n {
		sum := b[i]
		for k := range i {
			sum -= l[i][k] * z[k]
		}
		z[i] = sum / l[i][i]
	}
	out := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		sum := z[i]
		for k := i + 1; k < n; k++ {
			sum -= l[k][i] * out[k]
		}
		out[i] = sum / l[i][i]
	}
	return out, nil
}

// logistic fits P(y=1) = σ(x·β) by iteratively reweighted least squares with an L2 penalty.
func logistic(x [][]float64, y []float64, lambda float64) ([]float64, error) {
	beta := make([]float64, len(x[0]))
	for range 50 {
		w := make([]float64, len(x))
		z := make([]float64, len(x))
		for r, row := range x {
			eta := dot(row, beta)
			p := sigmoid(eta)
			v := max(p*(1-p), 1e-6)
			w[r] = v
			z[r] = eta + (y[r]-p)/v
		}
		next, err := ridge(x, z, w, lambda)
		if err != nil {
			return nil, err
		}
		change := 0.0
		for i := range beta {
			change = max(change, math.Abs(next[i]-beta[i]))
		}
		beta = next
		if change < 1e-8 {
			break
		}
	}
	return beta, nil
}

func dot(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

func sigmoid(z float64) float64 { return 1 / (1 + math.Exp(-z)) }

// grid returns n log-spaced points from lo to hi.
func grid(lo, hi float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = lo * math.Pow(hi/lo, float64(i)/float64(n-1))
	}
	return out
}
