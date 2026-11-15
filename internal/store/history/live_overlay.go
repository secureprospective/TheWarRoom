package history

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sync/atomic"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/deepclone"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// LiveOverlay holds the current week and the season's finished weeks. Published generations are
// detached and never mutated. A finished week is held so its results survive the NFL week rolling
// over: the pinned football evidence carries no statistics, so without it last week's games would
// read "No result yet" the moment the board moved on. It deliberately does not enter the desktop's
// durable live chronology.
type LiveOverlay struct {
	current atomic.Pointer[LiveGeneration]
	closed  atomic.Pointer[map[[2]int]*LiveGeneration]
}

// LiveView is one reading's captured overlay: the current week and every finished week, as they
// stood when the reading began.
type LiveView struct {
	Current *LiveGeneration
	Closed  map[[2]int]*LiveGeneration
}

// Week is the generation that scores season/week: the current one when it is that week, otherwise
// the finished week held for it, otherwise nil.
func (v LiveView) Week(season, week int) *LiveGeneration {
	if v.Current != nil && v.Current.Season == season && v.Current.Week == week {
		return v.Current
	}
	return v.Closed[[2]int{season, week}]
}

// Key names the view for single-flight sharing: any new generation, current or finished, is a new key.
func (v LiveView) Key() string {
	if v.Current == nil && len(v.Closed) == 0 {
		return ""
	}
	key := ""
	if v.Current != nil {
		key = v.Current.ID
	}
	for _, scope := range slices.SortedFunc(maps.Keys(v.Closed), func(a, b [2]int) int {
		return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1]))
	}) {
		key += fmt.Sprintf("|%d/%d:%s", scope[0], scope[1], v.Closed[scope].ID)
	}
	return key
}

type LiveIdentity struct {
	GSIS    string
	Sleeper string
}

type LiveGeneration struct {
	Sources    map[string]time.Time
	Reduced    bool
	Identities map[string]LiveIdentity
	ID         string
	Season     int
	Week       int
	At         time.Time
	Authority  string
	Missing    []string
	Lines      map[string]source.PlayerWeek
}

// Validate bounds the detached current-week candidate before durable publication.
func (candidate LiveGeneration) Validate() error {
	if candidate.ID == "" || candidate.At.IsZero() || candidate.Season < 1920 || candidate.Week < 1 || candidate.Week > 22 || len(candidate.Lines) > 6000 || len(candidate.Identities) > 6000 || len(candidate.Sources) > 20 {
		return fmt.Errorf("invalid or oversized live generation")
	}

	size := 0
	for _, line := range candidate.Lines {
		if err := candidate.validateLine(line); err != nil {
			return err
		}
		body, err := json.Marshal(line)
		if err != nil {
			return fmt.Errorf("measure live line: %w", err)
		}
		size += len(body)
		if size > 16<<20 {
			return fmt.Errorf("live lines exceed 16 MiB")
		}
	}

	body, err := json.Marshal(candidate)
	if err != nil {
		return fmt.Errorf("measure live generation: %w", err)
	}
	if len(body) > 16<<20 {
		return fmt.Errorf("live generation exceeds 16 MiB")
	}
	return nil
}

func (candidate LiveGeneration) validateLine(line source.PlayerWeek) error {
	if line.Season != candidate.Season || line.Week != candidate.Week || len(line.Line.Quantities) > 300 || len(line.Line.Events) > 300 {
		return fmt.Errorf("live line exceeds scope or field bounds")
	}
	for _, events := range line.Line.Events {
		if len(events) > 1000 {
			return fmt.Errorf("live events exceed bounds")
		}
	}
	return nil
}

// LiveEpoch returns the current public score identity. Reading replies carry their captured identity instead.
func (s *Store) LiveEpoch() string {
	if generation := s.liveSnapshot().Current; generation != nil {
		return generation.ID
	}
	return ""
}

func (o *LiveOverlay) Publish(candidate LiveGeneration) error {
	if err := candidate.Validate(); err != nil {
		return err
	}
	held := deepclone.Of(candidate)
	o.current.Store(&held)
	return nil
}

// PublishClosed holds a finished week's generation, replacing any earlier one for that week. The
// current week is never held here: a week the board still shows is published with Publish.
func (o *LiveOverlay) PublishClosed(candidate LiveGeneration) error {
	if err := candidate.Validate(); err != nil {
		return err
	}
	held := deepclone.Of(candidate)
	for {
		old := o.closed.Load()
		next := map[[2]int]*LiveGeneration{}
		if old != nil {
			maps.Copy(next, *old)
		}
		next[[2]int{held.Season, held.Week}] = &held
		if o.closed.CompareAndSwap(old, &next) {
			return nil
		}
	}
}

