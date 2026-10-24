package source

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// LineRevision is one version of a player's regular-season week as the held files reported it. Versions
// are numbered from 0 in the order the feed gave them; a refetch that changed nothing is not a version, but
// a return to an earlier line is a new one. Held is the earliest file that carried this version.
type LineRevision struct {
	Seq    int
	Digest string
	Held   HeldWeek
}

// WeekKey names a player's week in a feed keyed by GSIS.
type WeekKey struct {
	GSIS string
	Week int
}

// LineDigest identifies what scoring reads from a player's week: the position and every recorded
// quantity. Two lines with equal digests count identically under any rules.
func LineDigest(week PlayerWeek) string {
	body, err := json.Marshal(struct {
		Position   string
		Quantities map[leaguerules.StatID]leaguerules.Decimal
		Events     map[leaguerules.StatID][]leaguerules.Decimal
	}{week.Position, week.Line.Quantities, week.Line.Events})
	if err != nil {
		// Maps of strings to integers always encode; this guards a future field type.
		panic("source: line digest: " + err.Error())
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// lineHistory collects one player-week's versions while files are offered newest first.
type lineHistory struct{ newestFirst []LineRevision }

func (h *lineHistory) add(week PlayerWeek, sha string, at time.Time) {
	h.addHeld(HeldWeek{SHA256: sha, FetchedAt: at, Week: &week, Missing: []string{missingPlayByPlay}})
}

func (h *lineHistory) addHeld(held HeldWeek) {
	digest := LineDigest(*held.Week)
	if n := len(h.newestFirst); n != 0 && h.newestFirst[n-1].Digest == digest {
		h.newestFirst[n-1].Held = held
		return
	}
	h.newestFirst = append(h.newestFirst, LineRevision{Digest: digest, Held: held})
}

func (h *lineHistory) versions() []LineRevision {
	out := slices.Clone(h.newestFirst)
	slices.Reverse(out)
	for i := range out {
		out[i].Seq = i
	}
	return out
}

// DistinctLines turns held files, newest first, into the versions of one player's week, oldest first. A
// file that does not report the player adds nothing: absence is not a version.
func DistinctLines(newestFirst []HeldWeek) []LineRevision {
	var history lineHistory
	for _, held := range newestFirst {
		if held.Week != nil {
			history.addHeld(held)
		}
	}
	return history.versions()
}

// LineBook gathers the versions of every regular-season player week in one season's files.
type LineBook struct {
	season  int
	history map[WeekKey]*lineHistory
}

func NewLineBook(season int) *LineBook {
	return &LineBook{season: season, history: map[WeekKey]*lineHistory{}}
}

// Add offers one held file's weeks. Files must be added newest first.
func (b *LineBook) Add(sha string, at time.Time, weeks []PlayerWeek) {
	for _, week := range weeks {
		if week.Season != b.season || week.SeasonType != "REG" || week.Week < 1 || week.Week > 22 {
			continue
		}
		key := WeekKey{GSIS: week.GSIS, Week: week.Week}
		if b.history[key] == nil {
			b.history[key] = &lineHistory{}
		}
		b.history[key].add(week, sha, at)
	}
}

// Versions is every player week's versions, oldest first.
func (b *LineBook) Versions() map[WeekKey][]LineRevision {
	out := make(map[WeekKey][]LineRevision, len(b.history))
	for key, history := range b.history {
		out[key] = history.versions()
	}
	return out
}
