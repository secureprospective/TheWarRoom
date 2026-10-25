package serverversion

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// State is what a server build may do with a league file.
type State int

const (
	// Fresh is an empty file from a new installation: serving creates the league storage.
	Fresh State = iota
	// Ready matches this build and may be served as it is.
	Ready
	// Older was last served by an earlier release: twr-upgrade backs it up, then updates it.
	Older
)

// Plan is a preflight verdict with the file's own versions.
type Plan struct {
	State State
	File  db.FileState
}

// Refusals name the one thing the operator does next. Every refusal leaves the file unchanged.
var (
	ErrNewerFile = errors.New("this league file was last used by a newer league server than this one. Reinstall the newer server bundle; the file was not changed (UPGRADE.md explains a proper rollback)")
	ErrDamaged   = errors.New("this league file failed its integrity check. Restore the league from its backup (RESTORE.md); the file was not changed")
	ErrNotLeague = errors.New("this is not a league server database; check the -db path. Nothing was changed")
	// ErrOlderFile is what serving refuses for an Older file.
	ErrOlderFile = errors.New("this league file is from an older server release. Run twr-upgrade from the new server bundle: it backs the league up, then updates it. The file was not changed")
)

// Preflight reads path without changing it and says whether this build may serve it. A file a
// newer schema or a newer release wrote is refused before SQLite opens it for writing, so
// putting an older binary back over newer data (a binary-only downgrade) never touches it.
func Preflight(ctx context.Context, path string) (Plan, error) {
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return Plan{}, ErrNotLeague
	}
	current := Current()
	file, err := db.Inspect(ctx, path, current.Schema)
	if errors.Is(err, db.ErrNotDatabase) {
		return Plan{File: file}, ErrNotLeague
	}
	if err != nil {
		return Plan{File: file}, fmt.Errorf("read league file versions: %w", err)
	}
	plan := Plan{File: file}
	switch {
	case file.Empty:
		plan.State = Fresh
	case file.Schema > current.Schema || file.Release > current.Release:
		return plan, ErrNewerFile
	case !file.IntegrityOK:
		return plan, ErrDamaged
	case file.Schema < current.Schema || (file.Release != 0 && file.Release < current.Release):
		plan.State = Older
	default:
		// An unstamped file at this schema was set up by this build's own local tooling.
		plan.State = Ready
	}
	return plan, nil
}

// Describe is the operator's one-line summary of a preflight verdict.
func (p Plan) Describe() string {
	current := Current()
	switch p.State {
	case Fresh:
		return "No league is stored here yet; the server sets up its storage when it starts."
	case Older:
		return fmt.Sprintf("Ready to upgrade: the league file is at version %d (release %d); this server brings it to version %d (release %d) after backing it up.",
			p.File.Schema, p.File.Release, current.Schema, current.Release)
	case Ready:
		return fmt.Sprintf("The league file is up to date for this server (version %d, release %d).", current.Schema, current.Release)
	}
	return "Check the league file before starting the server."
}
