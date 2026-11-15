package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// rawCheckpoint is the history checkpoint kept away from the server: with each off-site backup and on
// the commissioner's desktop. A later rewrite of history and its local hash chain together no longer
// matches it. A checkpoint of the whole server names both chains; a league set's names only the
// league chain and a sign-in set's only the security chain.
type rawCheckpoint struct {
	Version  int           `json:"version"`
	Captured time.Time     `json:"captured"`
	Ledger   *db.AuditHead `json:"ledger,omitempty"`
	Security *db.AuditHead `json:"security,omitempty"`
	// Leagues names each league file's own chain on a server whose leagues have their own files, and
	// Shared the head the one shared chain was closed at when they got them.
	Leagues map[string]db.AuditHead `json:"leagues,omitempty"`
	Shared  *db.AuditHead           `json:"shared,omitempty"`
	// Reference names a reference set's raw archive by content; it has no chain.
	Reference *referenceAnchor `json:"reference,omitempty"`
}

// referenceAnchor is a raw archive's body count and the digest of its sorted sha256 keys.
type referenceAnchor struct {
	Bodies  int64  `json:"bodies"`
	Archive string `json:"archive"`
}

func referenceSetCheckpoint(anchor history.ArchiveAnchor) ([]byte, error) {
	return encodeCheckpoint(rawCheckpoint{Version: 1, Captured: time.Now().UTC(),
		Reference: &referenceAnchor{Bodies: anchor.Rows, Archive: anchor.Digest}})
}

var errCheckpointMismatch = errors.New("league history on this server no longer matches the saved checkpoint")

// signInSource is where a league file's sign-in history is read: the sign-in file beside it, or the
// league file itself when it was never converted (a copy saved before sign-ins moved). Empty path:
// the league file holds none and was never converted.
type signInSource struct {
	path    string
	version int
}

// locateSignIns only reads. Once the league file records the move, only the sign-in file is read.
func locateSignIns(ctx context.Context, leaguePath string) (signInSource, error) {
	pools, err := db.OpenReadOnly(ctx, leaguePath, history.LeagueSchemaVersion())
	if err != nil {
		return signInSource{}, fmt.Errorf("open server database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	moved, err := hasTable(ctx, pools, signInMoveTable)
	if err != nil {
		return signInSource{}, err
	}
	if moved {
		path := signInPath(leaguePath)
		if !fileExists(path) {
			return signInSource{}, fmt.Errorf("the sign-in file %s is missing", signInFileName)
		}
		return signInSource{path: path, version: accountstore.ServerVersion()}, nil
	}
	held, err := hasTable(ctx, pools, "accounts")
	if err != nil || !held {
		return signInSource{}, err
	}
	return signInSource{path: leaguePath, version: history.LeagueSchemaVersion()}, nil
}

// runCheckpoint prints a verified checkpoint of the server's two histories, or with -check compares
// the server against one. It only reads: no migration, no initialization, no listener.
func runCheckpoint(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server checkpoint", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database")
	check := flags.String("check", "", "saved checkpoint file to compare against")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse checkpoint flags: %w", err)
	}
	if info, err := os.Stat(*path); err != nil || !info.Mode().IsRegular() || flags.NArg() != 0 {
		return fmt.Errorf("checkpoint needs an existing server database (-db)")
	}
	if *check != "" {
		return checkCheckpoint(ctx, *check, *path, output)
	}
	signIns, err := locateSignIns(ctx, *path)
	if err != nil {
		return err
	}
	current := rawCheckpoint{Version: 1, Captured: time.Now().UTC()}
	if err := serverLedgers(ctx, *path, &current); err != nil {
		return err
	}
	if signIns.path != "" {
		security, err := securityHead(ctx, signIns)
		if err != nil {
			return err
		}
		current.Security = &security
	}
	if err := json.NewEncoder(output).Encode(current); err != nil {
		return fmt.Errorf("write checkpoint: %w", err)
	}
	return nil
}

// serverLedgers names the server's league chains: each league file's and the closed shared head once
// its leagues have their own files, else server.db's one chain.
func serverLedgers(ctx context.Context, path string, into *rawCheckpoint) error {
	split, done, err := readSplit(ctx, path)
	if err != nil {
		return err
	}
	if !done {
		ledger, err := ledgerHead(ctx, path)
		if err != nil {
			return err
		}
		into.Ledger = &ledger
		return nil
	}
	into.Shared, into.Leagues = &split.Shared, map[string]db.AuditHead{}
	leagues, err := leagueFileList(leaguesRoot(path))
	if err != nil {
		return err
	}
	for _, league := range leagues {
		file, _ := leagueFilePath(leaguesRoot(path), league)
		head, err := ledgerHead(ctx, file)
		if err != nil {
			return fmt.Errorf("league %s: %w", league, err)
		}
		into.Leagues[league] = head
	}
	return nil
}

