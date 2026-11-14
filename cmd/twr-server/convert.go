package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// runConversionProof prints the conversion proof of a stopped server: league by league and table by
// table, the rows and digest server.db held when its leagues got their own files, beside what each
// league's own file holds now. twr-upgrade prints the same proof when it converts a server; before the
// server starts every line reads "same", and after that a league's new writes show as differing.
func runConversionProof(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server conversion-proof", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database (server stopped)")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse conversion-proof flags: %w", err)
	}
	if flags.NArg() != 0 || *path == "" {
		return fmt.Errorf("conversion-proof needs -db, with the server stopped")
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	return conversionProof(ctx, *path, time.Time{}, output)
}

// conversionProof reports the split's proof when the split happened at or after since.
func conversionProof(ctx context.Context, path string, since time.Time, output io.Writer) error {
	files, err := openStateFiles(ctx, path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	split, done, err := history.ReadLeagueSplit(ctx, files.shared)
	if err != nil {
		return err //nolint:wrapcheck // names the file
	}
	if !done {
		return fmt.Errorf("server.db holds no record of its leagues getting their own files")
	}
	if split.At.Before(since) {
		return nil
	}
	return reportConversion(ctx, files, split, output)
}

// reportConversion compares each league file with the proof the split recorded; any difference is an
// error naming how many tables differ.
func reportConversion(ctx context.Context, files stateFiles, split history.LeagueSplit, output io.Writer) error {
	var b strings.Builder
	fmt.Fprintf(&b, "server.db gave its %d league(s) their own files on %s.\n", len(split.Leagues), split.At.UTC().Format("2006-01-02 15:04:05 UTC"))
	if len(split.Proofs) == 0 && len(split.Leagues) > 0 {
		b.WriteString("This server was put back from backups, so there is no conversion proof to compare.\n")
	}
	differ := 0
	for _, league := range split.Leagues {
		was, proved := split.Proofs[league]
		if !proved {
			continue
		}
		fmt.Fprintf(&b, "League %s:\n", league)
		n, err := reportLeague(ctx, files, league, was, &b)
		if err != nil {
			return err
		}
		differ += n
	}
	if _, err := io.WriteString(output, b.String()); err != nil {
		return fmt.Errorf("report conversion: %w", err)
	}
	if differ > 0 {
		return fmt.Errorf("%d table(s) differ from what server.db held when its leagues got their own files", differ)
	}
	return nil
}

func reportLeague(ctx context.Context, files stateFiles, league string, was map[string]db.RowProof, b *strings.Builder) (int, error) {
	pools, ok := files.leagues.pools(league)
	if !ok {
		fmt.Fprintf(b, "  no league file: DIFFERS\n")
		return len(was), nil
	}
	now, err := history.LeagueProofs(ctx, pools, league, false)
	if err != nil {
		return 0, err //nolint:wrapcheck // names the table
	}
	tables := make([]string, 0, len(was))
	for table := range was {
		tables = append(tables, table)
	}
	slices.Sort(tables)
	differ := 0
	for _, table := range tables {
		state := "same"
		if now[table] != was[table] {
			state = fmt.Sprintf("DIFFERS (server.db held %d rows, digest %s)", was[table].Rows, short(was[table].Digest))
			differ++
		}
		fmt.Fprintf(b, "  %-24s %8d rows  digest %s  %s\n", table, now[table].Rows, short(now[table].Digest), state)
	}
	return differ, nil
}

func short(digest string) string { return digest[:min(len(digest), 12)] }
