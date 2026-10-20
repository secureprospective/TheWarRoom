package personal

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	personalstore "github.com/secureprospective/TheWarRoom/internal/store/personal"
)

func TestBaitUsesHeldNamesAndKeepsUnknownPlayers(t *testing.T) {
	t.Parallel()
	names, franchises := Names(snapshot.Snapshot{})
	names["0042"], franchises["0001"] = "Player", "Owner"
	rows := BaitRows([]ingestion.RawTradeBait{{Give: "0042,0999,", Franchise: "0001", Want: "Picks"}}, names, franchises)
	if len(rows) != 1 || !reflect.DeepEqual(rows[0].Cells, []string{"Owner", "Player\nName unavailable", "Picks"}) {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestPlayersUseHeldStatusAndDeduplicate(t *testing.T) {
	t.Parallel()
	entries := []personalstore.Entry{{Entity: "player", ID: "0042"}, {Entity: "franchise", ID: "0001"}, {Entity: "player", ID: "0042"}, {Entity: "player", ID: "0999"}}
	lines := func(id string) []string { return []string{id} }
	got := Players(entries, true, map[string]string{"0042": "IR"}, "2026-10-10T12:00:00Z", time.UTC, lines)
	if len(got) != 2 || got[0].Status != "IR · as of Oct 10, 2026, 12:00 PM" || got[1].Status != "No injury designation · as of Oct 10, 2026, 12:00 PM" || got[0].Market[0] != "0042" {
		t.Fatalf("players = %+v", got)
	}
	got = Players(entries, false, nil, "", time.UTC, lines)
	if got[0].Status != "NFL status unavailable" {
		t.Fatal(got)
	}
}
