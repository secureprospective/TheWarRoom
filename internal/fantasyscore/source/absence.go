package source

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RawAbsenceProof is positive participation or an exact-week Out designation, not missing data.
type RawAbsenceProof struct {
	GSIS, PFR, Season, Week, GameID, Team, Position, Evidence string
}

func (r RawAbsenceProof) Validate() error {
	season, err := strconv.Atoi(r.Season)
	if err != nil {
		return fmt.Errorf("absence season: %w", err)
	}
	week, err := strconv.Atoi(r.Week)
	if err != nil {
		return fmt.Errorf("absence week: %w", err)
	}
	if r.GSIS != "" {
		if err := (RawStatScope{GSIS: r.GSIS, Season: season, Week: week}).Validate(); err != nil {
			return err
		}
	}
	if err := r.validateGame(season, week); err != nil {
		return err
	}
	if r.Evidence != "positive snaps; no play-by-play stat events" && r.Evidence != "exact-week injury Out; no play-by-play stat events" {
		return fmt.Errorf("unknown absence evidence")
	}
	return nil
}

func (r RawAbsenceProof) validateGame(season, week int) error {
	if err := r.validatePeriod(season, week); err != nil {
		return err
	}
	if r.GameID != "" {
		parts := strings.Split(r.GameID, "_")
		if len(parts) != 4 || parts[0] != r.Season || parts[1] != fmt.Sprintf("%02d", week) || (r.Team != "" && parts[2] != r.Team && parts[3] != r.Team) {
			return fmt.Errorf("absence proof game must match its period and team")
		}
	}
	if r.PFR != "" && (len(r.PFR) > 200 || strings.TrimSpace(r.PFR) != r.PFR) {
		return fmt.Errorf("invalid absence PFR identity")
	}
	return nil
}

// ReadSnapProofs reads only affirmative snap counts. Missing and all-zero rows prove nothing.
func ReadSnapProofs(input io.Reader) ([]RawAbsenceProof, error) {
	var out []RawAbsenceProof
	err := readRecords(input, func(f map[string]string) error {
		if f["game_type"] != "REG" {
			return nil
		}
		positive := false
		for _, col := range []string{"offense_snaps", "defense_snaps", "st_snaps"} {
			n, err := strconv.Atoi(f[col])
			if err != nil || n < 0 {
				return fmt.Errorf("invalid snap count %s", col)
			}
			positive = positive || n > 0
		}
		if !positive {
			return nil
		}
		r := RawAbsenceProof{PFR: f["pfr_player_id"], Season: f["season"], Week: f["week"], GameID: f["game_id"], Team: f["team"], Position: f["position"], Evidence: "positive snaps; no play-by-play stat events"}
		if err := r.Validate(); err != nil {
			return err
		}
		out = append(out, r)
		return nil
	})
	return out, err
}

func ReadOutProofs(input io.Reader) ([]RawAbsenceProof, error) {
	var out []RawAbsenceProof
	err := readRecords(input, func(f map[string]string) error {
		if f["game_type"] != "REG" || f["report_status"] != "Out" {
			return nil
		}
		r := RawAbsenceProof{GSIS: f["gsis_id"], Season: f["season"], Week: f["week"], Team: f["team"], Position: f["position"], Evidence: "exact-week injury Out; no play-by-play stat events"}
		if err := r.Validate(); err != nil {
			return err
		}
		out = append(out, r)
		return nil
	})
	return out, err
}

