package history

import (
	"context"
	"errors"
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
	stats  []heldFile
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
	raw, err := decodeFootball(body, s.storeParts(ctx))
	if err != nil {
		return nil, err
	}
	stats, err := raw.scoringFiles(s.storeParts(ctx))
	if err != nil {
		return nil, err
	}
	return &PinnedFootball{digest: digest, raw: raw, stats: stats}, nil
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
		f := player.Facts
		if player.NFLRosterSeasons != nil {
			f.NFLSeasons, f.NFLSeasonsHeld = len(*player.NFLRosterSeasons), true
		}
		facts[id] = f
	}
	nativecheck.AddLineCards(facts)
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

// BidEnvironment uses the selected schedule, never the host's wall-clock week number.
// Playoff eligibility remains unknown unless league evidence supplies it.
func (p *PinnedFootball) BidEnvironment(ctx context.Context, _ string, at time.Time) (commissioner.BidEnvironment, error) {
	weeks, err := p.ScheduleDates(ctx, p.raw.Season)
	if err != nil {
		return commissioner.BidEnvironment{}, err
	}
	week, ok := leagueweek.LineupWeek(at, weeks...)
	if !ok {
		return commissioner.BidEnvironment{}, nil
	}
	number := week.Number
	if number == 1 && at.Before(leagueweek.FirstLock(week)) {
		number = 0
	}
	return commissioner.BidEnvironment{Season: nativecheck.BidSeason{Week: number, WeekKnown: true}}, nil
}

// SetPinnedFootball installs one immutable evidence selection for lineups and bids at startup.
func (s *Store) SetPinnedFootball(evidence *PinnedFootball) {
	s.SetLineupEnvironment(evidence)
	s.SetBidEnvironment(pinnedBidEnvironment{store: s, evidence: evidence})
}

// ErrBidSeasonEvidence marks selected football evidence that is not for the league's current season.
// Bids are refused; a bid reading still opens and says why.
var ErrBidSeasonEvidence = errors.New("select football evidence for the league's current season before bidding")

type pinnedBidEnvironment struct {
	store    *Store
	evidence *PinnedFootball
}

func (p pinnedBidEnvironment) BidEnvironment(ctx context.Context, league string, at time.Time) (commissioner.BidEnvironment, error) {
	assets, err := p.store.NativeAssets(ctx, league)
	if err != nil {
		return commissioner.BidEnvironment{}, fmt.Errorf("read bid season: %w", err)
	}
	if p.evidence == nil || assets.Snapshot == nil || assets.Snapshot.Season != p.evidence.raw.Season {
		return commissioner.BidEnvironment{}, ErrBidSeasonEvidence
	}
	return p.evidence.BidEnvironment(ctx, league, at)
}
