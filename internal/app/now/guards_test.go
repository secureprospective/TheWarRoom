package now

import (
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
)

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

func TestNowRowsNeverShowIDs(t *testing.T) {
	t.Parallel()
	franchiseID := regexp.MustCompile(`(^|[^\d.,])00\d\d([^\d.,]|$)`)
	for _, week := range []int{1, 5, 15} {
		in := realNow(t, week)
		ids := map[string]bool{}
		for _, w := range in.Race.Held.Weeks.Value {
			for _, s := range allSides(w) {
				for _, p := range s.Players {
					ids[p.ID.String()] = true
				}
			}
		}
		board := nowBoard(t, in)
		for _, r := range append(board.Board.Reports, board.Matchup) {
			for _, row := range append(slices.Clone(r.Rows), r.Summary...) {
				for _, cell := range append(slices.Clone(row.Cells), row.Group) {
					for _, word := range strings.FieldsFunc(cell, func(c rune) bool { return c == ' ' || c == '\n' }) {
						if ids[word] || franchiseID.MatchString(word) {
							t.Fatalf("week %d %s shows an id: %q", week, r.ID, cell)
						}
					}
					if strings.Contains(strings.ToLower(cell), "not held") {
						t.Fatalf("week %d %s: %q", week, r.ID, cell)
					}
				}
			}
		}
	}
}
