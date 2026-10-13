package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
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
	next := "Next draft: no date set by the commissioner."
	if a.whatif != nil {
		clock, err := a.TargetClock()
		if err != nil {
			next = "Next draft date unavailable: " + err.Error()
		} else {
			next = nextDraftDate(clock.Value.Windows, time.Now(), time.Local)
		}
	}
	board := buildDraftReports(a.league.Season(), results, future, names, a.rulebook.FranchiseNames(), values, valueNote, next, time.Local)
	if a.mflClient != nil && a.mflClient.LeagueHost() != "" {
		board.MFLURL = fmt.Sprintf("https://%s/%d/options?L=%s&O=17", a.mflClient.LeagueHost(), a.season, url.QueryEscape(ingestion.LeagueID))
	}
	return board, nil
}

func (a *App) draftReportValues(ctx context.Context) (map[string]float64, string) {
	values := map[string]float64{}
	run, ok, err := a.history.LatestRun(ctx, a.season, history.RunModel)
	if err != nil {
		return values, "Engine points unavailable: " + err.Error()
	}
	if !ok {
		return values, "Engine points unavailable; use Score League to value players."
	}
	scores, err := a.history.ModelScores(ctx, run.ID)
	if err != nil {
		return values, "Engine points unavailable: " + err.Error()
	}
	for _, s := range scores {
		values[s.MFLID] = s.DynastyPPG
	}
	return values, a.draftValueNote(run)
}

func (a *App) draftValueNote(run history.Run) string {
	note := "Engine run: " + run.AsOf.In(time.Local).Format("January 2, 2006, 3:04 PM") + "."
	missing := []string{}
	for _, m := range a.describeMeasures(run.MissingMeasures) {
		if m.Meaning != "" {
			missing = append(missing, strings.TrimSuffix(m.Meaning, "."))
		} else {
			missing = append(missing, "an unknown player measure")
		}
	}
	if len(missing) > 0 {
		note += " Missing player information: " + strings.Join(missing, "; ") + "."
	}
	return note
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

func nextDraftDate(windows []leagueclock.Window, now time.Time, zone *time.Location) string {
	for _, w := range windows {
		if w.Kind == leagueclock.Draft && w.At != nil && w.At.After(now) {
			return "Next draft: " + w.At.In(zone).Format("January 2, 2006, 3:04 PM")
		}
	}
	return "Next draft: no date set by the commissioner."
}
