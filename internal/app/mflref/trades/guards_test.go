package trades

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestCommandsMatchTheBoundSurface(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeFor[*API]()
	var methods, commands []string
	for i := range typ.NumMethod() {
		methods = append(methods, typ.Method(i).Name)
	}
	for _, c := range Commands() {
		commands = append(commands, c.Method)
	}
	if !slices.Equal(methods, commands) {
		t.Fatalf("methods %v, catalog %v", methods, commands)
	}
}

func TestNoSourceBuildsMFLsTradeActionLink(t *testing.T) {
	t.Parallel()
	needle := "trade_" + "response"
	var hits []string
	repo := filepath.Join("..", "..", "..", "..")
	for _, root := range []string{repo, filepath.Join(repo, "frontend", "src")} {
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
