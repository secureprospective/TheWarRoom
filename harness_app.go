package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/harness"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// sandboxCap is the harness league-cap source. v1 runs without a loaded MFL league, so it
// supplies a fixed cap amount; production wires *rulebook.Store (GetSalaryCap) here instead.
// The salary-% cap tiers it feeds are still the real, admin-tunable B4 values.
type sandboxCap struct{}

func (sandboxCap) GetSalaryCap() string { return "1000" }

// rubrics is the B5b Layer-4 registry. EMPTY today (identity L4 only): Module 1 shows the
// scouting baseline and the rubric-gated Module 3 cases report PENDING. Each B5b block adds
// its position here, which both differentiates the rankings and auto-evaluates its cases.
func (a *App) rubrics() harness.RubricRegistry { return harness.RubricRegistry{} }

// assembler builds the composition boundary over the real params store and the sandbox cap.
func (a *App) assembler() (*composition.Assembler, error) {
	if a.params == nil {
		return nil, fmt.Errorf("params store not initialized")
	}
	return composition.New(a.params, sandboxCap{}), nil
}

// RookiesResult is the Module 1 payload: the ranked rows plus the active Layer-4 mode so the
// UI can label the board honestly ("identity / scouting baseline" until B5b lands).
type RookiesResult struct {
	OK     bool                `json:"ok"`
	Error  string              `json:"error"`
	L4Mode string              `json:"l4Mode"`
	Rows   []harness.RookieRow `json:"rows"`
}

// ScoreRookies runs the sample rookie set through the composition boundary and the engine,
// returning every scored intermediate for inspection. Fully typed (ifaceguard).
func (a *App) ScoreRookies() RookiesResult {
	asm, err := a.assembler()
	if err != nil {
		return RookiesResult{OK: false, Error: err.Error()}
	}
	rows := harness.RankRookies(asm, harness.SampleRookies(), a.rubrics())
	return RookiesResult{OK: true, L4Mode: "identity / scouting baseline", Rows: rows}
}

// ValidationResult is the Module 3 payload: the 12 case results plus the pass/fail/pending
// tally for the dashboard header. OK means the suite EXECUTED, not that every case passed —
// a real failure is read from Summary.Fail (the UI keys red/green off the tally, never OK),
// so PENDING is never conflated with FAIL (GLM review m2).
type ValidationResult struct {
	OK      bool                 `json:"ok"`
	Cases   []harness.CaseResult `json:"cases"`
	Summary harness.Summary      `json:"summary"`
}

// RunValidationSuite evaluates the 12 architectural cases against the current rubric
// registry and returns their PASS/FAIL/PENDING states.
func (a *App) RunValidationSuite() ValidationResult {
	cases := harness.RunValidationSuite(a.rubrics())
	return ValidationResult{OK: true, Cases: cases, Summary: harness.Summarize(cases)}
}

// ParamsResult is the admin panel payload: the live calibration definitions (key, default,
// range, current effective value is the default unless overridden).
type ParamsResult struct {
	OK     bool              `json:"ok"`
	Error  string            `json:"error"`
	Params []params.ParamDef `json:"params"`
}

// GetParams returns the B4 calibration definitions for the admin panel.
func (a *App) GetParams() ParamsResult {
	if a.params == nil {
		return ParamsResult{OK: false, Error: "params store not initialized"}
	}
	return ParamsResult{OK: true, Params: a.params.Definitions()}
}

// SetParamResult is the admin-write payload.
type SetParamResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// SetParam applies a live admin override to a global calibration value (cap-tier %, decay
// rate), so the operator can change a param and re-score to see the board move — the
// sandbox's whole point. Range/validation is enforced by the params store.
func (a *App) SetParam(key string, value float64) SetParamResult {
	if a.params == nil {
		return SetParamResult{OK: false, Error: "params store not initialized"}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
	defer cancel()
	if err := a.params.SetOverride(ctx, key, "", value, "harness admin panel"); err != nil {
		return SetParamResult{OK: false, Error: err.Error()}
	}
	return SetParamResult{OK: true}
}
