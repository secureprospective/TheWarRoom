package main

import (
	"fmt"
	"time"
)

// Freshness is the ONE degradation contract shared by every IPC result whose data depends
// on a network read (B-5). Before this existed, each module invented its own answer to
// "the fetch failed" — M1 carried a free-text Warning, M2 treated a standings failure as
// fatal, Home simply refused to show anything network-backed. Three ad-hoc answers to one
// question is how a UI ends up lying in a different way on every screen.
//
// The rule this encodes: a surface must DEGRADE HONESTLY rather than fail or lie. Data the
// app still holds stays on screen and stays legible; what changes is that the surface
// states its age. Session-C locked the visual side to match — freshness is an EDGE
// treatment (--fresh-live / --fresh-stale / --fresh-fail), never a blur, a hide, or a
// spinner over real data.
//
// State is a closed set of three, deliberately:
//   - live  — this data came from a successful fetch just now.
//   - stale — the fetch FAILED and this data is a last-known-good cache. FetchedAt is
//     mandatory here: stale data that will not say how old it is reads as current, which
//     is the exact lie this type exists to prevent.
//   - fail  — no data at all. The surface engraves; it does not pretend.
//
// "Offseason" is deliberately NOT a fourth state. It is not a freshness question — the
// data is perfectly fresh, it just describes a season that has ended. That is a SEASON
// PHASE concern, carried separately (see the phase field on the result DTOs) so the UI can
// say "final, season complete" instead of the nonsense "stale by 4 months".
type Freshness struct {
	State string `json:"state"`
	// FetchedAt is RFC3339 UTC, or "" when not applicable (fail) / not tracked.
	FetchedAt string `json:"fetchedAt"`
	// Note is a short human-readable reason, shown verbatim next to the edge treatment.
	// It carries the underlying error for stale so the cause is visible without a log dive.
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

// staleFreshness marks a result as served from cache after a failed fetch. cause is the
// fetch error that forced the fallback; it is surfaced so the user can tell an MFL outage
// from a local network problem without opening a log.
func staleFreshness(fetchedAt time.Time, cause error) Freshness {
	return Freshness{
		State:     FreshStale,
		FetchedAt: fetchedAt.UTC().Format(time.RFC3339),
		Note:      fmt.Sprintf("live fetch failed, showing last known data: %v", cause),
	}
}

// localFreshness marks a result that involved NO network read at all — it came entirely
// from local SQLite. It is reported as live because it is: persisted engine output is not
// stale merely because it is old, it is the authoritative result of the last scoring run.
// Treating local reads as perpetually stale would train the user to ignore the signal,
// which costs exactly the times it matters.
func localFreshness() Freshness {
	return Freshness{State: FreshLive, Note: "local data"}
}
