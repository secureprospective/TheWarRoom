package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// A preview and its later append share the exact command, request and session.
func proveActPreview(t *testing.T, app *App, raw commissioner.RawCommand) {
	t.Helper()
	before, err := app.leagueRecords(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := app.TargetCommissionerPreview(string(body))
	if err != nil {
		t.Fatalf("%s preview: %v", raw.Command, err)
	}
	if len(preview.Blockers) != 0 || len(preview.Fields) == 0 {
		t.Fatalf("%s legal preview: %+v", raw.Command, preview)
	}
	if len(preview.Fields) == 1 && preview.Fields[0].Label == "League ledger" &&
		preview.Fields[0].When != "Already recorded" {
		t.Fatalf("%s has no current-value preview: %+v", raw.Command, preview)
	}
	for _, f := range preview.Fields {
		if f.Before == "" || f.After == "" || f.BeforeSource == "" || f.AfterSource == "" {
			t.Fatalf("%s incomplete before/after: %+v", raw.Command, f)
		}
		if strings.Contains(f.After, "preview is not available") {
			t.Fatalf("%s generic preview", raw.Command)
		}
	}
	after, err := app.leagueRecords(t.Context(), raw.League)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("%s preview wrote: %v", raw.Command, err)
	}
	illegal := raw
	illegal.Visibility = "not-an-audience"
	bad, err := json.Marshal(illegal)
	if err != nil {
		t.Fatal(err)
	}
	blocked, err := app.TargetCommissionerPreview(string(bad))
	if err != nil {
		t.Fatal(err)
	}
	_, executeErr := app.TargetCommissionerExecute(string(bad))
	if executeErr == nil || len(blocked.Blockers) != 1 || blocked.Blockers[0] != previewExecuteReason(executeErr) {
		t.Fatalf("%s refusal differs: %+v / %v", raw.Command, blocked.Blockers, executeErr)
	}
	t.Logf("preview proof: %s", raw.Command)
}

func previewExecuteReason(err error) string {
	words := strings.TrimPrefix(err.Error(), "league change: ")
	return strings.ToUpper(words[:1]) + words[1:]
}

func TestStartingAssetsPreviewUsesTheSeedRequestAndRecordsNothing(t *testing.T) {
	t.Parallel()
	app := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, app, commissioner.PresetLegacy)
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := app.TargetCommissionerPreview(string(body))
	if err != nil || len(preview.Blockers) != 0 {
		t.Fatalf("starting assets: %+v %v", preview, err)
	}
	if len(preview.Fields) < 4 {
		t.Fatalf("starting roster and contracts absent: %+v", preview)
	}
	assets := readNativeTest(t, app, input.League)
	if assets.Seeded || assets.Revision != input.Expected {
		t.Fatal("preview recorded starting assets")
	}
	receipt := seedNativeTest(t, app, input)
	if receipt.Revision != input.Expected+1 {
		t.Fatal("preview reserved the seed request")
	}
	blocked, err := app.TargetCommissionerPreview(string(body))
	if err != nil || len(blocked.Blockers) != 0 {
		t.Fatalf("seed retry: %+v %v", blocked, err)
	}
}

// The tray opens before a reason is typed: the preview must still show the starting assets and the
// league version, and owe only the reason.
func TestStartingAssetsPreviewShowsBeforeTheReasonIsTyped(t *testing.T) {
	t.Parallel()
	app := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, app, commissioner.PresetLegacy)
	input.Reason = ""
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := app.TargetCommissionerPreview(string(body))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Target.League != input.League || preview.Target.Version != input.Expected {
		t.Fatalf("preview target %+v, want %s at %d", preview.Target, input.League, input.Expected)
	}
	if len(preview.Fields) < 4 {
		t.Fatalf("starting roster and contracts absent: %+v", preview)
	}
	if len(preview.Blockers) != 1 || preview.Blockers[0] != "Give a reason for this change" {
		t.Fatalf("blockers %q, want only the reason", preview.Blockers)
	}
	if _, err := app.TargetNativeSeed(input); err == nil {
		t.Fatal("recorded starting assets without a reason")
	}
}

