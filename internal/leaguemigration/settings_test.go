package leaguemigration

import "testing"

func TestDefaultPresetMatchesTeamCount(t *testing.T) {
	t.Parallel()
	for count, want := range map[int]string{16: "mirror16", 32: "legacy_nfl", 12: "default32", 20: "default32"} {
		if got := DefaultPreset(count); got != want {
			t.Errorf("%d teams: %s, want %s", count, got, want)
		}
	}
	if _, err := Base("invented"); err == nil {
		t.Fatal("accepted an unknown settings bundle")
	}
}
