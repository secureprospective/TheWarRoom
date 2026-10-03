package domain

import "testing"

func TestFranchiseLabel(t *testing.T) {
	names := map[string]string{"0001": "Alpha", "0002": ""}
	for _, c := range []struct {
		names map[string]string
		id    string
		want  string
	}{
		{names, "0001", "Alpha"},
		{names, "0002", "Franchise 0002"}, // blank name
		{names, "0099", "Franchise 0099"}, // unmapped
		{nil, "0001", "Franchise 0001"},   // no rulebook loaded
	} {
		if got := FranchiseLabel(c.names, c.id); got != c.want {
			t.Errorf("FranchiseLabel(%v, %q) = %q, want %q", c.names, c.id, got, c.want)
		}
	}
}
