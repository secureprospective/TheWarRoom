package labcompile

import "strings"

func classLabel(c string) string {
	kind, band, _ := strings.Cut(c, ":")
	if strings.HasPrefix(kind, "pick") {
		round := map[string]string{"pick1": "1st", "pick2": "2nd", "pick3": "3rd or later"}[kind]
		when := "a later draft"
		if band == "next" {
			when = "next draft"
		}
		return round + "-round pick, " + when
	}
	if kind == "player" {
		return "Player with no record"
	}
	names := map[string]string{"QB": "Quarterback", "RB": "Running back", "WR": "Wide receiver", "TE": "Tight end", "PK": "Kicker", "DL": "Defensive lineman", "LB": "Linebacker", "DB": "Defensive back"}
	ages := map[string]string{"young": "24 or younger", "prime": "25 to 27", "older": "28 or older"}
	return names[kind] + ", " + ages[band]
}
func priceInputs(rows []tradeValue) []valuedSides {
	out := make([]valuedSides, len(rows))
	for i, row := range rows {
		for s, side := range row.Sides {
			for _, it := range side {
				out[i][s] = append(out[i][s], it.valuedAsset)
			}
		}
	}
	return out
}
func marketSplits(rows []tradeValue) map[string][]tradeValue {
	out := map[string][]tradeValue{"odd": {}, "even": {}, "early": {}, "late": {}, "spring": {}, "summer": {}, "season": {}}
	for _, row := range rows {
		parity := "even"
		if row.Season%2 != 0 {
			parity = "odd"
		}
		era := "late"
		if row.Season <= 2021 {
			era = "early"
		}
		out[parity] = append(out[parity], row)
		out[era] = append(out[era], row)
		out[row.Window] = append(out[row.Window], row)
	}
	return out
}
func allCosted(row tradeValue) bool {
	for _, side := range row.Sides {
		for _, it := range side {
			if it.Cost == nil {
				return false
			}
		}
	}
	return true
}
func market(rows []tradeValue, classes []string, delivered map[string]*MarketDelivered) Market {
	full, meta := fitPrices(priceInputs(rows), classes, false)
	out := Market{Classes: []MarketClass{}, Anchor: anchor, Discount: discount}
	if meta == nil {
		return out
	}
	out.PerAsset = pythonRound(meta.PerAsset, 4)
	out.PerAssetSE = pythonRound(meta.PerAssetSE, 4)
	out.Trades = int64(meta.Trades)
	splits := marketSplits(rows)
	fitted := map[string]map[string]classPrice{}
	for name, subset := range splits {
		prices, _ := fitPrices(priceInputs(subset), classes, false)
		fitted[name] = prices
	}
	costed := []tradeValue{}
	for _, row := range rows {
		if allCosted(row) {
			costed = append(costed, row)
		}
	}
	_, costMeta := fitPrices(priceInputs(costed), classes, true)
	if costMeta != nil {
		out.PerCapMillion = ptr(pythonRound(costMeta.PerCapMillion, 4))
		out.PerCapMillionSE = ptr(pythonRound(costMeta.PerCapMillionSE, 4))
		out.CapTrades = int64(costMeta.Trades)
	}
	for _, c := range classes {
		f := full[c]
		values := classValues(rows, c)
		row := MarketClass{ID: c, Label: classLabel(c), N: int64(len(values)), Price: pythonRound(f.Price, 3), Lo: pythonRound(f.Price-1.96*f.SE, 3), Hi: pythonRound(f.Price+1.96*f.SE, 3), AvgValue: pythonRound(preciseMean(values), 3), Delivered: delivered[c]}
		row.Odd = priceSplit(splits["odd"], fitted["odd"], c)
		row.Even = priceSplit(splits["even"], fitted["even"], c)
		row.Early = priceSplit(splits["early"], fitted["early"], c)
		row.Late = priceSplit(splits["late"], fitted["late"], c)
		row.Spring = priceSplit(splits["spring"], fitted["spring"], c)
		row.Summer = priceSplit(splits["summer"], fitted["summer"], c)
		row.Season = priceSplit(splits["season"], fitted["season"], c)
		if d := row.Delivered; d != nil {
			if interval := deliveredInterval(row.Price, f.SE, d.Ratio, d.SE); interval != nil {
				row.PerDelivered = &DeliveredPrice{interval[0], interval[1], interval[2]}
			}
		}
		out.Classes = append(out.Classes, row)
	}
	return out
}
func classValues(rows []tradeValue, class string) []float64 {
	out := []float64{}
	for _, row := range rows {
		for _, side := range row.Sides {
			for _, it := range side {
				if it.Class == class {
					out = append(out, discounted(it.Value, discount))
				}
			}
		}
	}
	return out
}
func priceSplit(rows []tradeValue, prices map[string]classPrice, class string) *PriceSplit {
	p, ok := prices[class]
	n := len(classValues(rows, class))
	if !ok || n == 0 {
		return nil
	}
	return &PriceSplit{pythonRound(p.Price, 3), pythonRound(p.SE, 3), int64(n)}
}
