package history

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// heldWeeklyFiles offers each held copy of a season's weekly file to accept, newest first, parsed once. A
// refetch of the body just offered is not offered again; a return to an earlier body is, because the
// feed really did change back.
func (s *Store) heldWeeklyFiles(ctx context.Context, season int,
	accept func(sha string, at time.Time, weeks []source.PlayerWeek)) error {
	return s.readWeeklyFiles(ctx, season, func(sha string, at time.Time, weeks []source.PlayerWeek) bool {
		accept(sha, at, weeks)
		return false
	})
}

func (s *Store) readWeeklyFiles(ctx context.Context, season int,
	accept func(sha string, at time.Time, weeks []source.PlayerWeek) bool) error {
	feed, ok := s.reg.Feed("stats_player_week")
	if !ok {
		return fmt.Errorf("weekly NFL statistics feed is not registered")
	}
	if season < feed.FirstSeason {
		return fmt.Errorf("%w", source.ErrSeasonNotLoaded)
	}
	wanted := feed.URLFor(season)
	var parseErr error
	var previous string
	_, err := s.ExactArchivedBodies(ctx, wanted, func(fetchedURL string, body []byte, at time.Time) bool {
		if fetchedURL != wanted {
			return false
		}
		hash := sha256.Sum256(body)
		sha := hex.EncodeToString(hash[:])
		if sha == previous {
			return false
		}
		previous = sha
		weeks, err := s.decodedWeeks(sha, body)
		if err != nil {
			parseErr = fmt.Errorf("read held weekly statistics %s: %w", sha, err)
			return true
		}
		return accept(sha, at, weeks)
	})
	if err != nil {
		return fmt.Errorf("read weekly statistics archive: %w", err)
	}
	return parseErr
}

// PlayerWeekStats reads the original weekly files, not the reduced floating-point observations.
// Missing keeps absent play-by-play explicit until acquisition has held it.
func (s *Store) PlayerWeekStats(ctx context.Context, scope source.RawStatScope) ([]source.HeldWeek, error) {
	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("read held player statistics: %w", err)
	}
	files, err := s.heldSeason(ctx, scope.Season)
	if err != nil {
		return nil, err
	}
	return selectHeldWeeks(scope, files), nil
}

// heldFile is one held copy of a season's weekly file, already parsed.
type heldFile struct {
	sha            string
	complete       bool
	at             time.Time
	weeks          []source.PlayerWeek
	canonical      map[canonicalWeek]source.PlayerWeek
	unknownSleeper map[int]int
	index          source.HeldWeekIndex // built only by readings that look up many players
	// lines are the revision's offensive line games, nil when it holds no advanced rushing.
	lineSHA string
	lines   map[source.LineGame]fantasyscore.UnitLine
}

// heldSeason reads every held copy of a season's weekly file once, newest first.
func (s *Store) heldSeason(ctx context.Context, season int) ([]heldFile, error) {
	files := []heldFile{}
	err := s.heldWeeklyFiles(ctx, season, func(sha string, at time.Time, weeks []source.PlayerWeek) {
		files = append(files, heldFile{sha: sha, at: at, weeks: weeks})
	})
	if err != nil {
		return nil, err
	}
	return s.completeNewestFile(ctx, season, files)
}

