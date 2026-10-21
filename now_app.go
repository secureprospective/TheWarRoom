package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"time"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// nowHeld is League Pulse › Now's own held data: the NFL schedule of each week opened, the
// season's bye weeks, and each week's first held scores, which official stat changes compare with.
type nowHeld struct {
	schedules map[int]heldSeasonFeed[[]nflschedule.RawWeek]
	byes      heldSeasonFeed[[]nflschedule.RawBye]
	firsts    scoreFirsts
}

// scoreFirsts is, per week, the oldest archived weekly results that had the week scored. through
// is the newest copy already read, so a later read only looks at newer copies.
type scoreFirsts struct {
	weeks   map[int]firstScores
	through time.Time
	copies  int
}

type firstScores struct {
	at   time.Time
	week leaguefeed.WeekResult
}

// archivedBodies is the history store's ArchivedBodies, as a seam for tests.
type archivedBodies func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error)

// readScoreFirsts reads archived season results newer than prev.through, newest first, and keeps
// for each week not already held the oldest copy that had it scored.
func readScoreFirsts(ctx context.Context, prev scoreFirsts, season int, bodies archivedBodies) (scoreFirsts, error) {
	next := scoreFirsts{weeks: maps.Clone(prev.weeks), through: prev.through, copies: prev.copies}
	if next.weeks == nil {
		next.weeks = map[int]firstScores{}
	}
	var lastSum [sha256.Size]byte
	var last []leaguefeed.WeekResult
	_, err := bodies(ctx, "TYPE="+weeklyresults.Export, func(src string, body []byte, at time.Time) bool {
		if !at.After(prev.through) {
			return true
		}
		u, err := url.Parse(src)
		if err != nil || u.Query().Get("W") != "YTD" || !sameExport(src, weeklyresults.Export, season, false) {
			return false
		}
		if sum := sha256.Sum256(body); last == nil || !bytes.Equal(sum[:], lastSum[:]) {
			weeks, perr := weeklyresults.Parse(body)
			if perr != nil {
				return false
			}
			last, lastSum = weeks, sum
		}
		next.copies++
		if at.After(next.through) {
			next.through = at
		}
		for _, w := range last {
			if _, held := prev.weeks[w.Week]; !held && weekScored(w) {
				next.weeks[w.Week] = firstScores{at: at, week: w}
			}
		}
		return false
	})
	if err != nil {
		return prev, fmt.Errorf("official stat changes: %w", err)
	}
	return next, nil
}

func (a *App) refreshScoreFirsts(ctx context.Context) {
	a.core.SeasonMu.Lock()
	prev := a.now.firsts
	a.core.SeasonMu.Unlock()
	next, err := readScoreFirsts(ctx, prev, a.core.Season, a.core.History.ArchivedBodies)
	if err != nil {
		a.core.FallbackLog("the war room: " + err.Error())
		return
	}
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	a.now.firsts = next
}

// refreshNowWeek reads the chosen week's NFL schedule and the season's byes, each live or from
// the archive, unless held and fresh; force always reads them.
func (a *App) refreshNowWeek(ctx context.Context, week int, force bool) {
	pause := a.core.SeasonPause
	if pause == nil {
		pause = seasonPause
	}
	year := strconv.Itoa(a.core.Season)
	a.core.SeasonMu.Lock()
	held, byes := a.now.schedules[week], a.now.byes
	a.core.SeasonMu.Unlock()
	if force || !feedFresh(held.FetchedAt, held.RefreshError) {
		rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, a.scheduleFeed(week), a.core.FallbackLog)
		a.core.SeasonMu.Lock()
		schedules := maps.Clone(a.now.schedules)
		if schedules == nil {
			schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{}
		}
		h := schedules[week]
		holdSeasonFeed(&h, rows, fresh, err)
		schedules[week] = h
		a.now.schedules = schedules
		a.core.SeasonMu.Unlock()
		if err := pause(ctx, time.Second); err != nil {
			return
		}
	}
	if !force && byes.RefreshError == "" && time.Since(byes.FetchedAt) < 24*time.Hour {
		return
	}
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[nflschedule.RawBye]{
		Export: nflschedule.ByeExport, Leagueless: true,
		Fetch: func(ctx context.Context) ([]nflschedule.RawBye, error) {
			return nflschedule.FetchByes(ctx, a.core.MFLClient, year)
		},
		Parse: nflschedule.ParseByes, Bodies: a.core.History.ArchivedBodies,
	}, a.core.FallbackLog)
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	holdSeasonFeed(&a.now.byes, rows, fresh, err)
}

