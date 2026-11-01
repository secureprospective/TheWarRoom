package history

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

func TestLatestFeatureKeepsValueAndTextFromNewestRow(t *testing.T) {
	s, c, _ := newStore(t)
	for _, value := range []struct{ sacks, status string }{{"8", "Out"}, {"2", "Active"}} {
		_, err := s.Ingest(t.Context(), measures.Batch{Source: "alpha", Facts: []measures.Fact{
			{IDType: measures.IDTypeMFL, ID: "13604", Season: 2025, Week: 3, Field: "Sacks", Raw: value.sacks},
			{IDType: measures.IDTypeMFL, ID: "13604", Season: 2025, Week: 3, Field: "Status", Raw: value.status},
		}})
		if err != nil {
			t.Fatal(err)
		}
		c.now = c.now.Add(time.Hour)
	}
	for _, tc := range []struct {
		at     time.Time
		sacks  float64
		status string
	}{{t0(), 8, "Out"}, {c.now, 2, "Active"}} {
		got, err := s.Features(t.Context(), FeatureQuery{AsOf: tc.at, Season: 2025, Measures: []string{"outcome.sacks", "availability.game_status"}})
		if err != nil || len(got) != 2 {
			t.Fatalf("features=%+v error=%v", got, err)
		}
		for _, f := range got {
			if f.Measure == "outcome.sacks" && f.Value != tc.sacks || f.Measure == "availability.game_status" && f.Text != tc.status {
				t.Fatalf("wrong latest row: %+v", f)
			}
		}
	}
}

func TestValueEvidenceSurvivesReopenAndFollowsChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	open := func() (*Store, *db.Pools) {
		t.Helper()
		pools, err := db.Open(context.Background(), path, SchemaVersion())
		if err != nil {
			t.Fatal(err)
		}
		s, _, _ := reopen(t, pools, registry(t, ""))
		return s, pools
	}
	evidence := func(s *Store) string {
		t.Helper()
		e, err := s.ValueEvidence(t.Context())
		if err != nil || e == "" {
			t.Fatalf("evidence %q: %v", e, err)
		}
		return e
	}
	s, pools := open()
	if _, err := s.Ingest(t.Context(), solo("alpha", "Solo", "13604", "2")); err != nil {
		t.Fatal(err)
	}
	first := evidence(s)
	_ = pools.Close()
	s, pools = open()
	defer func() { _ = pools.Close() }()
	if got := evidence(s); got != first {
		t.Fatal("unchanged data fingerprinted differently after reopening")
	}
	if _, err := s.Ingest(t.Context(), solo("alpha", "Solo", "13604", "2")); err != nil {
		t.Fatal(err)
	}
	if got := evidence(s); got != first {
		t.Fatal("a load that added nothing changed the fingerprint")
	}
	if _, err := s.Ingest(t.Context(), solo("alpha", "Solo", "13605", "1")); err != nil {
		t.Fatal(err)
	}
	added := evidence(s)
	if added == first {
		t.Fatal("added observation left the fingerprint unchanged")
	}
	if _, err := pools.Write().ExecContext(t.Context(), `UPDATE source_fields SET priority = priority + 1`); err != nil {
		t.Fatal(err)
	}
	if evidence(s) == added {
		t.Fatal("changed source priority left the fingerprint unchanged")
	}
}
