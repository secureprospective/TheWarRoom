package report

import "testing"

func TestPluralNoun(t *testing.T) {
	for _, tc := range []struct {
		n    float64
		want string
	}{{0, "games"}, {1, "game"}, {1.5, "games"}, {2, "games"}} {
		if got := PluralNoun(tc.n, "game"); got != tc.want {
			t.Errorf("PluralNoun(%v, game) = %q, want %q", tc.n, got, tc.want)
		}
	}
}
