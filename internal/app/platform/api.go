// Package platform binds the app-wide reads every screen leans on: the target snapshot, the
// league clock, the move rail with its IR, taxi, hand-off and withdraw acts, and player photos.
package platform

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/mflref/planwatch"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// Host runs the snapshot, clock and move-log work against the shared stores, keeping each
// binding's original locks, startup check and error text.
type Host interface {
	Snapshot() (snapshot.Snapshot, error)
	Clock() (snapshot.Sourced[leagueclock.Reading], error)
	planwatch.Host
	// PlayerName is a player's directory name, if the directory has one.
	PlayerName(mflID string) (string, bool)
	PhotosOff() bool
}

// API is the platform binding. It holds no state.
type API struct {
	host Host
	*planwatch.API
}

func New(host Host) *API { return &API{host: host, API: planwatch.New(host)} }

// PhotoView is what a player card needs to show a photo: Go decides whether to ask for one and
// writes its alt text (W1). An empty Src means show the fallback without any request.
type PhotoView struct {
	Src string `json:"src"`
	Alt string `json:"alt"`
}

// Photo is one player's photo under a prose name, or none when photos are off.
func Photo(mflID, name string, off bool) PhotoView {
	if off {
		return PhotoView{Alt: name}
	}
	return PhotoView{Src: "/headshot/" + mflID, Alt: "Photo of " + name}
}

// PhotoRoutes passes only /headshot/ paths to the photo cache; every other path the window asks
// for is 404 from the first request. Startup runs alongside the page load on Linux, and the
// shell's native probe (GET /desktop/native) reads anything but 404 as a broken startup.
func PhotoRoutes(photos http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/headshot/") {
			http.NotFound(w, r)
			return
		}
		photos.ServeHTTP(w, r)
	})
}

func (a *API) TargetSnapshot() (snapshot.Snapshot, error) {
	return a.host.Snapshot() //nolint:wrapcheck // Preserve the binding's existing error text.
}

// TargetClock reads the phase log and commissioner calendar from the local store; no network.
func (a *API) TargetClock() (snapshot.Sourced[leagueclock.Reading], error) {
	return a.host.Clock() //nolint:wrapcheck // Preserve the binding's existing error text.
}

// TargetPhoto is a player card's photo: its address and alt text, or no address when photos are
// off or the player has none on file.
func (a *API) TargetPhoto(mflID string) PhotoView {
	name, ok := a.host.PlayerName(mflID)
	if !ok {
		name = "this player"
	}
	return Photo(mflID, report.ProseName(name), a.host.PhotosOff())
}

// Commands is the platform catalog.
func Commands() []catalog.Command { return catalog.DesktopCommands("platform", nil) }

// PhotoRuntime resolves IDs lazily; installing the cache never reads the crosswalk.
type PhotoRuntime struct {
	Available func() bool
	SourceID  func(context.Context, string, string) (string, error)
	Install   func(func(context.Context, string) (string, string, error))
	Settings  func() (bool, error)
	SetOff    func(bool)
}

func (s PhotoRuntime) Initialize() {
	s.Install(s.IDs)
	if off, err := s.Settings(); err == nil {
		s.SetOff(off)
	}
}

func (s PhotoRuntime) IDs(ctx context.Context, mflID string) (string, string, error) {
	if !s.Available() {
		return "", "", fmt.Errorf("history unavailable")
	}
	espn, err := s.SourceID(ctx, mflID, "espn")
	if err != nil {
		return "", "", fmt.Errorf("app: photo ids: %w", err)
	}
	sleeper, err := s.SourceID(ctx, mflID, "sleeper")
	if err != nil {
		return "", "", fmt.Errorf("app: photo ids: %w", err)
	}
	return espn, sleeper, nil
}
