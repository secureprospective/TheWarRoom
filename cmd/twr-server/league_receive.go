package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"filippo.io/age"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// runImportLeague takes a league from another server's move file on a stopped server. The move code
// comes on stdin. It prints one claim link per person the league seats.
func runImportLeague(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server import-league", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database (server stopped)")
	in := flags.String("in", "", "the move file")
	stage := flags.String("stage", "", "new private working folder")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse import-league flags: %w", err)
	}
	if flags.NArg() != 0 || *path == "" || *in == "" || *stage == "" {
		return fmt.Errorf("import-league needs -db, -in and a new -stage folder, with the server stopped")
	}
	line, err := bufio.NewReader(io.LimitReader(input, 512)).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read the move code: %w", err)
	}
	code, err := age.ParseX25519Identity(strings.ToUpper(strings.Join(strings.Fields(line), "")))
	if err != nil {
		return fmt.Errorf("that move code is not right: use the line beginning AGE-SECRET-KEY-1 the old server printed")
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	files, err := openStateFiles(ctx, *path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	return importLeague(ctx, files, *in, code, *stage, time.Now(), output)
}

func importLeague(ctx context.Context, files stateFiles, in string, code *age.X25519Identity, stage string, now time.Time, output io.Writer) error {
	if err := os.Mkdir(stage, 0o700); err != nil {
		return fmt.Errorf("create the working folder: %w", err)
	}
	defer func() { _ = os.RemoveAll(stage) }()
	manifest, football, err := readMoveFile(in, code, stage)
	if err != nil {
		return fmt.Errorf("%w; nothing was changed", err)
	}
	league := manifest.League
	if _, ok := files.leagues.pools(league); ok {
		return fmt.Errorf("league %s is already on this server; nothing was changed", league)
	}
	staged := filepath.Join(stage, leagueFileName)
	if err := checkMovedLeague(ctx, staged, league); err != nil {
		return fmt.Errorf("%w; nothing was changed", err)
	}
	for digest, stored := range football {
		if err := history.PutArchiveBody(ctx, files.reference, digest, stored); err != nil {
			return fmt.Errorf("%w; nothing was changed", err)
		}
	}
	recovered, err := movedPlayers(ctx, files, staged)
	if err != nil {
		return err
	}
	if _, err := installLeagueFile(ctx, files, league, staged, now); err != nil {
		return err
	}
	if _, held := football[manifest.Football]; held {
		if err := signInStore(files).PinFootball(ctx, league, manifest.Football); err != nil {
			return fmt.Errorf("league %s is on this server, but its football version could not be pinned: %w", league, err)
		}
	}
	claims, err := issueMoveClaims(ctx, files, league)
	if err != nil {
		return fmt.Errorf("league %s is on this server, but its claim links could not be made: %w", league, err)
	}
	return reportImport(output, manifest, recovered, claims)
}

// movedPlayers puts the moved league's players into this server's football reference file from the
// league's own restore points; it never mints.
func movedPlayers(ctx context.Context, files stateFiles, staged string) (history.IdentityRecovery, error) {
	pools, err := db.OpenReadOnly(ctx, staged, history.LeagueSchemaVersion())
	if err != nil {
		return history.IdentityRecovery{}, fmt.Errorf("open the moved league file: %w", err)
	}
	defer func() { _ = pools.Close() }()
	recovered, err := history.RecoverLeagueIdentities(ctx, pools, files.reference)
	if err != nil {
		return recovered, fmt.Errorf("%w; nothing was changed", err)
	}
	return recovered, nil
}

// issueMoveClaims gives each person the moved league seats, and each commissioner it names who holds
// no franchise, a one-time claim.
func issueMoveClaims(ctx context.Context, files stateFiles, league string) ([]accounts.IssuedSeatClaim, error) {
	store, ok := files.leagues.store(league)
	pools, _ := files.leagues.pools(league)
	if !ok {
		return nil, errNoLeague
	}
	seatsOf := leagueservice.JoinSeats{Store: store}
	franchises, err := seatsOf.Franchises(ctx, league)
	if err != nil {
		return nil, err //nolint:wrapcheck // names what was read
	}
	var seats []accounts.SeatClaim
	seated := map[governance.Principal]bool{}
	for _, f := range franchises {
		if f.Seat == "" {
			continue
		}
		seats = append(seats, accounts.SeatClaim{Principal: governance.Principal(f.Seat), Franchise: f.ID, Name: f.Name})
		seated[governance.Principal(f.Seat)] = true
	}
	named, err := history.NamedPrincipals(ctx, pools)
	if err != nil {
		return nil, err //nolint:wrapcheck // names what was read
	}
	slices.Sort(named)
	for _, p := range named {
		principal := governance.Principal(p)
		if seated[principal] {
			continue
		}
		if yes, err := seatsOf.IsCommissioner(ctx, league, principal); err == nil && yes {
			seats = append(seats, accounts.SeatClaim{Principal: principal, Franchise: accounts.CommissionerSeat, Name: "Commissioner"})
		}
	}
	return accounts.IssueSeatClaims(ctx, signInStore(files), league, seats) //nolint:wrapcheck // names the step
}

func reportImport(output io.Writer, manifest rawMoveManifest, recovered history.IdentityRecovery, claims []accounts.IssuedSeatClaim) error {
	var b strings.Builder
	fmt.Fprintf(&b, "League %s is on this server, as the old server sealed it on %s.\n", manifest.League, manifest.At.UTC().Format("2006-01-02 15:04:05 UTC"))
	if recovered.Installed > 0 {
		fmt.Fprintf(&b, "Added %d player(s) to this server's football data from the league's own record.\n", recovered.Installed)
	}
	if len(recovered.Conflicts) > 0 {
		fmt.Fprintf(&b, "%d player(s) conflict with this server's football data and were not merged: %s.\n", len(recovered.Conflicts), strings.Join(recovered.Conflicts, "; "))
	}
	b.WriteString("Send each person their own claim link (private: it works once). Each makes a new sign-in on this server:\n")
	for _, claim := range claims {
		fmt.Fprintf(&b, "  %-24s %s\n", claim.Name, serverhttp.JoinPageLink(manifest.League, claim.Code, claim.Franchise))
	}
	b.WriteString("Then finish the move on the old server, so the league there is archived.\n")
	if _, err := io.WriteString(output, b.String()); err != nil {
		return fmt.Errorf("report import (the league is already on this server): %w", err)
	}
	return nil
}

// runFinishMove ends a move on the old server: the league's file goes to the archive for 30 days.
func runFinishMove(ctx context.Context, args []string, output io.Writer) error {
	return moveDecision(ctx, "finish-move", args, output, func(files stateFiles, league string) (string, error) {
		if err := files.leagues.remove(league); err != nil {
			return "", err
		}
		now := time.Now()
		target, err := archiveLeague(files.leagues.root, league, now)
		if err != nil {
			return "", err
		}
		if err := signInStore(files).ForgetLeague(context.WithoutCancel(ctx), league); err != nil {
			return "", fmt.Errorf("league %s is archived at %s, but its directory row could not be removed: %w", league, target, err)
		}
		return fmt.Sprintf("League %s has moved. Its file here is kept in %s until %s, then this server deletes it.\n",
			league, target, now.Add(archiveDays*24*time.Hour).Format("2006-01-02")), nil
	})
}

// runCancelMove ends a move on the old server the other way: the league changes here again.
func runCancelMove(ctx context.Context, args []string, output io.Writer) error {
	return moveDecision(ctx, "cancel-move", args, output, func(files stateFiles, league string) (string, error) {
		marker, err := movingPath(files.leagues.root, league)
		if err != nil {
			return "", err
		}
		if err := os.Remove(marker); err != nil {
			return "", fmt.Errorf("cancel the move: %w", err)
		}
		if err := syncPath(filepath.Dir(marker)); err != nil {
			return "", err
		}
		return fmt.Sprintf("The move of league %s is cancelled; the league can change on this server again. Do not use the move file on another server.\n", league), nil
	})
}

func moveDecision(ctx context.Context, name string, args []string, output io.Writer, decide func(stateFiles, string) (string, error)) error {
	flags := flag.NewFlagSet("twr-server "+name, flag.ContinueOnError)
	path := flags.String("db", "", "existing server database (server stopped)")
	league := flags.String("league", "", "league ID (native.…)")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse %s flags: %w", name, err)
	}
	if flags.NArg() != 0 || *path == "" || *league == "" {
		return fmt.Errorf("%s needs -db and -league, with the server stopped", name)
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	files, err := openStateFiles(ctx, *path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	if !files.leagues.moving(*league) {
		return fmt.Errorf("league %s is not moving; nothing was changed", *league)
	}
	words, err := decide(files, *league)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(output, words); err != nil {
		return fmt.Errorf("report %s (it is already done): %w", name, err)
	}
	return nil
}
