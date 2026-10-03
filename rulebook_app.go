package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// LeagueSettingResult is one rulebook setting's effective value, overrides applied. For the IR
// and taxi controls, "0" means off and any other value is the slot count.
type LeagueSettingResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Value string `json:"value"`
}

// GetLeagueSetting returns one rulebook setting (e.g. "taxiSquad"), overrides applied.
func (a *App) GetLeagueSetting(key string) LeagueSettingResult {
	if a.rulebook == nil {
		return LeagueSettingResult{OK: false, Error: "rulebook store not initialized"}
	}
	if strings.TrimSpace(key) == "" {
		return LeagueSettingResult{OK: false, Error: "key must be non-empty"}
	}
	v, ok := a.rulebook.GetSetting(key)
	if !ok {
		return LeagueSettingResult{OK: false, Error: fmt.Sprintf("unknown league setting %q", key)}
	}
	return LeagueSettingResult{OK: true, Value: v}
}

// SetLeagueSettingResult is the admin-write payload for a rulebook override.
type SetLeagueSettingResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// SetLeagueSettingOverride applies a commissioner override to a rulebook setting. Rulebook
// writes are admin-only and go to the store directly, not through the transaction coordinator.
func (a *App) SetLeagueSettingOverride(key, value, note string) SetLeagueSettingResult {
	if a.rulebook == nil {
		return SetLeagueSettingResult{OK: false, Error: "rulebook store not initialized"}
	}
	if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
		return SetLeagueSettingResult{OK: false, Error: "key and value must be non-empty"}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
	defer cancel()
	if err := a.rulebook.SetOverride(ctx, "setting", key, value, note); err != nil {
		return SetLeagueSettingResult{OK: false, Error: err.Error()}
	}
	return SetLeagueSettingResult{OK: true}
}