// AddProvenZeros requires a declared end-of-game record, complete credits and affirmative
// evidence. Any player event wins over an Out designation. Unknown identities stay absent.
func AddProvenZeros(weeks []PlayerWeek, plays []RawPlay, proofs []RawAbsenceProof) ([]PlayerWeek, error) {
	known := make(map[string]bool)
	for _, w := range weeks {
		known[playKey(w.GSIS, w.Season, w.Week)] = true
	}
	final, actors, err := zeroCoverage(plays)
	if err != nil {
		return nil, err
	}
	out := append([]PlayerWeek(nil), weeks...)
	for _, r := range proofs {
		if err := r.Validate(); err != nil {
			return nil, err
		}
		if r.GSIS == "" {
			continue
		}
		season, _ := strconv.Atoi(r.Season)
		week, _ := strconv.Atoi(r.Week)
		key := playKey(r.GSIS, season, week)
		if known[key] || actors[key] {
			continue
		}
		game := r.GameID
		if game == "" {
			game = proofGame(r, final)
		}
		if !final[game] {
			continue
		}
		w := PlayerWeek{GSIS: r.GSIS, Season: season, Week: week, SeasonType: "REG", GameID: game, Position: r.Position, Line: emptyRecordedLine()}
		w.Provenance = "nflverse participation evidence"
		w.Review = "nflverse missing row"
		w.Evidence = r.Evidence
		out = append(out, w)
		known[key] = true
	}
	return out, nil
}

func proofGame(r RawAbsenceProof, final map[string]bool) string {
	week, _ := strconv.Atoi(r.Week)
	prefix := fmt.Sprintf("%s_%02d_", r.Season, week)
	found := ""
	for game := range final {
		parts := strings.Split(game, "_")
		if strings.HasPrefix(game, prefix) && len(parts) == 4 && (parts[2] == r.Team || parts[3] == r.Team) {
			if found != "" {
				return ""
			}
			found = game
		}
	}
	return found
}

func (r RawPlayerCredits) actors() []string {
	var out []string
	for column, value := range r {
		if strings.HasSuffix(column, "player_id") {
			out = append(out, value)
		}
	}
	return out
}

func emptyRecordedLine() fantasyscore.StatLine {
	line := fantasyscore.StatLine{Quantities: make(map[leaguerules.StatID]leaguerules.Decimal), Events: make(map[leaguerules.StatID][]leaguerules.Decimal)}
	for id := range mappings() {
		line.Quantities[id] = 0
	}
	for _, id := range edgeStatIDs() {
		line.Quantities[id] = 0
	}
	for _, id := range []leaguerules.StatID{"pat_made", "pat_missed", "pat_blocked", "fg_blocked", "fumbles_total", "fumble_recovery_tds", "def_tds", "fg_made_60_99", "interception_return_td", "punt_return_td", "kickoff_return_td", "other_return_td", "blocked_punt_td", "blocked_fg_td"} {
		line.Quantities[id] = 0
	}
	for _, id := range []leaguerules.StatID{"long_pass", "long_rush", "long_reception", "chunk_play", "fg_made_distance", "fg_missed_distance"} {
		line.Events[id] = []leaguerules.Decimal{}
	}
	return line
}

func zeroCoverage(plays []RawPlay) (map[string]bool, map[string]bool, error) {
	games := make(map[string]bool)
	actors := make(map[string]bool)
	for _, p := range plays {
		if err := p.Validate(); err != nil {
			return nil, nil, err
		}
		if _, ok := games[p.GameID]; !ok {
			games[p.GameID] = true
		}
		games[p.GameID] = games[p.GameID] && p.Credits.complete()
		season, _ := strconv.Atoi(p.Season)
		week, _ := strconv.Atoi(p.Week)
		for _, id := range append([]string{p.Passer, p.Receiver, p.Rusher, p.Scorer}, p.Credits.actors()...) {
			for _, part := range strings.Split(id, ";") {
				if part != "" && part != "NA" {
					actors[playKey(part, season, week)] = true
				}
			}
		}
	}
	final := make(map[string]bool)
	for _, p := range plays {
		if p.EndGame == "END GAME" && p.GameSeconds == "0" && games[p.GameID] {
			final[p.GameID] = true
		}
	}
	return final, actors, nil
}

func (r RawAbsenceProof) validatePeriod(season, week int) error {
	if season < 1920 || season > 9999 || week < 1 || week > 22 || (r.GSIS == "" && r.PFR == "") || (r.GameID == "" && r.Team == "") {
		return fmt.Errorf("absence proof needs identity, period and game or team")
	}
	return nil
}
