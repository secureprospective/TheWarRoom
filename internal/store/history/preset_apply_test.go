package history

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func presetStore(t *testing.T, create commissioner.RawCommand) *Store {
	t.Helper()
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), create, "creator", nil); err != nil {
		t.Fatal(err)
	}
	return s
}

func presetComparison(t *testing.T, s *Store, league, preset string) commissioner.PresetComparison {
	t.Helper()
	records, err := s.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	c, err := commissioner.PresetCompare(league, records, preset, s.now())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func presetChanges(t *testing.T, s *Store, apply commissioner.RawPresetApply) []commissioner.RawCommand {
	t.Helper()
	raws, err := commissioner.PresetRuleChanges(apply, presetComparison(t, s, apply.League, apply.Preset))
	if err != nil {
		t.Fatal(err)
	}
	return raws
}

func TestPresetCompareFixtureLeagues(t *testing.T) {
	t.Parallel()
	native := presetStore(t, createWith(commissioner.PresetDefault32, 32))
	same := presetComparison(t, native, "native.test", commissioner.PresetDefault32)
	if len(same.Differences) != 0 || same.Summary != "This league already matches the 32-Team Default on every setting." {
		t.Fatalf("Default32 against itself: %+v", same)
	}
	imported := presetStore(t, importedLegacy(32))
	got := presetComparison(t, imported, "native.test", commissioner.PresetDefault32)
	var fields []string
	for _, d := range got.Differences {
		fields = append(fields, d.Field)
		if d.Label == "" || d.League == "" || d.Preset == "" || d.League == d.Preset || len(d.Value) == 0 {
			t.Errorf("difference not in plain words: %+v", d)
		}
	}
	want := []string{"league.startingAssets", "league.divisionConferences", "roster.limits", "roster.positionBounds",
		"lineup.lock", "lineup.positions", "lineup.groups", "scoring.catalogue", "scoring.extras", "scoring.ranges",
		"scoring.scales", "scoring.totals", "scoring.lineUnit", "ir.autoReturn", "ir.eligible", "taxi.autoPromote",
		"taxi.basis", "draft.startupRounds"}
	if !slices.Equal(fields, want) || got.Summary != "18 settings differ from the 32-Team Default." {
		t.Fatalf("imported Legacy against Default32:\n got %v\nwant %v (%s)", fields, want, got.Summary)
	}
	if got.Differences[2].League != "18 to 80" || got.Differences[2].Preset != "18 to 44" {
		t.Fatalf("roster limits words: %+v", got.Differences[2])
	}
	if _, err := commissioner.PresetCompare("native.test", nil, commissioner.PresetEmpty, s0(imported)); err == nil {
		t.Fatal("compared with no league")
	}
	records, _ := imported.nativeRecords(t.Context(), "native.test")
	if _, err := commissioner.PresetCompare("native.test", records, commissioner.PresetEmpty, s0(imported)); err == nil {
		t.Fatal("compared with a preset that holds no values")
	}
}

func s0(s *Store) (at time.Time) { return s.now() }

// A preset applied to a subset records exactly the ledger the same edits made by hand record.
func TestPresetApplySubsetMatchesHandEdits(t *testing.T) {
	t.Parallel()
	byPreset, byHand := presetStore(t, importedLegacy(32)), presetStore(t, importedLegacy(32))
	apply := commissioner.RawPresetApply{League: "native.test", Preset: commissioner.PresetDefault32,
		Fields: []string{"taxi.basis", "roster.limits", "ir.autoReturn"}, Reason: "Adopt the 32-Team Default",
		Expected: 1, Request: "preset", Effective: "next_rollover"}
	raws := presetChanges(t, byPreset, apply)
	if got := []string{raws[0].Args.Field, raws[1].Args.Field, raws[2].Args.Field}; !slices.Equal(got,
		[]string{"roster.limits", "ir.autoReturn", "taxi.basis"}) {
		t.Fatalf("changes not in catalogue order: %v", got)
	}
	receipts, err := byPreset.ExecuteRulesBatchWithFacts(t.Context(), raws, "creator", nil, nil, nil)
	if err != nil || len(receipts) != 3 {
		t.Fatalf("apply: %v %v", receipts, err)
	}
	var hand []commissioner.Receipt
	for i, raw := range raws {
		raw.Expected = int64(1 + i)
		receipt, err := byHand.ExecuteCommissioner(t.Context(), raw, "creator", nil)
		if err != nil {
			t.Fatal(err)
		}
		hand = append(hand, receipt)
	}
	if !reflect.DeepEqual(receipts, hand) {
		t.Fatalf("receipts differ:\npreset %+v\n  hand %+v", receipts, hand)
	}
	presetRecords, _ := byPreset.nativeRecords(t.Context(), "native.test")
	handRecords, _ := byHand.nativeRecords(t.Context(), "native.test")
	if !reflect.DeepEqual(presetRecords, handRecords) {
		t.Fatal("ledgers differ between the preset apply and the same edits by hand")
	}
	for _, s := range []*Store{byPreset, byHand} {
		if row := ruleRow(t, s, "roster.limits"); len(row.Pending) != 1 {
			t.Fatalf("limits not waiting for the rollover: %+v", row)
		}
	}
	if left := presetComparison(t, byPreset, "native.test", commissioner.PresetDefault32); len(left.Differences) != 18 {
		t.Fatalf("waiting changes are not in force yet; still %d differences", len(left.Differences))
	}
	// A chosen setting that already matches is refused, as a hand edit that changes nothing is.
	apply.Fields = []string{"league.teams"}
	if _, err := commissioner.PresetRuleChanges(apply, presetComparison(t, byPreset, "native.test",
		commissioner.PresetDefault32)); err == nil {
		t.Fatal("applied a setting that does not differ")
	}
}

// A setting that cannot change once play has begun is refused for a preset exactly as by hand, and the
// whole preset change records nothing.
func TestPresetApplyRefusedMidSeasonLikeAHandEdit(t *testing.T) {
	t.Parallel()
	played := func() *Store {
		s := presetStore(t, createWith(commissioner.PresetMirror16, 16))
		generate := commissioner.RawCommand{League: "native.test", Command: commissioner.ScheduleGenerate,
			Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 3}}, Expected: 1,
			Request: "generate", Effective: "now", Visibility: "league"}
		if _, err := s.ExecuteCommissioner(t.Context(), generate, "creator", nil); err != nil {
			t.Fatal(err)
		}
		return s
	}
	byPreset, byHand := played(), played()
	apply := commissioner.RawPresetApply{League: "native.test", Preset: commissioner.PresetDefault32,
		Reason: "Adopt the 32-Team Default", Expected: 2, Request: "preset", Effective: "now"}
	raws := presetChanges(t, byPreset, apply)
	if raws[0].Args.Field != "players.pools" {
		t.Fatalf("first change %s", raws[0].Args.Field)
	}
	_, presetErr := byPreset.ExecuteRulesBatchWithFacts(t.Context(), raws, "creator", nil, nil, nil)
	_, handErr := byHand.ExecuteCommissioner(t.Context(), raws[0], "creator", nil)
	var refusal *commissioner.Refusal
	if !errors.As(presetErr, &refusal) || !errors.As(handErr, &refusal) ||
		!strings.Contains(handErr.Error(), "player pools can't change once the season has games") ||
		!strings.HasSuffix(presetErr.Error(), handErr.Error()) {
		t.Fatalf("refusals differ:\npreset %v\n  hand %v", presetErr, handErr)
	}
	if revision(t, byPreset) != 2 {
		t.Fatal("a refused preset change recorded part of itself")
	}
	// Without the refused setting the rest applies.
	apply.Fields = []string{"roster.positionBounds", "draft.startupRounds"}
	if _, err := byPreset.ExecuteRulesBatchWithFacts(t.Context(), presetChanges(t, byPreset, apply), "creator",
		nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if revision(t, byPreset) != 4 {
		t.Fatalf("revision %d", revision(t, byPreset))
	}
}

