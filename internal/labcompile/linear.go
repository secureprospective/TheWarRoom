// Package labcompile contains the numerical kernels of the archive Lab compiler.
package labcompile

import (
	"math"
	"slices"
)

func matrix(n int) [][]float64 {
	out := make([][]float64, n)
	for i := range out {
		out[i] = make([]float64, n)
	}
	return out
}

func solve(x [][]float64, y []float64, ridge float64, free int, weights []float64) ([]float64, [][]float64) {
	if len(x) == 0 {
		return nil, nil
	}
	n := len(x[0])
	a, b := matrix(n), make([]float64, n)
	for r, row := range x {
		w := 1.0
		if len(weights) > 0 {
			w = weights[r]
		}
		accumulateNormal(a, b, row, y[r], w)
	}
	for i := free; i < n; i++ {
		a[i][i] += ridge
	}
	return gauss(a, b), a
}

func accumulateNormal(a [][]float64, b, row []float64, y, w float64) {
	for i, v := range row {
		xi := v * w
		if xi == 0 {
			continue
		}
		b[i] += xi * y
		for j, xj := range row {
			a[i][j] += xi * xj
		}
	}
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
		eliminate(m, c)
	}
	result := make([]float64, n)
	for i := range result {
		if math.Abs(m[i][i]) > 1e-12 {
			result[i] = m[i][n] / m[i][i]
		}
	}
	return result
}

func eliminate(m [][]float64, c int) {
	for r := range m {
		if r == c || m[r][c] == 0 {
			continue
		}
		f := m[r][c] / m[c][c]
		for j := c; j < len(m[r]); j++ {
			m[r][j] -= f * m[c][j]
		}
	}
}

func inverse(a [][]float64) [][]float64 {
	out := matrix(len(a))
	for j := range a {
		e := make([]float64, len(a))
		e[j] = 1
		col := gauss(a, e)
		for i := range a {
			out[i][j] = col[i]
		}
	}
	return out
}

func localLinear(points [][2]float64, at, bandwidth float64) *float64 {
	var sw, swx, swy, swxx, swxy float64
	for _, p := range points {
		distance := (p[0] - at) / bandwidth
		w := math.Exp(-0.5 * (distance * distance))
		if w < 1e-6 {
			continue
		}
		dx := p[0] - at
		sw += w
		swx += w * dx
		swy += w * p[1]
		swxx += w * dx * dx
		swxy += w * dx * p[1]
	}
	if sw == 0 {
		return nil
	}
	det := sw*swxx - swx*swx
	result := swy / sw
	if det > 1e-9 {
		result = math.Max(0, (swxx*swy-swx*swxy)/det)
	}
	return &result
}
