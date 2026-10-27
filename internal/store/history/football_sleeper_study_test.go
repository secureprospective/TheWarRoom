package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// TestSleeperUnknownIDReport tracks unresolved nonzero rows, not guessed player eligibility.
// The inputs contain no fantasy rosters: roster membership is unknown, not zero.
func TestSleeperUnknownIDReport(t *testing.T) {
	crosswalkDir, sleeperDir := os.Getenv("TWR_STUDY_CACHE"), os.Getenv("TWR_SLEEPER_STUDY_CACHE")
	if crosswalkDir == "" || sleeperDir == "" {
		t.Skip("set TWR_STUDY_CACHE and TWR_SLEEPER_STUDY_CACHE to the p5/p6 caches")
	}
	known := make(map[string]bool)
	for _, row := range studyCSV(t, filepath.Join(crosswalkDir, "db_playerids.csv")).rows {
		if row["sleeper_id"] != "" && row["sleeper_id"] != "NA" {
			known[row["sleeper_id"]] = true
		}
	}
	for _, tc := range []struct{ season, want int }{{2023, 3233}, {2024, 3347}, {2025, 3496}} {
		nonzero, ids := 0, make(map[string]bool)
		for week := 1; week <= 13; week++ {
			rows := studySleeperRows(t, filepath.Join(sleeperDir, fmt.Sprintf("sleeper_%d_%d.json", tc.season, week)))
			for id, raw := range rows {
				if !sleeperPlayerID(id) || known[id] {
					continue
				}
				line, err := raw.Convert()
				if err != nil {
					t.Fatal(err)
				}
				for _, quantity := range line.Quantities {
					if quantity != 0 {
						nonzero++
						ids[id] = true
						break
					}
				}
			}
		}
		t.Logf("season=%d unknown-ID nonzero rows=%d distinct IDs=%d rostered share=unknown (no roster evidence)", tc.season, nonzero, len(ids))
		if nonzero != tc.want {
			t.Errorf("%d unknown nonzero rows=%d, want %d", tc.season, nonzero, tc.want)
		}
	}
}

func studySleeperRows(t *testing.T, path string) map[string]source.RawSleeperStats {
	t.Helper()
	body, err := os.ReadFile(path) //nolint:gosec // operator-selected opt-in study cache
	if err != nil {
		t.Fatal(err)
	}
	var rows map[string]source.RawSleeperStats
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatal(err)
	}
	for id, raw := range rows {
		if err := raw.Validate(); err != nil {
			t.Fatalf("%s %s: %v", path, id, err)
		}
	}
	return rows
}
