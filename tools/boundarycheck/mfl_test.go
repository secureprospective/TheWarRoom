package main

import (
	"strings"
	"testing"
)

func TestMFLNativeClosureRejectsTransitiveDebtAndSeamLeaks(t *testing.T) {
	for _, bridge := range []string{"internal/shared", "internal/composition"} {
		t.Run(bridge, func(t *testing.T) {
			root := boundaryFixture(t, "package composition\n")
			fixtureWrite(t, root, "docs/architecture/boundary-mfl.tsv", "fence\tinternal/mfl\t-\nnative\tinternal/native\t-\nseam\tinternal/composition\t-\n"+map[bool]string{true: "debt\tinternal/shared\tinternal/mfl\n"}[bridge == "internal/shared"])
			fixtureWrite(t, root, bridge+"/leak.go", "package bridge\nimport _ \"example.com/project/internal/mfl\"\n")
			fixtureWrite(t, root, "internal/native/logic.go", "package native\nimport _ \"example.com/project/"+bridge+"\"\n")
			err := checkMFL(root)
			if err == nil || !strings.Contains(err.Error(), "MFL transitive boundary: internal/native -> "+bridge+" -> internal/mfl") {
				t.Fatal(err)
			}
		})
	}
}

func TestMFLNativeClosureAllowsPureSharedCyclesAndMigrationSource(t *testing.T) {
	root := boundaryFixture(t, "package composition\n")
	fixtureWrite(t, root, "docs/architecture/boundary-mfl.tsv", "fence\tinternal/mfl\t-\nfence\tinternal/leaguemigration/mflsource\t-\nnative\tinternal/leaguemigration\t-\n")
	fixtureWrite(t, root, "internal/leaguemigration/mflsource/read.go", "package mflsource\nimport _ \"example.com/project/internal/mfl\"\n")
	fixtureWrite(t, root, "internal/leaguemigration/writer.go", "package leaguemigration\nimport _ \"example.com/project/internal/shared\"\n")
	fixtureWrite(t, root, "internal/shared/pure.go", "package shared\nimport _ \"example.com/project/internal/leaguemigration\"\n")
	if err := checkMFL(root); err != nil {
		t.Fatal(err)
	}
	fixtureWrite(t, root, "internal/shared/pure.go", "package shared\nimport _ \"example.com/project/internal/leaguemigration/mflsource\"\n")
	if err := checkMFL(root); err == nil || !strings.Contains(err.Error(), "imports fenced") {
		t.Fatal(err)
	}
}
