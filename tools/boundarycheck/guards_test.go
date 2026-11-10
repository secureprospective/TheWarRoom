package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestBoundaryRejectsStraightLineAndDeclarationChanges(t *testing.T) {
	for _, source := range []string{
		"package composition\nfunc adapter() int { return policy() }\nfunc policy() int { return 2 }",
		"package composition\nvar policy = func(cap int) bool { return cap > 1 }",
		"package composition\nconst minimum = 2\nfunc adapter() int { return minimum }",
		"package composition\ntype rule struct { minimum int }; var thresholds = map[string]int{\"cap\":2}",
	} {
		root := boundaryFixture(t, "package composition\nfunc adapter() int { return 1 }")
		approveFixture(t, root)
		fixtureWrite(t, root, "internal/composition/adapter.go", source)
		err := run(root, false, &bytes.Buffer{})
		if err == nil || (!strings.Contains(err.Error(), "changed boundary source") && !strings.Contains(err.Error(), "new decision syntax")) {
			t.Fatal(err)
		}
	}
}

func TestBoundaryRejectsNestedSourcesAndPackageInitializers(t *testing.T) {
	for _, directory := range []string{"internal/composition/nested", "internal/app/core/nested"} {
		root := boundaryFixture(t, "package composition\n")
		approveFixture(t, root)
		path := directory + "/policy.go"
		fixtureWrite(t, root, path, "package nested\nvar rule = func(cap int) bool { return cap > 1 }")
		err := run(root, false, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "unreviewed boundary source") {
			t.Fatal(err)
		}
	}
}

func TestMFLBoundaryAllowsSourceAdapterNotDestinationWriter(t *testing.T) {
	root := boundaryFixture(t, "package composition\n")
	fixtureWrite(t, root, "docs/architecture/boundary-mfl.tsv",
		"fence\tinternal/mfl\t-\nfence\tinternal/leaguemigration/mflsource\t-\n")
	fixtureWrite(t, root, "internal/leaguemigration/mflsource/source.go",
		"package mflsource\nimport _ \"example.com/project/internal/mfl\"\n")
	if err := checkMFL(root); err != nil {
		t.Fatal(err)
	}
	fixtureWrite(t, root, "internal/leaguemigration/writer.go",
		"package leaguemigration\nimport _ \"example.com/project/internal/mfl\"\n")
	if err := checkMFL(root); err == nil || !strings.Contains(err.Error(),
		"internal/leaguemigration imports fenced internal/mfl") {
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
