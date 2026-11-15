// Package seams holds the gate that keeps the future-seam markers in the code and the registry in
// docs/architecture/future-seams.md in step, so a later build finds every place it should plug in.
package seams

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestFutureSeamsAreRegistered(t *testing.T) {
	root := filepath.Join("..", "..")
	doc, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "future-seams.md"))
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]string{}
	row := regexp.MustCompile("(?m)^\\| `([a-z-]+)` \\| `([^`]+)` \\|")
	for _, m := range row.FindAllStringSubmatch(string(doc), -1) {
		rows[m[1]] = m[2]
	}
	if len(rows) == 0 {
		t.Fatal("registry has no seam rows")
	}
	marker := regexp.MustCompile(`//\s*FUTURE-SEAM\(([a-z-]+)\):`)
	repo, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	tree := repo.FS()
	found := map[string]string{}
	err = fs.WalkDir(tree, "internal", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		src, err := fs.ReadFile(tree, path)
		if err != nil {
			return err
		}
		for _, m := range marker.FindAllStringSubmatch(string(src), -1) {
			if prev, dup := found[m[1]]; dup {
				t.Errorf("seam %q marked twice: %s and %s", m[1], prev, path)
			}
			found[m[1]] = path
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, file := range found {
		if want, ok := rows[name]; !ok {
			t.Errorf("%s: marker FUTURE-SEAM(%s) has no row in docs/architecture/future-seams.md", file, name)
		} else if want != file {
			t.Errorf("seam %q: registry says %s, marker is in %s", name, want, file)
		}
	}
	for name, file := range rows {
		if _, ok := found[name]; !ok {
			t.Errorf("registry row %q names %s, but no FUTURE-SEAM(%s) marker is there", name, file, name)
		}
	}
}
