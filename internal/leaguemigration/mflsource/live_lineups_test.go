package mflsource_test

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func ids(t *testing.T, raw ...string) []playerid.PlayerID {
	t.Helper()
	out := make([]playerid.PlayerID, 0, len(raw))
	for _, r := range raw {
		id, err := playerid.New(r)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	return out
}

// A week in progress is not scored, so without LiveWeekResult the import reads no lineup at all
// for the week being played.
func TestLiveWeekResultRecordsTheWeekInPlay(t *testing.T) {
	t.Parallel()
	live := leaguefeed.Lineups{Week: 8,
		Matchups: []leaguefeed.Matchup{{Home: "0001", Away: "0002"}, {Home: "0003", Away: "0004"}},
		Franchises: []leaguefeed.Lineup{
			{Franchise: "0001", Starters: ids(t, "0099", "1000"), NonStarters: ids(t, "1001")},
			{Franchise: "0002", Starters: ids(t, "1002")},
			{Franchise: "0003", Starters: ids(t, "1003")},
			{Franchise: "0004"},
			{Franchise: "0005", Starters: ids(t, "1004")},
		}}
	got := mflsource.LiveWeekResult(live)
	if got.Week != 8 {
		t.Fatalf("week %d", got.Week)
	}
	if len(got.Games) != 1 || got.Games[0].Home.Franchise != "0001" || got.Games[0].Away.Franchise != "0002" {
		t.Fatalf("games %+v", got.Games)
	}
	if !got.Games[0].Home.Scored || len(got.Games[0].Home.Players) != 3 {
		t.Fatalf("home side %+v", got.Games[0].Home)
	}
	starters := 0
	for _, p := range got.Games[0].Home.Players {
		if p.Starter {
			starters++
		}
	}
	if starters != 2 {
		t.Fatalf("starters %d", starters)
	}
	// 0004 set no lineup: it stays out, so it is still a missed lineup. 0003 (its opponent) and
	// the unpaired 0005 are both still recorded.
	others := map[string]bool{}
	for _, s := range got.Others {
		others[s.Franchise] = true
	}
	if !others["0003"] || !others["0005"] || others["0004"] || len(others) != 2 {
		t.Fatalf("others %+v", got.Others)
	}
}

func TestLiveWeekResultFeedsThePlayedLineupImport(t *testing.T) {
	t.Parallel()
	live := leaguefeed.Lineups{Week: 8, Matchups: []leaguefeed.Matchup{{Home: "0001", Away: "0002"}},
		Franchises: []leaguefeed.Lineup{
			{Franchise: "0001", Starters: ids(t, "0099")},
			{Franchise: "0002", Starters: ids(t, "1002")},
		}}
	got, err := mflsource.Lineups([]leaguefeed.WeekResult{mflsource.LiveWeekResult(live)}, 2026,
		map[string]string{"0001": "franchise.01", "0002": "franchise.02"},
		map[string]string{"0099": "0099", "1002": "1002"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("lineups %+v", got)
	}
	for _, lineup := range got {
		if lineup.Week != 8 || lineup.Season != 2026 || len(lineup.Starters) != 1 {
			t.Fatalf("lineup %+v", lineup)
		}
	}
}
