package freeagents

import (
	"strings"
	"testing"
)

// The bodies follow MFL's documented JSON shape for TYPE=freeAgents; no live body was captured.
const (
	manyBody   = `{"version":"1.0","freeAgents":{"leagueUnit":{"unit":"LEAGUE","player":[{"id":"0531"},{"id":"13593"},{"id":"0531"}]}}}`
	oneBody    = `{"freeAgents":{"leagueUnit":{"unit":"LEAGUE","player":{"id":"9999"}}}}`
	unitsBody  = `{"freeAgents":{"leagueUnit":[{"unit":"A","player":[{"id":"1"}]},{"unit":"B","player":[{"id":"2"},{"id":"1"}]}]}}`
	emptyBody  = `{"freeAgents":{"leagueUnit":{"unit":"LEAGUE"}}}`
	noneBody   = `{"freeAgents":{}}`
	errorBody  = `{"error":{"$t":"API requires valid credentials"}}`
	badIDBody  = `{"freeAgents":{"leagueUnit":{"player":[{"id":"x1"}]}}}`
	otherReply = `{"version":"1.0","rosters":{}}`
)

func ids(t *testing.T, body string) []string {
	t.Helper()
	raw, err := Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	typed, err := ToPlayerIDs(raw)
	if err != nil {
		t.Fatalf("ToPlayerIDs: %v", err)
	}
	out := []string{}
	for _, id := range typed {
		out = append(out, id.String())
	}
	return out
}

func TestParseReadsEveryShapeMFLSends(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		want string
	}{
		"many, leading zeros kept, repeat dropped": {manyBody, "0531,13593"},
		"one player as a bare object":              {oneBody, "9999"},
		"two units share a player":                 {unitsBody, "0001,0002"},
		"a unit with no players":                   {emptyBody, ""},
		"a section with no unit":                   {noneBody, ""},
	} {
		if got := strings.Join(ids(t, tc.body), ","); got != tc.want {
			t.Errorf("%s: ids = %q, want %q", name, got, tc.want)
		}
	}
}

func TestParseRefusesWhatIsNotAPool(t *testing.T) {
	for name, body := range map[string]string{"MFL error envelope": errorBody, "a bad id": badIDBody,
		"another export's reply": otherReply, "not JSON": "<html>", "empty body": ""} {
		if raw, err := Parse([]byte(body)); err == nil {
			t.Errorf("%s: Parse returned %v", name, raw)
		}
	}
}
