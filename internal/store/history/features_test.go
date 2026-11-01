package history

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"strings"
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

func TestFeatureReadIndexesBothSourceAndMeasure(t *testing.T) {
	s, _, _ := newStore(t)
	query := featureReadSQL(1)
	rows, err := s.pools.Read().QueryContext(t.Context(), "EXPLAIN QUERY PLAN "+query,
		formatTime(t0()), 2025, "outcome.sacks")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	indexed := false
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(detail, "INDEX") && strings.Contains(detail, "source=? AND measure=?") {
			indexed = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !indexed {
		t.Fatal("feature join scans every source field instead of indexing source and measure")
	}
}

func TestStreamedFeaturesMatchSQLSelection(t *testing.T) {
	_, _, pools := newStore(t)
	s, c, _ := reopen(t, pools, registry(t, betaField))
	for _, value := range []string{"2", "8", "0"} {
		for _, batch := range []measures.Batch{
			solo("alpha", "Solo", "13604", value),
			solo("beta", "soloTackles", "13604", "99"),
			solo("beta", "soloTackles", "13605", value),
			{Source: "alpha", Facts: []measures.Fact{
				{IDType: measures.IDTypeMFL, ID: "13604", Season: 2025, Week: 3, Field: "Sacks", Raw: value},
				{IDType: measures.IDTypeMFL, ID: "13604", Season: 2025, Week: 3, Field: "Status", Raw: "Active"},
			}},
		} {
			if _, err := s.Ingest(t.Context(), batch); err != nil {
				t.Fatal(err)
			}
			c.now = c.now.Add(time.Hour)
		}
	}
	for _, season := range []int{2024, 2025} {
		for _, at := range []time.Time{t0().Add(-time.Second), t0(), t0().Add(5 * time.Hour), c.now} {
			q := FeatureQuery{AsOf: at, Season: season, Measures: []string{
				"outcome.tackles_solo", "outcome.sacks", "availability.game_status"}}
			got, err := s.Features(t.Context(), q)
			if err != nil {
				t.Fatal(err)
			}
			if want := sqlFeatures(t, s, q); !reflect.DeepEqual(got, want) {
				t.Fatalf("season %d as-of %s: streamed %+v; SQL %+v", season, at, got, want)
			}
		}
	}
}

func sqlFeatures(t *testing.T, s *Store, q FeatureQuery) []Feature {
	t.Helper()
	where := " AND season = ? AND measure IN (?" + strings.Repeat(", ?", len(q.Measures)-1) + ")"
	args := []any{formatTime(q.AsOf), q.Season}
	for _, m := range q.Measures {
		args = append(args, m)
	}
	rows, err := s.pools.Read().QueryContext(t.Context(), featuresSQL("?", where)+" ORDER BY player_id, week, measure", args...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []Feature
	for rows.Next() {
		var f Feature
		var stamp string
		var value sql.NullFloat64
		var text sql.NullString
		if err := rows.Scan(&f.PlayerID, &f.Season, &f.Week, &f.Measure, &f.Source, &stamp, &value, &text); err != nil {
			t.Fatal(err)
		}
		f.AsOf, err = parseTime(stamp)
		if err != nil {
			t.Fatal(err)
		}
		f.Value, f.Text = value.Float64, text.String
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
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
