package commissioner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// A preset is a bundle of setting values. Comparing one with a league reads both through the setting
// catalogue the rules screen uses; applying it is an ordinary rules change for each chosen setting, so a
// preset has no checks, wording or gravity of its own.

// PresetDifference is one setting where the league and the preset disagree, in plain words.
type PresetDifference struct {
	Field   string `json:"field"`
	Label   string `json:"label"`
	Section string `json:"section"`
	League  string `json:"league"`
	Preset  string `json:"preset"`
	// Value is the preset's value as the plain JSON a rules change is sent in.
	Value json.RawMessage `json:"value"`
	// NowOnly says the setting can only change now, as on the rules screen.
	NowOnly bool `json:"nowOnly,omitempty"`
}

// PresetComparison lists every setting where the league differs from the preset.
type PresetComparison struct {
	League      string             `json:"league"`
	Preset      string             `json:"preset"`
	PresetLabel string             `json:"presetLabel"`
	Revision    int64              `json:"revision"`
	Summary     string             `json:"summary"`
	Differences []PresetDifference `json:"differences"`
	Presets     []StartingPreset   `json:"presets"`
}

// ComparablePresets are the catalog presets that carry values to compare with.
func ComparablePresets() []StartingPreset {
	out := []StartingPreset{}
	for _, p := range presetCatalog() {
		if p.rules != nil {
			out = append(out, p.StartingPreset)
		}
	}
	return out
}

// PresetCompare reads the league's settings in force against the preset's, sized to the league's teams.
func PresetCompare(league string, records []Record, preset string, now time.Time) (PresetComparison, error) {
	if len(records) == 0 {
		return PresetComparison{}, errors.New("this app league has not been created")
	}
	entry, ok := presetByID(preset)
	if !ok || entry.rules == nil {
		return PresetComparison{}, Refused(fmt.Errorf("compare with %s", comparableChoices()))
	}
	rules, err := RulesNow(records, now)
	if err != nil {
		return PresetComparison{}, err
	}
	teams := legacyTeams
	if rules.League.Teams.Value > 0 {
		teams = rules.League.Teams.Value
	}
	bundle := entry.rules(teams)
	fields := leaguerules.Catalogue()
	leagueWords, presetWords := leaguerules.ReadSettings(rules), leaguerules.ReadSettings(bundle)
	leagueNow, presetNow := leaguerules.Current(rules), leaguerules.Current(bundle)
	out := PresetComparison{League: league, Preset: entry.ID, PresetLabel: entry.Label,
		Revision: int64(len(records)), Differences: []PresetDifference{}, Presets: ComparablePresets()}
	for i, field := range fields {
		path := string(field.Path)
		if bytes.Equal(leagueNow[i], presetNow[i]) || !presetSettable(field.Path, presetNow[i]) {
			continue
		}
		out.Differences = append(out.Differences, PresetDifference{Field: path, Label: field.Label,
			Section: field.Section, League: leagueWords[i].Words, Preset: presetWords[i].Words,
			Value: presetNow[i], NowOnly: shapeField(path) || path == poolsField})
	}
	switch n := len(out.Differences); n {
	case 0:
		out.Summary = fmt.Sprintf("This league already matches the %s on every setting.", entry.Label)
	case 1:
		out.Summary = fmt.Sprintf("1 setting differs from the %s.", entry.Label)
	default:
		out.Summary = fmt.Sprintf("%d settings differ from the %s.", n, entry.Label)
	}
	return out, nil
}

// presetSettable keeps a difference only where a hand edit could send the preset's value: a value the
// preset leaves waiting for a ruling, or one the league counts for itself, is not a change anyone sends.
func presetSettable(path leaguerules.FieldPath, value json.RawMessage) bool {
	if string(value) == "null" || path == alignmentField || path == sidesField || path == startupListField {
		return false
	}
	_, err := leaguerules.DecodeSetting(path, value)
	return err == nil
}

func comparableChoices() string {
	var names []string
	for _, p := range presetCatalog() {
		if p.rules != nil {
			names = append(names, p.choice)
		}
	}
	last := len(names) - 1
	return strings.Join(names[:last], ", ") + " or " + names[last]
}

// RawPresetApply asks for the chosen differences (every one when Fields is empty) as one rules change.
type RawPresetApply struct {
	League    string   `json:"league"`
	Preset    string   `json:"preset"`
	Fields    []string `json:"fields"`
	Reason    string   `json:"reason"`
	Expected  int64    `json:"expected"`
	Request   string   `json:"request"`
	Effective string   `json:"effective"`
	// DryRun returns the rules changes without recording them, for the ordinary preview of each.
	DryRun bool `json:"dryRun,omitempty"`
}

// DecodePresetApply reads a preset apply body strictly.
func DecodePresetApply(body string) (RawPresetApply, error) {
	var raw RawPresetApply
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return raw, fmt.Errorf("read preset change: %w", err)
	}
	if strings.TrimSpace(raw.Request) == "" {
		return raw, errors.New("preset change needs a request key")
	}
	return raw, nil
}

// PresetRuleChanges turns the chosen differences into the rules changes a commissioner would send by
// hand, one per setting, in catalogue order. A chosen setting that does not differ is refused.
func PresetRuleChanges(raw RawPresetApply, comparison PresetComparison) ([]RawCommand, error) {
	chosen := raw.Fields
	if len(chosen) == 0 {
		for _, d := range comparison.Differences {
			chosen = append(chosen, d.Field)
		}
	}
	if len(chosen) == 0 {
		return nil, Refused(errors.New("this league already matches the preset. Nothing would change"))
	}
	for _, field := range chosen {
		if !slices.ContainsFunc(comparison.Differences, func(d PresetDifference) bool { return d.Field == field }) {
			return nil, Refused(fmt.Errorf("%s does not differ from the preset; choose a setting from the comparison", field))
		}
	}
	var out []RawCommand
	for _, d := range comparison.Differences {
		if !slices.Contains(chosen, d.Field) {
			continue
		}
		out = append(out, RawCommand{League: raw.League, Command: RulesSet, Reason: raw.Reason,
			Expected: raw.Expected, Request: raw.Request + "." + d.Field, Effective: raw.Effective,
			Visibility: "league", Args: RawArgs{Field: d.Field, Value: slices.Clone(d.Value)}})
	}
	return out, nil
}
