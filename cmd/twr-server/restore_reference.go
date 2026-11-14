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

// A league set saved after the reference move needs a reference file beside it. Restore takes the
// reference set nearest after the league set (it is captured after the league set in every round) that
// holds everything the leagues need, puts each raw capture back from its own object, and proves the
// archive against the set's checkpoint.
//
// Football data never holds a league back (Christopher, 2026-11-14: "league comes back, football
// waits"). When no reference set holds everything, the nearest one that opens is used, or an empty
// reference file when none opens; it is completed from what is still at hand: each pinned football
// version from its own off-site evidence object, each owned player from the league's own restore
// points. What is still missing is said in plain words, and the restore goes on. A wrong key or a set
// from a newer release still stops it.

// attachReference puts the best reference file beside a restored league file that needs one.
func attachReference(ctx context.Context, sets recoverySets, identity *age.X25519Identity, dir string, captured time.Time, output io.Writer) error {
	if wanted, err := referenceWanted(ctx, dir); err != nil || !wanted {
		return err
	}
	need, err := referenceNeeds(ctx, dir)
	if err != nil {
		return err
	}
	names, err := sets.KindSets(ctx, offsite.ReferenceSet)
	if err != nil {
		return fmt.Errorf("list football reference backups: %w", err)
	}
	fallback, fallbackName := "", ""
	for i, name := range nearestAfter(names, captured)[:min(len(names), 64)] {
		sub := filepath.Join(filepath.Dir(dir), fmt.Sprintf("%s-reference-%02d", filepath.Base(dir), i))
		lacks, err := unlockReference(ctx, sets, name, identity, sub, need)
		if errors.Is(err, offsite.ErrRestoreKey) || errors.Is(err, errNewerSet) {
			return errors.Join(err, os.RemoveAll(sub), removeIf(fallback))
		}
		if err == nil && lacks == "" {
			err = os.Rename(filepath.Join(sub, referenceFileName), filepath.Join(dir, referenceFileName))
			return errors.Join(err, os.RemoveAll(sub), removeIf(fallback))
		}
		if err == nil && fallback == "" {
			fallback, fallbackName = sub, name // the nearest set that opens; kept in case none is complete
			continue
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
	return completeReference(ctx, sets, identity, dir, fallback, fallbackName, need, output)
}

// referenceWanted says whether a staged set keeps football reference data in a file of its own: a set
// saved before the move carries it in its league file.
func referenceWanted(ctx context.Context, dir string) (bool, error) {
	if stagedSplit(dir) {
		return true, nil
	}
	return leagueHas(ctx, filepath.Join(dir, "server.db"), history.ReferenceMoveTable)
}

func removeIf(dir string) error {
	if dir == "" {
		return nil
	}
	return os.RemoveAll(dir) //nolint:wrapcheck // joined into the caller's error
}

// completeReference installs the nearest reference set that opened (or an empty reference file when
// none did), completes it from what is still at hand, and reports what is still missing.
func completeReference(ctx context.Context, sets recoverySets, identity *age.X25519Identity, dir, fallback, name string,
	need referenceNeed, output io.Writer) error {
	target := filepath.Join(dir, referenceFileName)
	from := "no football reference backup could be opened, so an empty football reference file was made"
	if fallback != "" {
		if err := os.Rename(filepath.Join(fallback, referenceFileName), target); err != nil {
			return errors.Join(fmt.Errorf("install football reference backup: %w", err), os.RemoveAll(fallback))
		}
		if err := os.RemoveAll(fallback); err != nil {
			return fmt.Errorf("remove staged backup: %w", err)
		}
		from = "football reference backup " + name + " is older than this league backup"
	}
	recovered, err := fillReference(ctx, sets, identity, dir, need)
	if err != nil {
		return err
	}
	lacks, err := referenceLacks(ctx, target, need)
	if err != nil {
		return err
	}
	words := fmt.Sprintf("The league is restored with its rosters and contracts, but %s. Put back from what was at hand: %d player(s) from the league's own record, %d football version(s) from off-site evidence.",
		from, recovered.players, recovered.pins)
	if len(recovered.conflicts) > 0 {
		words += " Left out because another player already holds the same outside id: " + strings.Join(recovered.conflicts, "; ") + "."
	}
	if lacks != "" {
		words += " Still missing: " + strings.TrimPrefix(lacks, "it ") + ". Anything that needs it waits until it is back; the server says so on its status page."
	}
	if _, err := fmt.Fprintln(output, words); err != nil {
		return fmt.Errorf("report restore: %w", err)
	}
	return nil
}

type referenceFill struct {
	players, pins int
	conflicts     []string
}

// fillReference completes the installed reference file: an empty one gets its schema, each pinned
// football version it lacks comes from its own evidence object when that object exists, and each player
// the leagues own comes from the league file's restore points.
func fillReference(ctx context.Context, sets recoverySets, identity *age.X25519Identity, dir string, need referenceNeed) (referenceFill, error) {
	var out referenceFill
	reference, err := db.Open(ctx, filepath.Join(dir, referenceFileName), history.ReferenceSchemaVersion())
	if err != nil {
		return out, fmt.Errorf("open football reference file: %w", err)
	}
	defer func() { _ = reference.Close() }()
	if err := history.PrepareReferenceFile(ctx, reference); err != nil {
		return out, err //nolint:wrapcheck // names the file
	}
	digests := make([]string, 0, len(need.pins))
	for digest := range need.pins {
		digests = append(digests, digest)
	}
	slices.Sort(digests)
	_, missing, err := history.MissingReference(ctx, reference, nil, digests)
	if err != nil {
		return out, err //nolint:wrapcheck // names what was read
	}
	for _, digest := range missing {
		stored, err := openEvidence(ctx, sets, identity, digest)
		if err != nil {
			continue // not off-site either; reported as still missing
		}
		if err := history.PutArchiveBody(ctx, reference, digest, stored); err != nil {
			return out, err //nolint:wrapcheck // names the capture
		}
		out.pins++
	}
	files, err := stagedLeagues(dir)
	if err != nil {
		return out, err
	}
	for _, file := range files {
		league, err := db.OpenReadOnly(ctx, file, history.LeagueSchemaVersion())
		if err != nil {
			return out, fmt.Errorf("open restored database: %w", err)
		}
		recovered, err := history.RecoverLeagueIdentities(ctx, league, reference)
		if err = errors.Join(err, league.Close()); err != nil {
			return out, err
		}
		out.players += recovered.Installed
		out.conflicts = append(out.conflicts, recovered.Conflicts...)
	}
	return out, nil
}

// referenceNeed is what a restored league file and its sign-in file ask of a reference file: the
// player ids the leagues own, and each pinned football digest with the leagues that pin it.
type referenceNeed struct {
	players []string
	pins    map[string][]string
}

func referenceNeeds(ctx context.Context, dir string) (referenceNeed, error) {
	need := referenceNeed{pins: map[string][]string{}}
	files, err := stagedLeagues(dir)
	if err != nil {
		return need, err
	}
	for _, file := range files {
		league, err := db.OpenReadOnly(ctx, file, history.LeagueSchemaVersion())
		if err != nil {
			return need, fmt.Errorf("open restored database: %w", err)
		}
		players, err := history.LeaguePlayers(ctx, league)
		if err = errors.Join(err, league.Close()); err != nil {
			return need, err
		}
		need.players = append(need.players, players...)
	}
	slices.Sort(need.players)
	need.players = slices.Compact(need.players)
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
	file, err := db.Inspect(ctx, path, history.ReferenceSchemaVersion())
	if file.Schema > history.ReferenceSchemaVersion() || file.Release > serverversion.Current().Release {
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
	object, err := sets.OpenEvidence(ctx, identity.Recipient().String(), sha)
	if err != nil {
		return nil, err //nolint:wrapcheck // names the capture
	}
	defer func() { _ = object.Close() }()
	return offsite.UnsealEvidence(object, identity) //nolint:wrapcheck // names the failure
}

func referenceLacks(ctx context.Context, path string, need referenceNeed) (string, error) {
	pools, err := db.OpenReadOnly(ctx, path, history.ReferenceSchemaVersion())
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
	pools, err := db.OpenReadOnly(ctx, league, history.LeagueSchemaVersion())
	if err != nil {
		return false, fmt.Errorf("open restored database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	return hasTable(ctx, pools, table)
}