// Closed reports whether a finished week is held for season/week.
func (o *LiveOverlay) Closed(season, week int) bool {
	held := o.closed.Load()
	return held != nil && (*held)[[2]int{season, week}] != nil
}

// SetLiveOverlay is a startup-only installation, shared by all league pin siblings.
func (s *Store) SetLiveOverlay(overlay *LiveOverlay) { s.overlay = overlay }

// View captures the current week and every finished week as they stand now.
func (o *LiveOverlay) View() LiveView {
	view := LiveView{Current: o.current.Load()}
	if closed := o.closed.Load(); closed != nil {
		view.Closed = *closed
	}
	return view
}

func (s *Store) liveSnapshot() LiveView {
	if s.overlay == nil {
		return LiveView{}
	}
	return s.overlay.View()
}

func (n *nativeSources) liveKey() string { return n.overlay.Key() }

// weekKey names the generation one week is scored from, so a new current generation leaves every
// finished week's cached scores alone.
func (n *nativeSources) weekKey(season, week int) string {
	if g := n.overlay.Week(season, week); g != nil {
		return g.ID
	}
	return ""
}

func (s *Store) overlayEvidence(evidence source.NativeEvidence, scope source.RawNativeScope, generation *LiveGeneration) source.NativeEvidence {
	if generation == nil || generation.Season != scope.Season || generation.Week != scope.Week {
		return evidence
	}
	line, ok := generation.Lines[scope.Player]
	if !ok {
		return evidence
	}
	if pinned, yes := s.lineupEnv.(*PinnedFootball); yes {
		if pinned.officialWeek(scope.Week) {
			return evidence // official statistics are final; a live line would only reopen them
		}
		player, exists := pinned.raw.Players[scope.Player]
		identity := generation.Identities[scope.Player]
		gsis := player.nflIdentity()
		if !exists || (gsis != "" && identity.GSIS != gsis) || (player.Sleeper != "" && identity.Sleeper != player.Sleeper) {
			return evidence
		}
	}
	line = clonePlayerWeek(line)
	leaguerules.DropTotals(line.Line.Quantities)
	live := heldLiveLine(line, generation.Authority, generation.At, generation.Missing)
	versions := append([]source.LineRevision{}, evidence.Versions()...)
	versions = append(versions, source.LineRevision{Seq: len(versions), Digest: source.LiveRevisionDigest(line), Held: live})
	evidence.Lines = source.ConsolidateFinal(versions)
	if len(evidence.Lines) > len(versions) {
		merged := &evidence.Lines[len(evidence.Lines)-1]
		merged.Held.Missing = append(append([]string{}, merged.Held.Missing...), generation.Missing...)
	}
	evidence.Held = &live
	evidence.Missing = append([]string{}, live.Missing...)
	return evidence
}

func (s *Store) readingLiveStats(ctx context.Context, evidence source.NativeEvidence, scope source.RawNativeScope, stats source.StatPort) (source.NativeEvidence, error) {
	if s.overlay == nil {
		return s.withLiveStats(ctx, evidence, scope)
	}
	view := s.liveSnapshot()
	if reading, ok := stats.(*nativeSources); ok {
		view = reading.overlay
	}
	return s.overlayEvidence(evidence, scope, view.Week(scope.Season, scope.Week)), nil
}

// scoreEvidence describes the live generation a reading of season/week was scored from: that week's
// own generation when it is held, else the current one (a season-wide reading passes week 0). A
// finished week is never stale: its games are over, only not yet official.
func (n *nativeSources) scoreEvidence(season, week int) *commissioner.ScoreEvidence {
	g := n.overlay.Week(season, week)
	finished := g != nil && g != n.overlay.Current
	if g == nil {
		g = n.overlay.Current
	}
	if g == nil {
		return nil
	}
	return &commissioner.ScoreEvidence{Live: g.ID, Season: g.Season, Week: g.Week, AsOf: g.At, Source: g.Authority, Provisional: true, Sources: maps.Clone(g.Sources), Reduced: g.Reduced, Stale: !finished && n.now.Sub(g.At) > 10*time.Minute}
}
