package history

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/model"
)

func exportFixtureStore(t *testing.T) *Store {
	t.Helper()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	// Ids keep their leading zeros; the best dynasty value is the highest id and the worst projection.
	run := NewModelRun{Season: 2026, AsOf: time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC),
		Params: ParamSet{Params: map[string]float64{"a": 1}, Measures: []string{}}, Engine: "test", InputsHash: "in-1",
		Scores: []ModelScore{
			{MFLID: "0800", Position: "QB", Value: model.Value{DynastyPPG: 5}},
			{MFLID: "9826", Position: "WR", Value: model.Value{DynastyPPG: 30}},
			{MFLID: "1002", Position: "RB", Value: model.Value{DynastyPPG: 12}},
		}}
	if _, _, err := s.WriteModelRun(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	return s
}

// A run exported from a desktop-shaped store and imported into an empty hosted-shaped store (reference
// attached) must reorder the draft pool by dynasty_ppg, with the ids the pool holds.
func TestModelRunExportImportRanksPoolByDynastyPPG(t *testing.T) {
	t.Parallel()
	file, err := exportFixtureStore(t).ExportModelRun(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ModelRunFile
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	dest, _ := hostedStatStore(t, filepath.Join(t.TempDir(), "league.db"))
	rank := func() []string {
		got, err := DynastyDraftRanking{Store: dest}.Rank(t.Context(), "native.x", 2026, []string{"0800", "1002", "9826"})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	run, held, err := dest.ImportModelRun(t.Context(), decoded)
	if err != nil || held {
		t.Fatalf("import = %+v held=%v err=%v", run, held, err)
	}
	if got := rank(); !slices.Equal(got, []string{"9826", "1002", "0800"}) {
		t.Fatalf("rank after import = %v, want dynasty_ppg order", got)
	}
}

func TestModelRunImportTwiceIsNoOp(t *testing.T) {
	t.Parallel()
	file, err := exportFixtureStore(t).ExportModelRun(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	dest, _ := hostedStatStore(t, filepath.Join(t.TempDir(), "league.db"))
	first, held, err := dest.ImportModelRun(t.Context(), file)
	if err != nil || held {
		t.Fatalf("first import held=%v err=%v", held, err)
	}
	second, held, err := dest.ImportModelRun(t.Context(), file)
	if err != nil || !held || second.ID != first.ID {
		t.Fatalf("second import = %+v held=%v err=%v, want the first run held", second, held, err)
	}
	var runs int
	if err := dest.pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM model_runs`).Scan(&runs); err != nil || runs != 1 {
		t.Fatalf("model_runs = %d err=%v, want 1", runs, err)
	}
}
