package commissioner

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func previewRuleBody(t *testing.T, field, value, effective, reason string, expected int64) string {
	t.Helper()
	raw := commissioner.RawCommand{League: "native.preview", Command: commissioner.RulesSet,
		Args: commissioner.RawArgs{Field: field, Value: json.RawMessage(value)}, Reason: reason,
		Expected: expected, Request: "preview.request", Effective: effective, Visibility: "league"}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func previewApp(t *testing.T) *ledgerFixture {
	t.Helper()
	app := commissionerTestApp(t)
	executeCommissionerTest(t, app, commissioner.RawCommand{League: "native.preview",
		Command: commissioner.Create,
		Args:    commissioner.RawArgs{Name: "Preview league", Preset: commissioner.PresetLegacy, Source: commissioner.SourceMFLImport, Season: 2026},
		Reason:  "Start our league", Request: "create.preview", Effective: "now", Visibility: "league"})
	return app
}

func TestCommissionerPreviewDoesNotBypassRequestChecks(t *testing.T) {
	t.Parallel()
	app := previewApp(t)
	body := previewRuleBody(t, "roster.limits", `{"min":20,"max":70}`, "next_rollover", "", 1)
	if _, err := app.TargetCommissionerExecute(body); err != nil {
		t.Fatal(err)
	}
	for index, candidate := range []string{body, strings.Replace(body, `"max":70`, `"max":71`, 1)} {
		preview, err := app.TargetCommissionerPreview(candidate)
		if err != nil || len(preview.Blockers) != index {
			t.Fatalf("used request must not offer another write: %+v %v", preview, err)
		}
	}
	rules, err := app.TargetCommissionerRules("native.preview")
	if err != nil || rules.Revision != 2 {
		t.Fatalf("preview wrote a retry: %+v %v", rules, err)
	}
}

func TestTargetWorkflowRunsALocalRead(t *testing.T) {
	t.Parallel()
	app := previewApp(t)
	body := `{"command":"standings.read","version":"1","mode":"read","arguments":{"league":"native.preview","season":2026}}`
	got, err := app.TargetWorkflow(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.Label == "" || !got.Writable || got.Football != "local league" || len(got.Data) == 0 {
		t.Fatalf("local workflow response: %+v", got)
	}
}

func TestTargetWorkflowRefusesAnUnknownAction(t *testing.T) {
	t.Parallel()
	app := previewApp(t)
	body := `{"command":"not.a.command","version":"1","mode":"read","arguments":{"league":"native.preview"}}`
	if _, err := app.TargetWorkflow(body); err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("unknown local workflow action: %v", err)
	}
}
