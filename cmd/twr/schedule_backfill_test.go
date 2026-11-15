package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestScheduleBackfillRequiresExplicitDestination(t *testing.T) {
	t.Parallel()
	output := &bytes.Buffer{}
	e := env{out: output, in: strings.NewReader("")}
	root := e.root()
	root.SetArgs([]string{"schedule-backfill"})
	if err := root.ExecuteContext(t.Context()); err == nil || !strings.Contains(err.Error(), "--db") {
		t.Fatalf("missing explicit destination accepted: %v", err)
	}
	root = e.root()
	root.SetArgs([]string{"schedule-backfill", "--help"})
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "--principal") {
		t.Fatal("backfill omitted commissioner identity")
	}
}
