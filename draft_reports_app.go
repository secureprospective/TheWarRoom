package main

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"

	draftapp "github.com/secureprospective/TheWarRoom/internal/app/draft"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
)

func (a *App) refreshDraftReports(ctx context.Context) {
	results, rf, re := liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[draftreports.Pick]{
		export: draftreports.ResultsExport,
		fetch: func(ctx context.Context) ([]draftreports.Pick, error) {
			return draftreports.FetchResults(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID)
		}, parse: draftreports.ParseResults, bodies: a.history.ArchivedBodies,
	}, a.fallbackLog)
	a.seasonMu.Lock()
	holdSeasonFeed(&a.seasonDraftResults, results, rf, re)
	a.seasonMu.Unlock()
	pause := a.seasonPause
	if pause == nil {
		pause = seasonPause
	}
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	future, ff, fe := liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[draftreports.FuturePick]{
		export: draftreports.FutureExport,
		fetch: func(ctx context.Context) ([]draftreports.FuturePick, error) {
			return draftreports.FetchFuture(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID)
		}, parse: draftreports.ParseFuture, bodies: a.history.ArchivedBodies,
	}, a.fallbackLog)
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	holdSeasonFeed(&a.seasonFuturePicks, future, ff, fe)
}

func (a *App) TargetDraftReportsRefresh(force bool) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("draft refresh: startup: %w", err)
	}
	a.seasonMu.Lock()
	results, future := a.seasonDraftResults, a.seasonFuturePicks
	a.seasonMu.Unlock()
	if !force && results.refreshError == "" && future.refreshError == "" &&
		time.Since(results.fetchedAt) < poolFreshFor && time.Since(future.fetchedAt) < poolFreshFor {
		return nil
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	a.refreshDraftReports(ctx)
	return nil
}

// TargetDraftReports reads held exports and held model scores; it never fetches names or scores.
func (a *App) TargetDraftReports() (ReportBoard, error) {
	if err := a.ready(); err != nil {
		return ReportBoard{}, fmt.Errorf("draft reports: startup: %w", err)
	}
	a.seasonMu.Lock()
	results, future := a.seasonDraftResults, a.seasonFuturePicks
	a.seasonMu.Unlock()
	a.lookupMu.Lock()
	lookup := a.lookup
	a.lookupMu.Unlock()
	names := draftapp.Names(results.value, func(id string) (string, bool) {
		f, ok := lookup.Facts(id)
		return f.Name, ok
	})
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	values, valueNote := a.draftReportValues(ctx)
	next := "Next draft: no date set by the commissioner."
	if a.whatif != nil {
		clock, err := a.TargetClock()
		if err != nil {
			next = "Next draft date unavailable: " + err.Error()
		} else {
			next = draftapp.NextDate(clock.Value.Windows, time.Now(), time.Local)
		}
	}
	board := buildDraftReports(a.league.Season(), results, future, names, a.rulebook.FranchiseNames(), values, valueNote, next, time.Local)
	if a.mflClient != nil && a.mflClient.LeagueHost() != "" {
		board.MFLURL = fmt.Sprintf("https://%s/%d/options?L=%s&O=17", a.mflClient.LeagueHost(), a.season, url.QueryEscape(ingestion.LeagueID))
	}
	return board, nil
}

// draftReportValues reads the engine values the draft and pool screens show, through the service.
func (a *App) draftReportValues(ctx context.Context) (map[string]float64, string) {
	return draftapp.Values(ctx, a.history, a.season, a.measureMeanings, time.Local)
}

// measureMeanings is describeMeasures reduced to each measure's plain meaning.
func (a *App) measureMeanings(names []string) []string {
	described := a.describeMeasures(names)
	out := make([]string, len(described))
	for i, m := range described {
		out[i] = m.Meaning
	}
	return out
}

// engineRunText is report.EngineRun, the one wording for when the engine last valued players.
func engineRunText(at time.Time, zone *time.Location) string { return report.EngineRun(at, zone) }

func (a *App) TargetOpenMFLDraftRoom() error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("open draft room: %w", err)
	}
	if a.mflClient == nil || a.mflClient.LeagueHost() == "" {
		return fmt.Errorf("open draft room: MFL host not held; refresh first")
	}
	target := fmt.Sprintf("https://%s/%d/options?L=%s&O=17", a.mflClient.LeagueHost(), a.season, url.QueryEscape(ingestion.LeagueID))
	a.openURL(a.ctx, target)
	return nil
}

// TargetNFLDraft reads the held detailed players export; it never fetches.
func (a *App) TargetNFLDraft() (Report, error) {
	if err := a.ready(); err != nil {
		return Report{}, fmt.Errorf("NFL draft: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	held := archivedOnly(ctx, a, "players", false, func(body []byte) ([]players.RawPlayer, error) {
		rows, err := players.Parse(ctx, body)
		if err != nil {
			return nil, fmt.Errorf("NFL draft: players: %w", err)
		}
		if !slices.ContainsFunc(rows, func(p players.RawPlayer) bool { return p.DraftRound != "" && p.DraftPick != "" }) {
			return nil, fmt.Errorf("NFL draft: detailed selection fields not held")
		}
		return rows, nil
	})
	return nflDraftReport(a.season, held), nil
}

// nflDraftReport builds the NFL draft report in the feature service, as its wire type.
func nflDraftReport(season int, held heldSeasonFeed[[]players.RawPlayer]) Report {
	return wireReport(draftapp.NFL(season, draftHeld(held), time.Local))
}

// buildDraftReports builds the board in the feature service and converts it to its wire type.
func buildDraftReports(season int, results heldSeasonFeed[[]draftreports.Pick], future heldSeasonFeed[[]draftreports.FuturePick],
	names, franchises map[string]string, values map[string]float64, valueNote, next string, zone *time.Location) ReportBoard {
	return wireBoard(draftapp.Build(draftapp.Input{Season: season, Results: draftHeld(results), Future: draftHeld(future),
		Names: names, Franchises: franchises, Values: values, ValueNote: valueNote, Next: next, Zone: zone}))
}

func draftHeld[T any](h heldSeasonFeed[[]T]) draftapp.Held[T] {
	return draftapp.Held[T]{Value: h.value, Feed: h.state()}
}
