// Package leaguerecovery rebuilds private roster documents offline, never hosted authority.
package leaguerecovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// RawRequest is local operator input; a league role or network token cannot invoke it.
type RawRequest struct {
	Source      string `json:"sourceDatabase"`
	League      string `json:"league"`
	Destination string `json:"destinationFolder"`
}

func (r RawRequest) Validate() error {
	if r.Source == "" || r.Destination == "" || !strings.HasPrefix(r.League, "native.") || r.League == "native." || strings.TrimSpace(r.League) != r.League {
		return fmt.Errorf("choose a source database, league and new destination folder")
	}
	return nil
}

// StandUp validates the source before creating anything, then creates a new
// private directory exclusively. Failure removes only this operation's directory.
func StandUp(ctx context.Context, raw RawRequest) (message string, err error) {
	if err := raw.Validate(); err != nil {
		return "", err
	}
	body, err := db.ReadLeagueRestoreFile(ctx, raw.Source, raw.League, history.SchemaVersion())
	if err != nil {
		return "", fmt.Errorf("read roster recovery document: %w", err)
	}
	var point commissioner.RawRosterRestorePoint
	if err := json.Unmarshal([]byte(body), &point); err != nil {
		return "", fmt.Errorf("decode roster recovery document: %w", err)
	}
	if err := point.Validate(); err != nil {
		return "", fmt.Errorf("validate roster recovery document: %w", err)
	}
	if err := os.Mkdir(raw.Destination, 0700); err != nil {
		return "", fmt.Errorf("create new recovery folder (existing folders are never overwritten): %w", err)
	}
	defer func() {
		if err != nil {
			if removeErr := os.RemoveAll(raw.Destination); removeErr != nil {
				err = errors.Join(err, fmt.Errorf("remove incomplete recovery: %w", removeErr))
			}
		}
	}()
	pools, err := db.Open(ctx, filepath.Join(raw.Destination, "history.db"), history.SchemaVersion())
	if err != nil {
		return "", fmt.Errorf("create fresh league database: %w", err)
	}
	defer func() {
		if closeErr := pools.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close recovered league: %w", closeErr))
		}
	}()
	ledger := history.New(pools, &measures.Registry{})
	if err := ledger.Initialize(ctx); err != nil {
		return "", fmt.Errorf("prepare fresh recovery database: %w", err)
	}
	message, err = ledger.StandUpRosters(ctx, point)
	if err != nil {
		return "", fmt.Errorf("stand the league back up: %w", err)
	}
	return message, nil
}
