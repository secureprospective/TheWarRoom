package composition

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestSourcesFactsSerializerKeepsTryLockAndCallbackSpan(t *testing.T) {
	s := coretest.New(t).Build()
	s.FactsMu.Lock()
	called := false
	msg, err := sourcesHost{State: s}.SerializeSourcesCompile(func() (string, error) { called = true; return "", nil })
	s.FactsMu.Unlock()
	if called || err != nil || msg != "The checked facts are already compiling." {
		t.Fatal(msg, err, called)
	}
	_, err = sourcesHost{State: s}.SerializeSourcesCompile(func() (string, error) {
		if s.FactsMu.TryLock() {
			s.FactsMu.Unlock()
			t.Error("compile callback lost original lock span")
		}
		return "compiled", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !s.FactsMu.TryLock() {
		t.Fatal("compile did not release serializer")
	}
	s.FactsMu.Unlock()
}

func TestSourcesSharedReadsDetached(t *testing.T) {
	s := coretest.New(t).Build()
	lookup, err := normalize.NewLookup([]players.RawPlayer{{ID: "0042", Name: "Held", Position: "QB"}})
	if err != nil {
		t.Fatal(err)
	}
	s.Lookup, s.HasLookup = lookup, true
	_, err = s.League.Replace(t.Context(), state.MirrorSnapshot{Season: 2026,
		Players: []state.MirrorPlayer{{MFLID: "0042", FranchiseID: "0001", RosterStatus: domain.RosterActive}}})
	if err != nil {
		t.Fatal(err)
	}
	positions, rostered := (sourcesHost{State: s}).SourcesPositions(), s.SourcesRostered()
	if positions["0042"] != "QB" || rostered["0042"] != "0001" {
		t.Fatal(positions, rostered)
	}
	positions["0042"], rostered["0042"] = "mutated", "mutated"
	if (sourcesHost{State: s}).SourcesPositions()["0042"] != "QB" || s.SourcesRostered()["0042"] != "0001" {
		t.Fatal("shared reads alias state")
	}
}
