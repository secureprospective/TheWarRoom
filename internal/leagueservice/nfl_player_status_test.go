package leagueservice

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

type nflInjuryFeedFunc func(context.Context) (NFLInjuryFeed, error)

func (f nflInjuryFeedFunc) ReadNFLInjuryFeed(ctx context.Context) (NFLInjuryFeed, error) {
	return f(ctx)
}

// ResolveNFLReturnWeeks stands in for the server adapter: a return date on a week's kickoff day is that week.
func (nflInjuryFeedFunc) ResolveNFLReturnWeeks(rows []leaguefeed.Injury, weeks []leagueweek.Week) []leaguefeed.Injury {
	out := slices.Clone(rows)
	for i := range out {
		for _, week := range weeks {
			if len(week.Games) > 0 && week.Games[0].Kickoff.Format("Jan 2, 2006") == out[i].ExpReturn {
				out[i].ReturnSeason, out[i].ReturnWeek = week.Games[0].Kickoff.Year(), week.Number
			}
		}
	}
	return out
}

type nflStatusTestHost struct {
	admissionHost
	now   time.Time
	weeks []leagueweek.Week
}

func (h nflStatusTestHost) CommissionerNow() time.Time { return h.now }

func (h nflStatusTestHost) NativeScheduleDates(context.Context, int, time.Time) ([]leagueweek.Week, error) {
	return h.weeks, nil
}

func TestTargetNFLPlayerStatusIsMemberSafeAndLabelsStaleFacts(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	fetched := now.Add(-time.Minute)
	id, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	feed := NFLInjuryFeed{Season: 2026, FetchedAt: fetched, LastAttemptAt: fetched, Available: true,
		Injuries: []leaguefeed.Injury{{ID: id, Status: "Out", Details: "Ankle - PCL", ExpReturn: "Oct 15, 2026"}}}
	weeks := make([]leagueweek.Week, 6)
	for i, date := range []string{"2026-09-10", "2026-09-17", "2026-09-24", "2026-10-01", "2026-10-08", "2026-10-15"} {
		kickoff, err := time.Parse("2006-01-02", date)
		if err != nil {
			t.Fatal(err)
		}
		weeks[i] = leagueweek.Week{Number: i + 1, Games: []leagueweek.Game{{Teams: [2]string{"A", "B"}, Kickoff: kickoff}}}
	}
	var admitted commissioner.Access
	service := NewHosted(nflStatusTestHost{admissionHost: admissionHost{}, now: now, weeks: weeks}, admissionFunc(func(_ context.Context, access commissioner.Access) error {
		admitted = access
		return nil
	})).WithNFLInjuryFeed(nflInjuryFeedFunc(func(context.Context) (NFLInjuryFeed, error) { return feed, nil }))
	reading, err := service.TargetNFLPlayerStatus("native.test")
	if err != nil {
		t.Fatal(err)
	}
	if admitted.League != "native.test" || admitted.Operation != "nfl.player-status.read" || admitted.Mode != "read" {
		t.Fatalf("wrong admission scope: %+v", admitted)
	}
	if reading.State != "available" || reading.FetchedAt == nil || !reading.FetchedAt.Equal(fetched) || len(reading.Players) != 1 {
		t.Fatalf("status report not held: %+v", reading)
	}
	fact := reading.Players[0]
	if fact.Player != "0099" || fact.Designation != "Out" || fact.BodyPart != "Ankle" || fact.ExpectedReturnWeek != 6 {
		t.Fatalf("wrong status facts: %+v", fact)
	}
	body, err := json.Marshal(reading)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "Oct 15") || strings.Contains(string(body), "PCL") || strings.Contains(string(body), "news") {
		t.Fatalf("report exposed source prose instead of facts: %s", body)
	}

	feed.LastAttemptAt = fetched.Add(time.Minute)
	reading, err = service.TargetNFLPlayerStatus("native.test")
	if err != nil || reading.State != "stale" || reading.FetchedAt == nil || !reading.FetchedAt.Equal(fetched) || len(reading.Players) != 1 {
		t.Fatalf("failed refresh did not preserve and mark the last good report: %+v, %v", reading, err)
	}
}

func TestTargetNFLPlayerStatusUnavailableAndDenied(t *testing.T) {
	service := NewHosted(nflStatusTestHost{}, admissionFunc(func(context.Context, commissioner.Access) error { return nil })).
		WithNFLInjuryFeed(nflInjuryFeedFunc(func(context.Context) (NFLInjuryFeed, error) {
			return NFLInjuryFeed{Season: 2026, Injuries: []leaguefeed.Injury{}}, nil
		}))
	reading, err := service.TargetNFLPlayerStatus("native.test")
	if err != nil || reading.State != "unavailable" || reading.FetchedAt != nil || reading.Players == nil || len(reading.Players) != 0 {
		t.Fatalf("missing report was not explicitly unavailable: %+v, %v", reading, err)
	}

	denied := errors.New("league member required")
	outsider := NewHosted(nflStatusTestHost{}, admissionFunc(func(_ context.Context, access commissioner.Access) error {
		if access.Operation != "nfl.player-status.read" || access.League != "native.test" {
			return errors.New("wrong denial scope")
		}
		return denied
	})).WithNFLInjuryFeed(nflInjuryFeedFunc(func(context.Context) (NFLInjuryFeed, error) {
		t.Fatal("denied reader reached the feed")
		return NFLInjuryFeed{}, nil
	}))
	if _, err := outsider.TargetNFLPlayerStatus("native.test"); !errors.Is(err, denied) {
		t.Fatalf("non-member read was not refused: %v", err)
	}
	if _, err := service.TargetNFLPlayerStatus("bad league"); err == nil {
		t.Fatal("invalid league accepted")
	}
}
