package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/leaguerecovery"
)

func runStandUp(ctx context.Context, args []string, output io.Writer) error {
	registry, err := catalog.NewOperatorRegistry()
	if err != nil {
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
	body, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("encode roster recovery request: %w", err)
	}
	answer, err := registry.Invoke(ctx, catalog.ConsoleDoor, "league.stand_up", catalog.Execute, body)
	if err != nil {
		return fmt.Errorf("recover league rosters: %w", err)
	}
	var message string
	if err := json.Unmarshal(answer, &message); err != nil {
		return fmt.Errorf("read roster recovery result: %w", err)
	}
	if _, err := fmt.Fprintln(output, message); err != nil {
		return fmt.Errorf("print roster recovery result (the recovery has already committed): %w", err)
	}
	return nil
}
