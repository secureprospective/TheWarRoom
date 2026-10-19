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
	out := HeldWeek{SHA256: sha, FetchedAt: at, Missing: []string{missingPlayByPlay}}
	for _, week := range weeks {
		if week.GSIS == scope.GSIS && week.Season == scope.Season && week.Week == scope.Week &&
			week.SeasonType == "REG" {
			out.Week = &week
			return out
		}
	}
	out.Missing = append(out.Missing, "This held file does not report the player's regular-season week")
	return out
}
