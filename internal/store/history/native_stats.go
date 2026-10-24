package history

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// seasonStatFeed is the held weekly files of one season, read for every league player the directory can
// match to an NFL id.
type seasonStatFeed struct {
	store  *Store
	season int
}

// Weeks reads each held file once and keeps, per player's week, only the versions that differ.
func (f seasonStatFeed) Weeks(ctx context.Context) ([]commissioner.StatWeek, error) {
	if pinned, ok := f.store.lineupEnv.(*PinnedFootball); ok {
		return pinned.statWeeks(ctx, f.season)
	}
	if f.store.statWeeks != nil {
		return f.store.statWeeks(ctx, f.season)
	}
	book := source.NewLineBook(f.season)
	err := f.store.heldWeeklyFiles(ctx, f.season, book.Add)
	if errors.Is(err, source.ErrSeasonNotLoaded) {
		return []commissioner.StatWeek{}, nil
	}
	if err != nil {
		return nil, err
	}
	gsis, err := f.store.SourceIDs(ctx, "gsis")
	if err != nil {
		return nil, fmt.Errorf("match league players to NFL statistics: %w", err)
	}
	players := make([]string, 0, len(gsis))
	for player := range gsis {
		players = append(players, player)
	}
	slices.Sort(players)
	owner := make(map[string]string, len(players))
	for _, player := range players {
		if _, taken := owner[gsis[player]]; !taken {
			owner[gsis[player]] = player
		}
	}
	out := []commissioner.StatWeek{}
	for key, versions := range book.Versions() {
		player, matched := owner[key.GSIS]
		if !matched {
			continue
		}
		newest := versions[len(versions)-1].Held
		out = append(out, commissioner.StatWeek{Player: player, Week: key.Week, Evidence: source.NativeEvidence{
			GSIS: key.GSIS, Held: &newest, Lines: versions, Missing: slices.Clone(newest.Missing)}})
	}
	slices.SortFunc(out, func(a, b commissioner.StatWeek) int {
		if a.Player != b.Player {
			return compareText(a.Player, b.Player)
		}
		return a.Week - b.Week
	})
	return out, nil
}

func compareText(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// Name is how the lineup environment calls the player; empty while the app holds no NFL players.
func (f seasonStatFeed) Name(player string) string {
	if f.store.lineupEnv == nil {
		return ""
	}
	return f.store.lineupEnv.PlayerFacts()[player].Name
}

// StatQueue lists the NFL's changes to statistics the league has counted, and what was decided about them.
func (s *Store) StatQueue(ctx context.Context, league string, season int) (commissioner.StatQueue, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.StatQueue{}, err
	}
	feed := seasonStatFeed{store: s, season: season}
	sources := s.nativeSources(league, season)
	weeks, err := feed.Weeks(ctx)
	if err != nil {
		return commissioner.StatQueue{}, fmt.Errorf("read the held weekly statistics: %w", err)
	}
	for _, week := range weeks {
		sources.held[source.RawNativeScope{Player: week.Player, Season: season, Week: week.Week}] = week.Evidence
	}
	out, err := commissioner.StatQueueReading(ctx, league, records, held{weeks, feed}, sources, season, sources.now)
	if err != nil {
		return commissioner.StatQueue{}, fmt.Errorf("read statistics queue: %w", err)
	}
	return out, nil
}

// held is a feed whose weeks were already read, so one reading scans the files once.
type held struct {
	weeks []commissioner.StatWeek
	seasonStatFeed
}

func (h held) Weeks(context.Context) ([]commissioner.StatWeek, error) { return h.weeks, nil }

// statWeekFor reads the held evidence for the player's week a stat command is about.
func (s *Store) statWeekFor(ctx context.Context, raw commissioner.RawCommand) (commissioner.StatWeek, error) {
	change, err := commissioner.DecodeStatChange(raw.Args, raw.Command)
	if err != nil {
		return commissioner.StatWeek{}, fmt.Errorf("read the statistic change: %w", commissioner.Refused(err))
	}
	player, season, week := change.Scope()
	evidence, err := s.nativeEvidence(ctx, s, source.RawNativeScope{Player: player, Season: season, Week: week})
	if errors.Is(err, source.ErrSeasonNotLoaded) {
		return commissioner.StatWeek{Player: player, Week: week}, nil
	}
	if err != nil {
		return commissioner.StatWeek{}, fmt.Errorf("read the player's held statistics: %w", err)
	}
	return commissioner.StatWeek{Player: player, Week: week, Evidence: evidence}, nil
}

// statChangePreview describes a stat command ahead of recording it.
func (s *Store) statChangePreview(ctx context.Context, records []commissioner.Record,
	raw commissioner.RawCommand) (commissioner.ScorePreview, error) {
	week, err := s.statWeekFor(ctx, raw)
	if err != nil {
		return commissioner.ScorePreview{}, err
	}
	change, err := commissioner.DecodeStatChange(raw.Args, raw.Command)
	if err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("read the statistic change: %w", err)
	}
	_, season, _ := change.Scope()
	src := s.nativeSources(raw.League, season)
	out, err := commissioner.StatChangePreview(ctx, raw.League, records, src,
		seasonStatFeed{store: s, season: season}, week, raw, src.now)
	if err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("preview statistic change: %w", err)
	}
	return out, nil
}
