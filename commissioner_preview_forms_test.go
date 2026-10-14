package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func previewCommandBody(t *testing.T, raw commissioner.RawCommand) string {
	t.Helper()
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func roleCommand(command, person string, role governance.Role, reason string) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.preview", Command: command, Reason: reason, Expected: 1,
		Request: "role.request", Effective: "now", Visibility: "private",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{Principal: person, Role: role}}}
}

func atVersion(raw commissioner.RawCommand, version int64) commissioner.RawCommand {
	raw.Expected = version
	return raw
}

func createCommand(reason string) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.fresh", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Fresh league", Preset: commissioner.PresetLegacy, FranchiseCount: 12},
		Reason: reason, Request: "create.request", Effective: "now", Visibility: "league"}
}

func mustPreview(t *testing.T, app *App, raw commissioner.RawCommand) commissioner.Preview {
	t.Helper()
	preview, err := app.TargetCommissionerPreview(previewCommandBody(t, raw))
	if err != nil {
		t.Fatal(err)
	}
	return preview
}

func TestCommissionerPreviewDescribesStartingALeague(t *testing.T) {
	app := commissionerTestApp(t)
	preview := mustPreview(t, app, createCommand("Move our league here"))
	if len(preview.Blockers) != 0 || preview.Safeguard != "reason" || preview.Target.Version != 0 ||
		preview.Target.Name != "Fresh league" || preview.Target.League != "native.fresh" ||
		preview.Question == "" || len(preview.Fields) != 3 || preview.Fields[1].After != "12 franchises" ||
		preview.Fields[2].After != "Legacy NFL rules" || preview.Affects.Count == nil || *preview.Affects.Count != 0 {
		t.Fatalf("start a league: %+v", preview)
	}
	ledger, err := app.TargetCommissionerLeagues()
	if err != nil || len(ledger) != 0 {
		t.Fatalf("a dry run started a league: %+v %v", ledger, err)
	}
}

func TestCommissionerPreviewAsksForTheReasonAndStillDescribes(t *testing.T) {
	app := commissionerTestApp(t)
	preview := mustPreview(t, app, createCommand("  "))
	if preview.Safeguard != "reason" || len(preview.Fields) != 3 ||
		strings.Join(preview.Blockers, "|") != "Give a reason for this change" {
		t.Fatalf("the confirm Go asks for must show before the reason is typed: %+v", preview)
	}
	_, err := app.TargetCommissionerExecute(previewCommandBody(t, createCommand("  ")))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "give a reason for this change") {
		t.Fatalf("execute must still refuse it: %v", err)
	}
}

func TestCommissionerPreviewDescribesARoleChangeAndRefusesOneThatChangesNothing(t *testing.T) {
	app := previewApp(t)
	grant := roleCommand("commish.role.grant", "owner-two", governance.CoCommissioner, "Owner two will help")
	preview := mustPreview(t, app, grant)
	if len(preview.Blockers) != 0 || preview.Safeguard != "reason" || len(preview.Fields) != 1 ||
		preview.Fields[0].Label != "Co-commissioner for owner-two" ||
		preview.Fields[0].Before != "Does not hold the role" || preview.Fields[0].After != "Holds the role" ||
		preview.Target.Name != "Preview league" || preview.Target.Version != 1 ||
		preview.Affects.Count == nil || *preview.Affects.Count != 1 ||
		preview.Affects.Words != "1 owner reading uses this rule" {
		t.Fatalf("grant preview: %+v", preview)
	}
	for _, tc := range []struct{ name, command, person, want string }{
		{"already holds", "commish.role.grant", "creator", "This person already holds that role. Nothing would change"},
		{"does not hold", "commish.role.revoke", "owner-two", "This person does not hold that role. Nothing would change"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			role := governance.CoCommissioner
			if tc.person == "creator" {
				role = governance.Commissioner
			}
			raw := roleCommand(tc.command, tc.person, role, "Agreed")
			blocked := mustPreview(t, app, raw)
			if strings.Join(blocked.Blockers, "|") != tc.want {
				t.Fatalf("preview blockers: %+v", blocked.Blockers)
			}
			_, refusal := app.TargetCommissionerExecute(previewCommandBody(t, raw))
			if refusal == nil || refusal.Error() != tc.want {
				t.Fatalf("execute must refuse in the same words: %v", refusal)
			}
		})
	}
}

