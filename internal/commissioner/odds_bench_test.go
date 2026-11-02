package commissioner

import (
	"fmt"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func oddsFixture32() (NativeStandings, map[string]float64, competition.FieldInput) {
	s := NativeStandings{criteria: []leaguerules.Criterion{leaguerules.WinPercentage, leaguerules.Points}, counted: seasonGames{points: map[string]leaguerules.Decimal{}, allPlay: map[string]competition.Tally{}}}
	field := competition.FieldInput{Teams: map[string]competition.FieldTeam{}, Conferences: []string{"c"}, Rules: competition.PlayoffRules{PerConference: 8, WildCards: 8}}
	expected := map[string]float64{}
	for i := range 32 {
		id := fmt.Sprintf("%02d", i)
		s.teams = append(s.teams, competition.RankTeam{ID: id, Division: "d", Conference: "c"})
		field.Teams[id] = competition.FieldTeam{Division: "d", Conference: "c"}
		expected[id] = float64(90 + i)
	}
	for w := 1; w <= 14; w++ {
		for i := 0; i < 32; i += 2 {
			s.counted.open = append(s.counted.open, OpenGame{Week: w, Home: fmt.Sprintf("%02d", i), Away: fmt.Sprintf("%02d", i+1)})
		}
	}
	return s, expected, field
}

func TestOddsGolden32(t *testing.T) {
	s, expected, field := oddsFixture32()
	made, words := playOutChunks(s, expected, field, 7)
	if words != "" {
		t.Fatal(words)
	}
	want := "map[00:360 01:405 02:367 03:421 04:411 05:424 06:359 07:478 08:362 09:441 10:390 11:501 12:419 13:526 14:418 15:548 16:464 17:556 18:490 19:540 20:479 21:600 22:552 23:574 24:517 25:598 26:604 27:618 28:616 29:680 30:629 31:653]"
	if got := fmt.Sprint(made); got != want {
		t.Fatalf("seeded odds changed: %s", got)
	}
}

func BenchmarkPlayOut32(b *testing.B) {
	s, expected, field := oddsFixture32()
	b.ResetTimer()
	for range b.N {
		_, words := playOutChunks(s, expected, field, 7)
		if words != "" {
			b.Fatal(words)
		}
	}
}
