package main

import (
	"cmp"
	"context"
	"slices"
	"time"
)

// The admin console: every calibration parameter with the value in effect, and admin overrides.

// ParamView is one calibration parameter as the admin panel shows it: its definition and the
// value in effect. Position is empty for a league-wide parameter.
type ParamView struct {
	Key         string  `json:"key"`
	Label       string  `json:"label"`
	Group       string  `json:"group"`
	Position    string  `json:"position"`
	Description string  `json:"description"`
	Default     float64 `json:"default"`
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
	Value       float64 `json:"value"`
	Calibrated  bool    `json:"calibrated"` // the default came from the fit tool, not a hand setting
	Overridden  bool    `json:"overridden"` // set in Admin: pinned against later shipped defaults
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
		label, group := parameterWords(d.Key, d.Description)
		out[i] = ParamView{Key: d.Key, Label: label, Group: group, Position: d.Position, Description: d.Description,
			Default: d.Default, Min: d.Min, Max: d.Max, Value: v, Calibrated: d.IsCalibrated,
			Overridden: a.params.Overridden(d.Key, d.Position)}
	}
	slices.SortFunc(out, func(a, b ParamView) int {
		if n := cmp.Compare(a.Group, b.Group); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Label, b.Label); n != 0 {
			return n
		}
		return cmp.Compare(a.Position, b.Position)
	})
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

// ResetParam clears a parameter's admin override, so it follows its shipped default again.
func (a *App) ResetParam(key, position string) SetParamResult {
	if err := a.ready(); err != nil {
		return SetParamResult{OK: false, Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
	defer cancel()
	if err := a.params.ClearOverride(ctx, key, position); err != nil {
		return SetParamResult{OK: false, Error: err.Error()}
	}
	return SetParamResult{OK: true}
}