// completeNewestFile creates a new evidence revision, rather than rewriting older weekly revisions.
func (s *Store) completeNewestFile(ctx context.Context, season int, files []heldFile) ([]heldFile, error) {
	if len(files) == 0 {
		return files, nil
	}
	weekly := files[0]
	proofs, proofSHA, proofAt, err := s.heldAbsenceProofs(ctx, season)
	if err != nil {
		return nil, err
	}
	var completed heldFile
	var parseErr error
	found, err := s.ExactArchivedBodies(ctx, source.PlaysURL(season), func(_ string, body []byte, at time.Time) bool {
		hash := sha256.Sum256(append([]byte(weekly.sha+proofSHA), body...))
		sha := hex.EncodeToString(hash[:])
		weeks, err := s.completedWeeks(sha, weekly.weeks, body, proofs)
		if err != nil {
			parseErr = err
			return false
		}
		if weekly.at.After(at) {
			at = weekly.at
		}
		if proofAt.After(at) {
			at = proofAt
		}
		completed = heldFile{sha: sha, at: at, weeks: weeks, complete: true}
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("read held plays: %w", err)
	}
	if !found {
		if parseErr != nil {
			return nil, parseErr
		}
		return files, nil
	}
	return append([]heldFile{completed}, files...), nil
}

func (s *Store) completedWeeks(sha string, weekly []source.PlayerWeek, body []byte, proofs []source.RawAbsenceProof) ([]source.PlayerWeek, error) {
	s.weeklyMu.Lock()
	defer s.weeklyMu.Unlock()
	if file, ok := s.weekly[sha]; ok {
		return file.weeks, file.err
	}
	plays, err := source.ReadCompressedPlays(body)
	if err != nil {
		return nil, fmt.Errorf("read held compressed plays: %w", err)
	}
	weeks := make([]source.PlayerWeek, len(weekly))
	for i := range weekly {
		weeks[i] = clonePlayerWeek(weekly[i])
	}
	weeks, err = source.MissingPlayWeeks(weeks, plays)
	if err != nil {
		return nil, fmt.Errorf("build missing player weeks: %w", err)
	}
	if err := source.AttachPlays(weeks, plays); err != nil {
		return nil, fmt.Errorf("attach held plays: %w", err)
	}
	weeks, err = source.AddProvenZeros(weeks, plays, proofs)
	if err != nil {
		return nil, fmt.Errorf("verify empty player weeks: %w", err)
	}
	if s.weekly == nil {
		s.weekly = map[string]weeklyFile{}
	}
	s.weekly[sha] = weeklyFile{weeks: weeks}
	return weeks, nil
}

// selectHeldWeeks picks the player's week out of each held copy.
func selectHeldWeeks(scope source.RawStatScope, files []heldFile) []source.HeldWeek {
	out := make([]source.HeldWeek, 0, len(files))
	for _, f := range files {
		var held source.HeldWeek
		if f.index != nil {
			held = f.index.Select(scope, f.weeks, f.sha, f.at)
		} else {
			held = source.SelectHeldWeek(scope, f.weeks, f.sha, f.at)
		}
		if f.complete && held.Week != nil {
			held.Missing = held.Missing[1:]
		}
		if held.Week != nil {
			week := clonePlayerWeek(*held.Week)
			held.Week = &week
		}
		out = append(out, held)
	}
	return out
}

func (s *Store) NativePlayerScore(ctx context.Context, league string,
	scope source.RawNativeScope) (commissioner.NativePlayerScore, error) {
	if err := scope.Validate(); err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("read player score: %w", err)
	}
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativePlayerScore{}, err
	}
	return s.scorePlayer(ctx, league, records, scope)
}

// scorePlayer scores one player under the given ledger. An unseeded league reads no NFL statistics, so
// an ordinal season is never taken for an NFL one.
func (s *Store) scorePlayer(ctx context.Context, league string, records []commissioner.Record,
	scope source.RawNativeScope) (commissioner.NativePlayerScore, error) {
	evidence := source.NativeEvidence{}
	if nativeSeeded(records) {
		var err error
		if evidence, err = s.nativeEvidence(ctx, s, scope); err != nil {
			return commissioner.NativePlayerScore{}, fmt.Errorf("read native score evidence: %w", err)
		}
	}
	reading, err := commissioner.NativePlayerScoreReading(league, scope, records, evidence, s.now())
	if err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("read native score: %w", err)
	}
	return reading, nil
}

func nativeSeeded(records []commissioner.Record) bool {
	for _, record := range records {
		if record.Command.Command == commissioner.NativeSeed {
			return true
		}
	}
	return false
}

