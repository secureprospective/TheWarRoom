package playoffs

import (
	"os"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestParseRealBrackets(t *testing.T) {
	list, err := ParseList(fixture(t, "playoffBrackets-2025.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[1].ID != 4 || list[1].Name != "AFC Conference" || list[1].StartWeek != 14 || list[1].Teams != 7 ||
		list[0].Title != "2025 Super Bowl Champion" {
		t.Fatalf("brackets = %+v", list)
	}
	afc, err := ParseBracket(4, fixture(t, "playoffBracket_4-2025.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := leaguefeed.BracketGame{ID: 1, Week: 14,
		Home: leaguefeed.BracketSide{Franchise: "0015", Seed: 2, Scored: true, Points: 27668},
		Away: leaguefeed.BracketSide{Franchise: "0005", Seed: 7, Scored: true, Points: 17981}}
	if len(afc) != 6 || afc[0] != want || afc[5].Week != 16 || afc[5].Home.WinnerOf != 5 || afc[4].Home.Seed != 1 {
		t.Fatalf("AFC = %+v", afc)
	}
	sb, err := ParseBracket(1, fixture(t, "playoffBracket_1-2025.json"))
	if err != nil || len(sb) != 1 || sb[0].Away.FromBracket != 4 || sb[0].Away.WinnerOf != 6 || sb[0].Away.Points != 30785 {
		t.Fatalf("Super Bowl = %+v %v", sb, err)
	}
}

func TestParseUnseededBracket(t *testing.T) {
	afc, err := ParseBracket(4, fixture(t, "playoffBracket_4-2026.json"))
	if err != nil {
		t.Fatal(err)
	}
	if afc[0].Home.Franchise != "" || afc[0].Home.Seed != 2 || afc[0].Home.Scored {
		t.Fatalf("unseeded = %+v", afc[0])
	}
}

func TestParseRejectsMalformedBrackets(t *testing.T) {
	if _, err := ParseBracket(5, fixture(t, "playoffBracket_4-2025.json")); err == nil {
		t.Fatal("accepted the wrong bracket")
	}
	for name, body := range map[string]string{
		"empty side": `{"playoffBracket":{"bracket_id":"1","playoffRound":{"week":"17","playoffGame":{"game_id":"1","home":{},"away":{"seed":"1"}}}}}`,
		"bad points": `{"playoffBracket":{"bracket_id":"1","playoffRound":{"week":"17","playoffGame":{"game_id":"1","home":{"seed":"2","points":"x"},"away":{"seed":"1"}}}}}`,
		"bad team":   `{"playoffBracket":{"bracket_id":"1","playoffRound":{"week":"17","playoffGame":{"game_id":"1","home":{"franchise_id":"15"},"away":{"seed":"1"}}}}}`,
		"bad week":   `{"playoffBracket":{"bracket_id":"1","playoffRound":{"week":"x","playoffGame":[]}}}`,
	} {
		if _, err := ParseBracket(1, []byte(body)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := ParseList([]byte(`{"playoffBrackets":{"playoffBracket":{"id":"x","startWeek":"14","teamsInvolved":"7"}}}`)); err == nil {
		t.Fatal("accepted a bracket without an id")
	}
}
