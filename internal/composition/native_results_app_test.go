package composition

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type nativeResultLineups struct{}

func (nativeResultLineups) WeekLineup(context.Context, []commissioner.Record, string, string, int, int) (commissioner.WeekLineup, error) {
	return commissioner.WeekLineup{Set: true, Starters: []commissioner.Starter{{Player: "1000", Name: "Held starter"}}}, nil
}

func TestNativeResultsIPCUsesHeldIncompleteScores(t *testing.T) {
	t.Parallel()
	f := scoreApp(t)
	f.app.core.History.SetNativeLineups(nativeResultLineups{})
	raw := commissioner.RawCommand{League: legalityLeague, Command: commissioner.ScheduleGenerate,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 13}}, Expected: 2, Request: "generate-results",
		Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, f.app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 3, "publish-results"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, f.app, raw)
	for range 2 {
		results, err := newCommissionerAPI(f.app).TargetNativeResults(legalityLeague, 2026, 1)
		if err != nil || len(results.Games) != 16 {
			t.Fatalf("results: %+v %v", results, err)
		}
		if results.Games[0].Winner != "" || results.Games[0].Status != commissioner.GameNoResult ||
			!strings.Contains(results.Games[0].Words, "long plays and return touchdowns") {
			t.Fatal(results.Games[0])
		}
		standings, err := newCommissionerAPI(f.app).TargetNativeStandings(legalityLeague, 2026)
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
		preview, err := newCommissionerAPI(f.app).TargetNativeScorePreview(string(body))
		if err != nil || preview.Flip.Status != commissioner.FlipUnknown || len(preview.Warnings) != 1 {
			t.Fatal(preview, err)
		}
		after, err := newCommissionerAPI(f.app).TargetNativeResults(legalityLeague, 2026, 1)
		if err != nil || after.Revision != 4 {
			t.Fatal("preview recorded a change", err)
		}
		f.reopen(t)
		f.app.core.History.SetNativeLineups(nativeResultLineups{})
	}
	f.app.core.History.SetNativeLineups(nil)
	results, err := newCommissionerAPI(f.app).TargetNativeResults(legalityLeague, 2026, 1)
	if err != nil || !strings.Contains(results.Games[0].Words, "no lineup set") {
		t.Fatal(results, err)
	}
}

func TestNativeBracketIPCSaysWhyTheBracketIsNotBuiltYet(t *testing.T) {
	t.Parallel()
	f := scoreApp(t)
	f.app.core.History.SetNativeLineups(nativeResultLineups{})
	raw := commissioner.RawCommand{League: legalityLeague, Command: commissioner.ScheduleGenerate,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 13}}, Expected: 2, Request: "generate-bracket",
		Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, f.app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 3, "publish-bracket"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, f.app, raw)
	for range 2 {
		bracket, err := newCommissionerAPI(f.app).TargetNativeBracket(legalityLeague, 2026)
		if err != nil || len(bracket.Rounds) != 0 || bracket.Champion != nil || bracket.Title.Label != "2026 League Champion" ||
			!strings.HasPrefix(bracket.Status, "Needs a ruling") && !strings.HasPrefix(bracket.Status, "Waiting") {
			t.Fatalf("a bracket without results must say why: %+v %v", bracket, err)
		}
		t.Log(bracket.Status, bracket.Words)
		if bracket.Words == "" {
			t.Fatal("a missing bracket owes a sentence")
		}
		f.reopen(t)
		f.app.core.History.SetNativeLineups(nativeResultLineups{})
	}
}
