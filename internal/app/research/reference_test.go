package research

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type referenceFixture struct{ name, limit string }

func (r referenceFixture) FranchiseName(string) (string, bool) { return r.name, true }
func (r referenceFixture) StarterLimit(string) (string, bool)  { return r.limit, true }

func TestResearchUsesHeldReferencePort(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	in.Config = referenceFixture{name: "Held franchise", limit: "1"}
	in.Mirror = state.MirrorSnapshot{Players: []state.MirrorPlayer{{MFLID: "16162", FranchiseID: "0001"}}}
	v := researchView{Snapshot: in, scope: Scope{Group: "K"}}
	if got := v.player("16162"); got.Owner != "Held franchise" || got.FranchiseID != "0001" {
		t.Fatalf("reference directory not used: %+v", got)
	}
	if note := v.slotNote(); note != "" {
		t.Fatal(note)
	}
	in.Config = nil
	v.Snapshot = in
	if got := v.player("16162"); got.Owner != "Franchise 0001" {
		t.Fatalf("missing directory fallback changed: %+v", got)
	}
	if note := v.slotNote(); note != " This league has no K starter slots; MFL’s numbers are still shown." {
		t.Fatal(note)
	}
}
