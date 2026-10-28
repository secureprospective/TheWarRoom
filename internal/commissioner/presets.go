package commissioner

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// The preset catalog is the only place a preset is named. A preset is a named bundle of setting values
// applied when a league is created; the commissioner can reach the same league by setting each value by
// hand, so nothing outside this file asks which preset a league started from.
const (
	PresetLegacy = "legacy_nfl"
	PresetEmpty  = "empty"
	// PresetDefault32 is the 32-Team Default (DEFAULT-32 spec), offered only at 32 teams.
	PresetDefault32 = "default_32"
)

// StartingPreset is what the start form shows of a catalog entry.
type StartingPreset struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	MinTeams    int    `json:"minTeams"`
	MaxTeams    int    `json:"maxTeams"`
}

// presetEntry is one entry of the catalog: what the form shows, the wording Go writes in the ledger, the
// preview and each rule's citation, and the bundle of values itself.
type presetEntry struct {
	StartingPreset

	native       bool   // offered for a new app league; otherwise only an MFL import or a recovery uses it
	source       string // where a league from this preset comes from when the caller does not say
	defaultTeams int    // the size a create without one takes; zero asks the caller
	choice       string // how a refusal names it among the choices
	history      string // the ledger's "Created league … with …"
	preview      string // the preview's starting rules
	citation     string // where a starting value came from, on the rules reading
	// bracket and bracketShape name the built-in bracket shape this preset's playoff graph offers, in the
	// bracket reading's words and in the preview's; empty offers none.
	bracket, bracketShape string
	// rules is the bundle: every setting value the preset applies, alignment, division conferences,
	// the O-Line card and the startup draft included. Nil leaves every rule waiting for a ruling.
	rules func(teams int) leaguerules.RuleSet
}

// presetCatalog lists the presets in the order a refusal and the start form name them.
func presetCatalog() []presetEntry {
	return []presetEntry{
		{StartingPreset: StartingPreset{ID: PresetDefault32, Label: "32-Team Default",
			Description: "The 32-Team Default is for 32 franchises: the NFL's divisions, a startup draft " +
				"and the DEFAULT-32 lineup, roster and scoring. Every value can be changed afterwards.",
			MinTeams: legacyTeams, MaxTeams: legacyTeams},
			native: true, source: SourceNative, defaultTeams: legacyTeams, choice: "the 32-Team Default",
			history: "the 32-Team Default rules", preview: "32-Team Default rules", citation: "32-Team Default rules",
			rules: func(int) leaguerules.RuleSet { return leaguerules.Default32() }},
		{StartingPreset: StartingPreset{ID: PresetLegacy, Label: "Legacy NFL",
			Description: "The Legacy NFL's rules, as imported from MFL.",
			MinTeams:    minFranchises, MaxTeams: maxFranchises},
			source: SourceMFLImport, defaultTeams: legacyTeams, choice: "the Legacy NFL preset",
			history: "the Legacy NFL rules", preview: "Legacy NFL rules", citation: "Legacy NFL rules",
			bracket: "the Legacy NFL bracket", bracketShape: "the Legacy NFL playoff shape",
			rules: leaguerules.LegacyNFLForTeams},
		{StartingPreset: StartingPreset{ID: PresetEmpty, Label: "Start empty: every rule Needs a ruling",
			Description: "An empty league works with 2 to 32 franchises; every rule waits for a ruling.",
			MinTeams:    minFranchises, MaxTeams: maxFranchises},
			native: true, source: SourceNative, choice: "an empty league",
			history: "every rule waiting for a ruling", preview: "Every rule waits for a ruling",
			citation: "Empty league default"},
	}
}

// StartingPresets are the presets a new app league may start from, for the start form.
func StartingPresets() []StartingPreset {
	var out []StartingPreset
	for _, p := range presetCatalog() {
		if p.native {
			out = append(out, p.StartingPreset)
		}
	}
	return out
}

func presetByID(id string) (presetEntry, bool) {
	i := slices.IndexFunc(presetCatalog(), func(p presetEntry) bool { return p.ID == id })
	if i < 0 {
		return presetEntry{}, false
	}
	return presetCatalog()[i], true
}

// MFLImportPreset is the preset an imported or recovered league starts from when its record names none.
func MFLImportPreset() string {
	i := slices.IndexFunc(presetCatalog(), func(p presetEntry) bool { return p.source == SourceMFLImport })
	return presetCatalog()[i].ID
}

// storedPreset reads a preset id from a record; an id the catalog does not hold reads as unnamed. A
// league recorded before presets were named started empty in the ledger's and preview's words, but its
// citations came from the MFL import.
func storedPreset(id string, unnamed func() string) presetEntry {
	if p, ok := presetByID(id); ok {
		return p
	}
	p, _ := presetByID(unnamed())
	return p
}

func emptyPreset() string {
	i := slices.IndexFunc(presetCatalog(), func(p presetEntry) bool { return p.rules == nil })
	return presetCatalog()[i].ID
}

func (p presetEntry) seeds() bool { return p.rules != nil }

func (p presetEntry) sizeError(count int) error {
	if count >= p.MinTeams && count <= p.MaxTeams {
		return nil
	}
	if p.MinTeams == p.MaxTeams {
		return fmt.Errorf("the %s needs exactly %d franchises", p.Label, p.MinTeams)
	}
	return fmt.Errorf("choose between %d and %d franchises", p.MinTeams, p.MaxTeams)
}

func unknownPresetError() error {
	var names []string
	for _, p := range presetCatalog() {
		names = append(names, p.choice)
	}
	last := len(names) - 1
	return fmt.Errorf("start from %s or %s", strings.Join(names[:last], ", "), names[last])
}

func presetHistoryLabel(preset string) string { return storedPreset(preset, emptyPreset).history }

func presetPreviewLabel(preset string) string { return storedPreset(preset, emptyPreset).preview }

// builtInBracket is a bracket shape the app carries by name: the league's own rules, or a preset's graph.
type builtInBracket struct {
	id, words, shape string
	graph            func(rules leaguerules.RuleSet) []leaguerules.BracketNode
}

func builtInBrackets() []builtInBracket {
	out := []builtInBracket{{id: presetLeagueRules, words: "the league's playoff rules",
		shape: "the league playoff rules",
		graph: func(rules leaguerules.RuleSet) []leaguerules.BracketNode { return rules.Playoffs.Graph.Value }}}
	for _, p := range presetCatalog() {
		if p.bracket != "" {
			out = append(out, builtInBracket{id: p.ID, words: p.bracket, shape: p.bracketShape,
				graph: func(leaguerules.RuleSet) []leaguerules.BracketNode {
					return p.rules(legacyTeams).Playoffs.Graph.Value
				}})
		}
	}
	return out
}

func builtInBracketByID(id string) (builtInBracket, bool) {
	i := slices.IndexFunc(builtInBrackets(), func(b builtInBracket) bool { return b.id == id })
	if i < 0 {
		return builtInBracket{}, false
	}
	return builtInBrackets()[i], true
}
