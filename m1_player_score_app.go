package main

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// PlayerScoreDTO is one player's full breakdown for the inspector: the adjusted score, the
// layer values that build it, and the contract and cap block. FilmRaw is a debug field and never
// crosses to the UI.
type PlayerScoreDTO struct {
	MFLID         string `json:"mflID"`
	Name          string `json:"name"`
	Position      string `json:"position"`
	FranchiseID   string `json:"franchiseID"`
	FranchiseName string `json:"franchiseName"`

	BasePoints        float64 `json:"basePoints"`
	AgePull           float64 `json:"agePull"`
	FilmEffective     float64 `json:"filmEffective"`
	RASEffective      float64 `json:"rasEffective"`
	BreakoutEffective float64 `json:"breakoutEffective"`
	L4Combined        float64 `json:"l4Combined"`

	ScoutingAdjusted float64 `json:"scoutingAdjusted"`
	AdjustedScore    float64 `json:"adjustedScore"`

	Salary        float64 `json:"salary"` // $M at the display edge
	CapMultiplier float64 `json:"capMultiplier"`
	CapTier       string  `json:"capTier"`
	CapEff        float64 `json:"capEff"`   // AdjustedScore per $M
	CapEffOK      bool    `json:"capEffOK"` // false when salary ≤ 0 (undefined, not zero)
	IsVeteran     bool    `json:"isVeteran"`
}

// PlayerScoreResult: Found=false means no score row for this id on the latest board,
// which is not an error. Warning reports a names outage; Label is the base-points honesty
// string every score surface shows.
type PlayerScoreResult struct {
	OK      bool           `json:"ok"`
	Found   bool           `json:"found"`
	Error   string         `json:"error"`
	Warning string         `json:"warning"`
	Label   string         `json:"label"`
	Player  PlayerScoreDTO `json:"player"`
}

// GetPlayerScore returns one player's stored breakdown on the latest board. A names
// outage warns but still returns the numbers.
func (a *App) GetPlayerScore(mflID string) PlayerScoreResult {
	if err := a.m1Ready(); err != nil {
		return PlayerScoreResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()

	run, ok, err := a.latestBoard(ctx)
	if err != nil {
		return PlayerScoreResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	if !ok {
		return PlayerScoreResult{OK: true, Found: false, Label: a.proxyLabel()}
	}
	s, found, err := a.history.RunScore(ctx, run.ID, mflID)
	if err != nil {
		return PlayerScoreResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	if !found {
		return PlayerScoreResult{OK: true, Found: false, Label: a.proxyLabel()}
	}

	// Names are display-only: an outage must not hide a stored breakdown.
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
	if p, ok := a.league.Reader().Player(s.MFLID); ok {
		dto.FranchiseID = p.FranchiseID
		dto.FranchiseName = domain.FranchiseLabel(a.rulebook.FranchiseNames(), p.FranchiseID)
		dto.Salary = p.CapSalary.Millions()
		if dto.Salary > 0 {
			dto.CapEff, dto.CapEffOK = s.AdjustedScore/dto.Salary, true
		}
	}
	return PlayerScoreResult{OK: true, Found: true, Warning: warning, Label: a.proxyLabel(), Player: dto}
}
