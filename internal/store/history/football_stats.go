package history

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// RawFootballStats embeds the exact source bodies, newest revision first. Keeping
// them inside the selected evidence makes fetch-archive retention irrelevant. Rushing is
// PFR advanced weekly rushing; with it the plays body must carry the line columns and the
// revision holds offensive line facts (football_line.go).
type RawFootballStats struct {
	AsOf    time.Time `json:"asOf"`
	Weekly  string    `json:"weekly"`
	Plays   string    `json:"plays"`
	Rushing string    `json:"rushing,omitempty"`
}

func (r RawFootballStats) Validate() error {
	if r.AsOf.IsZero() || r.Weekly == "" || r.Plays == "" {
		return fmt.Errorf("statistics need a source instant, weekly CSV and play-by-play CSV")
	}
	return nil
}

func (r RawFootballEvidence) scoringFiles() ([]heldFile, error) {
	if len(r.Stats) > 100 {
		return nil, fmt.Errorf("football evidence exceeds 100 statistic revisions")
	}
	if err := r.validatePFR(); err != nil {
		return nil, err
	}
	files := make([]heldFile, 0, len(r.Stats))
	for i, raw := range r.Stats {
		if err := raw.Validate(); err != nil {
			return nil, err
		}
		if raw.AsOf.After(r.AsOf) || (i > 0 && !raw.AsOf.Before(r.Stats[i-1].AsOf)) {
			return nil, fmt.Errorf("statistic revisions must be newest first, no later than the import")
		}
		file, err := r.scoringFile(raw)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	if len(files) != 0 {
		for _, player := range r.Players {
			if err := (source.RawStatScope{GSIS: player.Identity, Season: r.Season, Week: 1}).Validate(); err != nil {
				return nil, fmt.Errorf("pinned statistics require GSIS identities: %w", err)
			}
		}
	}
	return files, nil
}

// scoringFile reads one statistic revision: its completed player weeks and, when it carries
// advanced rushing, its offensive line games.
func (r RawFootballEvidence) scoringFile(raw RawFootballStats) (heldFile, error) {
	weeks, err := source.ReadWeeks(strings.NewReader(raw.Weekly))
	if err != nil {
		return heldFile{}, fmt.Errorf("read pinned weekly statistics: %w", err)
	}
	for _, week := range weeks {
		if week.Season != r.Season {
			return heldFile{}, fmt.Errorf("statistics name a different season")
		}
	}
	plays, err := source.ReadPlays(strings.NewReader(raw.Plays))
	if err != nil {
		return heldFile{}, fmt.Errorf("read pinned plays: %w", err)
	}
	if err := source.AttachPlays(weeks, plays); err != nil {
		return heldFile{}, fmt.Errorf("complete pinned statistics: %w", err)
	}
	lineSHA, lines, err := lineFile(raw, r.leagueRB())
	if err != nil {
		return heldFile{}, err
	}
	return heldFile{sha: footballDigest([]byte(fmt.Sprintf("%d:%s%s", len(raw.Weekly), raw.Weekly, raw.Plays))), at: raw.AsOf,
		weeks: weeks, index: source.IndexHeldWeeks(weeks), lineSHA: lineSHA, lines: lines}, nil
}

func (p *PinnedFootball) SourceID(ctx context.Context, player, kind string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("read pinned identity: %w", err)
	}
	if kind != "gsis" {
		return "", fmt.Errorf("pinned statistics use GSIS identities")
	}
	if len(p.stats) == 0 {
		return "", nil
	}
	return p.raw.Players[player].Identity, nil
}

func (p *PinnedFootball) PlayerWeekStats(ctx context.Context, scope source.RawStatScope) ([]source.HeldWeek, error) {
	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("read pinned statistics: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("read pinned statistics: %w", err)
	}
	if scope.Season != p.raw.Season {
		return nil, fmt.Errorf("pinned statistics do not hold season %d", scope.Season)
	}
	weeks := selectHeldWeeks(scope, p.stats)
	for i := range weeks {
		// Every imported line passed AttachPlays coverage validation.
		weeks[i].Missing = weeks[i].Missing[1:]
	}
	return weeks, nil
}

func (p *PinnedFootball) statWeeks(ctx context.Context, season int) ([]commissioner.StatWeek, error) {
	out := []commissioner.StatWeek{}
	for _, player := range slices.Sorted(maps.Keys(p.raw.Players)) {
		for week := 1; week <= 22; week++ {
			evidence, err := source.ReadNativeEvidence(ctx, p, p, source.RawNativeScope{Player: player, Season: season, Week: week})
			if err != nil {
				return nil, fmt.Errorf("read pinned statistic revisions: %w", err)
			}
			if len(evidence.Lines) != 0 {
				out = append(out, commissioner.StatWeek{Player: player, Week: week, Evidence: evidence})
			}
		}
	}
	return out, nil
}
