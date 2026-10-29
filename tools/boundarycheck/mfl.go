package main

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type mflRules struct {
	fences map[string]bool
	seams  map[string]bool
	debt   map[string]bool
}

func readMFLRules(root string) (mflRules, error) {
	rules := mflRules{map[string]bool{}, map[string]bool{}, map[string]bool{}}
	f, err := os.Open(filepath.Join(root, "docs/architecture/boundary-mfl.tsv"))
	if err != nil {
		return rules, fmt.Errorf("open MFL boundary: %w", err)
	}
	defer func() { _ = f.Close() }()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if err := rules.addRow(line); err != nil {
			return rules, err
		}
	}
	if err := s.Err(); err != nil {
		return rules, fmt.Errorf("read MFL boundary: %w", err)
	}
	if len(rules.fences) == 0 {
		return rules, fmt.Errorf("empty MFL fence")
	}
	return rules, nil
}

func (rules mflRules) addRow(line string) error {
	fields := strings.Split(line, "\t")
	if len(fields) != 3 || !strings.HasPrefix(fields[1], "internal/") {
		return fmt.Errorf("invalid MFL boundary row: %s", line)
	}
	switch fields[0] {
	case "fence", "seam":
		if fields[2] != "-" {
			return fmt.Errorf("invalid MFL boundary row: %s", line)
		}
		target := rules.fences
		if fields[0] == "seam" {
			target = rules.seams
		}
		if target[fields[1]] {
			return fmt.Errorf("duplicate MFL boundary row: %s", line)
		}
		target[fields[1]] = true
	case "debt":
		if !strings.HasPrefix(fields[2], "internal/") {
			return fmt.Errorf("invalid MFL debt: %s", line)
		}
		key := fields[1] + "\t" + fields[2]
		if rules.debt[key] {
			return fmt.Errorf("duplicate MFL debt: %s", line)
		}
		rules.debt[key] = true
	default:
		return fmt.Errorf("invalid MFL boundary row: %s", line)
	}
	return nil
}

func checkMFL(root string) error {
	rules, err := readMFLRules(root)
	if err != nil {
		return err
	}
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return fmt.Errorf("read module: %w", err)
	}
	fields := strings.Fields(string(mod))
	if len(fields) < 2 || fields[0] != "module" {
		return fmt.Errorf("invalid module")
	}
	prefix := fields[1] + "/"
	seen := map[string]bool{}
	err = filepath.WalkDir(filepath.Join(root, "internal"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk MFL boundary: %w", walkErr)
		}
		return inspectMFLImports(root, path, prefix, entry, rules, seen)
	})
	if err != nil {
		return fmt.Errorf("check MFL imports: %w", err)
	}
	for key := range rules.debt {
		pair := strings.Split(key, "\t")
		if !rules.fences[pair[1]] || rules.fences[pair[0]] || rules.seams[pair[0]] || !seen[key] {
			return fmt.Errorf("stale MFL debt: %s", key)
		}
	}
	return nil
}

func inspectMFLImports(root, path, prefix string, entry fs.DirEntry, rules mflRules, seen map[string]bool) error {
	if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
		return nil
	}
	rel, relErr := filepath.Rel(root, filepath.Dir(path))
	if relErr != nil {
		return fmt.Errorf("MFL importer path: %w", relErr)
	}
	importer := filepath.ToSlash(rel)
	file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if parseErr != nil {
		return fmt.Errorf("parse imports %s: %w", path, parseErr)
	}
	for _, imp := range file.Imports {
		value, unquoteErr := strconv.Unquote(imp.Path.Value)
		if unquoteErr != nil {
			return fmt.Errorf("import %s: %w", path, unquoteErr)
		}
		if !strings.HasPrefix(value, prefix) {
			continue
		}
		target := strings.TrimPrefix(value, prefix)
		if !rules.fences[target] || rules.fences[importer] || rules.seams[importer] {
			continue
		}
		key := importer + "\t" + target
		if !rules.debt[key] {
			return fmt.Errorf("MFL boundary: %s imports fenced %s (%s)", importer, target, path)
		}
		seen[key] = true
	}
	return nil
}
