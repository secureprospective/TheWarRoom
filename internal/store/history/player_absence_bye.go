package history

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

func (s *Store) heldRosterByes(ctx context.Context, season int, depth []source.RawDepth, snaps []source.RawSnap) ([]source.RawAbsenceProof, []byte, time.Time, error) {
	var byes []source.RawBye
	var body []byte
	var at time.Time
	var parseErr error
	_, err := s.ArchivedBodies(ctx, "TYPE=nflByeWeeks", func(u string, b []byte, stamp time.Time) bool {
		if !strings.Contains(u, fmt.Sprintf("/%d/export?", season)) {
			return false
		}
		byes, parseErr = source.ParseByes(b)
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

func rosterByes(season int, depth []source.RawDepth, snaps []source.RawSnap, byes []source.RawBye) []source.RawAbsenceProof {
	starts := weekStarts(snaps)
	playing := make(map[string]bool)
	for _, snap := range snaps {
		playing[snap.Proof.Week+"/"+source.CanonicalTeam(snap.Proof.Team)] = true
	}
	var out []source.RawAbsenceProof
	for _, bye := range byes {
		cutoff, ok := starts[bye.Week]
		if !ok || playing[bye.Week+"/"+source.CanonicalTeam(bye.Team)] {
			continue
		}
		for _, r := range depthAt(depth, cutoff) {
			if r.GSIS == nil || *r.GSIS == "" || source.CanonicalTeam(r.Team) != source.CanonicalTeam(bye.Team) {
				continue
			}
			proof := source.RawAbsenceProof{GSIS: *r.GSIS, Season: strconv.Itoa(season), Week: bye.Week, Team: r.Team, Position: r.Position, Evidence: "team bye per schedule"}
			if proof.Validate() == nil {
				out = append(out, proof)
			}
		}
	}
	return out
}
