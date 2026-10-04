package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/harness"
)

// rubrics is the Layer-4 registry built from the current params, as a board run builds it.
func (a *App) rubrics() (harness.RubricRegistry, error) {
	reg, err := a.assembler().Rubrics()
	if err != nil {
		return nil, fmt.Errorf("app: build rubrics: %w", err)
	}
	return reg, nil
}

// assembler builds the composition boundary over the params store and the league's real cap,
// so the harness and the M1 board score against the same cap.
func (a *App) assembler() *composition.Assembler {
	return composition.New(a.params.Snapshot(), a.rulebook)
}

// RookiesResult is the rookie sandbox payload: ranked rows plus the active Layer-4 mode, so the
// UI can label the board honestly.
type RookiesResult struct {
	OK     bool                `json:"ok"`
	Error  string              `json:"error"`
	L4Mode string              `json:"l4Mode"`
	Rows   []harness.RookieRow `json:"rows"`
}

// ScoreRookies scores the sample rookie set and returns every intermediate for inspection.
func (a *App) ScoreRookies() RookiesResult {
	if err := a.ready(); err != nil {
		return RookiesResult{OK: false, Error: err.Error()}
	}
	reg, err := a.rubrics()
	if err != nil {
		return RookiesResult{OK: false, Error: err.Error()}
	}
	rows := harness.RankRookies(a.assembler(), harness.SampleRookies(), reg)
	return RookiesResult{OK: true, L4Mode: "identity / scouting baseline", Rows: rows}
}

// ValidationResult is the validation-suite payload. OK means the suite ran, not that every
// case passed: read failures from Summary.Fail, so PENDING is never mistaken for FAIL.
type ValidationResult struct {
	OK      bool                 `json:"ok"`
	Error   string               `json:"error"`
	Cases   []harness.CaseResult `json:"cases"`
	Summary harness.Summary      `json:"summary"`
}

// RunValidationSuite evaluates the architectural cases against the current rubric registry.
func (a *App) RunValidationSuite() ValidationResult {
	reg, err := a.rubrics()
	if err != nil {
		return ValidationResult{OK: false, Error: err.Error()}
	}
	cases := harness.RunValidationSuite(reg)
	return ValidationResult{OK: true, Cases: cases, Summary: harness.Summarize(cases)}
}

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