func TestCommissionerPreviewDescribesUndoingAChange(t *testing.T) {
	app := previewApp(t)
	made := executeCommissionerTest(t, app, commissioner.RawCommand{League: "native.preview",
		Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: "roster.limits",
			Value: json.RawMessage(`{"min":20,"max":70}`)}, Expected: 1, Request: "rule.made",
		Effective: "next_rollover", Visibility: "league"})
	undo := commissioner.RawCommand{League: "native.preview", Command: commissioner.Revert,
		Args: commissioner.RawArgs{Entry: made.Entry}, Reason: "Wrong size", Expected: 2,
		Request: "undo.request", Effective: "now", Visibility: "league"}
	preview := mustPreview(t, app, undo)
	if len(preview.Blockers) != 0 || preview.Safeguard != "reason" || len(preview.Fields) != 1 ||
		!strings.Contains(preview.Fields[0].Before, "Active roster size") ||
		!strings.HasPrefix(preview.Fields[0].After, "Undone") || preview.Target.Version != 2 {
		t.Fatalf("undo preview: %+v", preview)
	}
	assertPreviewAffects(t, preview)
	undo.Args.Entry = "act.missing"
	missing := mustPreview(t, app, undo)
	if strings.Join(missing.Blockers, "|") != "That recorded change is not in this league" {
		t.Fatalf("an unknown entry must be refused: %+v", missing)
	}
}

func TestCommissionerPreviewFormsWire(t *testing.T) {
	app := previewApp(t)
	made := executeCommissionerTest(t, app, commissioner.RawCommand{League: "native.preview",
		Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: "roster.limits",
			Value: json.RawMessage(`{"min":20,"max":70}`)}, Expected: 1, Request: "rule.made",
		Effective: "next_rollover", Visibility: "league"})
	views := map[string]commissioner.Preview{
		"start": mustPreview(t, app, createCommand("Move our league here")),
		"role": mustPreview(t, app, atVersion(roleCommand("commish.role.grant", "owner-two",
			governance.CoCommissioner, "Owner two will help"), 2)),
		"undo": mustPreview(t, app, commissioner.RawCommand{League: "native.preview",
			Command: commissioner.Revert, Args: commissioner.RawArgs{Entry: made.Entry},
			Reason: "Wrong size", Expected: 2, Request: "undo.request", Effective: "now", Visibility: "league"}),
		"roleUnchanged": mustPreview(t, app, atVersion(roleCommand("commish.role.grant", "creator",
			governance.Commissioner, "Agreed"), 2)),
		"needsReason": mustPreview(t, app, createCommand("")),
	}
	unchanged, err := app.TargetCommissionerPreview(previewRuleBody(t, "roster.limits",
		`{"min":20,"max":70}`, "next_rollover", "", 2))
	if err != nil {
		t.Fatal(err)
	}
	views["ruleUnchanged"] = unchanged
	if got := strings.Join(views["ruleUnchanged"].Blockers, "|"); got !=
		"This change is already waiting for the next rollover. Nothing would change" {
		t.Fatalf("a rule change to the value already waiting: %q", got)
	}
	body, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/commissioner/preview-forms.wire.testdata.json", body, "preview forms")
}

func TestCommissionerCommandSafeguardsWire(t *testing.T) {
	body, err := json.MarshalIndent(commissioner.CommandSafeguards(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/commissioner/commands.wire.testdata.json", body, "command safeguards")
}
