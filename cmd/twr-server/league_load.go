package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueload"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// runLeagueLoad is a local console command, run with the hosted service stopped. It puts the league
// held in a desktop MFL mirror onto this server: the commissioner's account is created on the server
// (first account only, or the existing account after its password checks), the native league is
// built from held MFL reads only, then moved here by the confirmed seal/activate path. The mirror is
// opened read-only; no MFL call is made.
func runLeagueLoad(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server league-load", flag.ContinueOnError)
	path := flags.String("db", "", "server database (service stopped); created when absent from an empty state directory")
	mirrorPath := flags.String("mirror", "", "a copy of the commissioner's desktop history.db; opened read-only")
	mflLeague := flags.String("mfl-league", "", "MFL league number held in the mirror")
	pin := flags.String("football-pin", "", "file to receive the pinned football digest the service starts with; empty writes none")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse league load flags: %w", err)
	}
	if flags.NArg() != 0 || *mflLeague == "" {
		return fmt.Errorf("league-load accepts only -db, -mirror, -mfl-league and -football-pin")
	}
	raw, err := readConsoleCredentials(input)
	if err != nil {
		return err
	}
	if *path == "" {
		if err := prepareFirstLoad(*path); err != nil {
			return err
		}
		return checkHostedDatabase(*path, 0)
	}
	return withStateDirectoryOwnership(filepath.Dir(*path), func() error {
		if err := prepareFirstLoad(*path); err != nil {
			return err
		}
		if err := checkHostedDatabase(*path, 0); err != nil {
			return err
		}
		return loadLeague(ctx, *path, *mirrorPath, *mflLeague, *pin, raw, output)
	})
}

// prepareFirstLoad lets the first load run before the service has ever started: when -db is absent and
// its directory is empty, it creates the empty server database the installer would have made. Nothing
// else is touched; no network is used and no player identity is filled in. Every other case is left to
// checkHostedDatabase.
func prepareFirstLoad(path string) error {
	if path == "" {
		return nil
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return nil
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("first load: read the state directory: %w", err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("first load: %s is absent but its directory is not empty; refusing to create a server database beside other files", path)
	}
	owner, err := stateDirectoryOwner(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("first load: read the state directory owner: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("first load: create the server database: %w", err)
	}
	closeErr := file.Close()
	ownerErr := chownCreatedStatePaths(filepath.Dir(path), owner, []string{path})
	if err := errors.Join(closeErr, ownerErr); err != nil {
		return fmt.Errorf("first load: finish the server database: %w", err)
	}
	return nil
}

type stateFileOwner struct {
	uid int
	gid int
}

func stateDirectoryOwner(path string) (stateFileOwner, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return stateFileOwner{}, fmt.Errorf("inspect state directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return stateFileOwner{}, fmt.Errorf("state path is not a real directory")
	}
	return stateOwnerFromFileInfo(info)
}

// Sys is platform-specific; use its Unix uid/gid fields without tying server code to one Stat_t type.
func stateOwnerFromFileInfo(info os.FileInfo) (stateFileOwner, error) {
	stat := reflect.ValueOf(info.Sys())
	if stat.Kind() == reflect.Pointer {
		stat = stat.Elem()
	}
	if stat.Kind() != reflect.Struct {
		return stateFileOwner{}, fmt.Errorf("state directory has no Unix ownership")
	}
	uid, ok := statID(stat.FieldByName("Uid"))
	if !ok {
		return stateFileOwner{}, fmt.Errorf("state directory has no Unix uid")
	}
	gid, ok := statID(stat.FieldByName("Gid"))
	if !ok {
		return stateFileOwner{}, fmt.Errorf("state directory has no Unix gid")
	}
	return stateFileOwner{uid: uid, gid: gid}, nil
}

func statID(field reflect.Value) (int, bool) {
	if field.CanUint() {
		value := field.Uint()
		if value > uint64(^uint(0)>>1) {
			return 0, false
		}
		return int(value), true
	}
	if field.CanInt() {
		value := field.Int()
		if value < 0 || uint64(value) > uint64(^uint(0)>>1) {
			return 0, false
		}
		return int(value), true
	}
	return 0, false
}

// withStateDirectoryOwnership gives only paths created by this load the state directory's owner.
// It compares the fixed state layout before and after, then changes each new path individually.
func withStateDirectoryOwnership(stateDir string, action func() error) (result error) {
	owner, err := stateDirectoryOwner(stateDir)
	if err != nil {
		return fmt.Errorf("read state directory owner: %w", err)
	}
	before, err := statePathsSnapshot(stateDir)
	if err != nil {
		return err
	}
	defer func() {
		after, err := statePathsSnapshot(stateDir)
		if err != nil {
			result = errors.Join(result, fmt.Errorf("inspect created state paths: %w", err))
			return
		}
		created := make([]string, 0, len(after))
		for path := range after {
			if _, existed := before[path]; !existed {
				created = append(created, path)
			}
		}
		sort.Strings(created)
		result = errors.Join(result, chownCreatedStatePaths(stateDir, owner, created))
	}()
	return action()
}

// statePathsSnapshot inspects only state files and one league-directory level; it never follows links.
func statePathsSnapshot(stateDir string) (map[string]struct{}, error) {
	root, err := filepath.Abs(stateDir)
	if err != nil {
		return nil, fmt.Errorf("resolve state directory: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect state directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("state path is not a real directory")
	}
	paths := make(map[string]struct{})
	if err := snapshotDirectoryChildren(root, paths); err != nil {
		return nil, err
	}
	if err := snapshotLeagueChildren(filepath.Join(root, leaguesFolder), paths); err != nil {
		return nil, err
	}
	return paths, nil
}

func snapshotDirectoryChildren(path string, paths map[string]struct{}) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("read state directory: %w", err)
	}
	for _, entry := range entries {
		paths[filepath.Join(path, entry.Name())] = struct{}{}
	}
	return nil
}

func snapshotLeagueChildren(path string, paths map[string]struct{}) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect league directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("league path is not a real directory")
	}
	leagues, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("read league directory: %w", err)
	}
	for _, league := range leagues {
		if err := snapshotLeague(path, league.Name(), paths); err != nil {
			return err
		}
	}
	return nil
}

