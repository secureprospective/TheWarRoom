package history

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sync/atomic"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/deepclone"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// LiveOverlay owns only the current week. Published generations are detached and never mutated.
// It deliberately does not enter the desktop's durable live chronology.
type LiveOverlay struct {
	current atomic.Pointer[LiveGeneration]
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
	if generation := s.liveSnapshot(); generation != nil {
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

// SetLiveOverlay is a startup-only installation, shared by all league pin siblings.
func (s *Store) SetLiveOverlay(overlay *LiveOverlay) { s.overlay = overlay }

func (s *Store) liveSnapshot() *LiveGeneration {
	if s.overlay == nil {
		return nil
	}
	return s.overlay.current.Load()
}

func (n *nativeSources) liveKey() string {
	if n.overlay == nil {
		return ""
	}
	return n.overlay.ID
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
	generation := s.liveSnapshot()
	if reading, ok := stats.(*nativeSources); ok {
		generation = reading.overlay
	}
	return s.overlayEvidence(evidence, scope, generation), nil
}

func (n *nativeSources) scoreEvidence() *commissioner.ScoreEvidence {
	if n.overlay == nil {
		return nil
	}
	g := n.overlay
	return &commissioner.ScoreEvidence{Live: g.ID, Season: g.Season, Week: g.Week, AsOf: g.At, Source: g.Authority, Provisional: true, Sources: maps.Clone(g.Sources), Reduced: g.Reduced, Stale: n.now.Sub(g.At) > 10*time.Minute}
}
