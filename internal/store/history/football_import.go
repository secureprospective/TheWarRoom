package history

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// RawFootballEvidence is a commissioner import of reference facts, not an MFL credential
// or a fantasy roster. Its exact bytes, including identity mappings and provenance, are pinned.
type RawFootballEvidence struct {
	Version int                          `json:"version"`
	Season  int                          `json:"season"`
	Source  string                       `json:"source"`
	AsOf    time.Time                    `json:"asOf"`
	Players map[string]RawFootballPlayer `json:"players"`
	Weeks   []leagueweek.Week            `json:"weeks"`
	Stats   []RawFootballStats           `json:"stats,omitempty"`
}

// RawFootballPlayer: PFR is the player's Pro Football Reference id, needed only to count an RB's
// yards before contact for the offensive line unit.
type RawFootballPlayer struct {
	Identity string                  `json:"identity"`
	Facts    nativecheck.PlayerFacts `json:"facts"`
	PFR      string                  `json:"pfr,omitempty"`
	// NFLRosterSeasons lists NFL season labels in which he appeared on an NFL roster, not games or accrued seasons.
	NFLRosterSeasons *[]int `json:"nflRosterSeasons,omitempty"`
}

// Validate checks inline evidence; evidence split into week parts validates only with them (validate).
func (r RawFootballEvidence) Validate() error { return r.validate(nil) }

func (r RawFootballEvidence) validate(parts partSource) error {
	if r.Version != 1 || r.Season < 1920 || r.Season > 9999 || r.AsOf.IsZero() ||
		strings.TrimSpace(r.Source) == "" || len(r.Source) > 200 || len(r.Players) == 0 || len(r.Players) > 30000 {
		return fmt.Errorf("choose version 1, a season, provenance and player facts")
	}
	if err := r.validatePlayers(); err != nil {
		return err
	}
	if err := validateFootballWeeks(r.Weeks); err != nil {
		return err
	}
	_, err := r.scoringFiles(parts)
	return err
}

func (r RawFootballPlayer) Validate() error {
	f := r.Facts
	if err := r.validateRosterSeasons(); err != nil {
		return err
	}
	if strings.TrimSpace(r.Identity) == "" || len(r.Identity) > 200 || strings.TrimSpace(f.Name) == "" || len(f.Name) > 200 ||
		!footballTeam(f.Team) || f.CareerGames < 0 || f.CareerGames > 2000 ||
		(f.DraftYearHeld && (f.DraftYear < 1920 || f.DraftYear > 9999)) {
		return fmt.Errorf("invalid identity or player facts")
	}
	if err := footballPosition(f.Position); err != nil {
		return err
	}
	return validateFootballDesignation(f)
}

func validateFootballDesignation(f nativecheck.PlayerFacts) error {
	switch f.Designation {
	case "", leaguerules.InjuredReserve, leaguerules.PUP, leaguerules.NFI, leaguerules.Out, leaguerules.Suspended:
	default:
		return fmt.Errorf("unknown injury designation")
	}
	if !f.DesignationHeld && f.Designation != "" {
		return fmt.Errorf("injury designation is not held")
	}
	return nil
}

func footballPosition(p leaguerules.Position) error {
	switch p {
	case leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.TE, leaguerules.PK,
		leaguerules.DT, leaguerules.DE, leaguerules.LB, leaguerules.CB, leaguerules.S:
		return nil
	case leaguerules.OL:
		// A team's O-Line card is the league's own, not a football person the evidence describes.
		return fmt.Errorf("football evidence lists people, not O-Line cards")
	default:
		return fmt.Errorf("unknown player position")
	}
}

func footballTeam(team string) bool {
	if team == "" || team == "FA" {
		return true
	}
	if len(team) < 2 || len(team) > 3 {
		return false
	}
	for _, ch := range team {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}
	return true
}

