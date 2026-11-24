package source

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestParseSleeperProjectionSample(t *testing.T) {
	file, err := os.Open("testdata/sleeper-proj-2026-w12-sample.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		ID     string `json:"player_id"`
		Team   string `json:"team"`
		Player struct {
			Position string `json:"position"`
		} `json:"player"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatal(err)
	}
	ids := make(map[string]string, len(rows))
	for _, row := range rows {
		if row.ID != row.Team {
			ids[row.ID] = "canonical-" + row.ID
		}
	}
	if len(ids) > 0 {
		delete(ids, rows[0].ID)
	}
	projections, counts, unmapped, err := ParseSleeperProjections(body, ids)
	if err != nil {
		t.Fatal(err)
	}
	if unmapped < 1 {
		t.Fatalf("unmapped=%d, want unsupported and unresolved rows counted", unmapped)
	}
	for _, position := range []leaguerules.Position{"QB", "RB", "WR", "TE", "PK", "DT", "DE", "LB", "CB", "DEF"} {
		if counts[position] == 0 {
			t.Errorf("no %s projection in fixture; counts=%v", position, counts)
		}
	}
	if len(projections) == 0 {
		t.Fatal("no projections parsed")
	}
}
