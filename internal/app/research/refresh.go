package research

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// FinishedScores keeps the first observed finish for a week, then rereads its scores once.
// Pin must merge with the latest held state under the caller's publication lock.
type FinishedScoreRefresh struct {
	Week     func() int
	Schedule func(context.Context, int)
	Held     func(int) time.Time
	Archive  func(context.Context, int) time.Time
	Cutoff   func(int) (time.Time, bool)
	Pin      func(int, time.Time) time.Time
	Pause    func() func(context.Context, time.Duration) error
	Refresh  func(context.Context, string)
}

func (s FinishedScoreRefresh) RefreshDue(ctx context.Context) {
	week := s.Week() - 1
	if week < 1 {
		return
	}
	s.Schedule(ctx, week)
	at := s.Held(week)
	if at.IsZero() {
		at = s.Archive(ctx, week)
	}
	end, finished := s.Cutoff(week)
	if !finished {
		return
	}
	end = s.Pin(week, end)
	if !at.Before(end) {
		return
	}
	pause := s.Pause()
	for _, period := range []string{strconv.Itoa(week), "YTD"} {
		if err := pause(ctx, time.Second); err != nil {
			return
		}
		s.Refresh(ctx, period)
	}
}

func PinFinished(values *Values, week int, end time.Time) time.Time {
	if values.FinishedAt == nil {
		values.FinishedAt = map[int]time.Time{}
	}
	if first := values.FinishedAt[week]; !first.IsZero() {
		return first
	}
	values.FinishedAt[week] = end
	return end
}

func RefreshFeed(ctx context.Context, held Feed[[]researchfeeds.Feed], force bool,
	read func(context.Context) ([]researchfeeds.Feed, domain.Freshness, error), publish func(Feed[[]researchfeeds.Feed])) {
	if !force && held.RefreshError == "" && time.Since(held.FetchedAt) < 5*time.Minute {
		return
	}
	rows, fresh, err := read(ctx)
	report.HoldFeed(&held, rows, fresh, err)
	publish(held)
}

func PublishFeed(values *Values, kind, week string, held Feed[[]researchfeeds.Feed]) {
	if values.Feeds == nil {
		values.Feeds = map[string]Feed[[]researchfeeds.Feed]{}
	}
	values.Feeds[Key(kind, week)] = held
}

func DepthLinks(ctx context.Context, lookup normalize.Lookup, source func(context.Context, string) (map[string]string, error)) (map[string]string, error) {
	out := map[string]string{}
	for _, kind := range []string{"gsis", "espn"} {
		sources, err := source(ctx, kind)
		if err != nil {
			return nil, fmt.Errorf("depth crosswalk: %w", err)
		}
		for _, id := range lookup.IDs() {
			if value := sources[id]; value != "" {
				out[kind+":"+value] = id
			}
		}
	}
	return out, nil
}

func RefreshDepth(ctx context.Context, held Feed[[]researchfeeds.RawDepth], force bool,
	read func(context.Context) ([]researchfeeds.RawDepth, domain.Freshness, error),
	links func(context.Context) (map[string]string, error),
	publish func(Feed[[]researchfeeds.RawDepth], map[string]string, error)) {
	if !force && held.RefreshError == "" && time.Since(held.FetchedAt) < 24*time.Hour {
		return
	}
	rows, fresh, err := read(ctx)
	linked, linkErr := links(ctx)
	report.HoldFeed(&held, rows, fresh, err)
	publish(held, linked, linkErr)
}

func PublishDepth(values *Values, held Feed[[]researchfeeds.RawDepth], links map[string]string, err error) {
	values.Depth = held
	if err != nil {
		values.LinkError = err.Error()
	} else {
		values.Links, values.LinkError = links, ""
	}
}

func PrepareDepth(available bool, parent func() context.Context, timeout time.Duration, refresh func(context.Context, bool)) {
	if !available {
		return
	}
	ctx, cancel := context.WithTimeout(parent(), timeout)
	defer cancel()
	refresh(ctx, false)
}