// nativeEvidence reads the held NFL evidence for a player's week from stats; tests replace it with supplied
// evidence.
func (s *Store) nativeEvidence(ctx context.Context, stats source.StatPort,
	scope source.RawNativeScope) (source.NativeEvidence, error) {
	pinned, isPinned := s.lineupEnv.(*PinnedFootball)
	if _, card := nativecheck.LineCardTeam(scope.Player); card {
		return lineCardEvidence(ctx, pinned, isPinned, scope.Season)
	}
	if isPinned {
		out, err := source.ReadNativeEvidence(ctx, pinned, pinned, scope)
		if err != nil {
			return out, fmt.Errorf("read pinned native evidence: %w", err)
		}
		return s.withScoringPosition(out, scope.Player), nil
	}
	if s.evidence != nil {
		return s.evidence(ctx, scope)
	}
	out, err := source.ReadNativeEvidence(ctx, s, stats, scope)
	if err != nil {
		return out, fmt.Errorf("read held native evidence: %w", err)
	}
	return s.withScoringPosition(out, scope.Player), nil
}

func (s *Store) withScoringPosition(evidence source.NativeEvidence, player string) source.NativeEvidence {
	if s.lineupEnv != nil {
		evidence.ListedPosition = string(s.lineupEnv.PlayerFacts()[player].Position)
	}
	return evidence
}

// lineCardEvidence reads an O-Line card's evidence: the season's held line games. A card has no NFL
// statistics identity; only pinned evidence with advanced rushing holds line games.
func lineCardEvidence(ctx context.Context, pinned *PinnedFootball, isPinned bool, season int) (source.NativeEvidence, error) {
	if !isPinned {
		return source.NativeEvidence{}, nil
	}
	held, ok, err := pinned.LineGames(ctx, season)
	if err != nil || !ok {
		return source.NativeEvidence{}, err
	}
	return source.NativeEvidence{OLine: &source.HeldLineGames{SHA256: held.SHA, FetchedAt: held.At, Games: held.Games}}, nil
}

type weeklyFile struct {
	weeks []source.PlayerWeek
	err   error
}

func (s *Store) decodedWeeks(sha string, body []byte) ([]source.PlayerWeek, error) {
	s.weeklyMu.Lock()
	defer s.weeklyMu.Unlock()
	if file, ok := s.weekly[sha]; ok {
		return file.weeks, file.err
	}
	weeks, err := source.ReadWeeks(bytes.NewReader(body))
	if err != nil {
		err = fmt.Errorf("decode weekly statistics: %w", err)
	}
	if s.weekly == nil {
		s.weekly = map[string]weeklyFile{}
	}
	s.weekly[sha] = weeklyFile{weeks: weeks, err: err}
	return weeks, err
}

func clonePlayerWeek(week source.PlayerWeek) source.PlayerWeek {
	week.Line.Quantities = maps.Clone(week.Line.Quantities)
	week.Line.Labels = maps.Clone(week.Line.Labels)
	week.Line.Events = maps.Clone(week.Line.Events)
	for stat, events := range week.Line.Events {
		week.Line.Events[stat] = slices.Clone(events)
	}
	return week
}

// HeldWeeks is the newest held copy of an NFL season's weekly statistics and that copy's sha.
// held is empty, with no error, when the season has not been loaded: a valuation that reads it
// runs on the seasons still flowing (R11). It exists so a league valued from its own scoring
// settings reads exactly the weekly lines the projections read.
func (s *Store) HeldWeeks(ctx context.Context, season int) (held []source.PlayerWeek, sha string, err error) {
	err = s.readWeeklyFiles(ctx, season, func(hash string, _ time.Time, weeks []source.PlayerWeek) bool {
		held, sha = weeks, hash
		return true
	})
	if errors.Is(err, source.ErrSeasonNotLoaded) {
		return nil, "", nil
	}
	return held, sha, err
}
