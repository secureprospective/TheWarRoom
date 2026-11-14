package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// rawRestoreInput arrives on stdin from the CT restore script. Only the new backup settings
// are saved; their public recipient is derived from the restore key, which stays in memory.
type rawRestoreInput struct {
	Endpoint   string `json:"endpoint"`
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"access_key"`
	SecretKey  string `json:"secret_key"`
	RestoreKey string `json:"restore_key"`
	// Before is optional: restore the newest set saved on or before this UTC date (YYYY-MM-DD).
	Before string `json:"before,omitempty"`
}

func (rawRestoreInput) String() string     { return "[restore input redacted]" }
func (r rawRestoreInput) GoString() string { return r.String() }

var errNewerSet = errors.New("a backup in this bucket was saved by a newer league server than this one. Install the newest server bundle, then run the restore again; nothing was installed")

type recoverySets interface {
	KindSets(context.Context, offsite.SetKind) ([]string, error)
	Open(context.Context, string) (io.ReadCloser, error)
	// OpenEvidence streams one raw capture object, named by its archive key.
	OpenEvidence(context.Context, string) (io.ReadCloser, error)
}

// runRestore opens no listener. It leaves server.db (sign-ins already reset), the
// authenticator key and new backup settings in <stage>/ready for the script to install.
func runRestore(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server restore", flag.ContinueOnError)
	target := flags.String("db", "", "this freshly installed server's database; it must hold no league")
	stage := flags.String("stage", "", "new private working folder")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse restore flags: %w", err)
	}
	if flags.NArg() != 0 || *stage == "" {
		return fmt.Errorf("restore needs -db and a new -stage folder")
	}
	var raw rawRestoreInput
	decoder := json.NewDecoder(io.LimitReader(input, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return fmt.Errorf("restore input is not one valid settings object")
	}
	identity, err := age.ParseX25519Identity(strings.ToUpper(strings.Join(strings.Fields(raw.RestoreKey), "")))
	if err != nil {
		return fmt.Errorf("that restore key is not right: use the line beginning AGE-SECRET-KEY-1 from the sealed copy")
	}
	var before time.Time
	if raw.Before != "" {
		day, err := time.Parse(time.DateOnly, strings.TrimSpace(raw.Before))
		if err != nil {
			return fmt.Errorf("that date is not right: use YYYY-MM-DD, or leave it blank for the newest backup")
		}
		before = day.AddDate(0, 0, 1)
	}
	config := offsite.RawConfig{Endpoint: raw.Endpoint, Bucket: raw.Bucket, AccessKey: raw.AccessKey, SecretKey: raw.SecretKey, Recipient: identity.Recipient().String()}
	r2, err := offsite.NewR2(config)
	if err != nil {
		return fmt.Errorf("check the R2 settings: %w", err)
	}
	return restoreSpareHost(ctx, r2, identity, config, *target, *stage, before, output)
}

// restoreSpareHost installs the newest usable set; a non-zero before skips sets saved at or after it.
func restoreSpareHost(ctx context.Context, sets recoverySets, identity *age.X25519Identity, config offsite.RawConfig,
	target, stage string, before time.Time, output io.Writer) error {
	if info, err := os.Stat(target); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("restore needs this server's installed database")
	}
	if held, err := targetHolds(ctx, target); err != nil || held {
		return errors.Join(fmt.Errorf("this server already holds a league or accounts; restore only onto a freshly installed server"), err)
	}
	if err := os.Mkdir(stage, 0700); err != nil {
		return fmt.Errorf("create restore folder: %w", err)
	}
	byLeague, err := leagueSetNames(ctx, sets, before)
	if err != nil {
		return err
	}
	if len(byLeague) > 0 {
		dir := filepath.Join(stage, "set")
		captured, err := restoreLeagueFiles(ctx, sets, byLeague, identity, stage, dir, output)
		if err != nil {
			return err
		}
		return completeRestore(ctx, sets, identity, dir, stage, "each league's own backup", captured, config, output)
	}
	return restoreSharedSet(ctx, sets, identity, config, stage, before, output)
}

// completeRestore attaches the sign-in and football files matching an unlocked set, then installs it.
// A league set saved after sign-ins moved needs its sign-in set beside it; the pair is checked before
// anything is installed. Football comes after sign-ins: a league's football pin is read from its
// directory row in the sign-in file.
func completeRestore(ctx context.Context, sets recoverySets, identity *age.X25519Identity, dir, stage, name string,
	captured time.Time, config offsite.RawConfig, output io.Writer) error {
	if err := attachSignIns(ctx, sets, identity, dir, captured, output); err != nil {
		return err
	}
	if err := attachReference(ctx, sets, identity, dir, captured, output); err != nil {
		return err
	}
	return activateRestore(ctx, dir, filepath.Join(stage, "ready"), name, captured, config, output)
}

// restoreSharedSet installs the newest usable set saved before leagues had their own files.
func restoreSharedSet(ctx context.Context, sets recoverySets, identity *age.X25519Identity, config offsite.RawConfig,
	stage string, before time.Time, output io.Writer) error {
	names, err := candidateSets(ctx, sets, before)
	if err != nil {
		return err
	}
	for i, name := range names[:min(len(names), 64)] {
		dir := filepath.Join(stage, fmt.Sprintf("set-%02d", i))
		captured, err := unlockSet(ctx, sets, name, identity, dir)
		if errors.Is(err, offsite.ErrRestoreKey) || errors.Is(err, errNewerSet) {
			return err
		}
		if err != nil {
			if _, werr := fmt.Fprintf(output, "Skipping %s: %v\n", name, err); werr != nil {
				return fmt.Errorf("report restore: %w", werr)
			}
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("remove skipped backup: %w", err)
			}
			continue
		}
		return completeRestore(ctx, sets, identity, dir, stage, name, captured, config, output)
	}
	return fmt.Errorf("no usable backup was found; nothing was installed")
}

// candidateSets lists sets newest first; a non-zero before drops sets saved at or after it, unopened.
func candidateSets(ctx context.Context, sets recoverySets, before time.Time) ([]string, error) {
	names, err := sets.KindSets(ctx, offsite.LeagueSet)
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no backups were found in that bucket")
	}
	if !before.IsZero() {
		names = slices.DeleteFunc(names, func(name string) bool {
			at, err := offsite.SetTime(name)
			return err != nil || !at.Before(before)
		})
		if len(names) == 0 {
			return nil, fmt.Errorf("no backup was saved on or before that date; nothing was installed")
		}
	}
	return names, nil
}

// unlockSet decrypts and validates one set, compares it with the checkpoint saved inside
// it, and refuses a set holding no league (a fresh install backed up by mistake).
func unlockSet(ctx context.Context, sets recoverySets, name string, identity *age.X25519Identity, dir string) (time.Time, error) {
	body, err := sets.Open(ctx, name)
	if err != nil {
		return time.Time{}, fmt.Errorf("download backup: %w", err)
	}
	defer func() { _ = body.Close() }()
	manifest, err := offsite.Restore(body, identity, dir)
	if err != nil {
		return time.Time{}, fmt.Errorf("unlock backup: %w", err)
	}
	snapshot := filepath.Join(dir, "server.db")
	// Skipping a newer set would quietly restore an older league instead.
	if _, err := serverversion.Preflight(ctx, snapshot); errors.Is(err, serverversion.ErrNewerFile) {
		return time.Time{}, errNewerSet
	}
	if err := runCheckpoint(ctx, []string{"-db", snapshot, "-check", filepath.Join(dir, "checkpoint.json")}, io.Discard); err != nil {
		return time.Time{}, err
	}
	held, err := holdsLeague(ctx, snapshot)
	if err != nil {
		return time.Time{}, err
	}
	if !held {
		return time.Time{}, fmt.Errorf("it holds no league (saved by a freshly installed server)")
	}
	return manifest.Captured, nil
}

func holdsLeague(ctx context.Context, path string) (bool, error) {
	pools, err := db.Open(ctx, path, history.LeagueSchemaVersion())
	if err != nil {
		return false, fmt.Errorf("open server database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	for _, table := range []string{"league_event", "accounts"} {
		var exists, rows bool
		if err := pools.Read().QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)`, table).Scan(&exists); err != nil {
			return false, fmt.Errorf("inspect server database: %w", err)
		}
		if !exists {
			continue
		}
		if err := pools.Read().QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM `+table+`)`).Scan(&rows); err != nil {
			return false, fmt.Errorf("inspect server database: %w", err)
		}
		if rows {
			return true, nil
		}
	}
	return false, nil
}

type restoreReport struct {
	ledger, security db.AuditHead
	digests          history.RecoveryDigests
	leagues          []leagueHead
}

type leagueHead struct {
	league string
	head   db.AuditHead
}

func readRestoreReport(ctx context.Context, ledger *history.Store, security *accountstore.Store) (restoreReport, error) {
	var r restoreReport
	var err error
	if r.ledger, err = ledger.VerifyAudit(ctx); err != nil {
		return r, fmt.Errorf("verify restored league history: %w", err)
	}
	if r.security, err = security.VerifyAudit(ctx); err != nil {
		return r, fmt.Errorf("verify restored sign-in history: %w", err)
	}
	if r.digests, err = ledger.RecoveryDigests(ctx); err != nil {
		return r, fmt.Errorf("fingerprint restored league: %w", err)
	}
	return r, nil
}

// activateRestore resets sign-ins inside the staged copy, before any listener can open it.
func activateRestore(ctx context.Context, dir, ready, name string, captured time.Time, config offsite.RawConfig, output io.Writer) error {
	restored, err := openRestored(ctx, filepath.Join(dir, "server.db"))
	if err != nil {
		return err
	}
	defer func() { _ = restored.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load server measures: %w", err)
	}
	security := restored.store()
	if err := security.Initialize(ctx); err != nil {
		return fmt.Errorf("prepare restored accounts: %w", err)
	}
	before, err := restoredReport(ctx, restored, registry, security)
	if err != nil {
		return err
	}
	accounts, err := security.ResetAfterRestore(ctx, "restored onto a spare server from the backup saved "+captured.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("reset restored sign-ins: %w", err)
	}
	after, err := restoredReport(ctx, restored, registry, security)
	if err != nil {
		return err
	}
	if after.ledger != before.ledger || !slices.Equal(after.leagues, before.leagues) || !sameDigests(after.digests, before.digests) {
		return fmt.Errorf("the sign-in reset changed league history; nothing was installed")
	}
	if err := restored.finish(ctx); err != nil {
		return err
	}
	if err := stageReady(dir, ready, config); err != nil {
		return err
	}
	return printRestoreReport(output, name, captured, before, accounts)
}

func sameDigests(a, b history.RecoveryDigests) bool {
	left, lerr := json.Marshal(a)
	right, rerr := json.Marshal(b)
	return lerr == nil && rerr == nil && bytes.Equal(left, right)
}

// stageCompanions moves the restored league files, sign-in file and football file beside server.db.
func stageCompanions(dir, ready string) error {
	if stagedSplit(dir) {
		if err := os.Rename(filepath.Join(dir, leaguesFolder), filepath.Join(ready, leaguesFolder)); err != nil {
			return fmt.Errorf("stage restored league files: %w", err)
		}
	}
	for _, name := range []string{signInFileName, referenceFileName} {
		if !fileExists(filepath.Join(dir, name)) {
			continue
		}
		if info, err := os.Stat(filepath.Join(dir, name+"-wal")); err == nil && info.Size() > 0 {
			return fmt.Errorf("restored %s did not finish writing", name)
		}
		if err := os.Rename(filepath.Join(dir, name), filepath.Join(ready, name)); err != nil {
			return fmt.Errorf("stage restored %s: %w", name, err)
		}
	}
	return nil
}

func stageReady(dir, ready string, config offsite.RawConfig) error {
	if err := os.Mkdir(ready, 0700); err != nil {
		return fmt.Errorf("create restore output: %w", err)
	}
	if info, err := os.Stat(filepath.Join(dir, "server.db-wal")); err == nil && info.Size() > 0 {
		return fmt.Errorf("restored database did not finish writing")
	}
	if err := os.Rename(filepath.Join(dir, "server.db"), filepath.Join(ready, "server.db")); err != nil {
		return fmt.Errorf("stage restored database: %w", err)
	}
	if err := stageCompanions(dir, ready); err != nil {
		return err
	}
	key, err := readOfflineKey(filepath.Join(dir, "config", "authenticator.key"))
	if err != nil {
		return fmt.Errorf("restored authenticator key: %w", err)
	}
	settings, err := json.Marshal(config) //nolint:gosec // backup.json is the private 0600 settings file the server reads.
	if err != nil {
		return fmt.Errorf("encode backup settings: %w", err)
	}
	for file, body := range map[string][]byte{"authenticator.key": key, "backup.json": settings} {
		if err := os.WriteFile(filepath.Join(ready, file), body, 0600); err != nil {
			return fmt.Errorf("stage %s: %w", file, err)
		}
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove restore working copy: %w", err)
	}
	return nil
}

func printRestoreReport(output io.Writer, name string, captured time.Time, r restoreReport, accounts int) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Unlocked %s, saved %s.\n", name, captured.UTC().Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(&b, "League history matches the checkpoint saved with it.\n")
	if len(r.leagues) == 0 {
		fmt.Fprintf(&b, "League history: %d entries, fingerprint %s\n", r.ledger.Size, r.ledger.Hash)
	}
	for _, head := range r.leagues {
		fmt.Fprintf(&b, "League file %s: %d entries, fingerprint %s\n", head.league, head.head.Size, head.head.Hash)
	}
	fmt.Fprintf(&b, "Sign-in history: %d entries, fingerprint %s\n", r.security.Size, r.security.Hash)
	fmt.Fprintf(&b, "Evidence files: %d, fingerprint %s\n", r.digests.EvidenceFiles, r.digests.Evidence)
	for _, league := range r.digests.Leagues {
		fmt.Fprintf(&b, "League %q (%s): %d entries, replay fingerprint %s\n", league.Name, league.League, league.Entries, league.Replay)
	}
	fmt.Fprintf(&b, "Signed out all %d accounts: earlier sign-ins, helper tokens, printed recovery codes and unused invitations no longer work.\n", accounts)
	if _, err := io.WriteString(output, b.String()); err != nil {
		return fmt.Errorf("report restore: %w", err)
	}
	return nil
}

// runResetSignIns is the incident console command: stop the service first. The reason is
// one stdin line and is kept in the sign-in history.
func runResetSignIns(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server reset-sign-ins", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse reset flags: %w", err)
	}
	if info, err := os.Stat(*path); err != nil || !info.Mode().IsRegular() || flags.NArg() != 0 {
		return fmt.Errorf("reset needs an existing server database (-db)")
	}
	line, err := bufio.NewReader(io.LimitReader(input, 512)).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read reset reason: %w", err)
	}
	files, err := openSignInFiles(ctx, *path)
	if err != nil {
		return fmt.Errorf("open server database: %w", err)
	}
	defer func() { _ = files.Close() }()
	store := signInStore(files)
	if err := store.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize server accounts: %w", err)
	}
	accounts, err := store.ResetSignIns(ctx, strings.TrimSpace(line))
	if err != nil {
		return fmt.Errorf("reset sign-ins: %w", err)
	}
	if _, err := fmt.Fprintf(output, "Signed out every device and helper on %d accounts. Passwords and authenticators are unchanged; owners sign in again.\n", accounts); err != nil {
		return fmt.Errorf("report reset: %w", err)
	}
	return nil
}
