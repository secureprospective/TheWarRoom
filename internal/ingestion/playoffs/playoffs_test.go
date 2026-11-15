package playoffs

import (
	"os"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return body
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
