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
)

func (s *Store) heldDepthHistory(ctx context.Context, season int) ([]source.RawDepth, []byte, time.Time, error) {
	var rows []source.RawDepth
	var body []byte
	var at time.Time
	var parseErr error
	_, err := s.ExactArchivedBodies(ctx, source.DepthHistoryURL(season), func(_ string, b []byte, stamp time.Time) bool {
		rows, parseErr = source.ReadDepthHistory(b)
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
	proofs := source.NonParticipationProofs(snaps, roster)
	proofs = append(proofs, unrosteredWeeks(season, depth, snaps, pfrGSIS)...)
	byes, byeBody, byeAt, err := s.heldRosterByes(ctx, season, depth, snaps)
	if err != nil {
		return nil, nil, time.Time{}, err
	}
	if byeAt.After(at) {
		at = byeAt
	}
	return append(proofs, byes...), append(body, byeBody...), at, nil
}

// weekStarts dates each week by its earliest held game, so a week's membership is read as it stood
// before that week's first kickoff rather than against a later snapshot.
func weekStarts(snaps []source.RawSnap) map[string]time.Time {
	starts := make(map[string]time.Time)
	for _, snap := range snaps {
		day, _ := time.Parse("20060102", snap.Date)
		week := snap.Proof.Week
		if starts[week].IsZero() || day.Before(starts[week]) {
			starts[week] = day
		}
	}
	return starts
}

// unrosteredWeeks admits a zero for a player no NFL team listed that week. The deciding dated
// snapshot must cover every team the season's membership history knows: one short of a team cannot
// say a player is on none of them. Only a player that history lists somewhere in the season can be
// judged by it, because a player it never lists is outside its coverage and stays pending. A player
// who took a snap that week was on a roster whatever the snapshot says, so he is never admitted here.
func unrosteredWeeks(season int, depth []source.RawDepth, snaps []source.RawSnap,
	pfrGSIS map[string]string) []source.RawAbsenceProof {
	teams, players := make(map[string]bool), make(map[string]bool)
	for _, r := range depth {
		teams[source.CanonicalTeam(r.Team)] = true
		if r.GSIS != nil && *r.GSIS != "" {
			players[*r.GSIS] = true
		}
	}
	played := make(map[string]bool)
	for _, snap := range snaps {
		if gsis, ok := pfrGSIS[snap.Proof.PFR]; ok && gsis != "" {
			played[snap.Proof.Week+"/"+gsis] = true
		}
	}
	starts := weekStarts(snaps)
	var out []source.RawAbsenceProof
	for _, week := range slices.Sorted(maps.Keys(starts)) {
		covered, rostered := make(map[string]bool), make(map[string]bool)
		for _, r := range depthAt(depth, starts[week]) {
			covered[source.CanonicalTeam(r.Team)] = true
			if r.GSIS != nil {
				rostered[*r.GSIS] = true
			}
		}
		if len(covered) != len(teams) {
			continue
		}
		for _, gsis := range slices.Sorted(maps.Keys(players)) {
			if rostered[gsis] || played[week+"/"+gsis] {
				continue
			}
			proof := source.RawAbsenceProof{GSIS: gsis, Season: strconv.Itoa(season), Week: week,
				Evidence: source.UnrosteredEvidence}
			if proof.Validate() == nil {
				out = append(out, proof)
			}
		}
	}
	return out
}

// depthAt uses one complete dated snapshot, not the last row ever seen for a player
// who may have since left the team. No current affiliation is carried backwards.
func depthAt(rows []source.RawDepth, cutoff time.Time) []source.RawDepth {
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
	var out []source.RawDepth
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
