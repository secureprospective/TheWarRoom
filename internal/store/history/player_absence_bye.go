package history

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func (s *Store) heldRosterByes(ctx context.Context, season int, depth []researchfeeds.RawDepth, snaps []source.RawSnap) ([]source.RawAbsenceProof, []byte, time.Time, error) {
	var byes []nflschedule.RawBye
	var body []byte
	var at time.Time
	var parseErr error
	_, err := s.ArchivedBodies(ctx, "TYPE=nflByeWeeks", func(u string, b []byte, stamp time.Time) bool {
		if !strings.Contains(u, fmt.Sprintf("/%d/export?", season)) {
			return false
		}
		byes, parseErr = nflschedule.ParseByes(b)
		body, at = b, stamp
		return true
	})
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("read held NFL byes: %w", err)
	}
	if parseErr != nil {
		return nil, nil, time.Time{}, fmt.Errorf("validate held NFL byes: %w", parseErr)
	}
	return rosterByes(season, depth, snaps, byes), body, at, nil
}

func rosterByes(season int, depth []researchfeeds.RawDepth, snaps []source.RawSnap, byes []nflschedule.RawBye) []source.RawAbsenceProof {
	starts := make(map[string]time.Time)
	playing := make(map[string]bool)
	for _, snap := range snaps {
		day, _ := time.Parse("20060102", snap.Date)
		week := snap.Proof.Week
		if starts[week].IsZero() || day.Before(starts[week]) {
			starts[week] = day
		}
		playing[week+"/"+livegame.Canonical(snap.Proof.Team)] = true
	}
	var out []source.RawAbsenceProof
	for _, bye := range byes {
		cutoff, ok := starts[bye.Week]
		if !ok || playing[bye.Week+"/"+livegame.Canonical(bye.Team)] {
			continue
		}
		for _, r := range depthAt(depth, cutoff) {
			if r.GSIS == nil || *r.GSIS == "" || livegame.Canonical(r.Team) != livegame.Canonical(bye.Team) {
				continue
			}
			out = append(out, source.RawAbsenceProof{GSIS: *r.GSIS, Season: strconv.Itoa(season), Week: bye.Week, Team: r.Team, Position: r.Position, Evidence: "team bye per schedule"})
		}
	}
	return out
}