func (a *App) scheduleFeed(week int) archivedFeed[nflschedule.RawWeek] {
	year := strconv.Itoa(a.core.Season)
	return archivedFeed[nflschedule.RawWeek]{
		Export: nflschedule.Export, Leagueless: true,
		Fetch: func(ctx context.Context) ([]nflschedule.RawWeek, error) {
			raw, ferr := nflschedule.Fetch(ctx, a.core.MFLClient, year, week)
			if ferr != nil {
				return nil, fmt.Errorf("NFL schedule: %w", ferr)
			}
			return checkScheduleWeek(raw, week)
		},
		Parse: func(body []byte) ([]nflschedule.RawWeek, error) {
			raw, perr := nflschedule.Parse(body)
			if perr != nil {
				return nil, fmt.Errorf("NFL schedule: %w", perr)
			}
			return checkScheduleWeek(raw, week)
		},
		Bodies: a.scopedBodies("W", strconv.Itoa(week)),
	}
}

func (a *App) seedClockSchedules(ctx context.Context) {
	a.seedCurrentWeek(ctx)
	for _, rule := range leagueclock.Rules(a.lastRegularWeek()) {
		if rule.Week == 0 {
			continue
		}
		rows, at, found, err := readArchivedFeed(ctx, a.core.Season, a.scheduleFeed(rule.Week))
		if err != nil {
			a.core.FallbackLog(err.Error())
			continue
		}
		if !found {
			continue
		}
		a.core.SeasonMu.Lock()
		if a.now.schedules == nil {
			a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{}
		}
		a.now.schedules[rule.Week] = heldSeasonFeed[[]nflschedule.RawWeek]{Value: rows, FetchedAt: at}
		a.core.SeasonMu.Unlock()
	}
}

func checkScheduleWeek(raw nflschedule.RawWeek, week int) ([]nflschedule.RawWeek, error) {
	if raw.Week != strconv.Itoa(week) {
		return nil, fmt.Errorf("NFL schedule: requested week %d, received %s", week, raw.Week)
	}
	return []nflschedule.RawWeek{raw}, nil
}

func feedFresh(at time.Time, refreshError string) bool {
	return refreshError == "" && time.Since(at) < poolFreshFor
}

// TargetNowRefresh refreshes the league's weekly results, then the chosen week's NFL schedule
// (0: the current week), unless fresh; force always refreshes. Go checks the week is listed.
func (a *App) TargetNowRefresh(week int, force bool) error {
	if err := a.TargetRaceRefresh(force); err != nil {
		return fmt.Errorf("now refresh: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	a.refreshScoreFirsts(ctx)
	if week == 0 {
		week = a.marketWeek()
	}
	a.core.SeasonMu.Lock()
	listed := false
	for _, w := range a.race.weeks.Value {
		listed = listed || w.Week == week
	}
	a.core.SeasonMu.Unlock()
	// The current NFL week is always fetched: its schedule does not depend on the league's weeks being
	// held yet (just after launch they may not be). Any other week must be on the league's schedule.
	current := week == a.marketWeek()
	if !listed && !current {
		return fmt.Errorf("now refresh: week %d is not on the league's schedule", week)
	}
	if week < 1 {
		return nil
	}
	a.refreshNowWeek(ctx, week, force)
	return nil
}

// TargetNow reads held data only; refresh and scheduling remain owned by App.
func (a *App) TargetNow(week, reportWeek int, mine string) (NowBoard, error) {
	if err := a.ready(); err != nil {
		return NowBoard{}, fmt.Errorf("now: startup: %w", err)
	}
	in := a.nowReadInput(week, reportWeek, mine)
	board, err := nowapp.Build(in)
	out := wireNow(board)
	if err == nil && in.Race.Held.Weeks.FetchedAt.IsZero() && in.Race.Held.Weeks.RefreshError == "" {
		out.Waiting = "Reading the league's weeks from MFL…"
	}
	return out, err //nolint:wrapcheck // preserve binding validation text
}

// NowBoard is League Pulse › Now: the week shown, the GM's own matchup first, then the week's
// reports. Week is the week Go accepted.
type NowBoard struct {
	ReportWeek  int          `json:"reportWeek"`
	ReportWeeks []RaceChoice `json:"reportWeeks"`
	Week        int          `json:"week"`
	Weeks       []RaceChoice `json:"weeks"`
	Matchup     Report       `json:"matchup"`
	Board       ReportBoard  `json:"board"`
	// Waiting is set while the league's weeks have not been read yet (just after launch): the screen says so
	// instead of drawing an empty week. The season event re-reads the board when they arrive.
	Waiting string `json:"waiting"`
}
