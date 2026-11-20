package leagueservice

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

const nflPlayerStatusMaxAge = 4 * time.Hour

// NFLInjuryFeed is the validated, persisted public report and its fetch chronology.
type NFLInjuryFeed struct {
	Season        int
	FetchedAt     time.Time
	LastAttemptAt time.Time
	Injuries      []leaguefeed.Injury
	Available     bool
}

// NFLInjuryFeedSource reads the latest validated report from held reference storage.
type NFLInjuryFeedSource interface {
	ReadNFLInjuryFeed(context.Context) (NFLInjuryFeed, error)
	// ResolveNFLReturnWeeks maps each expected return date onto the league's pinned NFL weeks.
	ResolveNFLReturnWeeks(rows []leaguefeed.Injury, weeks []leagueweek.Week) []leaguefeed.Injury
}

// NFLPlayerStatusFact contains only public status facts for one player.
type NFLPlayerStatusFact struct {
	Player             string `json:"player"`
	Designation        string `json:"designation"`
	BodyPart           string `json:"bodyPart,omitempty"`
	ExpectedReturnWeek int    `json:"expectedReturnWeek,omitempty"`
}

// NFLPlayerStatus is the member-safe NFL-wide report, with freshness kept separate from the facts.
type NFLPlayerStatus struct {
	League    string                `json:"league"`
	State     string                `json:"state"`
	Season    int                   `json:"season,omitempty"`
	FetchedAt *time.Time            `json:"fetchedAt"`
	Players   []NFLPlayerStatusFact `json:"players"`
}

// WithNFLInjuryFeed supplies the shared public status report held by the hosted server.
func (a *Service) WithNFLInjuryFeed(source NFLInjuryFeedSource) *Service {
	a.injuryFeed = source
	return a
}

// TargetNFLPlayerStatus reads public NFL status facts for any admitted league member.
func (a *Service) TargetNFLPlayerStatus(league string) (NFLPlayerStatus, error) {
	if err := (RawLeagueScheduleScope{League: league}).Validate(); err != nil {
		return NFLPlayerStatus{}, fmt.Errorf("NFL player status input: %w", err)
	}
	if err := a.admitRead(league, "nfl.player-status.read", ""); err != nil {
		return NFLPlayerStatus{}, err
	}
	if err := a.host.Ready(); err != nil {
		return NFLPlayerStatus{}, fmt.Errorf("NFL player status startup: %w", err)
	}
	reading := NFLPlayerStatus{League: league, State: "unavailable", Players: []NFLPlayerStatusFact{}}
	if a.injuryFeed == nil {
		return reading, nil
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	feed, err := a.injuryFeed.ReadNFLInjuryFeed(ctx)
	if err != nil {
		return NFLPlayerStatus{}, fmt.Errorf("read NFL injury reference: %w", err)
	}
	return a.reportNFLPlayerStatus(ctx, reading, feed), nil
}

func (a *Service) reportNFLPlayerStatus(ctx context.Context, reading NFLPlayerStatus, feed NFLInjuryFeed) NFLPlayerStatus {
	if !feed.Available || feed.FetchedAt.IsZero() {
		return reading
	}
	reading.Season = feed.Season
	fetchedAt := feed.FetchedAt.UTC()
	reading.FetchedAt = &fetchedAt
	reading.State = "available"
	now := a.host.CommissionerNow()
	if feed.LastAttemptAt.After(feed.FetchedAt) || now.Sub(feed.FetchedAt) > nflPlayerStatusMaxAge {
		reading.State = "stale"
	}
	weeks, scheduleErr := a.host.NativeScheduleDates(ctx, feed.Season, now)
	if scheduleErr != nil {
		weeks = nil
	}
	for _, injury := range a.injuryFeed.ResolveNFLReturnWeeks(feed.Injuries, weeks) {
		fact := NFLPlayerStatusFact{Player: injury.ID.String(), Designation: injury.Status, BodyPart: injuryBodyPart(injury.Details)}
		if injury.ReturnSeason == feed.Season {
			fact.ExpectedReturnWeek = injury.ReturnWeek
		}
		reading.Players = append(reading.Players, fact)
	}
	slices.SortFunc(reading.Players, func(left, right NFLPlayerStatusFact) int {
		if left.Player < right.Player {
			return -1
		}
		if left.Player > right.Player {
			return 1
		}
		return 0
	})
	return reading
}

func injuryBodyPart(details string) string {
	bodyPart, _, _ := strings.Cut(strings.TrimSpace(details), " - ")
	return strings.TrimSpace(bodyPart)
}