func snapshotLeague(parent, name string, paths map[string]struct{}) error {
	path := filepath.Join(parent, name)
	paths[path] = struct{}{}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect league path: %w", err)
	}
	if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		if err := snapshotDirectoryChildren(path, paths); err != nil {
			return fmt.Errorf("read league path: %w", err)
		}
	}
	return nil
}

func chownCreatedStatePaths(stateDir string, owner stateFileOwner, paths []string) error {
	root, err := filepath.Abs(stateDir)
	if err != nil {
		return fmt.Errorf("resolve state directory: %w", err)
	}
	var errs []error
	for _, path := range paths {
		if err := chownCreatedStatePath(root, owner, path); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func chownCreatedStatePath(root string, owner stateFileOwner, path string) error {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve created state path: %w", err)
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("created state path is outside the state directory")
	}
	if err := statePathParentsAreReal(root, relative); err != nil {
		return err
	}
	info, err := os.Lstat(resolved)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect created state path: %w", err)
	}
	if !info.Mode().IsRegular() && !info.IsDir() {
		return fmt.Errorf("created state path is not a regular file or directory: %s", filepath.Base(path))
	}
	if existingOwner, err := stateOwnerFromFileInfo(info); err == nil && existingOwner == owner {
		return nil
	}
	if err := os.Chown(resolved, owner.uid, owner.gid); err != nil {
		return fmt.Errorf("set owner on created state path %s: %w", filepath.Base(path), err)
	}
	return nil
}

func statePathParentsAreReal(root, relative string) error {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("state path is not under a real directory")
	}
	parent := filepath.Dir(relative)
	current := root
	if parent != "." {
		for _, part := range strings.Split(parent, string(filepath.Separator)) {
			current = filepath.Join(current, part)
			info, err := os.Lstat(current)
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("state path is not under a real directory")
			}
		}
	}
	return nil
}

func loadLeague(ctx context.Context, path, mirrorPath, mflLeague, pin string, raw accounts.RawCredentials, output io.Writer) error {
	mirrorPools, err := db.OpenReadOnly(ctx, mirrorPath, history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open the mirror read-only: %w", err)
	}
	defer func() { _ = mirrorPools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load server measures: %w", err)
	}
	mirror := leagueload.Mirror{Store: history.New(mirrorPools, registry)}
	held := heldLeague{MFLLeague: mflLeague, Source: leagueload.ReplaySource(ctx, mirror, mflLeague, time.Now),
		Football: func(ctx context.Context, season int) ([]byte, string, error) {
			return mirror.Football(ctx, season, time.Now())
		}}
	return loadHeld(ctx, path, held, pin, raw, output)
}

// heldLeague is what the loader reads the league from: the mirror's held MFL reads in production.
type heldLeague struct {
	MFLLeague string
	Source    func(leaguerules.RuleSet) (leaguemigration.Source, error)
	Football  func(context.Context, int) ([]byte, string, error)
}

