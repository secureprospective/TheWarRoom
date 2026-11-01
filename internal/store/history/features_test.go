package history

import (
	"testing"
	"time"

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