func readSplit(ctx context.Context, path string) (history.LeagueSplit, bool, error) {
	pools, err := db.OpenReadOnly(ctx, path, history.LeagueSchemaVersion())
	if err != nil {
		return history.LeagueSplit{}, false, fmt.Errorf("open server database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	return history.ReadLeagueSplit(ctx, pools) //nolint:wrapcheck // names the file
}

func ledgerHead(ctx context.Context, path string) (db.AuditHead, error) {
	var head db.AuditHead
	err := withLedger(ctx, path, func(ledger *history.Store) (err error) {
		head, err = ledger.VerifyAudit(ctx)
		return err //nolint:wrapcheck // wrapped below
	})
	if err != nil {
		return head, fmt.Errorf("verify league history: %w", err)
	}
	return head, nil
}

func securityHead(ctx context.Context, source signInSource) (db.AuditHead, error) {
	var head db.AuditHead
	err := withSecurity(ctx, source, func(security *accountstore.Store) (err error) {
		head, err = security.VerifyAudit(ctx)
		return err //nolint:wrapcheck // wrapped below
	})
	if err != nil {
		return head, fmt.Errorf("verify security history: %w", err)
	}
	return head, nil
}

func withLedger(ctx context.Context, path string, use func(*history.Store) error) error {
	pools, err := db.Open(ctx, path, history.LeagueSchemaVersion())
	if err != nil {
		return fmt.Errorf("open server database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load server measures: %w", err)
	}
	return use(history.New(pools, registry))
}

// checkLedger checks a saved chain head against the file's chain. Once the file's leagues have their
// own files, the one shared chain was closed: it matches only a head saved exactly where it closed.
func checkLedger(ctx context.Context, saved db.AuditHead, path string) error {
	split, done, err := readSplit(ctx, path)
	if err != nil {
		return err
	}
	if done {
		if split.Shared != saved {
			return fmt.Errorf("the shared chain was closed at entry %d when each league got its own file; the checkpoint names entry %d", split.Shared.Size, saved.Size)
		}
		return nil
	}
	return withLedger(ctx, path, func(ledger *history.Store) error { return ledger.VerifyAuditAnchor(ctx, saved) })
}

func withSecurity(ctx context.Context, source signInSource, use func(*accountstore.Store) error) error {
	pools, err := db.Open(ctx, source.path, source.version)
	if err != nil {
		return fmt.Errorf("open sign-in file: %w", err)
	}
	defer func() { _ = pools.Close() }()
	return use(accountstore.New(pools))
}

// leagueSetCheckpoint and signInSetCheckpoint are the checkpoints saved inside each kind of set.
func leagueSetCheckpoint(ctx context.Context, snapshot string) ([]byte, error) {
	head, err := ledgerHead(ctx, snapshot)
	if err != nil {
		return nil, err
	}
	return encodeCheckpoint(rawCheckpoint{Version: 1, Captured: time.Now().UTC(), Ledger: &head})
}

func signInSetCheckpoint(ctx context.Context, snapshot string) ([]byte, error) {
	head, err := securityHead(ctx, signInSource{path: snapshot, version: accountstore.ServerVersion()})
	if err != nil {
		return nil, err
	}
	return encodeCheckpoint(rawCheckpoint{Version: 1, Captured: time.Now().UTC(), Security: &head})
}

func encodeCheckpoint(c rawCheckpoint) ([]byte, error) {
	body, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("write checkpoint: %w", err)
	}
	return append(body, '\n'), nil
}

func readCheckpoint(file string) (rawCheckpoint, error) {
	body, err := os.ReadFile(file)
	if err != nil {
		return rawCheckpoint{}, fmt.Errorf("read saved checkpoint: %w", err)
	}
	var saved rawCheckpoint
	if err := json.Unmarshal(body, &saved); err != nil || saved.Version != 1 || (saved.Ledger == nil && saved.Leagues == nil && saved.Security == nil && saved.Reference == nil) {
		return rawCheckpoint{}, fmt.Errorf("the saved checkpoint file is not one this version can read")
	}
	return saved, nil
}

// checkCheckpoint compares each history the saved checkpoint names with its file. A league set's
// checkpoint names only the league chain, so its sign-in file is looked for only when one is named.
func checkCheckpoint(ctx context.Context, file, leaguePath string, output io.Writer) error {
	saved, err := readCheckpoint(file)
	if err != nil {
		return err
	}
	var signIns signInSource
	if saved.Security != nil {
		if signIns, err = locateSignIns(ctx, leaguePath); err != nil {
			return err
		}
	}
	if err := checkAnchors(ctx, saved, leaguePath, signIns); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "League history matches the checkpoint saved %s.\n", saved.Captured.Format(time.DateOnly)); err != nil {
		return fmt.Errorf("report checkpoint: %w", err)
	}
	return nil
}

func checkAnchors(ctx context.Context, saved rawCheckpoint, leaguePath string, signIns signInSource) error {
	day := saved.Captured.Format(time.DateOnly)
	if saved.Ledger != nil {
		if leaguePath == "" {
			return fmt.Errorf("%w (league history, saved %s): no league file", errCheckpointMismatch, day)
		}
		if err := checkLedger(ctx, *saved.Ledger, leaguePath); err != nil {
			return fmt.Errorf("%w (league history, saved %s): %w", errCheckpointMismatch, day, err)
		}
	}
	for league, head := range saved.Leagues {
		file, err := leagueFilePath(leaguesRoot(leaguePath), league)
		if err == nil && !fileExists(file) {
			err = fmt.Errorf("its file is missing")
		}
		if err == nil {
			err = withLedger(ctx, file, func(ledger *history.Store) error { return ledger.VerifyAuditAnchor(ctx, head) })
		}
		if err != nil {
			return fmt.Errorf("%w (league %s, saved %s): %w", errCheckpointMismatch, league, day, err)
		}
	}
	if saved.Security != nil {
		if signIns.path == "" {
			return fmt.Errorf("%w (sign-in history, saved %s): no sign-in file", errCheckpointMismatch, day)
		}
		err := withSecurity(ctx, signIns, func(security *accountstore.Store) error { return security.VerifyAuditAnchor(ctx, *saved.Security) })
		if err != nil {
			return fmt.Errorf("%w (sign-in history, saved %s): %w", errCheckpointMismatch, day, err)
		}
	}
	return nil
}
