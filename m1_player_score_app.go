package main

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// This file is the M1 single-player breakdown IPC — the read behind the B-4a Contextual Inspector.
// GetRankings returns the summary board row (base / agePull / L4Combined / adjusted / capEff); the
// Inspector needs the FULL per-player score anatomy — the Layer-4 sub-signals (Film / RAS / Breakout)
// that compose Combined. Every field is already PERSISTED in season_scores and read back by the
// output store's Score() getter, so this is a pure read projection — no recompute, no schema change.
// Split from m1_app.go to keep both within the 400-line file cap (AD-14/AD-17).

// PlayerScoreDTO is one player's full scoring breakdown as the Inspector renders it — the score-
// dominant hero (AdjustedScore) plus the six layer bars that build it and the contract/cap block.
//
// The bars map to the real pipeline (engine: AdjustedScore ← ScoutingAdjusted = BasePoints × AgePull
// × Layer4.Combined, then the cap overlay): Base (L2) · Age (L3) · Film / Athleticism(RAS) / Breakout
// (the Layer-4 sub-signals) · Cap (CapMultiplier + tier). FilmRaw is DELIBERATELY absent — it is a
// DEBUG/sandbox-only engine field ("never UI", engine/types.go), so it never crosses this boundary.
type PlayerScoreDTO struct {
	MFLID       string `json:"mflID"`
	Name        string `json:"name"`
	Position    string `json:"position"`
	FranchiseID string `json:"franchiseID"`

	// Layer bars (the six that compose the score).
	BasePoints        float64 `json:"basePoints"`        // L2 — MFL YTD proxy (see Label)
	AgePull           float64 `json:"agePull"`           // L3 — age-decay multiplier
	FilmEffective     float64 `json:"filmEffective"`     // L4 — post-effective film signal
	RASEffective      float64 `json:"rasEffective"`      // L4 — athleticism signal
	BreakoutEffective float64 `json:"breakoutEffective"` // L4 — breakout signal
	L4Combined        float64 `json:"l4Combined"`        // L4 — the composed scouting multiplier

	// Composites + the hero number.
	ScoutingAdjusted float64 `json:"scoutingAdjusted"` // BasePoints × AgePull × L4Combined
	AdjustedScore    float64 `json:"adjustedScore"`    // hero — the ranked value (L6 tiebreak encoded upstream)

	// Contract / cap block.
	Salary        float64 `json:"salary"`        // $M at the display edge
	CapMultiplier float64 `json:"capMultiplier"` // the cap overlay factor
	CapTier       string  `json:"capTier"`
	CapEff        float64 `json:"capEff"`   // AdjustedScore per $M
	CapEffOK      bool    `json:"capEffOK"` // false when salary ≤ 0 (undefined, not zero)
	IsVeteran     bool    `json:"isVeteran"`
}

// PlayerScoreResult is the GetPlayerScore IPC payload. Found distinguishes "no score row for this id
// yet" (rescore needed / off-board player) from an error; Warning carries the same names-offline
// degradation GetRankings uses (the breakdown is persisted and complete even when the directory is
// unreachable). Label is the same BasePoints-proxy honesty string every score surface must render.
type PlayerScoreResult struct {
	OK      bool           `json:"ok"`
	Found   bool           `json:"found"`
	Error   string         `json:"error"`
	Warning string         `json:"warning"`
	Label   string         `json:"label"`
	Player  PlayerScoreDTO `json:"player"`
}

// GetPlayerScore returns one player's full persisted breakdown for the active scoring config — the
// Inspector's read. It mirrors GetRankings' resolution (active rulebook version → output reader) and
// its degrade-not-hide posture: a directory (names) outage warns but still returns the complete,
// persisted numbers. A missing score row is OK:true/Found:false (the board simply has no row for
// that id under the current config), never a hard error.
func (a *App) GetPlayerScore(mflID string) PlayerScoreResult {
	if err := a.m1Ready(); err != nil {
		return PlayerScoreResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()

	ver, err := a.rulebook.ActiveVersion(ctx)
	if err != nil {
		return PlayerScoreResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	s, found, err := a.output.Reader().Score(ctx, a.season, ver, mflID)
	if err != nil {
		return PlayerScoreResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	if !found {
		return PlayerScoreResult{OK: true, Found: false, Label: a.proxyLabel()}
	}

	// Names are DISPLAY-ONLY — a directory outage must not hide a fully-persisted breakdown that sits
	// in SQLite (the GetRankings posture). Degrade to the unknown-id fallback and say why.
	var warning string
	lk, derr := a.directory(ctx)
	if derr != nil {
		lk = normalize.Lookup{}
		warning = "player name unavailable (players-DB fetch failed: " + derr.Error() + ") — the score is persisted and complete"
	}

	dto := PlayerScoreDTO{
		MFLID:             s.MFLID,
		BasePoints:        s.BasePoints,
		AgePull:           s.AgePull,
		FilmEffective:     s.Layer4Output.FilmEffective,
		RASEffective:      s.Layer4Output.RASEffective,
		BreakoutEffective: s.Layer4Output.BreakoutEffective,
		L4Combined:        s.Layer4Output.Combined,
		ScoutingAdjusted:  s.ScoutingAdjusted,
		AdjustedScore:     s.AdjustedScore,
		CapMultiplier:     s.CapMultiplier,
		CapTier:           string(s.CapTier),
		IsVeteran:         s.Tiebreaker.IsVeteran,
	}
	if f, ok := lk.Facts(s.MFLID); ok {
		dto.Name, dto.Position = f.Name, string(f.Position)
	} else {
		dto.Name = fmt.Sprintf("(unknown id %s)", s.MFLID)
	}
	if p, ok := a.state.Reader().Player(s.MFLID); ok {
		dto.FranchiseID = p.FranchiseID
		dto.Salary = p.CapSalary.Millions() // money → $M at the display edge
		if dto.Salary > 0 {
			dto.CapEff, dto.CapEffOK = s.AdjustedScore/dto.Salary, true
		}
	}
	return PlayerScoreResult{OK: true, Found: true, Warning: warning, Label: a.proxyLabel(), Player: dto}
}
