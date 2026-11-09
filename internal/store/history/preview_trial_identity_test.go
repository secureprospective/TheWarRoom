package history

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// proveTrialIdentityIgnored: the owner readings a preview compares never read the trial entry's Principal
// or Revision, so the one PreviewTrial (which sets both) reads exactly as the bare trial the readings had
// before it (neither set), and as trials naming the previewer or a stranger at another revision.
// proveStorePreview runs it for every built preview, so it covers every reading kind those tests reach.
func proveTrialIdentityIgnored(t *testing.T, s *Store, raw commissioner.RawCommand, principal string) {
	t.Helper()
	records, err := s.leagueRecords(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	facts, err := s.appendFactsFor(t.Context(), raw)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := commissioner.PreviewTrial(raw, records, s.now())
	if err != nil {
		t.Fatalf("%s: build the preview trial: %v", raw.Command, err)
	}
	trials := map[string][]commissioner.Record{"preview trial": shared}
	for _, id := range []struct {
		name, principal string
		revision        int64
	}{
		{"bare entry", "", 0},
		{"the previewer, another revision", principal, raw.Expected + 41},
		{"a stranger", "someone.else", raw.Expected + 1},
	} {
		trial := slices.Clone(shared)
		last := trial[len(trial)-1]
		last.Principal, last.Receipt.Revision = id.principal, id.revision
		trial[len(trial)-1] = last
		trials[id.name] = commissioner.WithReplayCache(trial)
	}
	var want string
	for _, name := range []string{"bare entry", "preview trial", "the previewer, another revision", "a stranger"} {
		readings, err := s.previewReadings(t.Context(), raw, principal, trials[name], facts)
		body, jerr := json.Marshal([]any{readings, fmt.Sprint(err)})
		if jerr != nil {
			t.Fatal(jerr)
		}
		if want == "" {
			want = string(body)
		} else if string(body) != want {
			t.Fatalf("%s: the %s reads differently from the bare entry:\n got %s\nwant %s", raw.Command, name, body, want)
		}
	}
}

// TestPreviewReadingsIgnoreTheTrialIdentity covers the reading kinds no proveStorePreview caller reaches: a
// lineup set read by its owner and by the commissioner who created the league (the trial's principal).
// It must see readings at all, or it proves nothing.
func TestPreviewReadingsIgnoreTheTrialIdentity(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	f.mustSet(t, f.a, f.legal(f.a))
	for _, starters := range [][]string{f.legal(f.a), f.legal(f.a)[:20]} {
		raw := f.command(t, commissioner.LineupSet, f.a, 1, starters)
		for _, principal := range []string{f.ownerOfA, f.creatorID} {
			proveTrialIdentityIgnored(t, f.s, raw, principal)
		}
		records, err := f.s.leagueRecords(t.Context(), raw.League)
		if err != nil {
			t.Fatal(err)
		}
		facts, err := f.s.appendFactsFor(t.Context(), raw)
		if err != nil {
			t.Fatal(err)
		}
		trial, err := commissioner.PreviewTrial(raw, records, f.at)
		if err != nil {
			t.Fatal(err)
		}
		readings, err := f.s.previewReadings(t.Context(), raw, f.ownerOfA, trial, facts)
		if err != nil || len(readings) < 2 {
			t.Fatalf("the lineup preview read nothing to compare: %v %v", readings, err)
		}
	}
}
