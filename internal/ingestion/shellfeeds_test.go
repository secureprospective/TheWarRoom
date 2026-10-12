package ingestion

import "testing"

func TestShellFeedBoundaries(t *testing.T) {
	for _, test := range []struct {
		name  string
		parse func([]byte) error
		good  string
		bad   []string
	}{
		{"calendar", func(b []byte) error { _, err := ParseCalendar(b); return err },
			`{"calendar":{"event":{"end_time":"","type":"DRAFT_START","happens":"","title":"","id":"8352818","start_time":"1784030400"}}}`,
			[]string{`{}`, `{"calendar":{"event":{"id":"1","type":"DRAFT","start_time":"oops"}}}`, `{"calendar":{"event":{"id":"1"}}}`}},
		{"bait", func(b []byte) error { _, err := ParseTradeBait(b); return err },
			`{"tradeBaits":{"tradeBait":[{"franchise_id":"0025","willGiveUp":"0816,15711","inExchangeFor":"Picks","timestamp":"1784759064"}]}}`,
			[]string{`{}`, `{"tradeBaits":{"tradeBait":{"franchise_id":"0025","willGiveUp":"bad"}}}`}},
		{"scores", func(b []byte) error { _, err := ParsePlayerScores(b); return err },
			`{"playerScores":{"week":"YTD","playerScore":{"id":"0816","score":"0.00"}}}`,
			[]string{`{}`, `{"playerScores":{"week":"YTD","playerScore":{"id":"0816","score":"NaN"}}}`,
				`{"playerScores":{"week":"0","playerScore":{"id":"0816","score":"3"}}}`}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.parse([]byte(test.good)); err != nil {
				t.Fatal(err)
			}
			for _, body := range append(test.bad, `{"error":"API failure"}`) {
				if err := test.parse([]byte(body)); err == nil {
					t.Fatalf("accepted %s", body)
				}
			}
		})
	}
}
