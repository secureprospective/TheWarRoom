package source

import (
	"fmt"
	"io"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
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
		if n < 0 || math.IsNaN(share) || math.IsInf(share, 0) || share < 0 || share > 1 || (n == 0 && share != 0) {
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

// NonParticipationProofs requires every unit's counts to reconcile to eleven players
// per snap and its rounded shares. A partial file cannot merely assert its own completeness.
func NonParticipationProofs(snaps []RawSnap, roster []RawAbsenceProof) []RawAbsenceProof {
	teams := make(map[string][]RawSnap)
	present := make(map[string]bool)
	for _, s := range snaps {
		teams[s.Proof.GameID+"/"+s.Proof.Team] = append(teams[s.Proof.GameID+"/"+s.Proof.Team], s)
		present[s.Proof.GameID+"/"+s.Proof.PFR] = true
	}
	complete := make(map[string]bool)
	for _, s := range snaps {
		game := s.Proof.GameID
		if _, seen := complete[game]; seen {
			continue
		}
		parts := strings.Split(game, "_")
		complete[game] = len(parts) == 4 && completeSnapTeam(teams[game+"/"+parts[2]]) && completeSnapTeam(teams[game+"/"+parts[3]])
	}
	var out []RawAbsenceProof
	for _, r := range roster {
		if r.GSIS == "" || r.PFR == "" {
			continue
		}
		for _, game := range slices.Sorted(maps.Keys(complete)) {
			ok := complete[game]
			if !ok || present[game+"/"+r.PFR] || (r.GameID != "" && r.GameID != game) {
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
		if total == 0 || total%11 != 0 {
			return false
		}
		denominator := float64(total / 11)
		for _, r := range rows {
			if math.Abs(float64(r.Counts[i])/denominator-r.Shares[i]) > 0.00501 {
				return false
			}
		}
	}
	return true
}
