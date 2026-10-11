// Package draftreports decodes MFL's draft results and future-pick ownership exports.
package draftreports

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const ResultsExport = "draftResults"
const FutureExport = "futureDraftPicks"

type RawPick struct {
	Round     string `json:"round"`
	Pick      string `json:"pick"`
	Player    string `json:"player"`
	Franchise string `json:"franchise"`
	Timestamp string `json:"timestamp"`
	Comments  string `json:"comments"`
}

func positive(raw string, maximum int) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > maximum {
		return 0, fmt.Errorf("invalid positive number %q (maximum %d)", raw, maximum)
	}
	return n, nil
}

func (r RawPick) Validate() error {
	if _, err := positive(r.Round, 100); err != nil {
		return fmt.Errorf("draft round: %w", err)
	}
	if _, err := positive(r.Pick, 1000); err != nil {
		return fmt.Errorf("draft pick: %w", err)
	}
	if err := ingestion.FeedFranchise(r.Franchise); err != nil {
		return fmt.Errorf("draft owner: %w", err)
	}
	if r.Player != "" {
		if _, err := playerid.New(r.Player); err != nil {
			return fmt.Errorf("draft player: %w", err)
		}
	}
	if r.Timestamp != "" && r.Timestamp != "0" {
		if _, err := ingestion.FeedTime(r.Timestamp); err != nil {
			return fmt.Errorf("draft time: %w", err)
		}
	}
	if r.Player == "" && r.Timestamp != "" && r.Timestamp != "0" {
		return fmt.Errorf("draft time without a player")
	}
	return nil
}

type Pick struct {
	Skipped     bool
	Round, Slot int
	Player      *playerid.PlayerID
	Franchise   string
	At          time.Time
	Comments    string
}

type RawFuturePick struct {
	Year     string `json:"year"`
	Round    string `json:"round"`
	Original string `json:"originalPickFor"`
	Owner    string `json:"-"`
}

func (r RawFuturePick) Validate() error {
	if _, err := positive(r.Year, 9999); err != nil {
		return fmt.Errorf("future year: %w", err)
	}
	if _, err := positive(r.Round, 100); err != nil {
		return fmt.Errorf("future round: %w", err)
	}
	for _, id := range []string{r.Owner, r.Original} {
		if err := ingestion.FeedFranchise(id); err != nil {
			return fmt.Errorf("future owner: %w", err)
		}
	}
	return nil
}

type FuturePick struct {
	Year, Round     int
	Original, Owner string
}

func section(body []byte, name string) (json.RawMessage, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("draft export: %w", err)
	}
	var env map[string]json.RawMessage
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("draft export: %w", err)
	}
	raw, ok := env[name]
	if !ok || string(raw) == "null" {
		return nil, fmt.Errorf("draft export: missing %s section", name)
	}
	return raw, nil
}

func ParseResults(body []byte) ([]Pick, error) {
	raw, err := section(body, ResultsExport)
	if err != nil {
		return nil, err
	}
	var env struct {
		Units ingestion.MFLList[struct {
			Unit  string                     `json:"unit"`
			Picks ingestion.MFLList[RawPick] `json:"draftPick"`
		}] `json:"draftUnit"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("draft results: %w", err)
	}
	out := []Pick{}
	seen := map[string]bool{}
	for _, unit := range env.Units {
		if unit.Unit != "" && unit.Unit != "LEAGUE" {
			return nil, fmt.Errorf("draft results: unsupported unit %q", unit.Unit)
		}
		for _, r := range unit.Picks {
			p, err := convertPick(r)
			if err != nil {
				return nil, err
			}
			key := fmt.Sprintf("%d:%d", p.Round, p.Slot)
			if seen[key] {
				return nil, fmt.Errorf("draft results: duplicate pick %s", key)
			}
			seen[key] = true
			out = append(out, p)
		}
	}
	return out, nil
}

func convertPick(r RawPick) (Pick, error) {
	if err := r.Validate(); err != nil {
		return Pick{}, err
	}
	round, err := positive(r.Round, 100)
	if err != nil {
		return Pick{}, err
	}
	slot, err := positive(r.Pick, 1000)
	if err != nil {
		return Pick{}, err
	}
	p := Pick{Round: round, Slot: slot, Franchise: r.Franchise, Comments: r.Comments}
	if r.Player == "0000" || r.Player == "0" {
		p.Skipped = true
	} else if r.Player != "" {
		id, err := playerid.New(r.Player)
		if err != nil {
			return Pick{}, fmt.Errorf("draft player: %w", err)
		}
		p.Player = &id
	}
	if r.Timestamp != "" && r.Timestamp != "0" {
		p.At, err = ingestion.FeedTime(r.Timestamp)
		if err != nil {
			return Pick{}, fmt.Errorf("draft timestamp: %w", err)
		}
	}
	return p, nil
}

func ParseFuture(body []byte) ([]FuturePick, error) {
	raw, err := section(body, FutureExport)
	if err != nil {
		return nil, err
	}
	var env struct {
		Franchises ingestion.MFLList[struct {
			ID    string                           `json:"id"`
			Picks ingestion.MFLList[RawFuturePick] `json:"futureDraftPick"`
		}] `json:"franchise"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("future picks: %w", err)
	}
	out := []FuturePick{}
	seen := map[string]bool{}
	for _, f := range env.Franchises {
		if err := ingestion.FeedFranchise(f.ID); err != nil {
			return nil, fmt.Errorf("future franchise: %w", err)
		}
		for _, r := range f.Picks {
			r.Owner = f.ID
			p, err := convertFuture(r)
			if err != nil {
				return nil, err
			}
			key := fmt.Sprintf("%d:%d:%s", p.Year, p.Round, p.Original)
			if seen[key] {
				return nil, fmt.Errorf("future picks: duplicate asset %s", key)
			}
			seen[key] = true
			out = append(out, p)
		}
	}
	return out, nil
}

func convertFuture(r RawFuturePick) (FuturePick, error) {
	if err := r.Validate(); err != nil {
		return FuturePick{}, err
	}
	year, err := positive(r.Year, 9999)
	if err != nil {
		return FuturePick{}, err
	}
	round, err := positive(r.Round, 100)
	if err != nil {
		return FuturePick{}, err
	}
	return FuturePick{Year: year, Round: round, Original: r.Original, Owner: r.Owner}, nil
}

func fetch(ctx context.Context, c *mfl.Client, export, year, leagueID string) ([]byte, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("draft host: %w", err)
	}
	body, err := ingestion.LeagueExport(ctx, c, export, year, leagueID, nil)
	if err != nil {
		return nil, fmt.Errorf("draft fetch: %w", err)
	}
	return body, nil
}

func FetchResults(ctx context.Context, c *mfl.Client, year, leagueID string) ([]Pick, error) {
	body, err := fetch(ctx, c, ResultsExport, year, leagueID)
	if err != nil {
		return nil, err
	}
	return ParseResults(body)
}

func FetchFuture(ctx context.Context, c *mfl.Client, year, leagueID string) ([]FuturePick, error) {
	body, err := fetch(ctx, c, FutureExport, year, leagueID)
	if err != nil {
		return nil, err
	}
	return ParseFuture(body)
}
