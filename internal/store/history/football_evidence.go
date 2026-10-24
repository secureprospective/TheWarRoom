package history

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// PinnedFootball is immutable after loading. Install it through SetLineupEnvironment
// at startup; replacing an import never silently changes a running authority's facts.
type PinnedFootball struct {
	digest string
	raw    RawFootballEvidence
}

// LoadFootball refuses missing or changed bodies, never substitutes the latest fetch.
func (s *Store) LoadFootball(ctx context.Context, digest string) (*PinnedFootball, error) {
	if len(digest) != 64 {
		return nil, fmt.Errorf("choose the football evidence digest")
	}
	body, err := s.Body(ctx, digest)
	if err != nil {
		return nil, fmt.Errorf("load pinned football evidence: %w", err)
	}
	if footballDigest(body) != digest {
		return nil, fmt.Errorf("held football evidence digest mismatch")
	}
	raw, err := decodeFootball(body)
	if err != nil {
		return nil, err
	}
	return &PinnedFootball{digest: digest, raw: raw}, nil
}

func (p *PinnedFootball) Digest() string { return p.digest }

// CheckDigest ties a request to the server's explicitly selected evidence.
func (p *PinnedFootball) CheckDigest(digest string) error {
	if digest == "" || digest != p.digest {
		return fmt.Errorf("reread the pinned football evidence before making this change")
	}
	return nil
}

func (p *PinnedFootball) PlayerFacts() nativecheck.Facts {
	facts := make(nativecheck.Facts, len(p.raw.Players))
	for id, player := range p.raw.Players {
		facts[id] = player.Facts
	}
	return facts
}

// Identities returns source identities keyed by canonical player id, never account aliases.
func (p *PinnedFootball) Identities() map[string]string {
	identities := make(map[string]string, len(p.raw.Players))
	for id, player := range p.raw.Players {
		identities[id] = player.Identity
	}
	return identities
}

func (p *PinnedFootball) LineupFacts(ctx context.Context, season, week int) (commissioner.LineupFacts, error) {
	weeks, err := p.ScheduleDates(ctx, season)
	if err != nil {
		return commissioner.LineupFacts{}, err
	}
	for _, held := range weeks {
		if held.Number == week {
			return commissioner.LineupFacts{Facts: p.PlayerFacts(), Locks: leagueweek.Locks(held), ScheduleHeld: true, Zone: time.UTC}, nil
		}
	}
	return commissioner.LineupFacts{}, fmt.Errorf("pinned football evidence does not hold NFL week %d", week)
}

func (p *PinnedFootball) ScheduleDates(ctx context.Context, season int) ([]leagueweek.Week, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("read pinned football schedule: %w", err)
	}
	if season != p.raw.Season {
		return nil, fmt.Errorf("pinned football evidence does not hold season %d", season)
	}
	weeks := make([]leagueweek.Week, len(p.raw.Weeks))
	for i, week := range p.raw.Weeks {
		weeks[i] = leagueweek.Week{Number: week.Number, Games: append([]leagueweek.Game(nil), week.Games...)}
	}
	return weeks, nil
}
