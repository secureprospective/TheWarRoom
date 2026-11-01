package history

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/contracts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func (s *Store) heldDepthHistory(ctx context.Context, season int) ([]researchfeeds.RawDepth, []byte, time.Time, error) {
	var rows []researchfeeds.RawDepth
	var body []byte
	var at time.Time
	var parseErr error
	_, err := s.ExactArchivedBodies(ctx, researchfeeds.DepthURL(season), func(_ string, b []byte, stamp time.Time) bool {
		rows, parseErr = contracts.ReadRows[researchfeeds.RawDepth](b)
		if parseErr == nil {
			for _, r := range rows {
				if parseErr = r.Validate(); parseErr != nil {
					break
				}
			}
		}
		body, at = b, stamp
		return true
	})
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("read held historical depth: %w", err)
	}
	if parseErr != nil {
		return nil, nil, time.Time{}, fmt.Errorf("validate held historical depth: %w", parseErr)
	}
	return rows, body, at, nil
}

func (s *Store) heldNonParticipation(ctx context.Context, season int, snapBody []byte, pfrGSIS map[string]string) ([]source.RawAbsenceProof, []byte, time.Time, error) {
	depth, body, at, err := s.heldDepthHistory(ctx, season)
	if err != nil || len(depth) == 0 || len(snapBody) == 0 {
		return nil, body, at, err
	}
	snaps, err := source.ReadSnaps(bytes.NewReader(snapBody))
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("validate full snap coverage: %w", err)
	}
	pfr := make(map[string]string)
	for id, gsis := range pfrGSIS {
		pfr[gsis] = id
	}
	dates := make(map[string]source.RawSnap)
	for _, snap := range snaps {
		dates[snap.Proof.GameID] = snap
	}
	var roster []source.RawAbsenceProof
	for _, game := range slices.Sorted(maps.Keys(dates)) {
		snap := dates[game]
		cutoff, _ := time.Parse("20060102", snap.Date)
		rows := depthAt(depth, cutoff)
		for _, row := range rows {
			if row.GSIS == nil || *row.GSIS == "" {
				continue
			}
			roster = append(roster, source.RawAbsenceProof{GSIS: *row.GSIS, PFR: pfr[*row.GSIS], Season: strconv.Itoa(season), Week: snap.Proof.Week, GameID: snap.Proof.GameID, Team: row.Team, Position: row.Position, Evidence: "complete snaps + pbp, no snap"})
		}
	}
	return source.NonParticipationProofs(snaps, roster), body, at, nil
}

// depthAt uses one complete dated snapshot, not the last row ever seen for a player
// who may have since left the team. No current affiliation is carried backwards.
func depthAt(rows []researchfeeds.RawDepth, cutoff time.Time) []researchfeeds.RawDepth {
	var latest time.Time
	parsed := make(map[string]time.Time)
	for _, r := range rows {
		at, ok := parsed[r.DT]
		if !ok {
			at, _ = time.Parse(time.RFC3339, r.DT)
			parsed[r.DT] = at
		}
		if !at.After(cutoff) && at.After(cutoff.AddDate(0, 0, -7)) && at.After(latest) {
			latest = at
		}
	}
	var out []researchfeeds.RawDepth
	if latest.IsZero() {
		return out
	}
	for _, r := range rows {
		at := parsed[r.DT]
		if at.Equal(latest) {
			out = append(out, r)
		}
	}
	return out
}
