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

	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
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
	a.seasonMu.Lock()
	prev := a.now.firsts
	a.seasonMu.Unlock()
	next, err := readScoreFirsts(ctx, prev, a.season, a.history.ArchivedBodies)
	if err != nil {
		a.fallbackLog("the war room: " + err.Error())
		return
	}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	a.now.firsts = next
}

// refreshNowWeek reads the chosen week's NFL schedule and the season's byes, each live or from
// the archive, unless held and fresh; force always reads them.
func (a *App) refreshNowWeek(ctx context.Context, week int, force bool) {
	pause := a.seasonPause
	if pause == nil {
		pause = seasonPause
	}
	year := strconv.Itoa(a.season)
	a.seasonMu.Lock()
	held, byes := a.now.schedules[week], a.now.byes
	a.seasonMu.Unlock()
	if force || !feedFresh(held.fetchedAt, held.refreshError) {
		rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.season, a.scheduleFeed(week), a.fallbackLog)
		a.seasonMu.Lock()
		schedules := maps.Clone(a.now.schedules)
		if schedules == nil {
			schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{}
		}
		h := schedules[week]
		holdSeasonFeed(&h, rows, fresh, err)
		schedules[week] = h
		a.now.schedules = schedules
		a.seasonMu.Unlock()
		if err := pause(ctx, time.Second); err != nil {
			return
		}
	}
	if !force && byes.refreshError == "" && time.Since(byes.fetchedAt) < 24*time.Hour {
		return
	}
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[nflschedule.RawBye]{
		export: nflschedule.ByeExport, leagueless: true,
		fetch: func(ctx context.Context) ([]nflschedule.RawBye, error) {
			return nflschedule.FetchByes(ctx, a.mflClient, year)
		},
		parse: nflschedule.ParseByes, bodies: a.history.ArchivedBodies,
	}, a.fallbackLog)
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	holdSeasonFeed(&a.now.byes, rows, fresh, err)
}

func (a *App) scheduleFeed(week int) archivedFeed[nflschedule.RawWeek] {
	year := strconv.Itoa(a.season)
	return archivedFeed[nflschedule.RawWeek]{
		export: nflschedule.Export, leagueless: true,
		fetch: func(ctx context.Context) ([]nflschedule.RawWeek, error) {
			raw, ferr := nflschedule.Fetch(ctx, a.mflClient, year, week)
			if ferr != nil {
				return nil, fmt.Errorf("NFL schedule: %w", ferr)
			}
			return checkScheduleWeek(raw, week)
		},
		parse: func(body []byte) ([]nflschedule.RawWeek, error) {
			raw, perr := nflschedule.Parse(body)
			if perr != nil {
				return nil, fmt.Errorf("NFL schedule: %w", perr)
			}
			return checkScheduleWeek(raw, week)
		},
		bodies: a.scopedBodies("W", strconv.Itoa(week)),
	}
}

func (a *App) seedClockSchedules(ctx context.Context) {
	a.seedCurrentWeek(ctx)
	for _, rule := range leagueclock.Rules(a.lastRegularWeek()) {
		if rule.Week == 0 {
			continue
		}
		rows, at, found, err := readArchivedFeed(ctx, a.season, a.scheduleFeed(rule.Week))
		if err != nil {
			a.fallbackLog(err.Error())
			continue
		}
		if !found {
			continue
		}
		a.seasonMu.Lock()
		if a.now.schedules == nil {
			a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{}
		}
		a.now.schedules[rule.Week] = heldSeasonFeed[[]nflschedule.RawWeek]{value: rows, fetchedAt: at}
		a.seasonMu.Unlock()
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
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	a.refreshScoreFirsts(ctx)
	if week == 0 {
		week = a.marketWeek()
	}
	a.seasonMu.Lock()
	listed := false
	for _, w := range a.race.weeks.value {
		listed = listed || w.Week == week
	}
	a.seasonMu.Unlock()
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

// TargetNow reads held data only. week is the week shown (0: the current NFL week) and mine the
// GM's franchise ("": none chosen); Go checks both.
func (a *App) TargetNow(week, reportWeek int, mine string) (NowBoard, error) {
	if err := a.ready(); err != nil {
		return NowBoard{}, fmt.Errorf("now: startup: %w", err)
	}
	current := a.marketWeek()
	projections := a.heldMarketForWeek(current)[marketfeeds.Projections]
	a.seasonMu.Lock()
	held, now, live := a.race, a.now, a.seasonLineups
	a.seasonMu.Unlock()
	a.lookupMu.Lock()
	lookup, hasLookup := a.lookup, a.hasLookup
	a.lookupMu.Unlock()
	board, err := buildNow(nowInput{
		race: raceInput{held: held, config: a.rulebook.ActiveConfig(), lookup: lookup, hasLookup: hasLookup,
			mine: mine, zone: time.Local},
		held: now, live: live, projections: projections, at: a.timeNow(), current: current, week: week, reportWeek: reportWeek,
	})
	if err == nil && held.weeks.fetchedAt.IsZero() && held.weeks.refreshError == "" {
		board.Waiting = "Reading the league's weeks from MFL…"
	}
	return board, err
}
