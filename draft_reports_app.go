package main

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	draftapp "github.com/secureprospective/TheWarRoom/internal/app/draft"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
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
	next := "Next draft: no date set by the commissioner."
	if h.Whatif != nil {
		clock, err := h.app.TargetClock()
		if err != nil {
			next = "Next draft date unavailable: " + err.Error()
		} else {
			next = draftapp.NextDate(clock.Value.Windows, time.Now(), time.Local)
		}
	}
	return next
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

func countLabel(n int, noun string) string { return report.Count(n, noun) }
func countNumber(n int) string             { return report.Number(n) }
