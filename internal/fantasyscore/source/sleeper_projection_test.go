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

func TestParseSleeperProjectionsKeepsGoodRowsAndIDPLabels(t *testing.T) {
	body := []byte(`[
 {"player_id":"1","team":"KC","week":12,"sport":"nfl","season_type":"regular","player":{"position":"QB"},"stats":{"pass_yd":250.5}},
 {"player_id":"","team":"KC","week":12,"sport":"nfl","season_type":"regular","player":{"position":"QB"},"stats":{"pass_yd":1}},
 {"player_id":"3","team":"KC","week":12,"sport":"nfl","season_type":"regular","player":{"position":"RB"}},
 {"player_id":"4","team":"KC","week":12,"sport":"nfl","season_type":"regular","player":{"position":"DB"},"stats":{"idp_tkl_solo":4.2}}
]`)
	projections, _, unmapped, err := ParseSleeperProjections(body, map[string]string{"1": "c1", "3": "c3", "4": "c4"})
	if err != nil {
		t.Fatalf("one broken row lost the snapshot: %v", err)
	}
	if len(projections) != 2 || unmapped != 2 {
		t.Fatalf("projections=%d unmapped=%d, want the QB and the DB kept and the two broken rows counted", len(projections), unmapped)
	}
	if projections[1].CanonicalID != "c4" || projections[1].Position != "" {
		t.Fatalf("DB row = %+v, want kept with its position left to the canonical player", projections[1])
	}
}
