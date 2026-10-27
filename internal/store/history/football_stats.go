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
// revision holds offensive line facts (football_line.go). A revision too large for one pinned item
// lists its week parts by digest instead (Parts, football_week.go); it carries no inline bodies.
type RawFootballStats struct {
	AsOf    time.Time               `json:"asOf"`
	Weekly  string                  `json:"weekly,omitempty"`
	Plays   string                  `json:"plays,omitempty"`
	Rushing string                  `json:"rushing,omitempty"`
	Parts   []string                `json:"parts,omitempty"`
	Sleeper []source.RawSleeperWeek `json:"sleeper,omitempty"`
}

func (r RawFootballStats) Validate() error {
	if r.AsOf.IsZero() {
		return fmt.Errorf("statistics need a source instant")
	}
	if len(r.Parts) != 0 {
		if r.Weekly != "" || r.Plays != "" || r.Rushing != "" || len(r.Sleeper) != 0 || len(r.Parts) > 22 {
			return fmt.Errorf("split statistics list one to 22 week parts and embed no bodies")
		}
		for _, digest := range r.Parts {
			if len(digest) != 64 {
				return fmt.Errorf("a week part is named by its digest")
			}
		}
		return nil
	}
	if r.Weekly == "" || r.Plays == "" {
		return fmt.Errorf("statistics need a source instant, weekly CSV and play-by-play CSV")
	}
	return nil
}

func (r RawFootballEvidence) scoringFiles(parts partSource) ([]heldFile, error) {
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
		file, err := r.scoringFile(raw, parts)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	if len(files) != 0 {
		for _, player := range r.Players {
			if player.Identity == "" && player.Sleeper != "" {
				continue
			}
			if err := (source.RawStatScope{GSIS: player.Identity, Season: r.Season, Week: 1}).Validate(); err != nil {
				return nil, fmt.Errorf("pinned statistics require GSIS identities: %w", err)
			}
		}
	}
	return files, nil
}

// scoringFile reads one statistic revision: its completed player weeks and, when it carries
// advanced rushing, its offensive line games. Inline and split revisions read the same bodies.
func (r RawFootballEvidence) scoringFile(raw RawFootballStats, parts partSource) (heldFile, error) {
	bodies, sha, err := r.revisionBodies(raw, parts)
	if err != nil {
		return heldFile{}, err
	}
	var weeks []source.PlayerWeek
	var plays []source.RawPlay
	for _, body := range bodies {
		w, p, err := body.weekFiles()
		if err != nil {
			return heldFile{}, err
		}
		weeks, plays = append(weeks, w...), append(plays, p...)
	}
	for _, week := range weeks {
		if week.Season != r.Season {
			return heldFile{}, fmt.Errorf("statistics name a different season")
		}
	}
	if err := source.AttachPlays(weeks, plays); err != nil {
		return heldFile{}, fmt.Errorf("complete pinned statistics: %w", err)
	}
	lineSHA, lines, err := lineFile(bodies, r.leagueRB())
	if err != nil {
		return heldFile{}, err
	}
	return heldFile{sha: sha, at: raw.AsOf,
		weeks: weeks, index: source.IndexHeldWeeks(weeks), lineSHA: lineSHA, lines: lines}, nil
}

// revisionBodies returns a revision's bodies and its statistics SHA. An inline revision is one
// unchecked-week body with its SHA as before; a split revision's SHA covers its part digests.
func (r RawFootballEvidence) revisionBodies(raw RawFootballStats, parts partSource) ([]RawFootballWeek, string, error) {
	var bodies []RawFootballWeek
	sha := footballDigest([]byte(fmt.Sprintf("%d:%s%s", len(raw.Weekly), raw.Weekly, raw.Plays)))
	if len(raw.Parts) == 0 {
		bodies = []RawFootballWeek{{Season: r.Season, Weekly: raw.Weekly, Plays: raw.Plays, Rushing: raw.Rushing, Sleeper: raw.Sleeper}}
	} else {
		var err error
		bodies, err = r.readParts(raw, parts)
		if err != nil {
			return nil, "", err
		}
		sha = footballDigest([]byte("parts:" + strings.Join(raw.Parts, ",")))
	}
	return r.sleeperRevision(bodies, sha)
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
