package sources

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Settings is the app's private settings.json configuration.
type Settings struct {
	LeagueArchiveDir string `json:"leagueArchiveDir"`
	PhotosOff        bool   `json:"photosOff,omitempty"`
}

func ReadSettings(path string) (Settings, error) {
	var s Settings
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, fmt.Errorf("app: read settings: %w", err)
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("app: settings.json: %w", err)
	}
	return s, nil
}

func WriteSettings(path string, s Settings) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("app: encode settings: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("app: write settings: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("app: write settings: %w", err)
	}
	return nil
}

type CacheScope struct {
	DataScope
	Run func(bool) error
}

type CacheRefreshers struct {
	Pool     func(bool) error
	Research func(bool) error
	Race     func(bool) error
	Comms    func(bool) error
	Draft    func(bool) error
}

func CacheScopes(refresh CacheRefreshers) []CacheScope {
	return []CacheScope{
		{DataScope{ID: "pool", Label: "Free agents and market reports", Note: "The free-agent list and the market reports built on it.",
			Action: "Read free agents again", Progress: "Reading…"}, refresh.Pool},
		{DataScope{ID: "research", Label: "Player research", Note: "Player scores, activity and depth charts.",
			Action: "Read player research again", Progress: "Reading…"}, refresh.Research},
		{DataScope{ID: "race", Label: "League race", Note: "The standings and the race for the playoffs.",
			Action: "Read the league race again", Progress: "Reading…"}, refresh.Race},
		{DataScope{ID: "comms", Label: "League board", Note: "The league's message board posts.",
			Action: "Read the league board again", Progress: "Reading…"}, refresh.Comms},
		{DataScope{ID: "draft", Label: "Draft reports", Note: "The draft boards and reports.",
			Action: "Read draft reports again", Progress: "Reading…"}, refresh.Draft},
	}
}

func CacheScopeReadings(scopes []CacheScope) []DataScope {
	out := []DataScope{}
	for _, scope := range scopes {
		out = append(out, scope.DataScope)
	}
	return out
}

func ClearCache(scopes []CacheScope, id string) (string, bool, error) {
	for _, scope := range scopes {
		if scope.ID == id {
			return scope.Label, true, scope.Run(true)
		}
	}
	return "", false, nil
}
