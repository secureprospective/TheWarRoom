package lab

import (
	"math"
	"slices"
)

// Constants match engine.js:5–7, 322; ordered operations match its dense solver.
const (
	Anchor   = "pick1:next"
	Ridge    = 0.25
	Z        = 1.96
	MinWhole = 80
	MinPart  = 60
)

type Asset struct {
	Kind  string  `json:"kind"`
	Value float64 `json:"value"`
}
type ValuedTrade struct {
	ID    int        `json:"id"`
	Sides [2][]Asset `json:"sides"`
}
type Price struct {
	Price float64 `json:"price"`
	SE    float64 `json:"se"`
}
type Fit struct {
	Prices     map[string]Price `json:"prices"`
	Kinds      []string         `json:"kinds"`
	Covariance [][]float64      `json:"covariance"`
	PerAsset   float64          `json:"perAsset"`
	Trades     int              `json:"trades"`
}

func ValuedTrades(d *Data, keep func(Trade) bool) []ValuedTrade {
	out := []ValuedTrade{}
	for _, t := range d.Trades {
		if t.ANet == nil || (keep != nil && !keep(t)) {
			continue
		}
		v := ValuedTrade{ID: t.ID, Sides: [2][]Asset{{}, {}}}
		for _, l := range t.Legs {
			if l.XValue != nil && l.XClass != nil {
				v.Sides[l.Side] = append(v.Sides[l.Side], Asset{*l.XClass, *l.XValue})
			}
		}
		out = append(out, v)
	}
	return out
}

func matrix(n int) [][]float64 {
	m := make([][]float64, n)
	for i := range m {
		m[i] = make([]float64, n)
	}
	return m
}

func gauss(a [][]float64, b []float64) []float64 {
	n := len(b)
	m := make([][]float64, n)
	for i := range m {
		m[i] = append(slices.Clone(a[i]), b[i])
	}
	for c := range n {
		p := c
		for r := c + 1; r < n; r++ {
			if math.Abs(m[r][c]) > math.Abs(m[p][c]) {
				p = r
			}
		}
		m[c], m[p] = m[p], m[c]
		if math.Abs(m[c][c]) < 1e-12 {
			continue
		}
		for r := range n {
			if r == c || m[r][c] == 0 {
				continue
			}
			f := m[r][c] / m[c][c]
			for j := c; j <= n; j++ {
				m[r][j] -= f * m[c][j]
			}
		}
	}
	out := make([]float64, n)
	for i := range out {
		if math.Abs(m[i][i]) > 1e-12 {
			out[i] = m[i][n] / m[i][i]
		}
	}
	return out
}

func inverse(a [][]float64) [][]float64 {
	out := matrix(len(a))
	for j := range a {
		b := make([]float64, len(a))
		b[j] = 1
		col := gauss(a, b)
		for i := range a {
			out[i][j] = col[i]
		}
	}
	return out
}

func design(trades []ValuedTrade, kinds []string) ([][]float64, []float64) {
	idx := make(map[string]int)
	for i, k := range kinds {
		idx[k] = i
	}
	p := len(kinds) + 1
	x := make([][]float64, len(trades))
	y := make([]float64, len(trades))
	for n, v := range trades {
		x[n] = make([]float64, p)
		for side, assets := range v.Sides {
			sign := -1.0
			if side == 1 {
				sign = 1
			}
			for _, a := range assets {
				if i, ok := idx[a.Kind]; ok {
					x[n][i] += sign * a.Value
				}
				x[n][p-1] += sign
				y[n] -= sign * a.Value
			}
		}
	}
	return x, y
}

func normal(x [][]float64, y []float64, p int, dropRidge bool) ([][]float64, []float64) {
	a := matrix(p)
	b := make([]float64, p)
	for n, row := range x {
		for i := range p {
			if row[i] == 0 {
				continue
			}
			b[i] += row[i] * y[n]
			for j := range p {
				a[i][j] += row[i] * row[j]
			}
		}
	}
	for i := range p {
		if !dropRidge || i != 0 {
			a[i][i] += Ridge
		}
	}
	return a, b
}

func sandwich(x [][]float64, y, beta []float64, inv [][]float64) [][]float64 {
	p := len(beta)
	meat := matrix(p)
	for n, row := range x {
		sum := 0.0
		for i, xi := range row {
			sum += xi * beta[i]
		}
		e := y[n] - sum
		for i, xi := range row {
			if xi == 0 {
				continue
			}
			for j, xj := range row {
				if xj != 0 {
					meat[i][j] += xi * xj * e * e
				}
			}
		}
	}
	half := matrix(p)
	v := matrix(p)
	for i := range p {
		for c := range p {
			for a, x := range inv[i] {
				if x != 0 {
					half[i][c] += x * meat[a][c]
				}
			}
		}
	}
	for i := range p {
		for j := range p {
			for c, x := range half[i] {
				v[i][j] += x * inv[c][j]
			}
		}
	}
	return v
}

func FitPrices(trades []ValuedTrade) Fit { return fitPrices(trades, false) }

func fitPrices(trades []ValuedTrade, dropRidge bool) Fit {
	seen := make(map[string]bool)
	kinds := []string{}
	for _, v := range trades {
		for _, side := range v.Sides {
			for _, a := range side {
				if a.Kind != Anchor && !seen[a.Kind] {
					kinds = append(kinds, a.Kind)
					seen[a.Kind] = true
				}
			}
		}
	}
	slices.Sort(kinds)
	x, y := design(trades, kinds)
	a, b := normal(x, y, len(kinds)+1, dropRidge)
	beta := gauss(a, b)
	v := sandwich(x, y, beta, inverse(a))
	f := Fit{Prices: make(map[string]Price), Kinds: append(slices.Clone(kinds), Anchor), Covariance: matrix(len(kinds) + 1), PerAsset: beta[len(kinds)], Trades: len(trades)}
	for i, k := range kinds {
		f.Prices[k] = Price{1 + beta[i], math.Sqrt(math.Max(0, v[i][i]))}
		copy(f.Covariance[i][:len(kinds)], v[i][:len(kinds)])
	}
	f.Prices[Anchor] = Price{1, 0}
	return f
}

func (f Fit) Cov(a, b string) float64 {
	i := slices.Index(f.Kinds, a)
	j := slices.Index(f.Kinds, b)
	if i < 0 || j < 0 {
		return 0
	}
	return f.Covariance[i][j]
}
