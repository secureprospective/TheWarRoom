package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
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
func (a *App) TargetDraftReports() (DraftReportBoard, error) {
	if err := a.ready(); err != nil {
		return DraftReportBoard{}, fmt.Errorf("draft reports: startup: %w", err)
	}
	a.seasonMu.Lock()
	results, future := a.seasonDraftResults, a.seasonFuturePicks
	a.seasonMu.Unlock()
	a.lookupMu.Lock()
	lookup := a.lookup
	a.lookupMu.Unlock()
	names := map[string]string{}
	for _, p := range results.value {
		if p.Player != nil {
			if f, ok := lookup.Facts(p.Player.String()); ok {
				names[p.Player.String()] = f.Name
			}
		}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	values, valueNote := a.draftReportValues(ctx)
	next := "Next draft: no dated commissioner calendar entry is held."
	if a.whatif != nil {
		clock, err := a.TargetClock()
		if err != nil {
			next = "Next draft date unavailable: " + err.Error()
		} else {
			for _, d := range clock.Value.Deadlines {
				if d.Label == "DRAFT" && d.At != nil && d.At.After(time.Now()) {
					next = "Next draft: " + d.At.Local().Format("2006-01-02 15:04:05 MST")
					break
				}
			}
		}
	}
	board := buildDraftReports(results, future, names, a.rulebook.FranchiseNames(), values, valueNote, next, time.Local)
	if a.mflClient != nil && a.mflClient.LeagueHost() != "" {
		board.MFLURL = fmt.Sprintf("https://%s/%d/options?L=%s&O=17", a.mflClient.LeagueHost(), a.season, url.QueryEscape(ingestion.LeagueID))
	}
	return board, nil
}

func (a *App) draftReportValues(ctx context.Context) (map[string]float64, string) {
	values := map[string]float64{}
	run, ok, err := a.history.LatestRun(ctx, a.season, history.RunModel)
	if err != nil {
		return values, "Model values unavailable: " + err.Error()
	}
	if !ok {
		return values, "No model run is held; player values are unavailable."
	}
	scores, err := a.history.ModelScores(ctx, run.ID)
	if err != nil {
		return values, "Model values unavailable: " + err.Error()
	}
	for _, s := range scores {
		values[s.MFLID] = s.DynastyPPG
	}
	return values, fmt.Sprintf("Value: held model run %d, as of %s; %s; missing measures: %v.", run.ID,
		run.AsOf.Local().Format("2006-01-02 15:04:05 MST"), run.Engine, run.MissingMeasures)
}

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
