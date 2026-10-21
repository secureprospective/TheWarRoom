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
	results, rf, re := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[draftreports.Pick]{
		Export: draftreports.ResultsExport,
		Fetch: func(ctx context.Context) ([]draftreports.Pick, error) {
			return draftreports.FetchResults(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
		}, Parse: draftreports.ParseResults, Bodies: a.core.History.ArchivedBodies,
	}, a.core.FallbackLog)
	a.core.SeasonMu.Lock()
	holdSeasonFeed(&a.core.SeasonDraftResults, results, rf, re)
	a.core.SeasonMu.Unlock()
	pause := a.core.SeasonPause
	if pause == nil {
		pause = seasonPause
	}
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	future, ff, fe := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[draftreports.FuturePick]{
		Export: draftreports.FutureExport,
		Fetch: func(ctx context.Context) ([]draftreports.FuturePick, error) {
			return draftreports.FetchFuture(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
		}, Parse: draftreports.ParseFuture, Bodies: a.core.History.ArchivedBodies,
	}, a.core.FallbackLog)
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	holdSeasonFeed(&a.core.SeasonFuturePicks, future, ff, fe)
}

func (a *App) TargetDraftReportsRefresh(force bool) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("draft refresh: startup: %w", err)
	}
	a.core.SeasonMu.Lock()
	results, future := a.core.SeasonDraftResults, a.core.SeasonFuturePicks
	a.core.SeasonMu.Unlock()
	if !force && results.RefreshError == "" && future.RefreshError == "" &&
		time.Since(results.FetchedAt) < poolFreshFor && time.Since(future.FetchedAt) < poolFreshFor {
		return nil
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	a.refreshDraftReports(ctx)
	return nil
}

// TargetDraftReports reads held exports and held model scores; it never fetches names or scores.
func (a *App) TargetDraftReports() (ReportBoard, error) {
	if err := a.ready(); err != nil {
		return ReportBoard{}, fmt.Errorf("draft reports: startup: %w", err)
	}
	a.core.SeasonMu.Lock()
	results, future := a.core.SeasonDraftResults, a.core.SeasonFuturePicks
	a.core.SeasonMu.Unlock()
	a.core.LookupMu.Lock()
	lookup := a.core.Lookup
	a.core.LookupMu.Unlock()
	names := draftapp.Names(results.Value, func(id string) (string, bool) {
		f, ok := lookup.Facts(id)
		return f.Name, ok
	})
	ctx, cancel := context.WithTimeout(a.core.Ctx, m4Timeout)
	defer cancel()
	values, valueNote := a.draftReportValues(ctx)
	next := "Next draft: no date set by the commissioner."
	if a.core.Whatif != nil {
		clock, err := a.TargetClock()
		if err != nil {
			next = "Next draft date unavailable: " + err.Error()
		} else {
			next = draftapp.NextDate(clock.Value.Windows, time.Now(), time.Local)
		}
	}
	board := buildDraftReports(a.core.League.Season(), results, future, names, a.core.Rulebook.FranchiseNames(), values, valueNote, next, time.Local)
	if a.core.MFLClient != nil && a.core.MFLClient.LeagueHost() != "" {
		board.MFLURL = fmt.Sprintf("https://%s/%d/options?L=%s&O=17", a.core.MFLClient.LeagueHost(), a.core.Season, url.QueryEscape(ingestion.LeagueID))
	}
	return board, nil
}

// draftReportValues reads the engine values the draft and pool screens show, through the service.
func (a *App) draftReportValues(ctx context.Context) (map[string]float64, string) {
	return draftapp.Values(ctx, a.core.History, a.core.Season, a.measureMeanings, time.Local)
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
	if a.core.MFLClient == nil || a.core.MFLClient.LeagueHost() == "" {
		return fmt.Errorf("open draft room: MFL host not held; refresh first")
	}
	target := fmt.Sprintf("https://%s/%d/options?L=%s&O=17", a.core.MFLClient.LeagueHost(), a.core.Season, url.QueryEscape(ingestion.LeagueID))
	a.core.OpenURL(a.core.Ctx, target)
	return nil
}

// TargetNFLDraft reads the held detailed players export; it never fetches.
func (a *App) TargetNFLDraft() (Report, error) {
	if err := a.ready(); err != nil {
		return Report{}, fmt.Errorf("NFL draft: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, m4Timeout)
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
	return nflDraftReport(a.core.Season, held), nil
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
	return draftapp.Held[T]{Value: h.Value, Feed: h.FeedState()}
}
