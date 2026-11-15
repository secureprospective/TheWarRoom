package market

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/appvalue"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// The model calls a kicker K and the lineup rules call him PK. The lens must use the lineup name,
// or kickers get no replacement level and read at full value.
func TestLensPlayersNameKickersAsTheLineupDoes(t *testing.T) {
	values := appvalue.Reading{Players: []appvalue.Player{
		{Player: "1", Position: "K", Available: true}, {Player: "2", Position: "S", Available: true}}}
	got := map[string]string{}
	for _, a := range lensPlayers(values, &commissioner.RawNativeSnapshot{}, 2026) {
		got[a.Token] = a.Position
	}
	if got["1"] != "PK" || got["2"] != "S" {
		t.Fatalf("positions = %v, want kicker as PK and safety unchanged", got)
	}
}
