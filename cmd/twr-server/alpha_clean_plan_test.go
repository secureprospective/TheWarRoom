package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAlphaCleanPlanDryRunLeavesThrowawayDatabaseUnchanged(t *testing.T) {
	fixture := newOneFileServer(t)
	before := digestFile(t, fixture.path)
	script, err := filepath.Abs("../../deploy/server/alpha-clean-plan.sh")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "sh", script, "--dry-run", "--state-db", fixture.path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run dry-run: %v\n%s", err, output)
	}
	for _, want := range []string{"PLAN ONLY", "commish.seat.remove", "twr-reset-sign-ins", "Sign-in backups", "No --apply mode"} {
		if !strings.Contains(string(output), want) {
			t.Errorf("plan output missing %q:\n%s", want, output)
		}
	}
	if after := digestFile(t, fixture.path); after != before {
		t.Fatal("dry-run changed the throwaway database")
	}

	apply := exec.CommandContext(t.Context(), "sh", script, "--apply", "--state-db", fixture.path)
	if output, err := apply.CombinedOutput(); err == nil || !strings.Contains(string(output), "unsupported option") {
		t.Fatalf("apply mode was not refused: err=%v output=%s", err, output)
	}
}
