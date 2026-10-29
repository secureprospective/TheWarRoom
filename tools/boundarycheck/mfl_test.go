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

func TestMFLNativeMathRootsAreDeclared(t *testing.T) {
	rules, err := readMFLRules("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"internal/engine/contractacts", "internal/model", "internal/domain"} {
		if !nativePackage(path, rules.native) {
			t.Fatalf("native math/domain root missing: %s", path)
		}
	}
}

func TestMFLFreezeRejectsNewCommandAndReferencePackage(t *testing.T) {
	root := boundaryFixture(t, "package composition\n")
	current := append(mflCommandSurface(), "mfl\tmfl.synthetic.new\tTargetSynthetic\t1\tdesktop")
	if err := checkMFLCommandSurface(root, current); err == nil || !strings.Contains(err.Error(), "unreviewed MFL command: mfl\tmfl.synthetic.new") {
		t.Fatal(err)
	}
	if err := checkMFLCommandSurface(root, mflCommandSurface()); err != nil {
		t.Fatal(err)
	}
	fixtureWrite(t, root, "internal/mflref/new/read.go", "package newref\n")
	if err := checkMFL(root); err == nil || !strings.Contains(err.Error(), "unreviewed MFL reference package: internal/mflref/new") {
		t.Fatal(err)
	}
}
