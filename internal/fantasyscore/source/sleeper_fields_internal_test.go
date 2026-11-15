package source

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestLiveSleeperEveryMappedFieldIsALeagueStat(t *testing.T) {
	for key, id := range sleeperFieldMap() {
		t.Run(key, func(t *testing.T) {
			if _, ok := leaguerules.StatByID(id); !ok {
				t.Fatalf("%s maps to unknown stat %s", key, id)
			}
			line, err := (RawLiveSleeperStats{key: json.Number("1")}).Convert()
			if err != nil || line.Quantities[id] != 10000 {
				t.Fatalf("%s: %+v %v", key, line, err)
			}
		})
	}
}
