package main

import (
	"context"
	"fmt"
	"time"
)

// LeagueSettingResult is one scalar rulebook setting's override-aware effective
// value — the pattern GetSetting/SetOverride already serve for the toggle-off /
// slot-count IR & taxi controls (Session 0): "0" reads as off, any other value is
// the slot count.
type LeagueSettingResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Value string `json:"value"`
}

// GetLeagueSetting returns one rulebook scalar setting (e.g. "taxiSquad",
// "injuredReserve"), with any commissioner override already applied.
func (a *App) GetLeagueSetting(key string) LeagueSettingResult {
	if a.rulebook == nil {
		return LeagueSettingResult{OK: false, Error: "rulebook store not initialized"}
	}
	v, ok := a.rulebook.GetSetting(key)
	if !ok {
		return LeagueSettingResult{OK: false, Error: fmt.Sprintf("unknown league setting %q", key)}
	}
	return LeagueSettingResult{OK: true, Value: v}
}

// SetLeagueSettingResult is the admin-write payload for a rulebook setting override.
type SetLeagueSettingResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// SetLeagueSettingOverride applies a commissioner override to a rulebook scalar
// setting (scope "setting" — validateOverride accepts any non-empty scalar).
// Rulebook writes are an admin-only path, never routed through the B7 Coordinator
// (AD-05) — this calls the store directly, not PreviewTransaction/ExecuteTransaction.
func (a *App) SetLeagueSettingOverride(key, value, note string) SetLeagueSettingResult {
	if a.rulebook == nil {
		return SetLeagueSettingResult{OK: false, Error: "rulebook store not initialized"}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
	defer cancel()
	if err := a.rulebook.SetOverride(ctx, "setting", key, value, note); err != nil {
		return SetLeagueSettingResult{OK: false, Error: err.Error()}
	}
	return SetLeagueSettingResult{OK: true}
}
