package main

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/defense"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/kicker"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/offense"
	"github.com/secureprospective/TheWarRoom/internal/harness"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// rubrics is the Layer-4 registry: one real rubric per position.
func (a *App) rubrics() harness.RubricRegistry {
	return harness.RubricRegistry{
		domain.PosQB: offense.NewQB(),
		domain.PosRB: offense.NewRB(),
		domain.PosWR: offense.NewWR(),
		domain.PosTE: offense.NewTE(),
		domain.PosDT: defense.NewDT(),
		domain.PosDE: defense.NewDE(),
		domain.PosLB: defense.NewLB(),
		domain.PosCB: defense.NewCB(),
		domain.PosS:  defense.NewS(),
		domain.PosK:  kicker.NewK(),
	}
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
	rows := harness.RankRookies(a.assembler(), harness.SampleRookies(), a.rubrics())
	return RookiesResult{OK: true, L4Mode: "identity / scouting baseline", Rows: rows}
}

// ValidationResult is the validation-suite payload. OK means the suite ran, not that every
// case passed: read failures from Summary.Fail, so PENDING is never mistaken for FAIL.
type ValidationResult struct {
	OK      bool                 `json:"ok"`
	Cases   []harness.CaseResult `json:"cases"`
	Summary harness.Summary      `json:"summary"`
}

// RunValidationSuite evaluates the architectural cases against the current rubric registry.
func (a *App) RunValidationSuite() ValidationResult {
	cases := harness.RunValidationSuite(a.rubrics())
	return ValidationResult{OK: true, Cases: cases, Summary: harness.Summarize(cases)}
}

// ParamsResult is the admin panel payload: each calibration parameter with its default, range
// and effective value.
type ParamsResult struct {
	OK     bool              `json:"ok"`
	Error  string            `json:"error"`
	Params []params.ParamDef `json:"params"`
}

// GetParams returns the calibration parameters for the admin panel.
func (a *App) GetParams() ParamsResult {
	if err := a.ready(); err != nil {
		return ParamsResult{OK: false, Error: err.Error()}
	}
	return ParamsResult{OK: true, Params: a.params.Definitions()}
}

// SetParamResult is the admin-write payload.
type SetParamResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// SetParam applies an admin override to a global calibration value. The params store
// validates the range.
func (a *App) SetParam(key string, value float64) SetParamResult {
	if err := a.ready(); err != nil {
		return SetParamResult{OK: false, Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
	defer cancel()
	if err := a.params.SetOverride(ctx, key, "", value, "harness admin panel"); err != nil {
		return SetParamResult{OK: false, Error: err.Error()}
	}
	return SetParamResult{OK: true}
}
