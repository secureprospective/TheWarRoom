package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// A league set saved after the reference move needs a reference set beside it. Restore takes the
// reference set nearest after the league set (it is captured after the league set in every round),
// puts each raw capture back from its own object, proves the archive against the set's checkpoint, and
// refuses a reference set that lacks a player id the league owns or a football version a league pins.

// attachReference puts the matching reference file beside a restored league file that needs one.
func attachReference(ctx context.Context, sets recoverySets, identity *age.X25519Identity, dir string, captured time.Time, output io.Writer) error {
	league := filepath.Join(dir, "server.db")
	moved, err := leagueHas(ctx, league, history.ReferenceMoveTable)
	if err != nil || !moved {
		return err // a set saved before the move carries reference data in its league file
	}
	need, err := referenceNeeds(ctx, dir)
	if err != nil {
		return err
	}
	names, err := sets.KindSets(ctx, offsite.ReferenceSet)
	if err != nil {
		return fmt.Errorf("list football reference backups: %w", err)
	}
	for i, name := range nearestAfter(names, captured)[:min(len(names), 64)] {
		sub := filepath.Join(filepath.Dir(dir), fmt.Sprintf("%s-reference-%02d", filepath.Base(dir), i))
		lacks, err := unlockReference(ctx, sets, name, identity, sub, need)
		if errors.Is(err, offsite.ErrRestoreKey) || errors.Is(err, errNewerSet) {
			return errors.Join(err, os.RemoveAll(sub))
		}
		if err == nil && lacks == "" {
			err = os.Rename(filepath.Join(sub, referenceFileName), filepath.Join(dir, referenceFileName))
			return errors.Join(err, os.RemoveAll(sub))
		}
		if err == nil {
			err = errors.New(lacks)
		}
		if _, werr := fmt.Fprintf(output, "Skipping football reference backup %s: %v\n", name, err); werr != nil {
			return fmt.Errorf("report restore: %w", werr)
		}
		if err := os.RemoveAll(sub); err != nil {
			return fmt.Errorf("remove skipped backup: %w", err)
		}
	}
	return fmt.Errorf("no football reference backup matches the league backup saved %s (it must hold every player the leagues own and every football version a league pins); nothing was installed",
		captured.UTC().Format(time.RFC3339))
}

// referenceNeed is what a restored league file and its sign-in file ask of a reference file: the
// player ids the leagues own, and each pinned football digest with the leagues that pin it.
type referenceNeed struct {
	players []string
	pins    map[string][]string
}

func referenceNeeds(ctx context.Context, dir string) (referenceNeed, error) {
	need := referenceNeed{pins: map[string][]string{}}
	league, err := db.OpenReadOnly(ctx, filepath.Join(dir, "server.db"), history.SchemaVersion())
	if err != nil {
		return need, fmt.Errorf("open restored database: %w", err)
	}
	need.players, err = history.LeaguePlayers(ctx, league)
	if err = errors.Join(err, league.Close()); err != nil {
		return need, err
	}
	path := filepath.Join(dir, signInFileName)
	if !fileExists(path) {
		return need, nil
	}
	signIns, err := db.OpenReadOnly(ctx, path, accountstore.ServerVersion())
	if err != nil {
		return need, fmt.Errorf("open restored sign-in file: %w", err)
	}
	defer func() { _ = signIns.Close() }()
	directory, err := accountstore.New(signIns).Directory(ctx)
	if err != nil {
		return need, err //nolint:wrapcheck // names what was read
	}
	for _, entry := range directory {
		if entry.Football != "" {
			need.pins[entry.Football] = append(need.pins[entry.Football], entry.League)
		}
	}
	return need, nil
}

// unlockReference decrypts one reference set, fills its raw captures, checks it against the checkpoint
// saved inside it, and says in plain words what it lacks ("" when nothing).
func unlockReference(ctx context.Context, sets recoverySets, name string, identity *age.X25519Identity, dir string, need referenceNeed) (string, error) {
	body, err := sets.Open(ctx, name)
	if err != nil {
		return "", fmt.Errorf("download backup: %w", err)
	}
	defer func() { _ = body.Close() }()
	manifest, err := offsite.Restore(body, identity, dir)
	if err != nil {
		return "", fmt.Errorf("unlock backup: %w", err)
	}
	if !manifest.Holds(offsite.ReferenceSet.File) {
		return "", fmt.Errorf("it is not a football reference backup")
	}
	path := filepath.Join(dir, referenceFileName)
	file, err := db.Inspect(ctx, path, history.SchemaVersion())
	if file.Schema > history.SchemaVersion() || file.Release > serverversion.Current().Release {
		return "", errNewerSet
	}
	if err != nil {
		return "", fmt.Errorf("read football reference file: %w", err)
	}
	saved, err := readCheckpoint(filepath.Join(dir, "checkpoint.json"))
	if err != nil {
		return "", err
	}
	if saved.Reference == nil {
		return "", fmt.Errorf("its checkpoint names no football reference archive")
	}
	anchor, err := history.FillArchiveBodies(ctx, path, func(sha string) ([]byte, error) { return openEvidence(ctx, sets, identity, sha) })
	if err != nil {
		return "", fmt.Errorf("put raw captures back: %w", err)
	}
	if anchor.Rows != saved.Reference.Bodies || anchor.Digest != saved.Reference.Archive {
		return "", fmt.Errorf("%w (football reference archive, saved %s)", errCheckpointMismatch, saved.Captured.Format(time.DateOnly))
	}
	return referenceLacks(ctx, path, need)
}

func openEvidence(ctx context.Context, sets recoverySets, identity *age.X25519Identity, sha string) ([]byte, error) {
	object, err := sets.OpenEvidence(ctx, sha)
	if err != nil {
		return nil, err //nolint:wrapcheck // names the capture
	}
	defer func() { _ = object.Close() }()
	return offsite.UnsealEvidence(object, identity) //nolint:wrapcheck // names the failure
}

func referenceLacks(ctx context.Context, path string, need referenceNeed) (string, error) {
	pools, err := db.OpenReadOnly(ctx, path, history.SchemaVersion())
	if err != nil {
		return "", fmt.Errorf("open football reference file: %w", err)
	}
	defer func() { _ = pools.Close() }()
	digests := make([]string, 0, len(need.pins))
	for digest := range need.pins {
		digests = append(digests, digest)
	}
	slices.Sort(digests)
	players, bodies, err := history.MissingReference(ctx, pools, need.players, digests)
	if err != nil {
		return "", err //nolint:wrapcheck // names what was read
	}
	var lacks []string
	if len(players) > 0 {
		lacks = append(lacks, fmt.Sprintf("it lacks %d player(s) the leagues own", len(players)))
	}
	for _, digest := range bodies {
		lacks = append(lacks, fmt.Sprintf("league %s pins football version %s, which it does not hold", strings.Join(need.pins[digest], ", "), digest))
	}
	return strings.Join(lacks, "; "), nil
}

func leagueHas(ctx context.Context, league, table string) (bool, error) {
	pools, err := db.OpenReadOnly(ctx, league, history.SchemaVersion())
	if err != nil {
		return false, fmt.Errorf("open restored database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	return hasTable(ctx, pools, table)
}
