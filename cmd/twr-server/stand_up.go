package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"reflect"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/leaguerecovery"
)

// This executable definition is console-only. Catalog metadata must never
// automatically publish offline filesystem authority through HTTP or Wails.
type standUpEntry struct {
	Metadata catalog.Command
	Door     string
	Handler  func(context.Context, leaguerecovery.RawRequest) (string, error)
}

func standUpDefinition() standUpEntry {
	return standUpEntry{
		Metadata: catalog.Command{ID: "league.stand_up", Method: "StandUpRosters", Version: "1", Role: "operator", Gravity: "G3", Availability: "server",
			Arguments:  catalog.Shape(reflect.TypeFor[leaguerecovery.RawRequest](), "leaguerecovery.RawRequest.Validate"),
			Permission: "offline-operator", Modes: []catalog.Mode{catalog.Execute}, Undo: "fresh-folder-only", Facts: "validated-rosters-and-contracts", Safeguards: "exclusive-private-folder; no-listener; read-only-league"},
		Door: "operator-console", Handler: leaguerecovery.StandUp,
	}
}

func runStandUp(ctx context.Context, args []string, output io.Writer) error {
	entry := standUpDefinition()
	if err := entry.Metadata.ValidatePublication(); err != nil {
		return fmt.Errorf("define offline recovery command: %w", err)
	}
	flags := flag.NewFlagSet("twr-server stand-up", flag.ContinueOnError)
	source := flags.String("from-db", "", "existing history.db or decrypted R2 server.db")
	league := flags.String("league", "", "league to recover")
	destination := flags.String("to-dir", "", "new private folder; never overwritten")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse roster recovery flags: %w", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("stand-up accepts only -from-db, -league and -to-dir")
	}
	raw := leaguerecovery.RawRequest{Source: *source, League: *league, Destination: *destination}
	if err := raw.Validate(); err != nil {
		return fmt.Errorf("check roster recovery request: %w", err)
	}
	message, err := entry.Handler(ctx, raw)
	if err != nil {
		return fmt.Errorf("recover league rosters: %w", err)
	}
	if _, err := fmt.Fprintln(output, message); err != nil {
		return fmt.Errorf("print roster recovery result (the recovery has already committed): %w", err)
	}
	return nil
}
