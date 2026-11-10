package freeagents

import (
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

func TestParseRefusesWhatIsNotAPool(t *testing.T) {
	for name, body := range map[string]string{"MFL error envelope": errorBody, "a bad id": badIDBody,
		"another export's reply": otherReply, "not JSON": "<html>", "empty body": ""} {
		if raw, err := Parse([]byte(body)); err == nil {
			t.Errorf("%s: Parse returned %v", name, raw)
		}
	}
}
