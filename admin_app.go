package main

import (
	"context"
	"time"
)

// The admin console: every calibration parameter with the value in effect, and admin overrides.

// ParamView is one calibration parameter as the admin panel shows it: its definition and the
// value in effect. Position is empty for a league-wide parameter.
type ParamView struct {
	Key         string  `json:"key"`
	Position    string  `json:"position"`
	Description string  `json:"description"`
	Default     float64 `json:"default"`
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
	Value       float64 `json:"value"`
	Calibrated  bool    `json:"calibrated"` // the default came from the fit tool, not a hand setting
}

// ParamsResult is the admin panel payload.
type ParamsResult struct {
	OK     bool        `json:"ok"`
	Error  string      `json:"error"`
	Params []ParamView `json:"params"`
}

// GetParams returns every calibration parameter with the value in effect.
func (a *App) GetParams() ParamsResult {
	if err := a.ready(); err != nil {
		return ParamsResult{OK: false, Error: err.Error()}
	}
	values := a.params.Snapshot()
	defs := a.params.Definitions()
	out := make([]ParamView, len(defs))
	for i, d := range defs {
		v, err := values.GetPosition(d.Key, d.Position)
		if err != nil {
			return ParamsResult{OK: false, Error: err.Error()}
		}
		out[i] = ParamView{Key: d.Key, Position: d.Position, Description: d.Description,
			Default: d.Default, Min: d.Min, Max: d.Max, Value: v, Calibrated: d.IsCalibrated}
	}
	return ParamsResult{OK: true, Params: out}
}

// SetParamResult is the admin-write payload.
type SetParamResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// SetParam applies an admin override to a parameter; position is empty for a league-wide one.
// The params store validates the range.
func (a *App) SetParam(key, position string, value float64) SetParamResult {
	if err := a.ready(); err != nil {
		return SetParamResult{OK: false, Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
	defer cancel()
	if err := a.params.SetOverride(ctx, key, position, value, "admin console"); err != nil {
		return SetParamResult{OK: false, Error: err.Error()}
	}
	return SetParamResult{OK: true}
}
