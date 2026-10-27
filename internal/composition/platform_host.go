package composition

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/headshot"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// platformHost is the platform API's Host. It is never bound: only platform.API exposes methods
// to Wails. Each method is the former binding's body, with its locks and error text unchanged.
type platformHost struct{ app *App }

// platformAPI is the platform API over this App; root callers use it, never an App wrapper.
func (a *App) platformAPI() *platformapp.API { return platformapp.New(platformHost{app: a}) }

func (h platformHost) Snapshot() (snapshot.Snapshot, error) { return h.app.takeSnapshot() }

func (h platformHost) Clock() (snapshot.Sourced[leagueclock.Reading], error) {
	return h.app.targetClock(h.app.core.Ctx)
}

func (h platformHost) DraftRosterMove(intent, franchiseID, playerID string) (envelope.Receipt, error) {
	return h.app.draftRosterMove(intent, franchiseID, playerID)
}

func (h platformHost) Moves(franchiseID string) ([]envelope.Receipt, error) {
	return h.app.listMoves(franchiseID)
}

func (h platformHost) HandOff(correlationID string) (envelope.Receipt, error) {
	return h.app.handOffMove(correlationID)
}

func (h platformHost) CheckMoves() error { return h.app.requestMovesPass() }

func (h platformHost) WithdrawRoster(correlationID string) (envelope.Receipt, error) {
	return h.app.withdrawRosterMove(correlationID)
}

func (h platformHost) PlayerName(mflID string) (string, bool) {
	h.app.core.LookupMu.Lock()
	defer h.app.core.LookupMu.Unlock()
	return h.app.core.Lookup.Name(mflID)
}

func (h platformHost) PhotosOff() bool { return h.app.core.PhotosOff.Load() }

// photoTimeout bounds one request to a photo host; a dead host never holds a screen (R11).
const photoTimeout = 10 * time.Second

// startHeadshots builds the photo cache in the data directory. Photos are personal use only
// (spec §7): never bundled, never in history.db.
func (a *App) startHeadshots() {
	(platformapp.PhotoRuntime{
		Available: func() bool { return a.core.History != nil },
		SourceID: func(ctx context.Context, id, source string) (string, error) {
			return a.core.History.SourceID(ctx, id, source)
		},
		Install: func(ids func(context.Context, string) (string, string, error)) {
			a.core.Headshots.Store(headshot.New(filepath.Join(a.core.Paths.DataDir, "headshots"), ids, &http.Client{Timeout: photoTimeout, Transport: a.core.Network}))
		}, Settings: func() (bool, error) { settings, err := a.readSettings(); return settings.PhotosOff, err }, SetOff: a.core.PhotosOff.Store,
	}).Initialize()
}

// photoHandler serves /headshot/<mfl id> to the webview from the cache, fetching on first use.
func (a *App) photoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		photos := a.core.Headshots.Load()
		if photos == nil {
			http.Error(w, "photos are not ready", http.StatusServiceUnavailable)
			return
		}
		photos.Handler(func() bool { return !a.core.PhotosOff.Load() }).ServeHTTP(w, r)
	})
}