// Applying a preset to one league leaves another league's readings byte for byte as they were.
func TestPresetApplyLeavesOtherLeaguesUntouched(t *testing.T) {
	t.Parallel()
	s := presetStore(t, importedLegacy(32))
	other := createWith(commissioner.PresetDefault32, 32)
	other.League = "native.other"
	if _, err := s.ExecuteCommissioner(t.Context(), other, "creator", nil); err != nil {
		t.Fatal(err)
	}
	read := func() []byte {
		rules, err := s.CommissionerRules(t.Context(), "native.other")
		if err != nil {
			t.Fatal(err)
		}
		ledger, err := s.CommissionerLedger(t.Context(), "native.other", "creator", nil)
		if err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal([]any{rules, ledger, presetComparison(t, s, "native.other", commissioner.PresetLegacy)})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	before := read()
	apply := commissioner.RawPresetApply{League: "native.test", Preset: commissioner.PresetDefault32,
		Reason: "Adopt the 32-Team Default", Expected: 1, Request: "preset", Effective: "next_rollover"}
	raws := slices.DeleteFunc(presetChanges(t, s, apply), func(r commissioner.RawCommand) bool {
		return r.Args.Field == "league.startingAssets"
	})
	if _, err := s.ExecuteRulesBatchWithFacts(t.Context(), raws, "creator", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if after := read(); string(after) != string(before) {
		t.Fatal("an untouched league's readings changed")
	}
}

func TestRulesBatchTakesOnlyRules(t *testing.T) {
	t.Parallel()
	s := presetStore(t, importedLegacy(32))
	if _, err := s.ExecuteRulesBatchWithFacts(t.Context(), []commissioner.RawCommand{createCommand()}, "creator",
		nil, nil, nil); err == nil {
		t.Fatal("a rules batch recorded a creation")
	}
	// Commissioner permission is the hand edit's: an owner is refused the same change both ways.
	raws := presetChanges(t, s, commissioner.RawPresetApply{League: "native.test", Preset: commissioner.PresetDefault32,
		Fields: []string{"roster.limits"}, Expected: 1, Request: "owner", Effective: "next_rollover"})
	_, batchErr := s.ExecuteRulesBatchWithFacts(t.Context(), raws, "owner", nil, nil, nil)
	_, handErr := s.ExecuteCommissioner(t.Context(), raws[0], "owner", nil)
	if batchErr == nil || handErr == nil || !strings.HasSuffix(batchErr.Error(), handErr.Error()) {
		t.Fatalf("owner refusals differ: %v / %v", batchErr, handErr)
	}
	if revision(t, s) != 1 {
		t.Fatal("a refused owner change was recorded")
	}
}
