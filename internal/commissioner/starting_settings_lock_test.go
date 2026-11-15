package commissioner

import (
	"strings"
	"testing"
)

// A league stood up from an MFL import without a recorded lock gets the same per-player lock as a new league.
func TestStartingDefaultsLockEachPlayerForImports(t *testing.T) {
	t.Parallel()
	for _, source := range []string{SourceMFLImport, SourceNative} {
		seed, err := startingDefaults(nil, source)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, s := range seed {
			if s.Field == "lineup.lock" {
				found = true
				if !strings.Contains(string(s.Value), "each_player_game") {
					t.Fatalf("%s lineup lock: %s", source, s.Value)
				}
			}
		}
		if !found {
			t.Fatalf("%s: no lineup lock supplied", source)
		}
	}
}