// Recording starting assets adds every rostered player at once. The preview sums them up per franchise, so
// the commissioner can read it and reach the reason, rather than listing each player three times.
func TestStartingAssetsPreviewSumsUpEachFranchise(t *testing.T) {
	t.Parallel()
	app := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, app, commissioner.PresetLegacy)
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := app.TargetCommissionerPreview(string(body))
	if err != nil || len(preview.Blockers) != 0 {
		t.Fatalf("starting assets: %+v %v", preview, err)
	}
	rosters, populated := 0, 0
	for _, field := range preview.Fields {
		if strings.Contains(field.After, " players: ") {
			populated++
		}
		if strings.HasSuffix(field.Label, " acquisition") || strings.HasSuffix(field.Label, " contract") {
			t.Fatalf("per-player row in the starting assets preview: %+v", field)
		}
		if strings.HasSuffix(field.Label, " cap charges") && field.After != "No cap charges" &&
			!strings.Contains(field.After, " totalling ") {
			t.Fatalf("cap charges are not summed up: %q", field.After)
		}
		if strings.HasSuffix(field.Label, " starting roster") {
			rosters++
			if field.After != "No rostered players" && (!strings.Contains(field.After, " players: ") ||
				!strings.Contains(field.After, " salary") || strings.Contains(field.After, ": ;")) {
				t.Fatalf("roster summary %q", field.After)
			}
		}
	}
	if rosters == 0 || populated == 0 {
		t.Fatalf("no franchise roster summaries: %+v", preview.Fields)
	}
}

func assertPreviewTransition(t *testing.T, preview commissioner.Preview, label, before, after string) {
	t.Helper()
	for _, field := range preview.Fields {
		if strings.Contains(field.Label, label) && strings.Contains(field.Before, before) &&
			strings.Contains(field.After, after) {
			return
		}
	}
	t.Fatalf("missing %q transition %q -> %q: %+v", label, before, after, preview.Fields)
}

func TestCommissionerPreviewShowsNativeContractAndCapValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ command, value, label, before, after string }{
		{commissioner.ContractEdit, `{"player":"0099","years":4,"salaryCents":2000000000}`,
			"contract", "$10,987,000", "$20,000,000"},
		{commissioner.CapAdjust, `{"franchise":"franchise.02","amountCents":-200000000,"season":2026}`,
			"counted salary", "$21,180,000", "$19,180,000"},
		{commissioner.CapRelief, `{"player":"0100","amountCents":300000000}`,
			"counted salary", "$21,180,000", "$18,180,000"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			t.Parallel()
			f := capApp(t)
			raw := overrideCommand(t, f, tc.command, tc.value)
			preview := mustPreview(t, f.app, raw)
			assertPreviewTransition(t, preview, tc.label, tc.before, tc.after)
			if preview.Safeguard != "reason" {
				t.Fatalf("Go refuses a blank reason, but the preview asks for %q", preview.Safeguard)
			}
			if !strings.Contains(preview.Affects.Words, "1 franchise affected") {
				t.Fatalf("affected teams: %+v", preview.Affects)
			}
			previewAndExecute(t, f.app, raw)
		})
	}
}

func TestCommissionerPreviewRefusalsUseTheAppendPreparation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		setup func(*testing.T) (*App, commissioner.RawCommand)
	}{
		{"waiver", func(t *testing.T) (*App, commissioner.RawCommand) {
			f := waiverApp(t)
			return f.app, f.raw(t, commissioner.WaiverProcess, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: "missing"}}, "league")
		}},
		{"bid", func(t *testing.T) (*App, commissioner.RawCommand) {
			f := bidApp(t)
			return f.app, f.raw(t, commissioner.BidsProcess, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5004"}}, "league")
		}},
		{"draft", func(t *testing.T) (*App, commissioner.RawCommand) {
			f := draftApp(t)
			f.scheduled(t, 3, 2, 1)
			f.now = draftStartAt().Add(time.Hour)
			return f.app, commissioner.RawCommand{League: draftLeague, Command: commissioner.DraftAssign,
				Args: commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: draftSeason,
					Act: &commissioner.RawDraftAct{Round: 1, Original: draftTeam(3), Player: "1000"}}},
				Expected: f.revision(t), Request: "refused-rookie", Reason: "Choose a rookie", Effective: "now", Visibility: "league"}
		}},
		{"trade", func(t *testing.T) (*App, commissioner.RawCommand) {
			f := tradeApp(t)
			offer := f.offer(t)
			f.accept(t, offer)
			f.app.core.CommissionerSession = testCommissionerSession{"creator"}
			return f.app, f.raw(t, commissioner.TradeOverride, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"}}, "league")
		}},
		{"pending trade", func(t *testing.T) (*App, commissioner.RawCommand) {
			f := tradeApp(t)
			f.offer(t)
			f.app.core.CommissionerSession = testCommissionerSession{"creator"}
			salary, years := domain.Money(200000000), 2
			return f.app, f.raw(t, commissioner.ContractEdit, commissioner.RawArgs{Override: &commissioner.RawNativeOverride{
				Player: "1000", Salary: &salary, Years: &years}}, "league")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, raw := tc.setup(t)
			preview := mustPreview(t, app, raw)
			body, err := json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			_, executeErr := app.TargetCommissionerExecute(string(body))
			if executeErr == nil || len(preview.Blockers) != 1 || preview.Blockers[0] != executeErr.Error() {
				t.Fatalf("real refusal differs: %+v / %v", preview.Blockers, executeErr)
			}
		})
	}
}

