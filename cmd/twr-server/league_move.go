package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// A league moves to another server as its own file (Christopher, 2026-11-14). The old server writes
// the league's file, sealed to a one-time move code, and from then on keeps the league readable but
// unchangeable. The new server takes the file and gives each person the league seats a one-time claim
// link: sign-ins never leave a server. Once the new server has it, the old server finishes the move,
// and the league's file is kept 30 days in the archive, as a deleted league's is; or it cancels the
// move, and the league changes there again.

const (
	movingFile    = "moving.json"
	moveManifest  = "manifest.json"
	moveFootball  = "football/"
	moveMaxFile   = 1 << 30
	moveExtension = ".twrleague"
)

var errLeagueMoving = errors.New("this league is moving to another server, so it cannot change here; the commissioner is finishing the move")

// rawMoveManifest names what a move file holds.
type rawMoveManifest struct {
	Version  int       `json:"version"`
	League   string    `json:"league"`
	At       time.Time `json:"at"`
	Football string    `json:"football,omitempty"`
}

// rawMoving is the old server's record that league is on its way to another server.
type rawMoving struct {
	File string    `json:"file"`
	At   time.Time `json:"at"`
}

func movingPath(root, league string) (string, error) {
	final, err := leagueFilePath(root, league)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(final), movingFile), nil
}

// moving says whether league is on its way to another server.
func (f *leagueFiles) moving(league string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	file := f.open[league]
	return file != nil && file.moving
}

// guard puts league's moving check before next, a write guard.
func (f *leagueFiles) guard(league string, next func() error) func() error {
	return func() error {
		if f.moving(league) {
			return errLeagueMoving
		}
		return next()
	}
}

// runExportLeague writes league's move file on a stopped server and seals the league here.
func runExportLeague(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server export-league", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database (server stopped)")
	league := flags.String("league", "", "league ID (native.…)")
	out := flags.String("out", "", "new folder for the move file")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse export-league flags: %w", err)
	}
	if flags.NArg() != 0 || *path == "" || *league == "" || *out == "" {
		return fmt.Errorf("export-league needs -db, -league and a new -out folder, with the server stopped")
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	files, err := openStateFiles(ctx, *path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	return exportLeague(ctx, files, *league, *out, time.Now(), output)
}

func exportLeague(ctx context.Context, files stateFiles, league, out string, now time.Time, output io.Writer) error {
	pools, ok := files.leagues.pools(league)
	if !ok {
		return fmt.Errorf("league %s is not on this server; nothing was changed", league)
	}
	if files.leagues.moving(league) {
		return fmt.Errorf("league %s is already moving; finish or cancel that move first", league)
	}
	pin, err := leaguePin(ctx, files, league)
	if err != nil {
		return err
	}
	if err := os.Mkdir(out, 0o700); err != nil {
		return fmt.Errorf("create the move folder: %w", err)
	}
	snapshot := filepath.Join(out, leagueFileName)
	if _, err := pools.Read().ExecContext(ctx, `VACUUM INTO ?`, snapshot); err != nil {
		return fmt.Errorf("copy league %s's file: %w", league, err)
	}
	football := map[string][]byte{}
	if pin != "" {
		body, held, err := history.ArchiveBody(ctx, files.reference, pin)
		if err != nil {
			return err //nolint:wrapcheck // names the capture
		}
		if held {
			football[pin] = body
		}
	}
	code, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("make the move code: %w", err)
	}
	name := filepath.Join(out, league+"-"+now.UTC().Format(archiveStamp)+moveExtension)
	manifest := rawMoveManifest{Version: 1, League: league, At: now.UTC(), Football: pin}
	if err := writeMoveFile(name, code.Recipient(), manifest, snapshot, football); err != nil {
		return errors.Join(err, os.Remove(name))
	}
	if err := os.Remove(snapshot); err != nil {
		return fmt.Errorf("remove the league copy: %w", err)
	}
	if err := sealMoving(files, league, rawMoving{File: name, At: now.UTC()}); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "League %s is ready to move.\n  move file  %s\n  move code  %s\nCopy the move file to the new server and keep the move code private: anyone holding both can take the league. Until you finish or cancel the move, the league stays readable here and cannot change.\n",
		league, name, code.String())
	if err != nil {
		return fmt.Errorf("report export (the league is already sealed for the move): %w", err)
	}
	return nil
}

// leaguePin is league's own pinned football version, "" when it uses the server's.
func leaguePin(ctx context.Context, files stateFiles, league string) (string, error) {
	directory, err := signInStore(files).Directory(ctx)
	if err != nil {
		return "", err //nolint:wrapcheck // names what was read
	}
	for _, entry := range directory {
		if entry.League == league {
			return entry.Football, nil
		}
	}
	return "", nil
}

