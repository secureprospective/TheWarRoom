package contracts

import (
	"strings"
	"testing"
)

func TestParseScorePeriod(t *testing.T) {
	t.Parallel()
	body := []byte(`{"playerScores":{"playerScore":[{"id":"0042","week":"YTD","score":"12.5"},{"id":"0042","week":"2","score":"4"}]}}`)
	for _, period := range []string{"YTD", "2"} {
		rows, err := ParseScorePeriod(body, period)
		if err != nil || len(rows) != 1 || rows[0].ID != "0042" || rows[0].Week != period {
			t.Fatalf("%s: %+v %v", period, rows, err)
		}
	}
	if rows, err := ParseScorePeriod(body, "3"); rows != nil || err == nil || err.Error() != "player scores: period 3 not held" {
		t.Fatalf("absent: %v %v", rows, err)
	}
	if rows, err := ParseScorePeriod([]byte(`{`), "YTD"); rows != nil || err == nil || !strings.HasPrefix(err.Error(), "player scores: parse: ") {
		t.Fatalf("bad body: %v %v", rows, err)
	}
}
