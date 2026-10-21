package main

import (
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// Freshness is the one contract for any IPC result that depends on a network read: a surface
// degrades honestly instead of failing or lying.
//
//	live   fetched just now
//	stale  the fetch failed; this is the last good copy, and FetchedAt says how old it is
//	fail   no data
//
// Offseason is not a fourth state. It is a season-phase question, carried separately.
type Freshness = domain.Freshness

const (
	FreshLive  = domain.FreshLive
	FreshStale = domain.FreshStale
	FreshFail  = domain.FreshFail
)

func liveFreshness(t time.Time) Freshness               { return core.LiveFreshness(t) }
func staleFreshness(t time.Time, cause error) Freshness { return core.StaleFreshness(t, cause) }
