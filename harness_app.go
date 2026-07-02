package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/defense"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/kicker"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/offense"
	"github.com/secureprospective/TheWarRoom/internal/harness"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// rubrics is the B5b Layer-4 registry. Each B5b block adds its position here, which both
// differentiates the rankings and auto-evaluates its Module-3 cases. QB (B5b-QB) is the
// first REAL offense rubric; DT (B5b-DT) is the first REAL defense rubric (the IDP
// skeleton-setter) and flips case 3F (SL-021 cushion guard). K (B5b-K) is the LAST L4 build:
// the real Madden-driven rubric (DECISION-011) — film ACTIVE (Madden 0.60 / NFLProduction 0.40),
// RAS + breakout forced to exactly 1.000, so Combined = film. Its RAS stays 1.000, keeping
// case 3C (SL-020, gates on BOTH QB and K) green. All 10/10 positions now run a real rubric.
func (a *App) rubrics() harness.RubricRegistry {
	return harness.RubricRegistry{
		domain.PosQB: offense.NewQB(),
		domain.PosRB: offense.NewRB(), // B5b-RB: standard offense rubric (flips case 3B)
		domain.PosWR: offense.NewWR(), // B5b-WR: the offense skill template (flips case 3A, 3K)
		domain.PosTE: offense.NewTE(), // B5b-TE: first SL-019 RAS-modulator instance (flips case 3M)
		domain.PosDT: defense.NewDT(), // B5b-DT: the defense skeleton-setter (first IDP rubric)
		domain.PosDE: defense.NewDE(), // B5b-DE: 2nd SL-019 instance, reuses curve.SL019 (flips case 3E)
		domain.PosLB: defense.NewLB(), // B5b-LB: first non-SL-019 IDP; SL-005 film + Medium RAS (flips 3D, 3J)
		domain.PosCB: defense.NewCB(), // B5b-CB: first NGS-anchor + 3rd SL-019 (reduced 0.30); flips case 3I
		domain.PosS:  defense.NewS(),  // B5b-S: 2nd NGS-anchor + 4th SL-019 (0.30); co-asserts case 3I
		domain.PosK:  kicker.NewK(),   // B5b-K: real Madden-driven film (DECISION-011); RAS+breakout forced 1.000 (keeps 3C green)
	}
}

// assembler builds the composition boundary over the real params store and the
// REAL rulebook cap (M1 — the sandboxCap fixed "1000" is gone: the league is
// loaded now, so the harness and the M1 board score against the same cap).
func (a *App) assembler() (*composition.Assembler, error) {
	if a.params == nil || a.rulebook == nil {
		return nil, fmt.Errorf("stores not initialized (params=%t rulebook=%t)", a.params != nil, a.rulebook != nil)
	}
	return composition.New(a.params, a.rulebook), nil
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
