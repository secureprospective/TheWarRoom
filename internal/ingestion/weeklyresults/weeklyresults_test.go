package weeklyresults

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestParseRejectsMalformedWeeks(t *testing.T) {
	game := func(a, b string) string {
		return `{"allWeeklyResults":{"weeklyResults":{"week":"1","matchup":{"regularSeason":"1","franchise":[` + a + `,` + b + `]}}}}`
	}
	home := `{"id":"0001","isHome":"1","score":"10","result":"W"}`
	for name, body := range map[string]string{
		"two home sides": game(home, `{"id":"0002","isHome":"1","score":"9","result":"L"}`),
		"half scored":    game(home, `{"id":"0002","isHome":"0"}`),
		"bad result":     game(home, `{"id":"0002","isHome":"0","score":"9","result":"X"}`),
		"bad score":      game(home, `{"id":"0002","isHome":"0","score":"NaN","result":"L"}`),
		"bad franchise":  game(home, `{"id":"2","isHome":"0","score":"9","result":"L"}`),
		"bad player":     game(home, `{"id":"0002","isHome":"0","score":"9","result":"L","player":{"id":"x","status":"starter"}}`),
		"bad set lineup": game(`{"id":"0001","isHome":"1"}`, `{"id":"0002","isHome":"0","player":{"id":"x","status":"starter"}}`),
		"no weeks":       `{"allWeeklyResults":{}}`,
		"bad week":       strings.Replace(game(home, `{"id":"0002","isHome":"0"}`), `"week":"1"`, `"week":"0"`, 1),
	} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	weeks, err := Parse([]byte(game(home, `{"id":"0002","isHome":"0","score":"-1.5","result":"L"}`)))
	if err != nil || weeks[0].Games[0].Away.Points != leaguefeed.Points(-150) {
		t.Fatal(weeks, err)
	}
}