type RefreshSteps struct {
	Week     func() int
	Status   func(string)
	Pause    func(context.Context, time.Duration) error
	Feed     func(context.Context, string, string, bool)
	Injuries func(context.Context)
	Schedule func(context.Context, int, bool)
	Depth    func(context.Context, bool)
}

func (s RefreshSteps) Run(ctx context.Context, force bool) error {
	week := s.Week()
	for _, kind := range Kinds() {
		s.Status("Reading " + FeedLabel(kind))
		if err := s.Pause(ctx, time.Second); err != nil {
			return err
		}
		window := ""
		if kind == "playerScores" {
			window = "YTD"
		}
		s.readFeed(ctx, kind, window, force)
	}
	for w := 1; w <= week; w++ {
		s.Status(fmt.Sprintf("Reading week %d of %d", w, week))
		if err := s.Pause(ctx, time.Second); err != nil {
			return err
		}
		s.readFeed(ctx, "playerScores", strconv.Itoa(w), force)
	}
	s.Status("Reading NFL injuries")
	s.Injuries(ctx)
	if week > 0 {
		s.Status("Reading this week’s NFL schedule")
		s.Schedule(ctx, week, force)
	}
	s.Status("Reading NFL depth charts")
	s.Depth(ctx, force)
	return nil
}

func (s RefreshSteps) readFeed(parent context.Context, kind, week string, force bool) {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	s.Feed(ctx, kind, week, force)
}

func ReadDepth(ctx, fallback context.Context, season int, bodies ArchivedBodies,
	fetch func(context.Context) ([]researchfeeds.RawDepth, error), log func(string)) ([]researchfeeds.RawDepth, domain.Freshness, error) {
	rows, err := fetch(ctx)
	if err == nil {
		return rows, report.LiveFreshness(time.Now()), nil
	}
	log(fmt.Sprintf("the war room: depth charts: live fetch failed: %v", err))
	return ReadDepthArchive(fallback, season, bodies, err)
}

// NewsRefresh keeps roster capture and publication lazy across startup and admission.
type NewsRefresh struct {
	Ready   func() error
	Names   func() map[string]string
	Parent  func() context.Context
	TryLock func() bool
	Unlock  func()
	Players func(string) []string
	Held    func() map[string]Feed[[]researchfeeds.News]
	Read    func(context.Context, map[string]Feed[[]researchfeeds.News], []string) error
	Publish func(map[string]Feed[[]researchfeeds.News])
}

func (s NewsRefresh) Run(mine string) error {
	if mine != "" {
		if err := (RawResearchScope{Mine: mine}).Validate(); err != nil {
			return err
		}
	}
	if err := s.Ready(); err != nil {
		return fmt.Errorf("news refresh: %w", err)
	}
	if mine != "" && s.Names()[mine] == "" {
		return fmt.Errorf("news: unknown franchise")
	}
	ctx, cancel := context.WithTimeout(s.Parent(), 30*time.Minute)
	defer cancel()
	if !s.TryLock() {
		return fmt.Errorf("news: already refreshing")
	}
	defer s.Unlock()
	ordered := s.Players(mine)
	held := s.Held()
	if held == nil {
		held = map[string]Feed[[]researchfeeds.News]{}
	}
	err := s.Read(ctx, held, ordered)
	s.Publish(held)
	return err
}

// RefreshReading admits the request before capturing resources; refresh status is cleared
// while still serialized, before releasing the caller's refresh lock.
func RefreshReading(force bool, ready func() error, parent func() context.Context, status func(string),
	serialize func(func() error) error, steps func() RefreshSteps) error {
	if err := ready(); err != nil {
		return fmt.Errorf("research refresh: %w", err)
	}
	status("Waiting for the MFL refresh to finish")
	return serialize(func() error {
		ctx, cancel := context.WithTimeout(parent(), 2*time.Minute)
		defer cancel()
		defer status("")
		return steps().Run(ctx, force)
	})
}
