package main

import (
	"strings"
	"testing"
)

func TestReferenceRestoreReportsMissingOwnedPlayer(t *testing.T) {
	h := newTwoFileHost(t)
	lacks, err := referenceLacks(t.Context(), referencePath(h.dir+"/server.db"), referenceNeed{
		players: []string{"player." + strings.Repeat("f", 32)},
	})
	if err != nil || lacks != "it lacks 1 player(s) the leagues own" {
		t.Fatalf("missing player not reported: %q (%v)", lacks, err)
	}
}
