package history

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"time"

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
	authority := s.live.authority
	if line.Provenance != "" {
		authority = line.Provenance
	}
	digest := source.LineDigest(line)
	sum := sha256.Sum256([]byte(authority + ":" + digest))
	return source.HeldWeek{Authority: authority, SHA256: hex.EncodeToString(sum[:]),
		FetchedAt: s.live.at, Week: &line, Missing: append([]string{}, s.live.missing...)}, true
}

// withLiveStats adds the live revision of a player's week as the newest version of the evidence.
// It is appended last, so the sequence numbers of the versions a commissioner may already have
// pinned never move. Which version counts stays the league's decision: the feed ladder prefers the
// final authority when one is held, and a commissioner's choice overrides both.
func (s *Store) withLiveStats(evidence source.NativeEvidence, scope source.RawNativeScope) source.NativeEvidence {
	live, ok := s.liveHeldWeek(scope.Player, scope.Season, scope.Week)
	if !ok {
		return evidence
	}
	versions := evidence.Versions()
	digest := source.LineDigest(*live.Week)
	if n := len(versions); n != 0 && versions[n-1].Digest == digest && versions[n-1].Held.Authority == live.Authority {
		versions[n-1].Held = live
	} else {
		versions = append(versions, source.LineRevision{Seq: len(versions), Digest: digest, Held: live})
	}
	evidence.Lines = versions
	evidence.Held = &live
	evidence.Missing = append([]string{}, live.Missing...)
	return evidence
}
