package main

import (
	"context"
	"fmt"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
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
	ctx, cancel := context.WithTimeout(a.core.Ctx, m1Timeout)
	defer cancel()
	health, err := a.healthInput(ctx)
	if err != nil {
		return ShellStatus{}, fmt.Errorf("shell status: %w", err)
	}
	state, label := sourceStanding(health)
	gameday := controlapp.Gameday(a.weekSnapshot(), a.timeNow())
	return ShellStatus{Health: label, State: state, Gameday: gameday}, nil
}
