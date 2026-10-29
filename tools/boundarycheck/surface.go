package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

func mflCommandSurface() []string {
	var rows []string
	add := func(area string, commands []catalog.Command) {
		for _, c := range commands {
			rows = append(rows, strings.Join([]string{area, c.ID, c.Method, c.Version, c.Availability}, "\t"))
		}
	}
	add("mfl", append(catalog.MFLCommands(catalog.MFLShapes{}), catalog.MFLPlanCommands()...))
	for _, feature := range []string{"contracts", "trades", "platform"} {
		add(feature, catalog.DesktopCommands(feature, nil))
	}
	for _, entry := range catalog.MigrationEntries(catalog.MigrationInputs{}) {
		add("migration", []catalog.Command{entry.Definition})
	}
	sort.Strings(rows)
	return rows
}

func checkMFLCommandSurface(root string, current []string) error {
	body, err := os.ReadFile(filepath.Join(root, "docs/architecture/boundary-mfl-commands.tsv"))
	if err != nil {
		return fmt.Errorf("read MFL command allowlist: %w", err)
	}
	approved := map[string]string{}
	for _, row := range strings.Split(string(body), "\n") {
		if row != "" && !strings.HasPrefix(row, "#") {
			approved[row] = row
		}
	}
	for _, row := range current {
		if _, ok := approved[row]; !ok {
			return fmt.Errorf("unreviewed MFL command: %s", row)
		}
		delete(approved, row)
	}
	if len(approved) > 0 {
		return fmt.Errorf("stale MFL command allowlist: %s", sortedKeys(approved)[0])
	}
	return nil
}
