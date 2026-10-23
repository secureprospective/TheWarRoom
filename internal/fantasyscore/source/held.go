package source

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrSeasonNotLoaded marks a season the weekly statistics feed does not reach.
var ErrSeasonNotLoaded = errors.New("weekly NFL statistics are not loaded for this season")

const missingPlayByPlay = "Held play-by-play is needed for long plays and return touchdowns"

// StatPort reads held revisions newest first, without fetching. A missing player in a revision
// stays missing; callers must not silently substitute that player's older revision.
type StatPort interface {
	PlayerWeekStats(context.Context, RawStatScope) ([]HeldWeek, error)
}

type RawStatScope struct {
	GSIS   string
	Season int
	Week   int
}

func (r RawStatScope) Validate() error {
	if r.Season < 1920 || r.Season > 9999 || r.Week < 1 || r.Week > 22 {
		return fmt.Errorf("choose an NFL season and week")
	}
	if len(r.GSIS) != 10 || r.GSIS[:3] != "00-" {
		return fmt.Errorf("choose a GSIS player id")
	}
	for _, digit := range r.GSIS[3:] {
		if digit < '0' || digit > '9' {
			return fmt.Errorf("choose a GSIS player id")
		}
	}
	return nil
}

// HeldWeek identifies the exact archived evidence. Week is nil when that version does not
// report the player, never a fabricated zero. Missing lists evidence needed before native scoring.
type HeldWeek struct {
	SHA256    string
	FetchedAt time.Time
	Week      *PlayerWeek
	Missing   []string
}

func SelectHeldWeek(scope RawStatScope, weeks []PlayerWeek, sha string, at time.Time) HeldWeek {
	for i := range weeks {
		week := &weeks[i] // by index: taking the range copy address heap-allocates every row scanned
		if week.GSIS == scope.GSIS && week.Season == scope.Season && week.Week == scope.Week &&
			week.SeasonType == "REG" {
			return heldWeekAt(weeks, i, sha, at)
		}
	}
	return heldWeekAt(weeks, -1, sha, at)
}

// HeldWeekIndex finds a player's regular-season week in one held file without scanning it: the first row
// for each player's week, as SelectHeldWeek finds it.
type HeldWeekIndex map[heldWeekKey]int

type heldWeekKey struct {
	gsis         string
	season, week int
}

func IndexHeldWeeks(weeks []PlayerWeek) HeldWeekIndex {
	index := make(HeldWeekIndex, len(weeks))
	for i := range weeks {
		week := &weeks[i]
		if week.SeasonType != "REG" {
			continue
		}
		key := heldWeekKey{week.GSIS, week.Season, week.Week}
		if _, seen := index[key]; !seen {
			index[key] = i
		}
	}
	return index
}

// Select is SelectHeldWeek for the file the index was built from.
func (index HeldWeekIndex) Select(scope RawStatScope, weeks []PlayerWeek, sha string, at time.Time) HeldWeek {
	i, ok := index[heldWeekKey{scope.GSIS, scope.Season, scope.Week}]
	if !ok {
		i = -1
	}
	return heldWeekAt(weeks, i, sha, at)
}

// heldWeekAt is the held evidence for row i of a file, or for a file that does not report the week when i is -1.
func heldWeekAt(weeks []PlayerWeek, i int, sha string, at time.Time) HeldWeek {
	out := HeldWeek{SHA256: sha, FetchedAt: at, Missing: []string{missingPlayByPlay}}
	if i < 0 {
		out.Missing = append(out.Missing, "This held file does not report the player's regular-season week")
		return out
	}
	found := weeks[i]
	out.Week = &found
	return out
}
