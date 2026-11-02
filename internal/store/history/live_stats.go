package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// liveStatBook is the newest live stat lines the app holds for one NFL week, by league player id.
// A live line is evidence like any other: it becomes the newest revision of a player's week, so the
// league's own score ladder decides what counts. Changed lines and exposed revisions are durable.
type liveStatBook struct {
	season, week int
	at           time.Time
	authority    string
	// missing is what the live source does not report, carried onto every line so a provisional
	// score says plainly what it is still waiting for.
	missing []string
	lines   map[string]source.PlayerWeek
}

type liveStatKey struct {
	player       string
	season, week int
}

// SetLiveStats replaces the live stat lines held for one NFL week. Lines are keyed by league player
// id. An empty map clears them, so a week that has finished stops offering a live revision.
func (s *Store) SetLiveStats(ctx context.Context, season, week int, at time.Time, authority string, missing []string,
	lines map[string]source.PlayerWeek) error {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	if err := s.restoreLiveStats(ctx); err != nil {
		return err
	}
	lines = deepClone(lines)
	for player, line := range lines {
		if line.Season == 0 {
			line.Season = season
		}
		if line.Week == 0 {
			line.Week = week
		}
		leaguerules.DropTotals(line.Line.Quantities)
		line.SeasonType = "REG"
		lines[player] = line
	}
	book := liveStatBook{season: season, week: week, at: at, authority: authority,
		missing: append([]string{}, missing...), lines: lines}
	if err := s.storeLiveBook(ctx, book); err != nil {
		return err
	}
	if s.liveHeld == nil {
		s.liveHeld = map[liveStatKey]source.HeldWeek{}
	}
	for player, line := range lines {
		s.liveHeld[liveStatKey{player, line.Season, line.Week}] = heldLiveLine(line, authority, at, missing)
		s.bumpLiveWeekLocked(line.Season, line.Week)
	}
	// The week the old book offered stops offering it.
	s.bumpLiveWeekLocked(s.live.season, s.live.week)
	s.bumpLiveWeekLocked(season, week)
	s.live = book
	return nil
}

// liveHeldWeek is the live revision of one player's week, when one is held for that exact week.
func (s *Store) liveHeldWeek(player string, season, week int) (source.HeldWeek, bool) {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	if s.live.lines == nil {
		return source.HeldWeek{}, false
	}
	line, ok := s.live.lines[player]
	if !ok || line.Season != season || line.Week != week {
		return source.HeldWeek{}, false
	}
	held, found := s.liveHeld[liveStatKey{player, season, week}]
	return deepClone(held), found
}

// withLiveStats adds the live revision of a player's week as the newest version of the evidence.
// It is appended last, so the sequence numbers of the versions a commissioner may already have
// pinned never move. Which version counts stays the league's decision: the feed ladder prefers the
// final authority when one is held, and a commissioner's choice overrides both.
func (s *Store) withLiveStats(ctx context.Context, evidence source.NativeEvidence, scope source.RawNativeScope) (source.NativeEvidence, error) {
	if err := s.RestoreLiveStats(ctx); err != nil {
		return evidence, err
	}
	live, ok := s.liveHeldWeek(scope.Player, scope.Season, scope.Week)
	if !ok {
		live, ok = s.finalLiveWeek(evidence, scope)
	}
	if !ok {
		return evidence, nil
	}
	versions, err := s.mergeLiveVersions(ctx, evidence.Versions(), live, scope)
	if err != nil {
		return evidence, err
	}
	evidence.Lines = versions
	evidence.Held = &live
	evidence.Missing = append([]string{}, live.Missing...)
	return evidence, nil
}

// The approval worker must see exactly the same live/merged revisions as score readings.
func (s *Store) withLiveStatWeeks(ctx context.Context, weeks []commissioner.StatWeek, season int) ([]commissioner.StatWeek, error) {
	if err := s.RestoreLiveStats(ctx); err != nil {
		return nil, err
	}
	type key struct {
		player string
		week   int
	}
	seen := map[key]bool{}
	for i := range weeks {
		scope := source.RawNativeScope{Player: weeks[i].Player, Season: season, Week: weeks[i].Week}
		evidence, err := s.withLiveStats(ctx, weeks[i].Evidence, scope)
		if err != nil {
			return nil, err
		}
		weeks[i].Evidence = evidence
		seen[key{scope.Player, scope.Week}] = true
	}
	s.liveMu.Lock()
	players := map[key]bool{}
	for heldKey := range s.liveHeld {
		if heldKey.season != season {
			continue
		}
		active, ok := s.live.lines[heldKey.player]
		if s.liveRevisions[heldKey] != nil || (ok && active.Season == season && active.Week == heldKey.week) {
			players[key{heldKey.player, heldKey.week}] = true
		}
	}
	s.liveMu.Unlock()
	for playerWeek := range players {
		player, week := playerWeek.player, playerWeek.week
		if seen[playerWeek] {
			continue
		}
		scope := source.RawNativeScope{Player: player, Season: season, Week: week}
		evidence, err := s.withLiveStats(ctx, source.NativeEvidence{}, scope)
		if err != nil {
			return nil, err
		}
		weeks = append(weeks, commissioner.StatWeek{Player: player, Week: week, Evidence: evidence})
	}
	slices.SortFunc(weeks, func(a, b commissioner.StatWeek) int {
		if a.Player != b.Player {
			return compareText(a.Player, b.Player)
		}
		return a.Week - b.Week
	})
	return weeks, nil
}

func heldLiveLine(line source.PlayerWeek, authority string, at time.Time, missing []string) source.HeldWeek {
	if line.Provenance != "" {
		authority = line.Provenance
	}
	digest := source.LiveRevisionDigest(line)
	sum := sha256.Sum256([]byte(authority + ":" + digest))
	return source.HeldWeek{Authority: authority, SHA256: hex.EncodeToString(sum[:]), FetchedAt: at, Week: &line, Missing: append([]string{}, missing...)}
}

func (s *Store) finalLiveWeek(evidence source.NativeEvidence, scope source.RawNativeScope) (source.HeldWeek, bool) {
	final := false
	for _, version := range evidence.Versions() {
		if version.Held.Authority == source.NFLVerse {
			final = true
			break
		}
	}
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	if !final && s.liveRevisions[liveStatKey{scope.Player, scope.Season, scope.Week}] == nil {
		return source.HeldWeek{}, false
	}
	held, ok := s.liveHeld[liveStatKey{scope.Player, scope.Season, scope.Week}]
	return deepClone(held), ok
}
