package draft

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// Host supplies detached shared feeds and composes cross-feature clock/value reads.
// Refresh serialization stays with the shared feed owner.
type Host interface {
	DraftReady() (context.Context, int, *time.Location, error)
	DraftInputs() (report.Held[draftreports.Pick], report.Held[draftreports.FuturePick], normalize.Lookup)
	DraftLeague() (int, map[string]string, *time.Location)
	DraftValues(context.Context) (map[string]float64, string)
	DraftNext() string
	DraftDestination() string
	DraftPlayers(context.Context) report.Held[players.RawPlayer]
	DraftFacts() (*lab.Data, string, error)
	RefreshDraftReports(bool) error
	OpenDraftRoom() error
}

type API struct{ host Host }

func New(host Host) *API { return &API{host: host} }

// LabReading is concrete so Wails discovers the nested surface models.
type LabReading report.Reading[report.Surface]

func (a *API) TargetDraftReportsRefresh(force bool) error {
	return a.host.RefreshDraftReports(force) //nolint:wrapcheck // Preserve the shared refresh refusal exactly.
}

// TargetDraftReports reads held exports and model scores, never fetching names or scores.
func (a *API) TargetDraftReports() (report.Board, error) {
	parent, _, _, err := a.host.DraftReady()
	if err != nil {
		return report.Board{}, fmt.Errorf("draft reports: startup: %w", err)
	}
	results, future, lookup := a.host.DraftInputs()
	names := Names(results.Value, func(id string) (string, bool) {
		f, ok := lookup.Facts(id)
		return f.Name, ok
	})
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	values, valueNote := a.host.DraftValues(ctx)
	next := a.host.DraftNext()
	season, franchises, zone := a.host.DraftLeague()
	board := Build(Input{Season: season, Results: Held[draftreports.Pick]{Value: results.Value, Feed: results.Feed},
		Future: Held[draftreports.FuturePick]{Value: future.Value, Feed: future.Feed}, Names: names,
		Franchises: franchises, Values: values, ValueNote: valueNote, Next: next, Zone: zone})
	if target := a.host.DraftDestination(); target != "" {
		board.MFLURL = target
	}
	return board, nil
}

func (a *API) TargetOpenMFLDraftRoom() error {
	return a.host.OpenDraftRoom() //nolint:wrapcheck // The host preserves readiness and destination check order.
}

// TargetNFLDraft reads the held detailed players export; it never fetches.
func (a *API) TargetNFLDraft() (report.Report, error) {
	parent, season, zone, err := a.host.DraftReady()
	if err != nil {
		return report.Report{}, fmt.Errorf("NFL draft: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	held := a.host.DraftPlayers(ctx)
	return NFL(season, Held[players.RawPlayer]{Value: held.Value, Feed: held.Feed}, zone), nil
}

// Commands records existing bindings without inventing publication metadata.
func Commands() []catalog.Command {
	return []catalog.Command{
		{ID: catalog.Unset, Method: "TargetDraftLab", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetDraftReports", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetDraftReportsRefresh", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetNFLDraft", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: "draft.reports", Method: "TargetOpenMFLDraftRoom", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "desktop"},
	}
}
