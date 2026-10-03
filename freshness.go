package main

import (
	"fmt"
	"time"
)

// Freshness is the one contract for any IPC result that depends on a network read: a surface
// degrades honestly instead of failing or lying.
//
//	live   fetched just now
//	stale  the fetch failed; this is the last good copy, and FetchedAt says how old it is
//	fail   no data
//
// Offseason is not a fourth state. It is a season-phase question, carried separately.
type Freshness struct {
	State string `json:"state"`
	// FetchedAt is RFC3339 UTC, or "" when not applicable (fail) / not tracked.
	FetchedAt string `json:"fetchedAt"`
	// Note is shown beside the edge treatment; for stale it carries the fetch error.
	Note string `json:"note"`
}

// Freshness states. Keep in sync with the TypeScript union in
// frontend/src/components/board/freshness.ts — these strings cross the IPC boundary.
const (
	FreshLive  = "live"
	FreshStale = "stale"
	FreshFail  = "fail"
)

// liveFreshness marks a result as served from a successful fetch at time t.
func liveFreshness(t time.Time) Freshness {
	return Freshness{State: FreshLive, FetchedAt: t.UTC().Format(time.RFC3339)}
}

// staleFreshness marks a cached result after a failed fetch. The cause is shown so an MFL
// outage reads differently from a local network fault.
func staleFreshness(fetchedAt time.Time, cause error) Freshness {
	return Freshness{
		State:     FreshStale,
		FetchedAt: fetchedAt.UTC().Format(time.RFC3339),
		Note:      fmt.Sprintf("live fetch failed, showing last known data: %v", cause),
	}
}

// localFreshness marks a result read only from SQLite. It is live: stored engine output is
// the last scoring run's result, not stale data.
func localFreshness() Freshness {
	return Freshness{State: FreshLive, Note: "local data"}
}
