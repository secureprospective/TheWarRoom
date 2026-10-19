package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MFL's trade-response action link acts at once on a GET (Ring1_Gap_Closure I7): accepting, rejecting
// or withdrawing an offer with no confirmation. The app hands every trade off to the trade desk
// (O=05) and must never build that link. The needle is assembled so this file does not match.
func TestNoSourceBuildsMFLsTradeActionLink(t *testing.T) {
	t.Parallel()
	needle := "trade_" + "response"
	var hits []string
	for _, root := range []string{".", "frontend/src"} {
		fsys := os.DirFS(root)
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				switch d.Name() {
				case "node_modules", "dist", "build", ".git", "testdata", "docs", "frontend":
					if path != "." {
						return fs.SkipDir
					}
				}
				return nil
			}
			if ext := filepath.Ext(path); ext != ".go" && ext != ".ts" && ext != ".tsx" {
				return nil
			}
			b, err := fs.ReadFile(fsys, path)
			if err != nil {
				return err
			}
			if strings.Contains(string(b), needle) {
				hits = append(hits, filepath.Join(root, path))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(hits) > 0 {
		t.Fatalf("source builds MFL's irreversible trade action link: %v", hits)
	}
}
