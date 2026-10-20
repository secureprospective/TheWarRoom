// Package seasontest supplies shared real-source season fixtures.
package seasontest

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playoffs"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

type Data struct {
	Weeks     []leaguefeed.WeekResult
	Brackets  []leaguefeed.Bracket
	Standings []leaguestandings.RawStanding
	Config    league.RawConfig
	Lookup    normalize.Lookup
}

func Sample(t *testing.T, feed, name string) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "..", "ingestion", feed, "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
func compressed(t *testing.T, feed, name string) string {
	t.Helper()
	zr, err := gzip.NewReader(strings.NewReader(Sample(t, feed, name)))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if err := zr.Close(); err != nil {
		t.Fatal(err)
	}
	return string(body)
}
func Load(t *testing.T, year string) Data {
	t.Helper()
	cfg, err := league.Parse([]byte(Sample(t, "league", "league-"+year+".json")),
		[]byte(Sample(t, "league", "rules-2025.json")))
	if err != nil {
		t.Fatal(err)
	}
	weeks, err := weeklyresults.Parse([]byte(compressed(t, "weeklyresults", "weeklyResults-"+year+".json.gz")))
	if err != nil {
		t.Fatal(err)
	}
	brackets, err := playoffs.ParseList([]byte(Sample(t, "playoffs", "playoffBrackets-"+year+".json")))
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range brackets {
		name := fmt.Sprintf("playoffBracket_%d-%s.json", b.ID, year)
		if brackets[i].Games, err = playoffs.ParseBracket(b.ID, []byte(Sample(t, "playoffs", name))); err != nil {
			t.Fatal(err)
		}
	}
	standings, err := leaguestandings.Parse([]byte(Sample(t, "leaguestandings", "leagueStandings-"+year+".json")))
	if err != nil {
		t.Fatal(err)
	}
	raws, err := players.Parse(context.Background(), []byte(compressed(t, "players", "players-2025.json.gz")))
	if err != nil {
		t.Fatal(err)
	}
	lookup, err := normalize.NewLookup(raws)
	if err != nil {
		t.Fatal(err)
	}
	return Data{Weeks: weeks, Brackets: brackets, Standings: standings, Config: cfg, Lookup: lookup}
}
func Power() season.PowerRankingsResult {
	return season.PowerRankingsResult{OK: true, View: season.ViewSeason, Weight: 4.0 / 7, WeeksScored: 3, ModelRunID: 3,
		AggMode: "lineup", Performance: "all-play", Rows: []season.PowerRow{
			{Rank: 1, FranchiseID: "0032", PowerScore: 1, RosterValue: 210.4, Results: 89.0 / 93, H2HW: 3, DeltaOK: true, RankDelta: 2},
			{Rank: 2, FranchiseID: "0001", PowerScore: 0.6, RosterValue: 205.1, Results: 0.7, H2HW: 3},
		}}
}
