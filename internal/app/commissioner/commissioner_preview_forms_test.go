package commissioner

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/commissioner/commissionertest"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func previewCommandBody(t *testing.T, raw commissioner.RawCommand) string {
	t.Helper()
	return commissionertest.Body(t, raw)
}

func roleCommand(command, person string, role governance.Role, reason string) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.preview", Command: command, Reason: reason, Expected: 1,
		Request: "role.request", Effective: "now", Visibility: "private",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{Principal: person, Role: role}}}
}

func createCommand(reason string) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.fresh", Command: commissioner.Create,
		Args: commissioner.RawArgs{Name: "Fresh league", Preset: commissioner.PresetLegacy, Source: commissioner.SourceMFLImport,
			FranchiseCount: 12, Season: 2026},
		Reason: reason, Request: "create.request", Effective: "now", Visibility: "league"}
}

func mustPreview(t *testing.T, app *ledgerFixture, raw commissioner.RawCommand) commissioner.Preview {
	t.Helper()
	return commissionertest.Preview(t, app.TargetCommissionerPreview, raw)
}

func TestCommissionerPreviewAsksForTheReasonAndStillDescribes(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	app := previewApp(t)
	grant := roleCommand("commish.role.grant", "owner-two", governance.CoCommissioner, "Owner two will help")
	preview := mustPreview(t, app, grant)
	if len(preview.Blockers) != 0 || preview.Safeguard != "reason" || len(preview.Fields) != 1 ||
		preview.Fields[0].Label != "Co-commissioner for owner-two" ||
		preview.Fields[0].Before != "Does not hold the role" || preview.Fields[0].After != "Holds the role" ||
		preview.Target.Name != "Preview league" || preview.Target.Version != 1 ||
		preview.Affects.Count == nil || *preview.Affects.Count != 1 ||
		preview.Affects.Words != "All 32 franchises affected" {
		t.Fatalf("grant preview: %+v", preview)
	}
	for _, tc := range []struct{ name, command, person, want string }{
		{"already holds", "commish.role.grant", "creator",
			"This person already holds that role. Nothing would change"},
		{"does not hold", "commish.role.revoke", "owner-two",
			"This person does not hold that role. Nothing would change"},
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
