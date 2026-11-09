package leagueservice

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestJoinNamePrefersTheLeaguesOwnName(t *testing.T) {
	imported := commissioner.NativeFranchise{Name: "Gridiron Gang", Abbreviation: "GG"}
	cases := []struct{ name, given, want string }{
		{"placeholder gives way to the imported name", "Franchise 3", "Gridiron Gang"},
		{"blank gives way to the imported name", "", "Gridiron Gang"},
		{"a rename wins over the import", "Renamed Club", "Renamed Club"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, _ := joinName(2, c.given, "X", imported); got != c.want {
				t.Fatalf("joinName(%q) = %q, want %q", c.given, got, c.want)
			}
		})
	}
	if got, _ := joinName(2, "Franchise 3", "F3", commissioner.NativeFranchise{}); got != "Franchise 3" {
		t.Fatalf("no import: got %q, want the created name", got)
	}
}
