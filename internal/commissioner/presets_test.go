package commissioner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func presetLeague(t *testing.T, preset string, teams int) []Record {
	t.Helper()
	raw, err := (RawCommand{League: "native.test", Command: Create,
		Args:   RawArgs{Name: "Ours", Preset: preset, FranchiseCount: teams, Season: 2026},
		Reason: "Start", Request: "create", Effective: "now", Visibility: "league"}).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	if err := raw.Args.validateCreation(); err != nil {
		t.Fatal(err)
	}
	return []Record{{Command: raw, Principal: "creator", Role: "commissioner",
		Receipt: Receipt{Entry: "create", League: "native.test", Revision: 1,
			RecordedAt: noChangeTime().Add(-time.Hour)}}}
}

// The catalog changes where a preset is named, not what a league starts with: each creation, its ledger
// and preview words and its rules in force hash to what they were before the catalog (20a00b25). The
// rules reading is pinned too where the citation wording did not change; the 32-Team Default's reading
// now cites its own preset (see TestStartingCitationNamesTheLeaguesOwnPreset).
func TestPresetCreationIsUnchangedByTheCatalog(t *testing.T) {
	for _, c := range []struct {
		preset        string
		teams         int
		records, read string
	}{
		{PresetDefault32, 0, "5b88192c625de822", ""},
		{PresetLegacy, 32, "e7b5be9d4e7ca9f7", "f56f1bd65f098a02"},
		{PresetLegacy, 12, "80efa35b8407cd77", "06afc5e0695f1638"},
		{PresetEmpty, 12, "2611477a23c36c30", "bcd70492a864aac2"},
	} {
		records := presetLeague(t, c.preset, c.teams)
		rules, err := RulesInForce(records, 2026, noChangeTime())
		if err != nil {
			t.Fatal(err)
		}
		reading, err := RulesReading("native.test", records, noChangeTime())
		if err != nil {
			t.Fatal(err)
		}
		if got := digest(t, records[0].Command, recordLabel(records[0]),
			presetPreviewLabel(c.preset), rules); got != c.records {
			t.Errorf("%s at %d: creation digest %s, want %s", c.preset, c.teams, got, c.records)
		}
		if got := digest(t, reading); c.read != "" && got != c.read {
			t.Errorf("%s at %d: reading digest %s, want %s", c.preset, c.teams, got, c.read)
		}
	}
}

