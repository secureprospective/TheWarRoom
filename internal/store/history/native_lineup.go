package history

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// LineupEnvironment supplies the football facts a lineup is judged against. It reads only what the app
// already holds; it never fetches, and never reads a fantasy league's lineup.
type LineupEnvironment interface {
	// PlayerFacts is who each player is: name, position and NFL team.
	PlayerFacts() nativecheck.Facts
	// LineupFacts adds the NFL kickoff times of one week, and the zone the owner reads them in.
	LineupFacts(ctx context.Context, season, week int) (commissioner.LineupFacts, error)
}

// SetLineupEnvironment installs the facts lineups are judged against. Set it at startup; without it a
// lineup cannot be set or corrected, because nothing could say who is locked.
func (s *Store) SetLineupEnvironment(env LineupEnvironment) { s.lineupEnv = env }

var errNoLineupFacts = errors.New("lineups can't be judged until the app holds the NFL players and schedule")

// lineupFactsFor gathers a lineup command's facts before the write begins; any other command needs none.
func (s *Store) lineupFactsFor(ctx context.Context, raw commissioner.RawCommand) (commissioner.LineupFacts, error) {
	if raw.Command == commissioner.BracketBuild || raw.Command == commissioner.BracketEdit ||
		raw.Command == commissioner.BracketSeed || raw.Command == commissioner.BracketPresetSave {
		return s.playoffFactsFor(ctx, raw)
	}
	if raw.Command != commissioner.LineupSet && raw.Command != commissioner.LineupCorrect {
		return commissioner.LineupFacts{}, nil
	}
	change, err := commissioner.DecodeLineupChange(raw.Args, raw.Command)
	if err != nil {
		return commissioner.LineupFacts{}, fmt.Errorf("validate league change: %w", commissioner.Refused(err))
	}
	return s.weekFacts(ctx, change.Season, change.Week)
}

// playoffFactsFor gathers the NFL kickoffs of the season's first playoff week, which say whether the playoffs
// have begun. Without the held schedule they are not known, and the bracket check takes the command as stated.
func (s *Store) playoffFactsFor(ctx context.Context, raw commissioner.RawCommand) (commissioner.LineupFacts, error) {
	change, err := commissioner.DecodeBracketChange(raw.Args, raw.Command)
	if err != nil {
		return commissioner.LineupFacts{}, fmt.Errorf("validate league change: %w", commissioner.Refused(err))
	}
	if s.lineupEnv == nil {
		return commissioner.LineupFacts{}, nil
	}
	records, err := s.nativeRecords(ctx, raw.League)
	if err != nil {
		return commissioner.LineupFacts{}, fmt.Errorf("read the league before checking its playoffs: %w", err)
	}
	week, ok := commissioner.PlayoffStartWeek(records, change.Season, s.now())
	if !ok {
		return commissioner.LineupFacts{}, nil
	}
	return s.weekFacts(ctx, change.Season, week)
}

func (s *Store) weekFacts(ctx context.Context, season, week int) (commissioner.LineupFacts, error) {
	if s.lineupEnv == nil {
		return commissioner.LineupFacts{}, fmt.Errorf("league change refused: %w", commissioner.Refused(errNoLineupFacts))
	}
	facts, err := s.lineupEnv.LineupFacts(ctx, season, week)
	if err != nil {
		return commissioner.LineupFacts{}, fmt.Errorf("read lineup facts: %w", err)
	}
	return facts, nil
}

// NativeLineup reads the lineup standing for a franchise's week at an instant, with locks in the owner's zone.
func (s *Store) NativeLineup(ctx context.Context, league, franchise, principal string, week int,
	at time.Time) (commissioner.NativeLineup, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeLineup{}, err
	}
	season, err := commissioner.SeasonAt(records, commissioner.ReadingInstant(records, at))
	if err != nil {
		return commissioner.NativeLineup{}, fmt.Errorf("read the lineup's season: %w", err)
	}
	env, err := s.weekFacts(ctx, season, week)
	if err != nil {
		return commissioner.NativeLineup{}, err
	}
	out, err := commissioner.NativeLineupReading(league, franchise, principal, week, records, at, env)
	if err != nil {
		return commissioner.NativeLineup{}, fmt.Errorf("read native lineup: %w", err)
	}
	out.Revision = commissioner.ViewRevision(ctx, records, out.Revision)
	return out, nil
}

func (s *Store) lineupCorrectionPreview(ctx context.Context, records []commissioner.Record,
	raw commissioner.RawCommand) (commissioner.ScorePreview, error) {
	change, err := commissioner.DecodeLineupChange(raw.Args, raw.Command)
	if err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("read the lineup correction: %w", err)
	}
	env, err := s.weekFacts(ctx, change.Season, change.Week)
	if err != nil {
		return commissioner.ScorePreview{}, err
	}
	src := s.nativeSources(raw.League, change.Season)
	out, err := commissioner.LineupCorrectionPreview(ctx, raw.League, records, src, raw, env, src.now)
	if err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("preview lineup correction: %w", err)
	}
	return out, nil
}
