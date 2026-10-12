package labcompile

import (
	"hash/crc32"
	"math"
	"math/big"
)

func foldOf(pid string) int { return int(crc32.ChecksumIEEE([]byte(pid)) % 5) }

func discounted(vector []float64, discount float64) float64 {
	total := 0.0
	for k, v := range vector {
		total += v * math.Pow(discount, float64(k))
	}
	return total
}

// Decimal ties are decided from the exact binary input, not from x*10^digits.
func pythonRound(x float64, digits int) float64 {
	if x == 0 || math.IsNaN(x) || math.IsInf(x, 0) || digits > 323 {
		return x
	}
	if digits < -308 {
		return math.Copysign(0, x)
	}
	r := new(big.Rat).SetFloat64(math.Abs(x))
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(absInt(digits))), nil)
	if digits >= 0 {
		r.Mul(r, new(big.Rat).SetInt(scale))
	} else {
		r.Quo(r, new(big.Rat).SetInt(scale))
	}
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(r.Num(), r.Denom(), rem)
	twice := new(big.Int).Lsh(rem, 1)
	comparison := twice.Cmp(r.Denom())
	if comparison > 0 || (comparison == 0 && q.Bit(0) == 1) {
		q.Add(q, big.NewInt(1))
	}
	result := new(big.Rat).SetInt(q)
	if digits >= 0 {
		result.Quo(result, new(big.Rat).SetInt(scale))
	} else {
		result.Mul(result, new(big.Rat).SetInt(scale))
	}
	value, _ := result.Float64()
	return math.Copysign(value, x)
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

type forecastFeatures struct {
	Age float64 `json:"age"`
	R0  float64 `json:"r0"`
	R1  float64 `json:"r1"`
	GM  float64 `json:"gm"`
}

func forecastX(r forecastFeatures) []float64 {
	a := r.Age - 26
	return []float64{1, r.R0, r.R1, a, a * a, r.R0 * a, r.GM}
}

type forecastAccuracy struct {
	N         int     `json:"n"`
	Explained float64 `json:"explained"`
}

func forecastCheck(pairs [][2]float64) *forecastAccuracy {
	if len(pairs) < 30 {
		return nil
	}
	values := make([]float64, len(pairs))
	for i, p := range pairs {
		values[i] = p[0]
	}
	mean := preciseMean(values)
	sse, sst := 0.0, 0.0
	for _, p := range pairs {
		errorTerm, deviation := p[0]-p[1], p[0]-mean
		sse += errorTerm * errorTerm
		sst += deviation * deviation
	}
	if sst == 0 || math.IsInf(sse, 0) || math.IsInf(sst, 0) {
		return nil
	}
	return &forecastAccuracy{len(pairs), pythonRound(1-sse/sst, 3)}
}

// Python statistics.mean sums exact float ratios before rounding once.
func preciseMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := new(big.Rat)
	for _, v := range values {
		total.Add(total, new(big.Rat).SetFloat64(v))
	}
	total.Quo(total, new(big.Rat).SetInt64(int64(len(values))))
	result, _ := total.Float64()
	return result
}

// Python 3.13's built-in sum retains a low-order compensation for float inputs.
func pythonSum(values []float64) float64 {
	hi, lo := 0.0, 0.0
	for _, x := range values {
		sum := hi + x
		if math.Abs(hi) >= math.Abs(x) {
			lo += (hi - sum) + x
		} else {
			lo += (x - sum) + hi
		}
		hi = sum
	}
	return hi + lo
}
