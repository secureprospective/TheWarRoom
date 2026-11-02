package history

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// liveStatBook is the newest live stat lines the app holds for one NFL week, by league player id.
// A live line is evidence like any other: it becomes the newest revision of a player's week, so the
// league's own score ladder decides what counts. Nothing here is written to the database: a game in
// progress is read again every poll, and only the final feed is archived.
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
func (s *Store) SetLiveStats(season, week int, at time.Time, authority string, missing []string,
	lines map[string]source.PlayerWeek) {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	if len(lines) == 0 {
		s.live = liveStatBook{}
		return
	}
	lines = maps.Clone(lines)
	for player, line := range lines {
		if line.Season == 0 {
			line.Season = season
		}
		if line.Week == 0 {
			line.Week = week
		}
		line.SeasonType = "REG"
		lines[player] = line
		if s.liveHeld == nil {
			s.liveHeld = map[liveStatKey]source.HeldWeek{}
		}
		s.liveHeld[liveStatKey{player, line.Season, line.Week}] = heldLiveLine(line, authority, at, missing)
	}
	s.live = liveStatBook{season: season, week: week, at: at, authority: authority,
		missing: append([]string{}, missing...), lines: lines}
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
	return heldLiveLine(line, s.live.authority, s.live.at, s.live.missing), true
}

// withLiveStats adds the live revision of a player's week as the newest version of the evidence.
// It is appended last, so the sequence numbers of the versions a commissioner may already have
// pinned never move. Which version counts stays the league's decision: the feed ladder prefers the
// final authority when one is held, and a commissioner's choice overrides both.
func (s *Store) withLiveStats(evidence source.NativeEvidence, scope source.RawNativeScope) source.NativeEvidence {
	live, ok := s.liveHeldWeek(scope.Player, scope.Season, scope.Week)
	if !ok {
		live, ok = s.finalLiveWeek(evidence, scope)
	}
	if !ok {
		return evidence
	}
	evidence.Lines = s.mergeLiveVersions(evidence.Versions(), live, scope)
	evidence.Held = &live
	evidence.Missing = append([]string{}, live.Missing...)
	return evidence
}

// The approval worker must see exactly the same live/merged revisions as score readings.
func (s *Store) withLiveStatWeeks(weeks []commissioner.StatWeek, season int) []commissioner.StatWeek {
	type key struct {
		player string
		week   int
	}
	seen := map[key]bool{}
	for i := range weeks {
		scope := source.RawNativeScope{Player: weeks[i].Player, Season: season, Week: weeks[i].Week}
		weeks[i].Evidence = s.withLiveStats(weeks[i].Evidence, scope)
		seen[key{scope.Player, scope.Week}] = true
	}
	s.liveMu.Lock()
	players := make(map[string]int, len(s.live.lines))
	for player, line := range s.live.lines {
		if line.Season == season {
			players[player] = line.Week
		}
	}
	s.liveMu.Unlock()
	for player, week := range players {
		if seen[key{player, week}] {
			continue
		}
		scope := source.RawNativeScope{Player: player, Season: season, Week: week}
		evidence := s.withLiveStats(source.NativeEvidence{}, scope)
		weeks = append(weeks, commissioner.StatWeek{Player: player, Week: week, Evidence: evidence})
	}
	return weeks
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
	return held, ok
}
