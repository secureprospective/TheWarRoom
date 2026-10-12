package main

import (
	"fmt"
	"strings"
	"time"
)

type ShellStatus struct {
	Health  string `json:"health"`
	State   string `json:"state"`
	Gameday string `json:"gameday"`
}

func (a *App) TargetShellStatus() (ShellStatus, error) {
	sources, err := a.TargetDataSources()
	if err != nil {
		return ShellStatus{}, fmt.Errorf("shell status: %w", err)
	}
	state, label := worstSourceHealth(sources.Health)
	a.weekMu.Lock()
	week, failed := a.week, a.weekRefreshError
	a.weekMu.Unlock()
	gameday := "Gameday unavailable"
	if failed == "" && len(week.Games) > 0 {
		playing, final := 0, 0
		now := time.Now()
		for _, game := range week.Games {
			if game.Final {
				final++
			} else if !game.Kickoff.After(now) {
				playing++
			}
		}
		gameday = fmt.Sprintf("Week %d · %d NFL games in progress · %d final", week.Number, playing, final)
	}
	return ShellStatus{Health: label, State: state, Gameday: gameday}, nil
}

func worstSourceHealth(report Report) (string, string) {
	state, label := "live", "Sources flowing"
	for _, row := range report.Rows {
		if len(row.Cells) < 3 {
			continue
		}
		age, failure := row.Cells[1], row.Cells[2]
		if strings.Contains(age, "Retired") {
			continue
		}
		if strings.Contains(age, "Lost ·") || strings.Contains(age, "No good read yet") {
			return "fail", "Sources missing · board may be reduced"
		}
		if failure != "None" && !strings.Contains(failure, "(recovered since)") {
			state, label = "stale", "Source failures · last good data"
		}
	}
	return state, label
}