func writeMoveFile(name string, recipient age.Recipient, manifest rawMoveManifest, league string, football map[string][]byte) (err error) {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create the move file: %w", err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	sealed, err := age.Encrypt(file, recipient)
	if err != nil {
		return fmt.Errorf("seal the move file: %w", err)
	}
	zipped := gzip.NewWriter(sealed)
	archive := tar.NewWriter(zipped)
	body, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encode the move manifest: %w", err)
	}
	if err := tarBytes(archive, moveManifest, body); err != nil {
		return err
	}
	data, err := os.ReadFile(league)
	if err != nil {
		return fmt.Errorf("read the league copy: %w", err)
	}
	if err := tarBytes(archive, leagueFileName, data); err != nil {
		return err
	}
	for digest, stored := range football {
		if err := tarBytes(archive, moveFootball+digest, stored); err != nil {
			return err
		}
	}
	if err := errors.Join(archive.Close(), zipped.Close(), sealed.Close()); err != nil {
		return fmt.Errorf("finish the move file: %w", err)
	}
	return file.Sync() //nolint:wrapcheck // names the file
}

func tarBytes(archive *tar.Writer, name string, body []byte) error {
	if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		return fmt.Errorf("write %s into the move file: %w", name, err)
	}
	if _, err := archive.Write(body); err != nil {
		return fmt.Errorf("write %s into the move file: %w", name, err)
	}
	return nil
}

// sealMoving records the move beside league's file and stops its changes at once.
func sealMoving(files stateFiles, league string, moving rawMoving) error {
	marker, err := movingPath(files.leagues.root, league)
	if err != nil {
		return err
	}
	body, err := json.Marshal(moving)
	if err != nil {
		return fmt.Errorf("record the move: %w", err)
	}
	if err := os.WriteFile(marker, body, 0o600); err != nil {
		return fmt.Errorf("record the move: %w", err)
	}
	if err := syncPath(filepath.Dir(marker)); err != nil {
		return err
	}
	files.leagues.mu.Lock()
	defer files.leagues.mu.Unlock()
	if file := files.leagues.open[league]; file != nil {
		file.moving = true
	}
	return nil
}

// readMoveFile opens a move file with its code into stage: the manifest, the league file and any
// football versions it carries. Nothing else is accepted.
func readMoveFile(name string, code *age.X25519Identity, stage string) (rawMoveManifest, map[string][]byte, error) {
	file, err := os.Open(name)
	if err != nil {
		return rawMoveManifest{}, nil, fmt.Errorf("open the move file: %w", err)
	}
	defer func() { _ = file.Close() }()
	opened, err := age.Decrypt(file, code)
	if err != nil {
		return rawMoveManifest{}, nil, fmt.Errorf("that move code does not open this move file")
	}
	zipped, err := gzip.NewReader(opened)
	if err != nil {
		return rawMoveManifest{}, nil, fmt.Errorf("the move file is damaged: %w", err)
	}
	entries := moveEntries{stage: stage, football: map[string][]byte{}}
	archive := tar.NewReader(zipped)
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return rawMoveManifest{}, nil, fmt.Errorf("the move file is damaged: %w", err)
		}
		if err := entries.take(header, archive); err != nil {
			return rawMoveManifest{}, nil, err
		}
	}
	if !entries.league || !entries.manifested {
		return rawMoveManifest{}, nil, fmt.Errorf("the move file is incomplete")
	}
	return entries.manifest, entries.football, nil
}

// moveEntries collects a move file's entries as they are read, each kind at most once.
type moveEntries struct {
	stage              string
	manifest           rawMoveManifest
	football           map[string][]byte
	league, manifested bool
}

func (m *moveEntries) take(header *tar.Header, archive io.Reader) error {
	if header.Typeflag != tar.TypeReg || header.Size < 0 || header.Size > moveMaxFile {
		return fmt.Errorf("the move file holds something it should not (%q)", header.Name)
	}
	body, err := io.ReadAll(io.LimitReader(archive, header.Size))
	if err != nil {
		return fmt.Errorf("the move file is damaged: %w", err)
	}
	name := header.Name
	switch {
	case name == moveManifest && !m.manifested:
		m.manifested = true
		if err := json.Unmarshal(body, &m.manifest); err != nil || m.manifest.Version != 1 || !validLeagueID(m.manifest.League) {
			return fmt.Errorf("the move file's manifest is not one this server reads")
		}
	case name == leagueFileName && !m.league:
		m.league = true
		if err := os.WriteFile(filepath.Join(m.stage, leagueFileName), body, 0o600); err != nil {
			return fmt.Errorf("stage the league file: %w", err)
		}
	case strings.HasPrefix(name, moveFootball) && validDigest(strings.TrimPrefix(name, moveFootball)):
		m.football[strings.TrimPrefix(name, moveFootball)] = body
	default:
		return fmt.Errorf("the move file holds something it should not (%q)", name)
	}
	return nil
}

func validDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// checkMovedLeague opens the staged league file read-only and proves it is league's, whole.
func checkMovedLeague(ctx context.Context, staged, league string) error {
	pools, err := db.OpenReadOnly(ctx, staged, history.LeagueSchemaVersion())
	if err != nil {
		return fmt.Errorf("open the moved league file: %w", err)
	}
	defer func() { _ = pools.Close() }()
	if err := checkLeagueFile(ctx, pools, league); err != nil {
		return fmt.Errorf("the moved league file: %w", err)
	}
	if _, err := history.New(pools, nil).VerifyAudit(ctx); err != nil {
		return fmt.Errorf("the moved league file's history does not check: %w", err)
	}
	return nil
}
