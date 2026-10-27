// boundarycheck freezes reviewed boundary wiring, including declarations and semantic directives.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	root := flag.String("root", ".", "repository root")
	inventory := flag.Bool("inventory", false, "print decision fingerprints for explicit review, not an accepted baseline")
	flag.Parse()
	if err := run(*root, *inventory, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "boundarycheck:", err)
		os.Exit(1)
	}
}

func run(root string, inventory bool, output io.Writer) error {
	current, err := decisions(root)
	if err != nil {
		return err
	}
	keys := sortedKeys(current)
	if inventory {
		for _, key := range keys {
			if _, err := fmt.Fprintf(output, "%s\t%s\treview\n", key, current[key]); err != nil {
				return fmt.Errorf("write inventory: %w", err)
			}
		}
		return nil
	}
	baseline, err := readBaseline(filepath.Join(root, "docs/architecture/boundary-adapters.tsv"))
	if err != nil {
		return err
	}
	for _, key := range keys {
		if strings.HasSuffix(key, ":#source") {
			continue
		}
		approved, found := baseline[key]
		if !found {
			return fmt.Errorf("%s: new decision syntax; move policy into its service (or explicitly review resource/transport wiring)", key)
		}
		if approved != current[key] {
			return fmt.Errorf("%s: changed decision-bearing function; move policy inward, do not refresh the allowlist to hide it", key)
		}
	}
	for _, key := range sortedKeys(baseline) {
		if _, found := current[key]; !found {
			return fmt.Errorf("%s: stale allowlist entry; remove it to tighten the boundary", key)
		}
	}
	if err := checkSources(keys, current, baseline); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "boundarycheck: %d frozen functions/sources; no unreviewed composition/core changes\n", len(current))
	if err != nil {
		return fmt.Errorf("write result: %w", err)
	}
	return nil
}

func checkSources(keys []string, current, baseline map[string]string) error {
	for _, key := range keys {
		if !strings.HasSuffix(key, ":#source") {
			continue
		}
		approved, found := baseline[key]
		if !found {
			return fmt.Errorf("%s: unreviewed boundary source; review declarations and straight-line wiring", key)
		}
		if approved != current[key] {
			return fmt.Errorf("%s: changed boundary source; review declarations and straight-line wiring, not just branches", key)
		}
	}
	return nil
}

func decisions(root string) (map[string]string, error) {
	out := map[string]string{}
	for _, directory := range []string{"internal/composition", "internal/app/core"} {
		paths, err := boundaryPaths(filepath.Join(root, directory))
		if err != nil {
			return nil, err
		}

		for _, path := range paths {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return nil, fmt.Errorf("boundary relative path: %w", err)
			}
			relative = filepath.ToSlash(relative)
			if strings.HasSuffix(path, "_test.go") || relative == "internal/composition/wails_surface.gen.go" {
				continue
			}
			if err := inspectFile(path, relative, out); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func boundaryPaths(directory string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk boundary: %w", err)
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list boundary sources: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%s: no boundary sources", directory)
	}
	return paths, nil
}

func inspectFile(path, relative string, out map[string]string) error {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", relative, err)
	}
	digest, err := fingerprint(file)
	if err != nil {
		return fmt.Errorf("fingerprint source %s: %w", relative, err)
	}
	out[relative+":#source"] = digest
	// Fixture code is not production policy, but its source still cannot change unnoticed.
	if strings.HasPrefix(relative, "internal/app/core/coretest/") {
		return nil
	}
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || !hasDecision(fn) {
			continue
		}
		name := fn.Name.Name
		if fn.Recv != nil {
			var receiver bytes.Buffer
			if err := format.Node(&receiver, token.NewFileSet(), fn.Recv.List[0].Type); err != nil {
				return fmt.Errorf("format %s receiver: %w", relative, err)
			}
			name = "(" + receiver.String() + ")." + name
		}
		digest, err := fingerprint(fn)
		if err != nil {
			return fmt.Errorf("fingerprint %s:%s: %w", relative, name, err)
		}
		out[relative+":"+name] = digest
	}
	return nil
}

func hasDecision(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SelectStmt, *ast.BinaryExpr:
			found = true
		case *ast.UnaryExpr:
			found = found || value.Op == token.NOT
		}
		return !found
	})
	return found
}

func fingerprint(fn ast.Node) (string, error) {
	var formatted bytes.Buffer
	if err := format.Node(&formatted, token.NewFileSet(), fn); err != nil {
		return "", fmt.Errorf("format function: %w", err)
	}
	// Hash Go tokens; semantic source directives are included separately.
	var scan scanner.Scanner
	files := token.NewFileSet()
	scan.Init(files.AddFile("", -1, formatted.Len()), formatted.Bytes(), nil, 0)
	var canonical bytes.Buffer
	for {
		_, kind, literal := scan.Scan()
		if kind == token.EOF {
			break
		}
		_, _ = fmt.Fprintf(&canonical, "%d:%s\x00", kind, literal)
	}
	if file, ok := fn.(*ast.File); ok {
		for _, group := range file.Comments {
			for _, comment := range group.List {
				if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "// +build ") || strings.HasPrefix(comment.Text, "//line ") {
					_, _ = fmt.Fprintf(&canonical, "directive:%s\x00", comment.Text)
				}
			}
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256(canonical.Bytes())), nil
}

func readBaseline(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open decision allowlist: %w", err)
	}
	defer func() { _ = file.Close() }()
	out := map[string]string{}
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		line := lines.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 3 || len(fields[1]) != 64 || (fields[2] != "adapter" && fields[2] != "legacy") {
			return nil, fmt.Errorf("invalid allowlist row: %s", line)
		}
		if _, exists := out[fields[0]]; exists {
			return nil, fmt.Errorf("duplicate allowlist entry: %s", fields[0])
		}
		out[fields[0]] = fields[1]
	}
	if err := lines.Err(); err != nil {
		return nil, fmt.Errorf("read decision allowlist: %w", err)
	}
	return out, nil
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