func validateFootballWeeks(weeks []leagueweek.Week) error {
	if len(weeks) == 0 || len(weeks) > 22 {
		return fmt.Errorf("supply one to 22 NFL weeks")
	}
	seen := make(map[int]bool, len(weeks))
	for _, week := range weeks {
		if week.Number < 1 || week.Number > 22 || seen[week.Number] || len(week.Games) == 0 || len(week.Games) > 16 {
			return fmt.Errorf("invalid or duplicate NFL week")
		}
		seen[week.Number] = true
		if err := validateFootballGames(week.Games); err != nil {
			return err
		}
	}
	return nil
}

func validateFootballGames(games []leagueweek.Game) error {
	teams := make(map[string]bool)
	for _, game := range games {
		if game.Kickoff.IsZero() {
			return fmt.Errorf("NFL kickoff is missing")
		}
		for _, team := range game.Teams {
			if team == "" || team == "FA" || !footballTeam(team) || teams[team] {
				return fmt.Errorf("invalid or repeated NFL team in week")
			}
			teams[team] = true
		}
	}
	return nil
}

func decodeFootball(body []byte, parts partSource) (RawFootballEvidence, error) {
	var raw RawFootballEvidence
	if len(body) > maxFootballItem {
		return raw, fmt.Errorf("football evidence exceeds 8 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return raw, fmt.Errorf("decode football evidence: %w", err)
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return raw, fmt.Errorf("football evidence must contain one JSON value")
	}
	if err := raw.validate(parts); err != nil {
		return raw, fmt.Errorf("validate football evidence: %w", err)
	}
	return raw, nil
}

func footballDigest(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

// ImportFootball validates everything before writing. It reuses history's immutable,
// matched-backup archive; no active environment is changed by an import. Split evidence's week
// parts must already be imported (ImportFootballWeek).
func (s *Store) ImportFootball(ctx context.Context, body []byte, digest string) error {
	if _, err := decodeFootball(body, s.storeParts(ctx)); err != nil {
		return err
	}
	if footballDigest(body) != digest {
		return fmt.Errorf("football evidence digest mismatch")
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(body); err != nil {
		return fmt.Errorf("compress football evidence: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close football evidence: %w", err)
	}
	if err := s.Record(ctx, archive.Fetch{URL: "urn:twr:football:" + digest, SHA256: digest,
		Size: int64(len(body)), Gzip: compressed.Bytes(), Status: 200, FetchedAt: s.now()}); err != nil {
		return fmt.Errorf("import football evidence: %w", err)
	}
	return nil
}

func (r RawFootballPlayer) validateRosterSeasons() error {
	if r.NFLRosterSeasons == nil {
		if r.Facts.NFLSeasonsHeld || r.Facts.NFLSeasons != 0 {
			return fmt.Errorf("NFL season counts need NFL roster season evidence")
		}
		return nil
	}
	seen := map[int]bool{}
	for _, season := range *r.NFLRosterSeasons {
		if season < 1920 || season > 9999 || seen[season] {
			return fmt.Errorf("invalid or repeated NFL roster season")
		}
		seen[season] = true
	}
	if r.Facts.NFLSeasons != 0 || r.Facts.NFLSeasonsHeld {
		return fmt.Errorf("supply NFL roster seasons, not a precomputed count")
	}
	return nil
}

func (r RawFootballPlayer) validateRosterThrough(season int) error {
	if r.NFLRosterSeasons != nil {
		for _, year := range *r.NFLRosterSeasons {
			if year > season {
				return fmt.Errorf("NFL roster season is later than the evidence season")
			}
		}
	}
	return nil
}

func (r RawFootballEvidence) validatePlayers() error {
	identities := make(map[string]bool, len(r.Players))
	for id, player := range r.Players {
		canonical, err := playerid.New(id)
		if err != nil || canonical.String() != id {
			return fmt.Errorf("football player id must be canonical: %q", id)
		}
		if err := player.Validate(); err != nil {
			return fmt.Errorf("football player %s: %w", id, err)
		}
		if err := player.validateRosterThrough(r.Season); err != nil {
			return err
		}
		if identities[player.Identity] {
			return fmt.Errorf("football identity maps to multiple players")
		}
		identities[player.Identity] = true
	}
	return nil
}