func TestReservePreviewShowsTheCurrentEligibilityAndAsksForTheReason(t *testing.T) {
	t.Parallel()
	f := legalityApp(t)
	raw := overrideCommand(t, f, commissioner.ReserveException, `{"player":"2002","bucket":"ir"}`)
	raw.Reason = ""
	preview := mustPreview(t, f.app, raw)
	if len(preview.Blockers) != 1 || preview.Blockers[0] != "Give a reason for this change" {
		t.Fatalf("missing reason: %+v", preview)
	}
	before := ""
	for _, player := range f.reserves(t).Players {
		if player.Player == "2002" {
			before = player.Words
		}
	}
	if before == "" {
		t.Fatal("test player has no reserve verdict")
	}
	assertPreviewTransition(t, preview, "eligibility", before, "May stay on IR")
	raw.Reason = "Medical ruling"
	previewAndExecute(t, f.app, raw)
}

func TestRunningDraftPreviewUsesTheRaisedSafeguard(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(time.Hour)
	raw := commissioner.RawCommand{League: draftLeague, Command: commissioner.DraftOrderSet,
		Args: commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: draftSeason, Round: 2,
			Order: draftOrder(3, 2, 1), Running: true}}, Expected: f.revision(t), Request: "raised-preview",
		Reason: "Set the running order", Effective: "now", Visibility: "league"}
	preview := mustPreview(t, f.app, raw)
	if len(preview.Blockers) != 0 || preview.Safeguard != "reason" {
		t.Fatalf("running draft safeguards: %+v", preview)
	}
}

func previewAndExecute(t *testing.T, app *App, raw commissioner.RawCommand) commissioner.Receipt {
	t.Helper()
	proveActPreview(t, app, raw)
	return executeCommissionerTest(t, app, raw)
}

func proveRefusedAct(t *testing.T, app *App, raw commissioner.RawCommand) {
	t.Helper()
	preview := mustPreview(t, app, raw)
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	_, executeErr := app.TargetCommissionerExecute(string(body))
	if executeErr == nil || len(preview.Blockers) != 1 || preview.Blockers[0] != executeErr.Error() {
		t.Fatalf("%s real refusal differs: %+v / %v", raw.Command, preview.Blockers, executeErr)
	}
	t.Logf("refusal-only proof: %s", raw.Command)
}

func TestStartingAssetsPreviewRejectsClientSuppliedAssets(t *testing.T) {
	t.Parallel()
	app := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, app, commissioner.PresetLegacy)
	raw, err := app.nativeSeedCommand(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := app.TargetCommissionerPreview(string(body))
	if err != nil || len(preview.Blockers) != 1 || preview.Blockers[0] != "Preview starting assets through the held-data request" {
		t.Fatalf("client starting assets accepted: %+v %v", preview, err)
	}
	input.Expected = 0
	body, err = json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	preview, err = app.TargetCommissionerPreview(string(body))
	_, executeErr := app.TargetNativeSeed(input)
	if err != nil || executeErr == nil || len(preview.Blockers) != 1 ||
		preview.Blockers[0] != previewExecuteReason(executeErr) {
		t.Fatalf("seed validation differs: %+v %v %v", preview, err, executeErr)
	}
}

func TestPreviewUsesTheCurrentFranchiseName(t *testing.T) {
	t.Parallel()
	f := capApp(t)
	assets := readNativeTest(t, f.app, legalityLeague)
	executeCommissionerTest(t, f.app, commissioner.RawCommand{League: legalityLeague, Command: "commish.franchise.edit",
		Args:     commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.02", Name: "Harbor", Abbreviation: "HBR"}},
		Expected: assets.Revision, Request: "rename-harbor", Reason: "Our franchise name", Effective: "now", Visibility: "league"})
	assets = readNativeTest(t, f.app, legalityLeague)
	raw := commissioner.RawCommand{League: legalityLeague, Command: commissioner.CorrectRoster,
		Args:     commissioner.RawArgs{Correction: &commissioner.RawCorrection{Player: "0099", Franchise: "franchise.02", Action: "remove"}},
		Expected: assets.Revision, Request: "remove-harbor-player", Reason: "Correct the roster", Effective: "now", Visibility: "league"}
	preview := mustPreview(t, f.app, raw)
	assertPreviewTransition(t, preview, "roster", "Harbor", "None recorded")
	if !strings.Contains(preview.Affects.Words, "Harbor") {
		t.Fatalf("old franchise name in effects: %+v", preview.Affects)
	}
	previewAndExecute(t, f.app, raw)
}
