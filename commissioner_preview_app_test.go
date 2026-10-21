package main

import (
	"encoding/json"
	"reflect"
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

func previewApp(t *testing.T) *App {
	t.Helper()
	app := commissionerTestApp(t)
	executeCommissionerTest(t, app, commissioner.RawCommand{League: "native.preview", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Preview league", Preset: commissioner.PresetLegacy, Season: 2026},
		Reason: "Start our league", Request: "create.preview", Effective: "now", Visibility: "league"})
	return app
}

func TestCommissionerPreviewRecordsNothing(t *testing.T) {
	t.Parallel()
	app := previewApp(t)
	before, err := app.TargetCommissionerLedger("native.preview")
	if err != nil {
		t.Fatal(err)
	}
	body := previewRuleBody(t, "roster.limits", `{"min":20,"max":70}`, "next_rollover", "", 1)
	for range 2 {
		preview, err := app.TargetCommissionerPreview(body)
		if err != nil || len(preview.Blockers) != 0 || preview.Target.Version != 1 {
			t.Fatalf("preview: %+v %v", preview, err)
		}
	}
	after, err := app.TargetCommissionerLedger("native.preview")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("preview recorded a change: before %+v after %+v, %v", before, after, err)
	}
	// The same request key is still available for a different value after a dry run.
	changed := strings.Replace(body, `"max":70`, `"max":71`, 1)
	receipt, err := app.TargetCommissionerExecute(changed)
	if err != nil || receipt.Revision != 2 {
		t.Fatalf("preview consumed the request or version: %+v %v", receipt, err)
	}
}

func TestCommissionerPreviewChecksMatchExecute(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, principal, field, value, effective, reason string
		expected                                         int64
	}{
		{"stale", "creator", "roster.limits", `{"min":20,"max":70}`, "next_rollover", "", 0},
		{"permission", "owner", "roster.limits", `{"min":20,"max":70}`, "next_rollover", "", 1},
		{"rules", "creator", "playoffs.startWeek", `18`, "now", "Delay playoffs", 1},
		{"validation", "creator", "roster.limits", `{"min":71,"max":70}`, "next_rollover", "", 1},
		{"reason", "creator", "roster.limits", `{"min":20,"max":70}`, "now", "", 1},
		{"unchanged", "creator", "roster.limits", `{"min":18,"max":80}`, "next_rollover", "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			app := previewApp(t)
			app.core.CommissionerSession = testCommissionerSession{tc.principal}
			body := previewRuleBody(t, tc.field, tc.value, tc.effective, tc.reason, tc.expected)
			preview, err := app.TargetCommissionerPreview(body)
			if err != nil || len(preview.Blockers) != 1 {
				t.Fatalf("preview must refuse: %+v %v", preview, err)
			}
			_, refusal := app.TargetCommissionerExecute(body)
			if refusal == nil || !strings.Contains(strings.ToLower(refusal.Error()),
				strings.ToLower(preview.Blockers[0])) {
				t.Fatalf("checks differ: preview %+v, execute %v", preview.Blockers, refusal)
			}
			rules, err := app.TargetCommissionerRules("native.preview")
			if err != nil || rules.Revision != 1 {
				t.Fatalf("refusal recorded: %+v %v", rules, err)
			}
		})
	}
}

func TestCommissionerPreviewWire(t *testing.T) {
	t.Parallel()
	app := previewApp(t)
	views := map[string]commissioner.Preview{}
	for _, tc := range []struct{ name, field, value, effective, reason string }{
		{"hold", "roster.limits", `{"min":20,"max":70}`, "next_rollover", ""},
		{"confirm", "calendar.timezone", `"America/Chicago"`, "next_rollover", ""},
		{"reason", "roster.limits", `{"min":20,"max":70}`, "now", "New roster vote"},
	} {
		reading, err := app.TargetCommissionerPreview(previewRuleBody(t, tc.field, tc.value,
			tc.effective, tc.reason, 1))
		if err != nil || len(reading.Blockers) != 0 || reading.Safeguard != tc.name {
			t.Fatalf("%s preview: %+v %v", tc.name, reading, err)
		}
		views[tc.name] = reading
	}
	roster := views["hold"]
	if roster.Question != "How small and how large may a roster be? Any position minimums?" ||
		roster.Target.Name != "Preview league" || len(roster.Fields) != 1 ||
		roster.Fields[0].Before != "18 to 80" || roster.Fields[0].After != "20 to 70" ||
		roster.Fields[0].BeforeSource != "Legacy NFL rules" ||
		roster.Fields[0].AfterSource != "Your proposed change" {
		t.Fatalf("question, frozen target, values and provenance: %+v", roster)
	}
	assertPreviewAffects(t, roster)
	stale, err := app.TargetCommissionerPreview(previewRuleBody(t, "roster.limits",
		`{"min":20,"max":70}`, "next_rollover", "", 0))
	if err != nil || len(stale.Blockers) != 1 || stale.Target.Version != 0 {
		t.Fatalf("stale preview must keep its frozen target: %+v %v", stale, err)
	}
	views["stale"] = stale
	body, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/commissioner/preview.wire.testdata.json", body, "preview")
}

func assertPreviewAffects(t *testing.T, preview commissioner.Preview) {
	t.Helper()
	// These four registry rows govern roster limits, not four players or four franchises.
	if preview.Affects.Count == nil || *preview.Affects.Count != 4 || len(preview.Affects.Readings) != 4 ||
		preview.Affects.Words != "All 32 franchises affected" {
		t.Fatalf("affected catalogue rows: %+v", preview.Affects)
	}
	if strings.Join(preview.Affects.Readings, ";") != "Rosters;Roster w/Stats;Submit Lineup;Drop Players" {
		t.Fatalf("affected reading names must come from the registry: %+v", preview.Affects.Readings)
	}
	for _, name := range preview.Affects.Readings {
		if strings.HasPrefix(name, "M-") || strings.Contains(name, "O=") || name == "" {
			t.Fatalf("reading has no GM name: %q", name)
		}
	}
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
