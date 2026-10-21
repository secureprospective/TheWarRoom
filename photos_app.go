package main

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/headshot"
)

// photoTimeout bounds one request to a photo host; a dead host never holds a screen (R11).
const photoTimeout = 10 * time.Second

// PhotoView is what a player card needs to show a photo: Go decides whether to ask for one and
// writes its alt text (W1). An empty Src means show the fallback without any request.
type PhotoView struct {
	Src string `json:"src"`
	Alt string `json:"alt"`
}

// startHeadshots builds the photo cache in the data directory. Photos are personal use only
// (spec §7): never bundled, never in history.db.
func (a *App) startHeadshots() {
	ids := func(ctx context.Context, mflID string) (string, string, error) {
		if a.core.History == nil {
			return "", "", fmt.Errorf("history unavailable")
		}
		espn, err := a.core.History.SourceID(ctx, mflID, "espn")
		if err != nil {
			return "", "", fmt.Errorf("app: photo ids: %w", err)
		}
		sleeper, err := a.core.History.SourceID(ctx, mflID, "sleeper")
		if err != nil {
			return "", "", fmt.Errorf("app: photo ids: %w", err)
		}
		return espn, sleeper, nil
	}
	a.core.Headshots.Store(headshot.New(filepath.Join(a.core.Paths.DataDir, "headshots"), ids, &http.Client{Timeout: photoTimeout}))
	if s, err := a.readSettings(); err == nil {
		a.core.PhotosOff.Store(s.PhotosOff)
	}
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

// TargetPhoto is a player card's photo: its address and alt text, or no address when photos are
// off or the player has none on file.
func (a *App) TargetPhoto(mflID string) PhotoView {
	a.core.LookupMu.Lock()
	name, ok := a.core.Lookup.Name(mflID)
	a.core.LookupMu.Unlock()
	if !ok {
		name = "this player"
	}
	return photoView(mflID, proseName(name), a.core.PhotosOff.Load())
}

func photoView(mflID, name string, off bool) PhotoView {
	if off {
		return PhotoView{Alt: name}
	}
	return PhotoView{Src: "/headshot/" + mflID, Alt: "Photo of " + name}
}

type DataActResult struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Summary string `json:"summary"`
}

// SetPhotos turns player photos on or off; off means no request reaches a photo host.
func (a *App) SetPhotos(on bool) DataActResult {
	s, err := a.readSettings()
	if err != nil {
		return DataActResult{Error: err.Error()}
	}
	s.PhotosOff = !on
	if err := a.writeSettings(s); err != nil {
		return DataActResult{Error: err.Error()}
	}
	a.core.PhotosOff.Store(!on)
	if on {
		return DataActResult{OK: true, Summary: "Player photos are on; each shows the first time its card is drawn."}
	}
	return DataActResult{OK: true, Summary: "Player photos are off; cards show the NFL team instead."}
}
