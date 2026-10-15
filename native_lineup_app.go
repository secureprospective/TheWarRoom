package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// lineupWindow is the calendar action that lets an owner set a lineup. Like every action the gate asks
// about, it is closed unless a window or an exception opens it.
const lineupWindow leaguerules.Action = "lineup"

// RawLineupSet is an owner's request to set the starters of their own franchise for a week.
type RawLineupSet struct {
	League    string   `json:"league"`
	Franchise string   `json:"franchise"`
	Season    int      `json:"season"`
	Week      int      `json:"week"`
	Starters  []string `json:"starters"`
	Expected  int64    `json:"expected"`
	Request   string   `json:"request"`
}

func decodeLineupSet(body []byte) (RawLineupSet, error) {
	var in RawLineupSet
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		return in, fmt.Errorf("read the lineup: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return in, fmt.Errorf("send exactly one lineup")
	}
	if !strings.HasPrefix(in.League, "native.") || in.League == "native." || strings.TrimSpace(in.League) != in.League ||
		strings.TrimSpace(in.Request) == "" || in.Expected < 0 {
		return in, fmt.Errorf("choose an app league and a valid request and league version")
	}
	return in, nil
}

// lineupSetAction is the registered owner action; native_gate_test.go holds it to the gate with the rest.
func lineupSetAction() NativeOwnerAction {
	return NativeOwnerAction{Name: "lineup.set", Opens: lineupWindow, Run: runLineupSet}
}

// runLineupSet records the lineup. It is reached only through runNativeAction, and it checks the permit
// it was handed is for this very lineup, so a permit for anything else cannot start it.
func runLineupSet(ctx context.Context, a *App, permit commissioner.Permit,
	input json.RawMessage) (commissioner.Receipt, error) {
	in, err := decodeLineupSet(input)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if permit.League() != in.League || permit.Franchise() != in.Franchise || permit.Action() != lineupWindow {
		return commissioner.Receipt{}, fmt.Errorf("this lineup was not cleared by the league calendar")
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	raw := commissioner.RawCommand{League: in.League, Command: commissioner.LineupSet, Expected: in.Expected,
		Request: in.Request, Effective: "now", Visibility: "private",
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: in.Season, Franchise: in.Franchise,
			Week: in.Week, Starters: in.Starters}}}
	ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()
	receipt, err := a.history.ExecuteCommissioner(ctx, raw, principal, a.commissionerAuthorizer)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record the lineup: %w", err)
	}
	return receipt, nil
}

// TargetNativeLineupSet sets an owner's lineup through the native action gate and the league ledger. The
// lineup must be legal under the league's lineup check, and no player whose game has started can move.
func (a *App) TargetNativeLineupSet(body string) (commissioner.Receipt, error) {
	in, err := decodeLineupSet([]byte(body))
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("lineup input: %w", err)
	}
	receipt, err := a.runNativeAction(nativeOwnerActions(), in.League, in.Franchise, "lineup.set", json.RawMessage(body))
	var refused *commissioner.Refusal
	if errors.As(err, &refused) {
		return commissioner.Receipt{}, errors.New(refused.Reason())
	}
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("set lineup: %w", err)
	}
	return receipt, nil
}

// TargetNativeLineup reads a franchise's lineup for a week: starters and bench, who is locked and when it
// locks in the owner's time, and the league's verdict. at is an RFC 3339 instant, or empty for now.
func (a *App) TargetNativeLineup(league, franchise string, week int, at string) (commissioner.NativeLineup, error) {
	if !strings.HasPrefix(league, "native.") || league == "native." || strings.TrimSpace(league) != league ||
		strings.TrimSpace(franchise) == "" || week < 1 || week > 22 {
		return commissioner.NativeLineup{}, fmt.Errorf("choose an app league, a franchise and a week")
	}
	instant := a.timeNow()
	if at != "" {
		parsed, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return commissioner.NativeLineup{}, fmt.Errorf("choose when to read the lineup: %w", err)
		}
		instant = parsed
	}
	if err := a.ready(); err != nil {
		return commissioner.NativeLineup{}, fmt.Errorf("native lineup startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeLineup(ctx, league, franchise, week, instant)
	if err != nil {
		return commissioner.NativeLineup{}, fmt.Errorf("native lineup: %w", err)
	}
	return reading, nil
}

// appLineupEnv gives lineups the football facts the app already holds: players from the held directory and
// kickoff times from the held NFL schedule, with no fetch.
type appLineupEnv struct{ app *App }

func (e appLineupEnv) PlayerFacts() nativecheck.Facts { return e.app.nativeFacts() }

func (e appLineupEnv) LineupFacts(ctx context.Context, season, week int) (commissioner.LineupFacts, error) {
	weeks, err := e.app.nativeScheduleDates(ctx, season, e.app.timeNow())
	if err != nil {
		return commissioner.LineupFacts{}, err
	}
	out := commissioner.LineupFacts{Facts: e.app.nativeFacts(), Zone: e.app.ownerZone()}
	for _, w := range weeks {
		if w.Number == week {
			out.Locks, out.ScheduleHeld = leagueweek.Locks(w), true
		}
	}
	return out, nil
}
