package composition

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	draftapp "github.com/secureprospective/TheWarRoom/internal/app/draft"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// draftHost composes the clock and shared engine-value read without a feature import in core.
// It is never bound: only draft.API exposes methods to Wails.
type draftHost struct {
	*core.State
	app *App
}

func (h draftHost) DraftValues(ctx context.Context) (map[string]float64, string) {
	return h.app.draftReportValues(ctx)
}

func (h draftHost) DraftNext() string {
	return draftapp.NextDraft(h.Whatif != nil, func() (snapshot.Sourced[leagueclock.Reading], error) { return h.app.targetClock(h.app.core.Ctx) }, time.Now, time.Local)
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

func (h draftHost) DraftDestination() string {
	host, season := h.CommsDestination()
	return draftapp.Destination(host, season)
}

func (h draftHost) DraftPlayers(ctx context.Context) report.Held[players.RawPlayer] {
	return h.ReadDraftPlayers(ctx, func(body []byte) ([]players.RawPlayer, error) { return draftapp.ParsePlayers(ctx, body) })
}

func (h draftHost) OpenDraftRoom() error {
	return draftapp.OpenRoom(h.Ready, h.DraftDestination, h.OpenBrowser) //nolint:wrapcheck // Preserve startup and absent-host refusal order.
}

func (a *App) refreshDraftReportsFeeds(ctx context.Context) {
	draftapp.RefreshFeeds(ctx, func() func(context.Context, time.Duration) error { return a.core.SeasonPause }, a.core.RefreshDraftResults, a.core.RefreshFuturePicks)
}
func (h draftHost) RefreshDraftReports(force bool) error {
	//nolint:wrapcheck // Preserve refresh readiness errors.
	return draftapp.RefreshReports(force, h.Ready, func() (report.Feed, report.Feed) {
		h.SeasonMu.Lock()
		defer h.SeasonMu.Unlock()
		return h.SeasonDraftResults.FeedState(), h.SeasonFuturePicks.FeedState()
	}, func() context.Context { return h.Ctx }, func(ctx context.Context) {
		h.RefreshMu.Lock()
		defer h.RefreshMu.Unlock()
		h.app.refreshDraftReportsFeeds(ctx)
	})
}
