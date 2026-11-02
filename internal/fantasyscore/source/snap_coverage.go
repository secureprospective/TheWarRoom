package source

import (
	"fmt"
	"io"
	"maps"
	"math"
	"slices"
	"strconv"
	"time"
)

// RawSnap retains the counts and shares needed to reject partial team-game files.
type RawSnap struct {
	Proof  RawAbsenceProof
	Date   string
	Counts [3]int
	Shares [3]float64
}

func (r RawSnap) Validate() error {
	if err := r.Proof.Validate(); err != nil {
		return err
	}
	if r.Proof.PFR == "" || r.Proof.GameID == "" || r.Proof.Team == "" {
		return fmt.Errorf("snap needs player, team and game")
	}
	if _, err := time.Parse("20060102", r.Date); err != nil {
		return fmt.Errorf("snap game date: %w", err)
	}
	for i, n := range r.Counts {
		share := r.Shares[i]
		if n < 0 || n > 1000 || math.IsNaN(share) || math.IsInf(share, 0) || share < 0 || share > 1 || (n == 0 && share != 0) {
			return fmt.Errorf("invalid snap count or share")
		}
	}
	return nil
}

func ReadSnaps(input io.Reader) ([]RawSnap, error) {
	var out []RawSnap
	err := readRecords(input, func(f map[string]string) error {
		if f["game_type"] != "REG" {
			return nil
		}
		r := RawSnap{Proof: RawAbsenceProof{PFR: f["pfr_player_id"], Season: f["season"], Week: f["week"], GameID: f["game_id"], Team: f["team"], Position: f["position"], Evidence: "positive snaps; no play-by-play stat events"}}
		if len(f["pfr_game_id"]) >= 8 {
			r.Date = f["pfr_game_id"][:8]
		}
		for i, col := range []string{"offense", "defense", "st"} {
			var err error
			if r.Counts[i], err = strconv.Atoi(f[col+"_snaps"]); err != nil {
				return fmt.Errorf("snap %s count: %w", col, err)
			}
			if r.Shares[i], err = strconv.ParseFloat(f[col+"_pct"], 64); err != nil {
				return fmt.Errorf("snap %s share: %w", col, err)
			}
		}
		if err := r.Validate(); err != nil {
			return err
		}
		out = append(out, r)
		return nil
	})
	return out, err
}

// games lists every NFL game the snap file covers.
func games(snaps []RawSnap) map[string]bool {
	out := make(map[string]bool, len(snaps))
	for _, s := range snaps {
		out[s.Proof.GameID] = true
	}
	return out
}

// NonParticipationProofs judges each team's file on its own, because whether a player took a snap
// for his team is answered by his team's rows; the opponent's are no evidence about him. A team
// whose rows are missing from the file certifies nothing. It requires every unit's recovered play count to account for at least eleven
// players a play. A file short of that is missing somebody's snaps, so it cannot prove non-participation;
// counts above it are over-manned plays, which leaving a player out could never produce. A partial file
// cannot merely assert its own completeness.
func NonParticipationProofs(snaps []RawSnap, roster []RawAbsenceProof) []RawAbsenceProof {
	teams := make(map[string][]RawSnap)
	present := make(map[string]bool)
	for _, s := range snaps {
		teams[s.Proof.GameID+"/"+s.Proof.Team] = append(teams[s.Proof.GameID+"/"+s.Proof.Team], s)
		present[s.Proof.GameID+"/"+s.Proof.PFR] = true
	}
	complete := make(map[string]bool)
	for key, rows := range teams {
		complete[key] = completeSnapTeam(rows)
	}
	held := slices.Sorted(maps.Keys(games(snaps)))
	var out []RawAbsenceProof
	for _, r := range roster {
		if r.GSIS == "" || r.PFR == "" {
			continue
		}
		for _, game := range held {
			if !complete[game+"/"+r.Team] || present[game+"/"+r.PFR] || (r.GameID != "" && r.GameID != game) {
				continue
			}
			proof := r
			proof.GameID = game
			proof.Evidence = "complete snaps + pbp, no snap"
			if proof.Validate() == nil {
				out = append(out, proof)
			}
		}
	}
	return out
}

func completeSnapTeam(rows []RawSnap) bool {
	if len(rows) == 0 {
		return false
	}
	seen := make(map[string]bool)
	totals := [3]int{}
	for _, r := range rows {
		if r.Validate() != nil || seen[r.Proof.PFR] {
			return false
		}
		seen[r.Proof.PFR] = true
		for i, n := range r.Counts {
			totals[i] += n
		}
	}
	for i, total := range totals {
		plays, ok := snapUnitPlays(rows, i, total)
		if !ok || total < 11*plays || fullTimeSnaps(rows, i, plays) > 11 {
			return false
		}
	}
	return true
}

// fullTimeSnaps counts the players the file says took every play of a unit. No more than eleven
// can have, so a file claiming more is holding somebody else's rows rather than this team's.
func fullTimeSnaps(rows []RawSnap, unit, plays int) int {
	count := 0
	for _, r := range rows {
		if r.Counts[unit] == plays {
			count++
		}
	}
	return count
}

// snapUnitPlays recovers how many plays a unit took from its rounded shares, which hold that count
// independently of the counts themselves. Only one play count may fit every share: a file whose
// shares leave the count ambiguous cannot say who is missing from it.
func snapUnitPlays(rows []RawSnap, unit, total int) (int, bool) {
	if total == 0 {
		return 0, false
	}
	found := 0
	for plays := 1; plays <= total; plays++ {
		if !snapUnitMatches(rows, unit, plays) {
			continue
		}
		if found != 0 {
			return 0, false
		}
		found = plays
	}
	return found, found != 0
}

func snapUnitMatches(rows []RawSnap, unit, denominator int) bool {
	if denominator < 1 {
		return false
	}
	for _, r := range rows {
		if r.Counts[unit] > denominator || math.Abs(float64(r.Counts[unit])/float64(denominator)-r.Shares[unit]) > 0.00501 {
			return false
		}
	}
	return true
}
