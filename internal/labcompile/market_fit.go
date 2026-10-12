package labcompile

import "math"

const (
	discount    = 0.75
	anchor      = "pick1:next"
	marketRidge = 0.25
)

type valuedAsset struct {
	Value []float64 `json:"value"`
	Class string    `json:"class"`
	Cost  []float64 `json:"cost"`
}
type valuedSides [2][]valuedAsset

type classPrice struct {
	Price float64 `json:"price"`
	SE    float64 `json:"se"`
}
type priceMeta struct {
	PerAsset        float64 `json:"perAsset"`
	PerAssetSE      float64 `json:"perAssetSe"`
	Trades          int     `json:"trades"`
	PerCapMillion   float64 `json:"perCapMillion"`
	PerCapMillionSE float64 `json:"perCapMillionSe"`
}

func fitPrices(trades []valuedSides, classes []string, withCost bool) (map[string]classPrice, *priceMeta) {
	if len(trades) == 0 {
		return nil, nil
	}
	idx := map[string]int{}
	for _, c := range classes {
		if c != anchor {
			idx[c] = len(idx)
		}
	}
	x, y := priceDesign(trades, idx, withCost)
	beta, a := solve(x, y, marketRidge, 0, nil)
	se := robustSE(x, y, beta, inverse(a))
	prices := make(map[string]classPrice, len(classes))
	for c, i := range idx {
		prices[c] = classPrice{1 + beta[i], se[i]}
	}
	prices[anchor] = classPrice{1, 0}
	meta := &priceMeta{PerAsset: beta[len(idx)], PerAssetSE: se[len(idx)], Trades: len(trades)}
	if withCost {
		meta.PerCapMillion = beta[len(beta)-1]
		meta.PerCapMillionSE = se[len(se)-1]
	}
	return prices, meta
}

func priceDesign(trades []valuedSides, idx map[string]int, withCost bool) ([][]float64, []float64) {
	extra := len(idx) + 1
	if withCost {
		extra++
	}
	x, y := make([][]float64, len(trades)), make([]float64, len(trades))
	for n, sides := range trades {
		x[n] = make([]float64, extra)
		for side, items := range sides {
			sign := -1.0
			if side == 1 {
				sign = 1
			}
			for _, it := range items {
				priceAsset(x[n], idx, it, sign, withCost)
			}
		}
		y[n] = sideValue(sides[0]) - sideValue(sides[1])
	}
	return x, y
}

func priceAsset(row []float64, idx map[string]int, it valuedAsset, sign float64, withCost bool) {
	if i, ok := idx[it.Class]; ok {
		row[i] += sign * discounted(it.Value, discount)
	}
	row[len(idx)] += sign
	if withCost {
		row[len(idx)+1] += sign * discounted(it.Cost, discount)
	}
}

func sideValue(items []valuedAsset) float64 {
	total := 0.0
	for _, it := range items {
		total += discounted(it.Value, discount)
	}
	return total
}

func robustSE(x [][]float64, y, beta []float64, inv [][]float64) []float64 {
	k := len(beta)
	meat := matrix(k)
	for n, row := range x {
		predicted := 0.0
		for i, xi := range row {
			predicted += beta[i] * xi
		}
		residualMeat(meat, row, y[n]-predicted)
	}
	tmp := matrix(k)
	for i := range k {
		for b := range k {
			for a := range k {
				tmp[i][b] += inv[i][a] * meat[a][b]
			}
		}
	}
	se := make([]float64, k)
	for i := range k {
		variance := 0.0
		for b := range k {
			variance += tmp[i][b] * inv[b][i]
		}
		se[i] = math.Sqrt(math.Max(0, variance))
	}
	return se
}

func residualMeat(meat [][]float64, row []float64, e float64) {
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

func deliveredInterval(price, se, ratio, ratioSE float64) *[3]float64 {
	if ratio <= 0 {
		return nil
	}
	per := price / ratio
	priceError, ratioError := se/math.Max(price, 1e-6), ratioSE/ratio
	rel := math.Sqrt(priceError*priceError + ratioError*ratioError)
	low, high := per*math.Exp(-1.96*rel), per*math.Exp(1.96*rel)
	if math.IsInf(low, 0) || math.IsInf(high, 0) || math.IsNaN(low) || math.IsNaN(high) {
		return nil
	}
	out := [3]float64{pythonRound(per, 3), pythonRound(low, 3), pythonRound(high, 3)}
	return &out
}
