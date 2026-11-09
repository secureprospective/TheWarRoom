package commissioner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// approvalScores scores each starter at a fixed number of points, plus bonus once the ledger (held or
// trial) holds an approval of one of his items.
type approvalScores struct {
	lineups map[string][]string
	points  map[string]leaguerules.Decimal
	bonus   leaguerules.Decimal
}

func (s approvalScores) Lineup(_ context.Context, _ []Record, id string, _ int) (WeekLineup, error) {
	l := WeekLineup{Set: len(s.lineups[id]) != 0}
	for _, p := range s.lineups[id] {
		l.Starters = append(l.Starters, Starter{Player: p})
	}
	return l, nil
}

func (s approvalScores) PlayerScore(_ context.Context, records []Record, id string, _ int) (NativePlayerScore, error) {
	points := s.points[id]
	for _, r := range records {
		if r.Command.Command == StatApprove && r.Command.Args.Stat != nil && !strings.HasPrefix(r.Command.Request, "pin.") &&
			strings.Contains(r.Command.Args.Stat.Item, "."+id+".") {
			points += s.bonus
			break
		}
	}
	return NativePlayerScore{Status: "Complete", Points: &points}, nil
}

// A queue read through the price cache and one shared trial ledger must be JSON-identical to pricing every
// item again on its own copy of the ledger, at every revision: the first reading, the same revision again, a
// decision on another player in the same week (which turns over his game) and a non-statistics entry. A cache
// that leaves the week's results out of its key must fail.
func TestStatPriceCacheMatchesFullPricing(t *testing.T) {
	for _, broken := range []bool{false, true} {
		mismatch := pricedAcrossRevisions(t, broken)
		if broken && mismatch == "" {
			t.Fatal("a cache keyed without the week's results still matched full pricing; the test proves nothing")
		}
		if !broken && mismatch != "" {
			t.Fatal(mismatch)
		}
	}
}

func pricedAcrossRevisions(t *testing.T, broken bool) string {
	t.Helper()
	b := pinnedStatLedger(t)
	directory, err := NativeDirectoryReading("native.test", b.records, seasonMoment(3))
	if err != nil || len(directory.Franchises) < 3 {
		t.Fatalf("directory: %v %+v", err, directory)
	}
	home, away, other := directory.Franchises[0].Franchise, directory.Franchises[1].Franchise, directory.Franchises[2].Franchise
	for i, command := range []string{ScheduleGenerate, SchedulePublish} {
		b.add(RawCommand{League: "native.test", Command: command, Args: RawArgs{Schedule: &RawScheduleArgs{}},
			Request: fmt.Sprintf("schedule.%d", i), Effective: "now", Visibility: "league"}, seasonMoment(3))
		if command == ScheduleGenerate {
			b.records[len(b.records)-1].GeneratedSchedule = []ScheduleWeek{{Week: 1,
				Games: []ScheduleGame{{ID: "g1", Home: home, Away: away}}}}
		}
	}
	first, second := tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6)
	allen := statWeek(first, second)
	other99 := allen
	other99.Player = "0100"
	src := approvalScores{bonus: 50000,
		lineups: map[string][]string{home: {"0099"}, away: {"0100"}, other: {"0101"}},
		points:  map[string]leaguerules.Decimal{"0099": 100000, "0100": 120000, "0101": 1}}
	feed := fixedFeed{[]StatWeek{allen, other99}}
	cache := NewStatPriceCache()
	cache.skipWeek = broken
	read := func(label string) string {
		t.Helper()
		got, err := StatQueueReadingWith(t.Context(), "native.test", b.records, feed, src, 2026, seasonMoment(20), cache)
		if err != nil {
			t.Fatal(label, err)
		}
		want, err := statQueueReading(t.Context(), "native.test", b.records, feed, src, 2026, seasonMoment(20), nil, false)
		if err != nil {
			t.Fatal(label, err)
		}
		g, _ := json.Marshal(got)
		w, _ := json.Marshal(want)
		if string(g) != string(w) {
			return fmt.Sprintf("%s: cached %s\nfull %s", label, g, w)
		}
		if len(got.Open) == 0 {
			t.Fatalf("%s: the fixture has nothing waiting: %s", label, g)
		}
		return ""
	}
	if m := read("first reading"); m != "" {
		return m
	}
	if m := read("same revision again"); m != "" {
		return m
	}
	if hits, _ := cache.Counts(); hits == 0 {
		t.Fatal("a second reading of the same revision priced every item again")
	}
	// Approving Allen's change lifts the home side to 150k, so approving 0100's change now turns the game.
	b.statCommand(StatApprove, RawStatChange{Item: StatItemID{Season: 2026, Week: 1, Player: "0099", Seq: 1,
		Digest: second.Digest[:statDigestLen]}.String()}, 7)
	if m := read("after a decision in the same week"); m != "" {
		return m
	}
	b.add(RawCommand{League: "native.test", Command: LineupCorrect, Request: "lineup.note", Effective: "now",
		Visibility: "league"}, seasonMoment(8))
	if m := read("after a non-statistics entry"); m != "" {
		return m
	}
	return ""
}
