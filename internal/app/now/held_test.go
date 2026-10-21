package now

import (
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func heldFirsts() FirstScores {
	week := leaguefeed.WeekResult{Week: 5, Games: []leaguefeed.Game{{
		Home: leaguefeed.Side{Franchise: "0001", Players: []leaguefeed.WeekPlayer{{Starter: true}}},
		Away: leaguefeed.Side{Franchise: "0002"}}}}
	return FirstScores{Weeks: map[int]FirstScore{5: {At: time.Unix(1, 0), Week: week}}, Copies: 1}
}

// Mutating what was published, or what a snapshot returned, never reaches the holder.
func TestHeldDetachesOnPublishAndSnapshot(t *testing.T) {
	t.Parallel()
	var h Held
	in := heldFirsts()
	h.PublishFirsts(in)
	in.Weeks[5].Week.Games[0].Home.Players[0].Starter = false
	delete(in.Weeks, 5)
	out := h.SnapshotFirsts()
	if !out.Weeks[5].Week.Games[0].Home.Players[0].Starter {
		t.Fatal("publication kept the caller's first scores")
	}
	out.Weeks[5].Week.Games[0].Home.Players[0].Starter = false
	if !h.SnapshotFirsts().Weeks[5].Week.Games[0].Home.Players[0].Starter {
		t.Fatal("snapshot shared the held first scores")
	}
	byes := Feed[[]nflschedule.RawBye]{Value: []nflschedule.RawBye{{Week: "7"}}}
	h.PublishByes(byes)
	byes.Value[0].Week = "changed"
	if got := h.SnapshotByes(); got.Value[0].Week != "7" {
		t.Fatal("publication kept the caller's byes")
	}
	live := LiveGame{Week: 5, Adapter: livegame.Adapter{Board: livegame.Held{Games: []livegame.Game{{ID: "g", Box: []livegame.Box{{Rows: []livegame.BoxRow{{Stats: []string{"1"}}}}}}}}}}
	h.PublishLive(live)
	live.Adapter.Board.Games[0].Box[0].Rows[0].Stats[0] = "changed"
	if got := h.SnapshotLive(); got.Adapter.Board.Games[0].Box[0].Rows[0].Stats[0] != "1" {
		t.Fatal("publication kept the caller's live board")
	}
}

// Publication and reads race freely under -race; the holder's lock is its own.
func TestHeldPublishReadRace(t *testing.T) {
	t.Parallel()
	var h Held
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(2)
		go func() { defer wg.Done(); f := heldFirsts(); f.Copies = i; h.PublishFirsts(f) }()
		go func() { defer wg.Done(); _ = h.SnapshotFirsts(); _ = h.SnapshotByes(); _ = h.SnapshotLive() }()
	}
	wg.Wait()
	if h.SnapshotFirsts().Weeks[5].Week.Week != 5 {
		t.Fatal("racing publications lost the held week")
	}
}