func digest(t *testing.T, parts ...any) string {
	t.Helper()
	h := sha256.New()
	for _, p := range parts {
		body, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		h.Write(body)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// A preset is only a bundle of setting values: a league started from it reads the same as an empty league
// whose commissioner set each of those values by hand. The one known difference is Legacy NFL's alignment:
// that bundle leaves it for the MFL import's own divisions, while setting Divisions by hand divides the
// franchises evenly (F5b), so by hand the alignment is set.
func TestAPresetIsTheSameAsSettingItsValuesByHand(t *testing.T) {
	for _, c := range []struct {
		preset  string
		teams   int
		differs []string
	}{{PresetDefault32, 32, nil}, {PresetLegacy, 32, []string{"league.alignment"}},
		{PresetLegacy, 12, []string{"league.alignment"}}} {
		preset, _ := presetByID(c.preset)
		started := presetLeague(t, c.preset, c.teams)
		byHand := presetLeague(t, PresetEmpty, c.teams)
		bundle := preset.rules(c.teams)
		values, current := leaguerules.ReadSettings(bundle), leaguerules.Current(bundle)
		for i, field := range leaguerules.Catalogue() {
			if values[i].State == "Needs a ruling" {
				continue
			}
			byHand = append(byHand, Record{Command: RawCommand{League: "native.test", Command: RulesSet,
				Request: fmt.Sprintf("hand.%d", i), Args: RawArgs{Field: string(field.Path), Value: current[i]},
				Effective: "now", Visibility: "league"}, Principal: "creator", Role: "commissioner",
				Receipt: Receipt{Entry: fmt.Sprintf("hand.%d", i), League: "native.test",
					Revision: int64(len(byHand) + 1), RecordedAt: noChangeTime().Add(-time.Minute)}})
		}
		want, err := RulesReading("native.test", started, noChangeTime())
		if err != nil {
			t.Fatal(err)
		}
		got, err := RulesReading("native.test", byHand, noChangeTime())
		if err != nil {
			t.Fatal(err)
		}
		if len(rows(got)) != len(rows(want)) {
			t.Fatalf("%s: %d rows by hand, %d from the preset", c.preset, len(rows(got)), len(rows(want)))
		}
		for i, w := range rows(want) {
			g := rows(got)[i]
			if slices.Contains(c.differs, w.Field) {
				if g.Value == w.Value {
					t.Errorf("%s at %d %s: expected to differ by hand, and now it does not", c.preset, c.teams, w.Field)
				}
				continue
			}
			if g.Field != w.Field || g.Value != w.Value || g.State != w.State || string(g.Current) != string(w.Current) {
				t.Errorf("%s at %d %s: by hand %q (%s) %s, from the preset %q (%s) %s", c.preset, c.teams, w.Field,
					g.Value, g.State, g.Current, w.Value, w.State, w.Current)
			}
		}
	}
}

func rows(r Rules) []RuleRow {
	var out []RuleRow
	for _, s := range r.Sections {
		out = append(out, s.Rows...)
	}
	return out
}

func TestStartingCitationNamesTheLeaguesOwnPreset(t *testing.T) {
	for _, c := range []struct{ preset, citation, want string }{
		{PresetDefault32, "Rulebook §4", "32-Team Default rules (rulebook section 4)"},
		{PresetDefault32, "DEFAULT-32 spec §2, locked 2026-10-26", "32-Team Default rules"},
		{PresetLegacy, "Rulebook §4", "Legacy NFL rules (rulebook section 4)"},
		{"", "Census CM-001", "Legacy NFL rules"},
		{PresetEmpty, "Rulebook §4", "Empty league default"},
	} {
		if got := startingSource(c.preset, "Set", c.citation); got != c.want {
			t.Errorf("%q %q: got %q, want %q", c.preset, c.citation, got, c.want)
		}
	}
}

// The start form offers what the catalog offers; Legacy NFL is the MFL import's bundle only.
func TestTheStartFormListsTheNativeCatalog(t *testing.T) {
	var ids []string
	for _, p := range ControlsReading().Presets {
		ids = append(ids, p.ID)
		if p.Label == "" || p.Description == "" || p.MinTeams < minFranchises || p.MaxTeams > maxFranchises {
			t.Errorf("%s: incomplete catalog entry %+v", p.ID, p)
		}
	}
	if !slices.Equal(ids, []string{PresetDefault32, PresetEmpty}) {
		t.Fatalf("start form presets: %v", ids)
	}
	if MFLImportPreset() != PresetLegacy {
		t.Fatalf("MFL import preset: %s", MFLImportPreset())
	}
}

// TestOnlyTheCatalogNamesAPreset allows the listed non-test Go files to name a preset; everything else asks
// the catalog.
func TestOnlyTheCatalogNamesAPreset(t *testing.T) {
	presetNameSites := map[string]string{
		"internal/commissioner/presets.go": "the catalog",
		"tools/hostedfixture/fixture.go":   "hosted wire fixture tool: records an empty test league",
		"tools/hostedfixture/workflows.go": "hosted wire fixture tool: records an MFL-import practice league",
	}
	names := regexp.MustCompile(`\b(PresetDefault32|PresetEmpty|PresetLegacy|IsDefault32)\b`)
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "frontend") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if _, allowed := presetNameSites[filepath.ToSlash(rel)]; allowed {
			return nil
		}
		body, err := os.ReadFile(path) //nolint:gosec // a repository source file
		if err != nil {
			return err
		}
		if names.Match(body) {
			t.Errorf("%s names a preset; read it from the catalog instead", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
