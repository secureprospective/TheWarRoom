package commissioner

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/commissioner/commissionertest"
)

type stoppedHost struct{ Host }

func (stoppedHost) Ready() error { return errors.New("startup failed") }
func repoPath(parts ...string) string {
	return commissionertest.RepoPath(parts...)
}
func checkWire(t *testing.T, path string, body []byte, name string) {
	t.Helper()
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(repoPath(path), append(bytes.TrimSpace(body), '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(repoPath(path))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(held), bytes.TrimSpace(body)) {
		t.Fatalf("%s wire changed: %s", name, body)
	}
}
func TestCommissionerControlsWire(t *testing.T) {
	t.Parallel()
	app := New(nil)
	reading := app.TargetCommissionerControls()
	body, err := json.MarshalIndent(reading, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/commissioner/controls.wire.testdata.json", body, "controls")
	// The landing asks the registry, generated from the census, for each row's question and name.
	file, err := os.Open(repoPath("docs/ui/endpoint-registry.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	table, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	question := slices.Index(table[0], "gm_question")
	name := slices.Index(table[0], "name")
	asked := map[string]bool{}
	for _, row := range table[1:] {
		if strings.HasPrefix(row[0], "CM-") && strings.TrimSpace(row[question]) != "" && row[name] != "" {
			asked[row[0]] = true
		}
	}
	count := 0
	for _, group := range reading.Groups {
		for _, entry := range group.Controls {
			count++
			if !asked[entry.ID] {
				t.Fatalf("%s has no question in the registry", entry.ID)
			}
		}
	}
	if count != 56 || len(asked) != 56 {
		t.Fatalf("the landing lists %d rows against %d in the registry", count, len(asked))
	}
}
func TestNativeResultsIPCScopeAndStartup(t *testing.T) {
	t.Parallel()
	a := New(stoppedHost{})
	for _, scope := range []RawNativeResultsScope{
		{League: "native.", Season: 2026, Week: 1}, {League: "imported", Season: 2026, Week: 1},
		{League: " native.test", Season: 2026, Week: 1}, {League: "native.test", Season: 0, Week: 1},
		{League: "native.test", Season: 10000, Week: 1}, {League: "native.test", Season: 2026, Week: 0},
		{League: "native.test", Season: 2026, Week: 23},
	} {
		if err := scope.Validate(); err == nil {
			t.Fatalf("scope validation accepted %+v", scope)
		}
		if _, err := a.TargetNativeResults(scope.League, scope.Season, scope.Week); err == nil {
			t.Fatalf("accepted %+v", scope)
		}
	}
	if _, err := a.TargetNativeStandings("native.test", 0); err == nil {
		t.Fatal("invalid standings season")
	}
	if _, err := a.TargetNativeResults("native.test", 2026, 1); err == nil {
		t.Fatal("results ignored startup")
	}
	if _, err := a.TargetNativeStandings("native.test", 2026); err == nil {
		t.Fatal("standings ignored startup")
	}
	if _, err := a.TargetNativeScorePreview(`{"command":"bad"}`); err == nil {
		t.Fatal("invalid preview")
	}
}
func TestNativeBracketIPCScopeAndStartup(t *testing.T) {
	t.Parallel()
	a := New(stoppedHost{})
	for _, scope := range []RawNativeResultsScope{{League: "native.", Season: 2026}, {League: "imported",
		Season: 2026},
		{League: "native.test", Season: 0}, {League: "native.test", Season: 10000}} {
		if _, err := a.TargetNativeBracket(scope.League, scope.Season); err == nil {
			t.Fatalf("accepted %+v", scope)
		}
	}
	if _, err := a.TargetNativeBracket("native.test", 2026); err == nil || !strings.Contains(err.Error(),
		"startup") {
		t.Fatalf("the bracket ignored a failed startup: %v", err)
	}
}
