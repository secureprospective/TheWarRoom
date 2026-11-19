package leagueservice

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/injuries"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
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
	var schedule []nflschedule.RawWeek
	if scheduleErr == nil {
		schedule = injurySchedule(weeks)
	}
	for _, injury := range injuries.ResolveReturnWeeks(feed.Injuries, schedule) {
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

func injurySchedule(weeks []leagueweek.Week) []nflschedule.RawWeek {
	out := make([]nflschedule.RawWeek, 0, len(weeks))
	for _, week := range weeks {
		raw := nflschedule.RawWeek{Week: strconv.Itoa(week.Number), Matchups: make([]nflschedule.RawMatchup, 0, len(week.Games))}
		for _, game := range week.Games {
			raw.Matchups = append(raw.Matchups, nflschedule.RawMatchup{Kickoff: strconv.FormatInt(game.Kickoff.Unix(), 10)})
		}
		out = append(out, raw)
	}
	return out
}
