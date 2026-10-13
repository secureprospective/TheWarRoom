package main

import (
	"context"
	"fmt"
)

type ShellStatus struct {
	Health  string `json:"health"`
	State   string `json:"state"`
	Gameday string `json:"gameday"`
}

func (a *App) TargetShellStatus() (ShellStatus, error) {
	if err := a.ready(); err != nil {
		return ShellStatus{}, fmt.Errorf("shell status: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()
	health, err := a.healthInput(ctx)
	if err != nil {
		return ShellStatus{}, fmt.Errorf("shell status: %w", err)
	}
	state, label := sourceStanding(health)
	a.weekMu.Lock()
	week, failed := a.week, a.weekRefreshError
	a.weekMu.Unlock()
	gameday := "Gameday unavailable"
	if failed == "" && len(week.Games) > 0 {
		playing, final := 0, 0
		now := a.timeNow()
		for _, game := range week.Games {
			started := !game.Kickoff.After(now)
			if started && game.Final {
				final++
			} else if started {
				playing++
			}
		}
		gameday = fmt.Sprintf("Week %d · %s in progress · %d final", week.Number, countText(playing, "NFL game"), final)
	}
	return ShellStatus{Health: label, State: state, Gameday: gameday}, nil
}
