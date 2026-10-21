// Package drafttest holds the draft report fixtures that both the draft service's tests and the
// root binding tests read, so neither copies them. They are plain values: each side wraps them
// in its own held-feed type.
package drafttest

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Draft is everything the draft reports read, with each feed's read state.
type Draft struct {
	Picks             []draftreports.Pick
	Future            []draftreports.FuturePick
	Read, FutureRead  report.Feed
	Names, Franchises map[string]string
	Values            map[string]float64
	ValueNote, Next   string
}

// Pick is a made pick of player id by owner at round.slot, at Unix second at.
func Pick(t *testing.T, round, slot int, owner, id string, at int64) draftreports.Pick {
	t.Helper()
	p, err := playerid.New(id)
	if err != nil {
		t.Fatal(err)
	}
	return draftreports.Pick{Round: round, Slot: slot, Franchise: owner, Player: &p, At: time.Unix(at, 0).UTC()}
}

// Held is a complete 2026 draft of three picks out of order, one future pick, names, franchise
// names and engine values, read at noon UTC on October 11, 2026.
func Held(t *testing.T) Draft {
	t.Helper()
	read := report.Feed{FetchedAt: time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)}
	return Draft{
		Picks:  []draftreports.Pick{Pick(t, 2, 1, "0002", "0531", 160), Pick(t, 1, 2, "0001", "0532", 130), Pick(t, 1, 1, "0002", "0533", 100)},
		Future: []draftreports.FuturePick{{Year: 2027, Round: 2, Original: "0001", Owner: "0002"}},
		Read:   read, FutureRead: read,
		Names: map[string]string{"0531": "Third", "0532": "Second", "0533": "First"}, Franchises: map[string]string{"0001": "One", "0002": "Two"},
		Values: map[string]float64{"0531": 50, "0532": 10, "0533": 30}, ValueNote: "Engine run: October 11, 2026, 12:00 PM.",
		Next: "Next draft: date unknown.",
	}
}

// Stale is a draft in progress whose last refresh failed: an unmade pick with no time and a
// skipped pick with MFL comments; the future picks' daily read is pending.
func Stale(t *testing.T) Draft {
	t.Helper()
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	pending := Pick(t, 1, 2, "0001", "0532", 0)
	pending.Player, pending.At = nil, time.Time{}
	skipped := Pick(t, 1, 1, "0002", "0533", 100)
	skipped.Player, skipped.Skipped, skipped.Comments = nil, true, "From One"
	return Draft{
		Picks: []draftreports.Pick{pending, skipped}, Future: []draftreports.FuturePick{},
		Read: report.Feed{FetchedAt: at, RefreshError: "timeout"}, FutureRead: report.Feed{FetchedAt: at, ReadPending: true},
		Franchises: map[string]string{"0001": "One"}, ValueNote: "Engine points unavailable.",
		Next: "Next draft: no date set by the commissioner.",
	}
}

// RepoPath resolves canonical fixtures independently of the test's working directory.
func RepoPath(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
	return filepath.Join(append([]string{root}, parts...)...)
}
