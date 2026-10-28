package commissioner

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestPlayerLookupsNameASide is the lint gate on the side seam: in the commissioner packages a line that
// compares a player's id must also name a side or a franchise, unless it is in player_sides.go or a file
// listed here with how many such lines it still holds. The allowance only shrinks: a new bare-id lookup
// fails, and so does one removed without lowering its count.
func TestPlayerLookupsNameASide(t *testing.T) {
	allowed := map[string]int{
		"correction_labels.go":          1, // a player's name for a label: every copy has the same name
		"result_flip.go":                1, // a lineup's starter within one franchise's lineup, not a copy
		"market/native_contract_cap.go": 1, // inside one franchise's roster, already filtered by franchise
		"preview_exceptions.go":         1, // preview words over every franchise's reserve verdicts
		"preview_affects.go":            1, // which franchise a previewed correction touches
		"draft_apply.go":                2, // the draft per side (F8c) narrows this to its side
		"draft_close.go":                1, // the draft per side (F8c) narrows this to its side
	}
	lookup := regexp.MustCompile(`(\bp|\bother|\bheld|\bstarter)\.(ID|Player) (==|!=) [A-Za-z_]|\.Players\[[^]]+\]\.ID|` +
		`Player\.ID (==|!=) [A-Za-z_]|\.Player == (a\.|player\b|id\b)`)
	named := regexp.MustCompile(`Side|Franchise|presetEntry`)
	found := map[string]int{}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") ||
			path == "player_sides.go" {
			return err
		}
		body, err := os.ReadFile(path) //nolint:gosec // a repository source file
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(body), "\n") {
			if lookup.MatchString(line) && !named.MatchString(line) {
				rel := filepath.ToSlash(path)
				found[rel]++
				if _, ok := allowed[rel]; !ok {
					t.Errorf("%s:%d looks a player up by id alone; use the side seam in player_sides.go", rel, i+1)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for file, n := range allowed {
		if found[file] != n {
			t.Errorf("%s: %d bare-id lookups, allowance %d; lower the allowance as sites move to the seam", file, found[file], n)
		}
	}
}
