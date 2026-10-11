package weeklyresults

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func season(t *testing.T, year string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/weeklyResults-" + year + ".json.gz")
	if err != nil {
		t.Fatal(err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestParseFinishedSeason(t *testing.T) {
	weeks, err := Parse(season(t, "2025"))
	if err != nil {
		t.Fatal(err)
	}
	if len(weeks) != 17 || weeks[0].Week != 1 || len(weeks[0].Games) != 16 || !weeks[0].Games[0].Regular {
		t.Fatalf("weeks = %d, week 1 games = %d", len(weeks), len(weeks[0].Games))
	}
	g := weeks[0].Games[0]
	if g.Home.Franchise != "0002" || g.Away.Franchise != "0001" || g.Away.Points != 30440 || g.Home.Points != 16650 ||
		g.Away.Result != "W" || g.Away.Best != 32980 {
		t.Fatalf("week 1 game = %+v", g)
	}
	if len(g.Away.Players) == 0 || !g.Away.Players[0].Starter || g.Away.Players[0].Points != 3450 || g.Away.Players[0].ID.String() != "15238" {
		t.Fatalf("player = %+v", g.Away.Players[0])
	}
	w14 := weeks[13]
	if w14.Week != 14 || len(w14.Games) != 6 || w14.Games[0].Regular || len(w14.Others) != 20 || !w14.Others[0].Scored {
		t.Fatalf("week 14 = %d games, %d others", len(w14.Games), len(w14.Others))
	}
}

func TestParseSeasonInProgressKeepsUnscoredWeeks(t *testing.T) {
	weeks, err := Parse(season(t, "2026"))
	if err != nil {
		t.Fatal(err)
	}
	if !weeks[2].Games[0].Home.Scored || weeks[3].Games[0].Home.Scored || weeks[3].Games[0].Home.Result != "" {
		t.Fatal("week 3 is scored and week 4 is not")
	}
	if len(weeks[3].Games[0].Away.Players) == 0 || weeks[3].Games[0].Away.Players[0].Points != 0 {
		t.Fatal("a set lineup is listed before the week is scored")
	}
}

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
