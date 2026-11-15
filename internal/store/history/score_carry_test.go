package history

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func carryRecord(entry, command, request string, lineup *commissioner.RawLineupChange) commissioner.Record {
	return commissioner.Record{Receipt: commissioner.Receipt{Entry: entry},
		Command: commissioner.RawCommand{Command: command, Request: request, Args: commissioner.RawArgs{Lineup: lineup}}}
}

func carryLineup(entry, franchise string, week int, request string) commissioner.Record {
	return carryRecord(entry, commissioner.LineupSet, request,
		&commissioner.RawLineupChange{Season: 2026, Franchise: franchise, Week: week, Starters: []string{"1001"}})
}

// A burst of owners setting lineups must not make the next standings reading rescore the whole season: a
// lineup counts for its own franchise-week only, so a revision that only adds lineups starts from the last
// one, less the franchise-weeks it touches. Anything else, or anything a lineup cannot vouch for, starts empty.
func TestScoreCacheCarriesEverythingALineupCannotTouch(t *testing.T) {
	t.Parallel()
	held := []commissioner.Record{carryRecord("e1", "commish.rules.set", "", nil), carryRecord("e2", "commish.rules.set", "", nil)}
	keys := []weekScoreKey{{franchise: "franchise.01", season: 2026, week: 9}, {franchise: "franchise.02", season: 2026, week: 9},
		{franchise: "franchise.01", season: 2026, week: 10}}
	seed := func() *seasonScoreCache {
		c := &seasonScoreCache{}
		c.carryLineups("r1", "base", held)
		for _, k := range keys {
			c.put("r1", k, commissioner.FranchiseWeekScore{Franchise: k.franchise})
		}
		return c
	}
	carried := func(c *seasonScoreCache, revision string) map[weekScoreKey]bool {
		out := map[weekScoreKey]bool{}
		for k := range c.revisions[revision].scores {
			out[k] = true
		}
		return out
	}

	c := seed()
	c.carryLineups("r2", "base", append(append([]commissioner.Record{}, held...), carryLineup("e3", "franchise.01", 9, "owner-1")))
	got := carried(c, "r2")
	if len(got) != 2 || got[keys[0]] || !got[keys[1]] || !got[keys[2]] {
		t.Fatalf("a franchise.01 week 9 lineup carried %v; want every franchise-week but franchise.01 week 9", got)
	}

	for name, added := range map[string]commissioner.Record{
		"another command":    carryRecord("e3", "commish.score.franchise", "", nil),
		"a synced lineup":    carryLineup("e3", "franchise.01", 9, "lineup.sync.mfl-9"),
		"a lineup with none": carryRecord("e3", commissioner.LineupCorrect, "", nil),
	} {
		c := seed()
		c.carryLineups("r2", "base", append(append([]commissioner.Record{}, held...), added))
		if n := len(c.revisions["r2"].scores); n != 0 {
			t.Fatalf("%s carried %d scores", name, n)
		}
	}

	c = seed()
	c.carryLineups("r2", "base", []commissioner.Record{held[0], carryLineup("e3", "franchise.01", 9, "owner-1")})
	if n := len(c.revisions["r2"].scores); n != 0 {
		t.Fatalf("a ledger missing a held entry carried %d scores", n)
	}
	c = seed()
	c.carryLineups("r2", "other rules", append(append([]commissioner.Record{}, held...), carryLineup("e3", "franchise.01", 9, "owner-1")))
	if n := len(c.revisions["r2"].scores); n != 0 {
		t.Fatalf("a different base (rules, archive or facts) carried %d scores", n)
	}
}
