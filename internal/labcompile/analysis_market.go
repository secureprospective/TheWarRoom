package labcompile

type Market struct {
	Classes         []MarketClass `json:"classes"`
	Anchor          string        `json:"anchor"`
	Discount        float64       `json:"discount"`
	PerAsset        float64       `json:"perAsset"`
	PerAssetSE      float64       `json:"perAssetSe"`
	Trades          int64         `json:"trades"`
	PerCapMillion   *float64      `json:"perCapMillion"`
	PerCapMillionSE *float64      `json:"perCapMillionSe"`
	CapTrades       int64         `json:"capTrades"`
}
type MarketClass struct {
	ID           string           `json:"id"`
	Label        string           `json:"label"`
	N            int64            `json:"n"`
	Price        float64          `json:"price"`
	Lo           float64          `json:"lo"`
	Hi           float64          `json:"hi"`
	AvgValue     float64          `json:"avgValue"`
	Odd          *PriceSplit      `json:"odd"`
	Even         *PriceSplit      `json:"even"`
	Early        *PriceSplit      `json:"early"`
	Late         *PriceSplit      `json:"late"`
	Spring       *PriceSplit      `json:"spring"`
	Summer       *PriceSplit      `json:"summer"`
	Season       *PriceSplit      `json:"season"`
	Delivered    *MarketDelivered `json:"delivered"`
	PerDelivered *DeliveredPrice  `json:"perDelivered"`
}
type PriceSplit struct {
	Price float64 `json:"price"`
	SE    float64 `json:"se"`
	N     int64   `json:"n"`
}
type DeliveredRatio struct {
	Ratio float64 `json:"ratio"`
	SE    float64 `json:"se"`
	N     int64   `json:"n"`
}
type MarketDelivered struct {
	Ratio float64         `json:"ratio"`
	SE    float64         `json:"se"`
	N     int64           `json:"n"`
	Early *DeliveredRatio `json:"early"`
	Late  *DeliveredRatio `json:"late"`
	Split int64           `json:"split"`
}
type DeliveredPrice struct {
	Value float64 `json:"value"`
	Lo    float64 `json:"lo"`
	Hi    float64 `json:"hi"`
}
type AnalysisChecks struct {
	Forecast     map[string]ForecastCheck `json:"forecast"`
	ValuedTrades int64                    `json:"valuedTrades"`
	Currency     *CurrencyCheck           `json:"currency"`
	Replacement  map[string]float64       `json:"replacement"`
	Backtest     *BacktestCheck           `json:"backtest"`
}
type ForecastCheck struct {
	N         int64   `json:"n"`
	Explained float64 `json:"explained"`
}
type CurrencyCheck struct {
	TeamSeasons int64   `json:"teamSeasons"`
	Correlation float64 `json:"correlation"`
}
type BacktestCheck struct {
	Trades  int64          `json:"trades"`
	Seasons [2]int64       `json:"seasons"`
	Bands   []BacktestBand `json:"bands"`
	Slope   float64        `json:"slope"`
	Won     float64        `json:"won"`
}
type BacktestBand struct {
	Expected float64 `json:"expected"`
	Realized float64 `json:"realized"`
	Won      float64 `json:"won"`
	N        int64   `json:"n"`
}
