package envelope

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestReasonNotesNeverJoinFullStopToSemicolon(t *testing.T) {
	if got := reasonNote([]string{"First.", "Second.", "", "Third"}); got != "First. Second. Third" {
		t.Fatal(got)
	}
	got := (ReleaseCheck{Offseason: false}).Evaluate(Spec{Intent: BuyoutIntent}, snapshot.Snapshot{})
	if strings.Contains(got.Note, ".;") {
		t.Fatal(got)
	}
}

func TestUnmappedNeverLeavesDraftOrBlocked(t *testing.T) {
	e, at, _ := testEnvelope(t)
	e.spec.Target = Target{Kind: Unmapped}
	for _, s := range []State{Draft, Blocked} {
		e.state = s
		if _, err := e.move(at, ChecksPass, ""); err == nil {
			t.Fatal("unmapped escaped", s)
		}
		if got, err := e.move(at, ChecksBlock, ""); err != nil || got.State() != Blocked {
			t.Fatal("unmapped could not block", s, err)
		}
	}
}
