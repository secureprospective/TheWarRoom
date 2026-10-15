package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type nativeResultLineups struct{}

func (nativeResultLineups) WeekLineup(context.Context, []commissioner.Record, string, string, int, int) (commissioner.WeekLineup, error) {
	return commissioner.WeekLineup{Set: true, Starters: []commissioner.Starter{{Player: "1000", Name: "Held starter"}}}, nil
}

func TestNativeResultsIPCUsesHeldIncompleteScores(t *testing.T) {
	f := scoreApp(t)
	f.app.history.SetNativeLineups(nativeResultLineups{})
	raw := commissioner.RawCommand{League: legalityLeague, Command: commissioner.ScheduleGenerate,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 13}}, Expected: 2, Request: "generate-results",
		Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, f.app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 3, "publish-results"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, f.app, raw)
	for range 2 {
		results, err := f.app.TargetNativeResults(legalityLeague, 2026, 1)
		if err != nil || len(results.Games) != 16 {
			t.Fatalf("results: %+v %v", results, err)
		}
		if results.Games[0].Winner != "" || results.Games[0].Status != commissioner.GameNoResult ||
			!strings.Contains(results.Games[0].Words, "long plays and return touchdowns") {
			t.Fatal(results.Games[0])
		}
		standings, err := f.app.TargetNativeStandings(legalityLeague, 2026)
		if err != nil || standings.Decided != 0 || standings.Pending != 208 || len(standings.Computed) != 32 {
			t.Fatal(standings, err)
		}
		change := commissioner.RawCommand{League: legalityLeague, Command: commissioner.ScorePlayer,
			Args:     commissioner.RawArgs{Score: &commissioner.RawScoreChange{Player: "1000", Season: 2026, Week: 1, Delta: scoreDelta(10000)}},
			Expected: 4, Request: "preview-results", Reason: "Correct this score", Effective: "now", Visibility: "league"}
		body, err := json.Marshal(change)
		if err != nil {
			t.Fatal(err)
		}
		preview, err := f.app.TargetNativeScorePreview(string(body))
		if err != nil || preview.Flip.Status != commissioner.FlipUnknown || len(preview.Warnings) != 1 {
			t.Fatal(preview, err)
		}
		after, err := f.app.TargetNativeResults(legalityLeague, 2026, 1)
		if err != nil || after.Revision != 4 {
			t.Fatal("preview recorded a change", err)
		}
		f.reopen(t)
		f.app.history.SetNativeLineups(nativeResultLineups{})
	}
	f.app.history.SetNativeLineups(nil)
	results, err := f.app.TargetNativeResults(legalityLeague, 2026, 1)
	if err != nil || !strings.Contains(results.Games[0].Words, "no lineup set") {
		t.Fatal(results, err)
	}
}

func TestNativeResultsIPCScopeAndStartup(t *testing.T) {
	a := &App{started: make(chan struct{}), startupErr: errors.New("startup failed")}
	close(a.started)
	for _, scope := range []RawNativeResultsScope{
		{League: "native.", Season: 2026, Week: 1}, {League: "imported", Season: 2026, Week: 1},
		{League: " native.test", Season: 2026, Week: 1}, {League: "native.test", Season: 0, Week: 1},
		{League: "native.test", Season: 10000, Week: 1}, {League: "native.test", Season: 2026, Week: 0},
		{League: "native.test", Season: 2026, Week: 23},
	} {
		if err := scope.Validate(); err == nil {
			t.Fatalf("scope validation accepted %+v", scope)
		}
		if _, err := a.TargetNativeResults(scope.League, scope.Season, scope.Week); err == nil {
			t.Fatalf("accepted %+v", scope)
		}
	}
	if _, err := a.TargetNativeStandings("native.test", 0); err == nil {
		t.Fatal("invalid standings season")
	}
	if _, err := a.TargetNativeResults("native.test", 2026, 1); err == nil {
		t.Fatal("results ignored startup")
	}
	if _, err := a.TargetNativeStandings("native.test", 2026); err == nil {
		t.Fatal("standings ignored startup")
	}
	if _, err := a.TargetNativeScorePreview(`{"command":"bad"}`); err == nil {
		t.Fatal("invalid preview")
	}
}
