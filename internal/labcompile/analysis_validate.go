package labcompile

import (
	"fmt"
	"math"
)

// RawAnalysis is the external recording at the value-analysis seam.
type RawAnalysis Analysis

func (r RawAnalysis) Validate() error {
	if r.Market.Anchor == "" || r.Market.Classes == nil || r.Checks2.Forecast == nil {
		return fmt.Errorf("analysis is missing market or checks")
	}
	for _, values := range [][]AnalysisPick{r.PickSlots, r.PickSeasons} {
		for _, p := range values {
			if len(p.Values) != 5 {
				return fmt.Errorf("analysis pick %d/%d/%d needs five horizons", p.Year, p.Round, p.Slot)
			}
			if err := finitePointers(p.Values...); err != nil {
				return err
			}
		}
	}
	for _, v := range r.Legs {
		if err := finitePointers(v.Value, v.Price, v.Real); err != nil {
			return err
		}
	}
	for _, v := range r.Delivered {
		if math.IsNaN(v.Expected) || math.IsInf(v.Expected, 0) || math.IsNaN(v.Delivered) || math.IsInf(v.Delivered, 0) {
			return fmt.Errorf("analysis delivery is not finite")
		}
	}
	return nil
}
func finitePointers(values ...*float64) error {
	for _, v := range values {
		if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0)) {
			return fmt.Errorf("analysis value is not finite")
		}
	}
	return nil
}
func (a Analysis) validateIndices(trades []Trade) error {
	if err := a.validateLegIndices(trades); err != nil {
		return err
	}
	deals := map[int]bool{}
	for _, v := range a.Deals {
		if v.Trade < 0 || v.Trade >= len(trades) || deals[v.Trade] {
			return fmt.Errorf("analysis deal has invalid or duplicate trade index")
		}
		deals[v.Trade] = true
	}
	for _, v := range a.Delivered {
		if v.Trade < 0 || v.Trade >= len(trades) {
			return fmt.Errorf("analysis delivery has invalid trade index")
		}
	}
	return nil
}

func (a Analysis) validateLegIndices(trades []Trade) error {
	legs := map[legKey]bool{}
	for _, v := range a.Legs {
		if v.Trade < 0 || v.Trade >= len(trades) || v.Side < 0 || v.Side > 1 {
			return fmt.Errorf("analysis leg has invalid trade or side")
		}
		assets := trades[v.Trade].AGave
		if v.Side == 1 {
			assets = trades[v.Trade].BGave
		}
		key := legKey{v.Trade, v.Side, v.Asset}
		if v.Asset < 0 || v.Asset >= len(assets) || legs[key] {
			return fmt.Errorf("analysis leg has invalid or duplicate asset index")
		}
		legs[key] = true
	}
	return nil
}