func loadHeld(ctx context.Context, path string, held heldLeague, pin string, raw accounts.RawCredentials, output io.Writer) error {
	files, err := openStateFiles(ctx, path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load server measures: %w", err)
	}
	created := &createdLeague{files: files.leagues}
	signIns := signInStore(files)
	commissioner, err := commissionerAccount(ctx, signIns, raw)
	if err != nil {
		return err
	}
	principal, err := commissioner.Principal()
	if err != nil {
		return fmt.Errorf("commissioner identity: %w", err)
	}
	workspace, cleanup, err := scratchHistory(ctx, registry)
	if err != nil {
		return err
	}
	defer cleanup()
	result, err := leagueload.Load(ctx, leagueload.Request{MFLLeague: held.MFLLeague, Commissioner: principal, Workspace: workspace,
		Server: created.create, Source: held.Source, Football: held.Football, Now: time.Now})
	if err != nil {
		// Nothing names a league whose load failed; its half-made file goes with it.
		return errors.Join(fmt.Errorf("load the league: %w", err), files.leagues.discard(created.league))
	}
	// The league has committed in the league file; only now does the directory name it.
	if err := recordDirectory(ctx, created.store, signIns); err != nil {
		return err
	}
	if err := reportLoad(output, result, commissioner.Username, pin); err != nil {
		return err
	}
	return nil
}

// createdLeague makes the loaded league's own file when the loader asks for it, and remembers it.
type createdLeague struct {
	files  *leagueFiles
	league string
	store  *history.Store
}

func (c *createdLeague) create(ctx context.Context, league string) (*history.Store, error) {
	store, err := c.files.create(ctx, league)
	if err != nil {
		return nil, err
	}
	c.league, c.store = league, store
	return store, store.Initialize(ctx) //nolint:wrapcheck // the loader names the step
}

// reportLoad writes the loaded league's football pin, when asked for, and says what was loaded.
func reportLoad(output io.Writer, result leagueload.Result, commissioner, pin string) error {
	if pin != "" {
		if err := writePin(pin, result.Football); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(output, "League %s is active on this server with %d franchises; %s is its commissioner. Football evidence %s.\n",
		result.League, result.Franchises, commissioner, result.Football)
	if err != nil {
		return fmt.Errorf("report league load (the load has already committed): %w", err)
	}
	return nil
}

// commissionerAccount creates the server's first account, or signs in the existing one by password.
func commissionerAccount(ctx context.Context, store *accountstore.Store, raw accounts.RawCredentials) (accounts.Account, error) {
	if err := store.Initialize(ctx); err != nil {
		return accounts.Account{}, fmt.Errorf("initialize server accounts: %w", err)
	}
	service := accounts.New(store)
	account, err := service.LocalBootstrap(ctx, raw)
	if errors.Is(err, accounts.ErrBootstrapClosed) {
		account, err = service.Authenticate(ctx, raw)
	}
	if err != nil {
		return accounts.Account{}, fmt.Errorf("commissioner account: %w", err)
	}
	return account, nil
}

// scratchHistory is the disposable workspace the native league is built in before it moves.
func scratchHistory(ctx context.Context, registry *measures.Registry) (*history.Store, func(), error) {
	pools, err := db.OpenScratch(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("open league workspace: %w", err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(ctx); err != nil {
		_ = pools.Close()
		return nil, nil, fmt.Errorf("initialize league workspace: %w", err)
	}
	return store, func() { _ = pools.Close() }, nil
}

func readConsoleCredentials(input io.Reader) (accounts.RawCredentials, error) {
	body, err := io.ReadAll(io.LimitReader(input, 4097))
	if err != nil || len(body) > 4096 {
		return accounts.RawCredentials{}, fmt.Errorf("commissioner input exceeds 4096 bytes or cannot be read")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	// RawCredentials keeps the password out of JSON on purpose; this console input is the one place it is read.
	var fields struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decoder.Decode(&fields); err != nil {
		return accounts.RawCredentials{}, fmt.Errorf("supply the commissioner's username and password as one JSON object on stdin")
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return accounts.RawCredentials{}, fmt.Errorf("supply exactly one JSON object on stdin")
	}
	raw := accounts.RawCredentials{Username: fields.Username, Password: fields.Password}
	if err := raw.Validate(); err != nil {
		return accounts.RawCredentials{}, fmt.Errorf("commissioner credentials: %w", err)
	}
	return raw, nil
}

// writePin replaces the pin file atomically; it holds only a public digest.
func writePin(path, digest string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".football-pin-")
	if err != nil {
		return fmt.Errorf("write football pin: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString(digest + "\n"); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write football pin: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write football pin: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write football pin: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("write football pin: %w", err)
	}
	return nil
}
