package archive

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

// Host supplies detached archive facts and resolves the live opponent under its owning locks.
type Host interface {
	Ready() error
	ArchiveContext() context.Context
	ArchiveOpponent(string) string
	ArchiveFacts(context.Context) (report.ArchiveSnapshot, bool, error)
}

type API struct{ host Host }

func New(host Host) *API { return &API{host: host} }

// ArchiveBoard preserves the Archive binding's wire contract.
//
//nolint:revive // Preserve the existing exported wire name.
type ArchiveBoard struct {
	Board   report.Board    `json:"board"`
	Season  int             `json:"season"`
	Seasons []report.Choice `json:"seasons"`
	Team    string          `json:"team"`
	Other   string          `json:"other"`
	Teams   []report.Choice `json:"teams"`
}

func (a *API) TargetArchive(season int, team, other string) (ArchiveBoard, error) {
	if err := a.host.Ready(); err != nil {
		return ArchiveBoard{}, fmt.Errorf("archive: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.ArchiveContext(), 5*time.Minute)
	defer cancel()
	if other == "" {
		other = a.host.ArchiveOpponent(team)
	}
	cache, stale, err := a.host.ArchiveFacts(ctx)
	if err != nil {
		return ArchiveBoard{}, err //nolint:wrapcheck // Preserve the existing archive error.
	}
	b, err := Build(cache, stale, season, team, other)
	if err != nil {
		return ArchiveBoard{}, err
	}
	return wireArchive(b), nil
}

func wireArchive(b Board) ArchiveBoard {
	return ArchiveBoard(b)
}

// Commands describes the compound archive read; registry pages use endpoint.open, not a dedicated verb.
func Commands() []catalog.Command { return catalog.DesktopCommands("archive", nil) }
