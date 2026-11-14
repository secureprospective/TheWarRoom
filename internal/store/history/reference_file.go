package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// Football reference data is shared by every league on a server: the raw archive (fetched bodies and
// the blobs a transfer brings), loads and observations, model and scoring runs, compiled inputs, live
// lines, permanent player identity with its outside links and records, and the measure registry
// mirror. On the hosted server it lives in its own file, attached to the league file's connections
// as schema "reference" (db.OpenWithReference). A league refers to reference data only by content:
// a raw archive sha256, a football evidence digest, a measure name. The one exception is player
// identity: league rows hold permanent canonical ids, and the league file keeps its own copy of the
// ids it owns (league_player) so its ownership foreign key stays inside one file.
//
// Each hosted file has its own migration list and user_version.

// ReferenceTables are the tables that move to the reference file, parents first: a move copies them
// in this order and drops them from the league file in the reverse order.
func ReferenceTables() []string {
	return []string{"canonical_player", "player_external_id", "player_record",
		"raw_archive", "fetch_log", "loads", "observations", "unresolved_observations",
		"param_sets", "scoring_runs", "run_scores", "model_runs", "model_scores", "quarantine_model_runs", "quarantine_model_scores",
		"input_manifests", "compiled_snapshots", "snapshot_checks",
		"live_evidence", "live_chronology", "live_board",
		"measures", "sources", "source_fields"}
}

// referenceViews read reference tables only; they go with them.
func referenceViews() []string { return []string{"player_ids", "season_features", "week_features"} }

// ReferenceMoveTable is the league file's record that its reference tables live in the reference file.
const ReferenceMoveTable = "reference_move"

// rawArchive is proved by its set of sha256 keys and each body hashing to its key, not by a row digest.
const rawArchive = "raw_archive"

// PrepareReferenceFile brings a reference file to this build's schema.
func PrepareReferenceFile(ctx context.Context, pools *db.Pools) error {
	if _, err := pools.Migrate(ctx, referenceMigrations()...); err != nil {
		return fmt.Errorf("history: reference file: %w", err)
	}
	return nil
}

// PrepareNewLeague creates only league tables in a brand-new hosted file.
func PrepareNewLeague(ctx context.Context, pools *db.Pools) error {
	var version int
	if err := pools.Read().QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("history: read the league file version: %w", err)
	}
	if version != 0 {
		return nil
	}
	return migrateLeagueFile(ctx, pools)
}

// MigrateLeague chooses the file's own list after the reference move. Combined desktop
// and pre-conversion files retain their original migrations until they are split.
func MigrateLeague(ctx context.Context, pools *db.Pools) error {
	split, err := db.HasTable(ctx, pools, LeagueSplitTable)
	if err != nil {
		return fmt.Errorf("history: read league layout: %w", err)
	}
	if split {
		// The shared file after the split holds no league tables: it is at every league version at once.
		var version int
		if err := pools.Read().QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
			return fmt.Errorf("history: read the split shared file's version: %w", err)
		}
		if version < LeagueSchemaVersion() {
			if _, err := pools.Write().ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version=%d`, LeagueSchemaVersion())); err != nil {
				return fmt.Errorf("history: stamp the split shared file: %w", err)
			}
		}
		return nil
	}
	moved, err := db.HasTable(ctx, pools, ReferenceMoveTable)
	if err != nil {
		return fmt.Errorf("history: read league layout: %w", err)
	}
	if moved || pools.Reference() != "" {
		return migrateLeagueFile(ctx, pools)
	}
	if _, err := pools.Migrate(ctx, historyMigrations()...); err != nil {
		return fmt.Errorf("history: migrate history file: %w", err)
	}
	return nil
}

func migrateLeagueFile(ctx context.Context, pools *db.Pools) error {
	if _, err := pools.MigrateFrom(ctx, leagueMigrationBase, leagueMigrations()...); err != nil {
		return fmt.Errorf("history: migrate league file: %w", err)
	}
	return nil
}

func referenceRows() []db.MovedRows {
	tables := ReferenceTables()
	out := make([]db.MovedRows, 0, len(tables))
	for _, t := range tables {
		out = append(out, db.MovedRows{Table: t})
	}
	return out
}

// CopyReference copies every reference table from the league file into the new reference file, as
// stored, in one transaction. Identity is copied before evidence, so the evidence identity triggers
// find every link already present and mint nothing.
func CopyReference(ctx context.Context, from, target *db.Pools) error {
	tx, err := target.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("history: begin the reference copy: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, rows := range referenceRows() {
		if err := db.CopyRows(ctx, from.Read(), tx, rows); err != nil {
			return fmt.Errorf("history: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: commit the reference copy: %w", err)
	}
	return nil
}

// ReferenceProof is the evidence a move was exact: per table its count and ordered-row digest, and for
// the raw archive its content digest (archiveKeys).
type ReferenceProof struct {
	Tables  map[string]db.RowProof
	Archive db.RowProof
}

// ProveReference reads the proof of pools' own reference tables.
func ProveReference(ctx context.Context, pools *db.Pools) (ReferenceProof, error) {
	proof := ReferenceProof{Tables: map[string]db.RowProof{}}
	for _, m := range referenceRows() {
		if m.Table == rawArchive {
			continue
		}
		p, err := db.ProveRows(ctx, pools.Read(), m)
		if err != nil {
			return proof, fmt.Errorf("history: %w", err)
		}
		proof.Tables[m.Table] = p
	}
	archive, err := archiveKeys(ctx, pools)
	if err != nil {
		return proof, err
	}
	proof.Archive = archive
	return proof, nil
}

// archiveKeys digests the raw archive by content, in key order: each body's key, recorded size and the
// sha256 of its stored (compressed) bytes, so a missing, extra or changed body changes the digest.
func archiveKeys(ctx context.Context, pools *db.Pools) (db.RowProof, error) {
	rows, err := pools.Read().QueryContext(ctx, `SELECT sha256, size, body FROM main.raw_archive ORDER BY sha256`)
	if err != nil {
		return db.RowProof{}, fmt.Errorf("history: read the raw archive: %w", err)
	}
	defer func() { _ = rows.Close() }()
	keys := sha256.New()
	var proof db.RowProof
	for rows.Next() {
		var key string
		var size int64
		var body []byte
		if err := rows.Scan(&key, &size, &body); err != nil {
			return db.RowProof{}, fmt.Errorf("history: read the raw archive: %w", err)
		}
		stored := sha256.Sum256(body)
		fmt.Fprintf(keys, "%s,%d,%x\n", key, size, stored)
		proof.Rows++
	}
	if err := rows.Err(); err != nil {
		return db.RowProof{}, fmt.Errorf("history: read the raw archive: %w", err)
	}
	proof.Digest = hex.EncodeToString(keys.Sum(nil))
	return proof, nil
}

// SameReference reports the first table where two proofs differ.
func SameReference(want, got ReferenceProof) error {
	if want.Archive != got.Archive {
		return fmt.Errorf("history: raw_archive did not copy exactly (%d bodies before, %d after)", want.Archive.Rows, got.Archive.Rows)
	}
	for _, table := range ReferenceTables() {
		if table == rawArchive {
			continue
		}
		if want.Tables[table] != got.Tables[table] {
			return fmt.Errorf("history: %s did not copy exactly (%d rows before, %d after)", table, want.Tables[table].Rows, got.Tables[table].Rows)
		}
	}
	return nil
}

// ReferenceMove is what the league file records about the move.
type ReferenceMove struct {
	File    string
	Bodies  int64
	Archive string
}

// CommitReferenceMove drops the reference views and tables from the league file and records the move,
// in one transaction. Dropping a table is not a row delete, so the append-only triggers go with it.
func CommitReferenceMove(ctx context.Context, league *db.Pools, record ReferenceMove) error {
	tx, err := league.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("history: begin the reference move: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, view := range referenceViews() {
		if _, err := tx.ExecContext(ctx, `DROP VIEW IF EXISTS main.`+view); err != nil {
			return fmt.Errorf("history: remove %s from the league file: %w", view, err)
		}
	}
	tables := slices.Clone(ReferenceTables())
	slices.Reverse(tables)
	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS main.`+table); err != nil {
			return fmt.Errorf("history: remove %s from the league file: %w", table, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE main.`+ReferenceMoveTable+` (
 id INTEGER PRIMARY KEY CHECK(id = 1), moved_at TEXT NOT NULL, file TEXT NOT NULL,
 bodies INTEGER NOT NULL, archive TEXT NOT NULL) STRICT`); err != nil {
		return fmt.Errorf("history: create the reference move record: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO main.`+ReferenceMoveTable+` VALUES (1, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339Nano), record.File, record.Bodies, record.Archive); err != nil {
		return fmt.Errorf("history: write the reference move record: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: commit the reference move: %w", err)
	}
	return nil
}

// CompactLeague returns the pages the move freed: the league file shrinks to league data. VACUUM is
// atomic; a crash during it leaves the file as it was before it.
func CompactLeague(ctx context.Context, league *db.Pools) error {
	if _, err := league.Write().ExecContext(ctx, `VACUUM`); err != nil {
		return fmt.Errorf("history: compact the league file: %w", err)
	}
	if _, err := league.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return fmt.Errorf("history: flush the league file: %w", err)
	}
	return nil
}

// ArchiveAnchor names a raw archive by content: how many bodies, and archiveKeys' digest.
type ArchiveAnchor = db.RowProof

// StripArchiveBodies prepares a staged copy of the reference file for its off-site set: put receives
// every raw capture (key and body), then each body is emptied in the copy, keeping its key and size,
// and the copy is compacted. The live file is never touched. The anchor is read before stripping.
func StripArchiveBodies(ctx context.Context, path string, put func(sha string, body []byte) error) (ArchiveAnchor, error) {
	return withStagedArchive(ctx, path, func(pools *db.Pools) (ArchiveAnchor, error) {
		anchor, err := archiveKeys(ctx, pools)
		if err != nil {
			return anchor, err
		}
		if err := eachArchiveBody(ctx, pools, func(sha string, body []byte) error { return put(sha, body) }); err != nil {
			return anchor, err
		}
		return anchor, rewriteBodies(ctx, pools, func(tx execer) error {
			_, err := tx.ExecContext(ctx, `UPDATE raw_archive SET body = X''`)
			return err //nolint:wrapcheck // wrapped by rewriteBodies
		})
	})
}

// FillArchiveBodies puts every emptied raw capture back into a restored reference file from get and
// returns the archive's anchor; the caller compares it with the anchor saved with the set.
func FillArchiveBodies(ctx context.Context, path string, get func(sha string) ([]byte, error)) (ArchiveAnchor, error) {
	return withStagedArchive(ctx, path, func(pools *db.Pools) (ArchiveAnchor, error) {
		missing, err := readStrings(ctx, pools, `SELECT sha256 FROM main.raw_archive WHERE size > 0 AND length(body) = 0 ORDER BY sha256`)
		if err != nil {
			return ArchiveAnchor{}, err
		}
		err = rewriteBodies(ctx, pools, func(tx execer) error {
			for _, sha := range missing {
				body, err := get(sha)
				if err != nil {
					return err
				}
				if _, err := tx.ExecContext(ctx, `UPDATE raw_archive SET body = ? WHERE sha256 = ?`, body, sha); err != nil {
					return err //nolint:wrapcheck // wrapped by rewriteBodies
				}
			}
			return nil
		})
		if err != nil {
			return ArchiveAnchor{}, err
		}
		return archiveKeys(ctx, pools)
	})
}

func withStagedArchive(ctx context.Context, path string, work func(*db.Pools) (ArchiveAnchor, error)) (anchor ArchiveAnchor, err error) {
	pools, err := db.Open(ctx, path, ReferenceSchemaVersion())
	if err != nil {
		return anchor, fmt.Errorf("history: open the staged reference file: %w", err)
	}
	defer func() { err = errors.Join(err, pools.Close()) }()
	if anchor, err = work(pools); err != nil {
		return anchor, err
	}
	if _, err := pools.Write().ExecContext(ctx, `VACUUM`); err != nil {
		return anchor, fmt.Errorf("history: compact the staged reference file: %w", err)
	}
	if _, err := pools.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return anchor, fmt.Errorf("history: finish the staged reference file: %w", err)
	}
	return anchor, nil
}

// rewriteBodies changes raw archive bodies in a staged copy only (never the live file): the append-only
// update guard is lifted inside the one transaction and put back before it commits.
func rewriteBodies(ctx context.Context, pools *db.Pools, change func(execer) error) error {
	tx, err := pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("history: begin the staged archive change: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DROP TRIGGER raw_archive_no_update`); err != nil {
		return fmt.Errorf("history: open the staged archive: %w", err)
	}
	if err := change(tx); err != nil {
		return fmt.Errorf("history: change the staged archive: %w", err)
	}
	if _, err := tx.ExecContext(ctx, appendOnlyDDL(rawArchive)); err != nil {
		return fmt.Errorf("history: close the staged archive: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: commit the staged archive change: %w", err)
	}
	return nil
}

func eachArchiveBody(ctx context.Context, pools *db.Pools, visit func(sha string, body []byte) error) error {
	rows, err := pools.Read().QueryContext(ctx, `SELECT sha256, body FROM raw_archive WHERE size > 0 ORDER BY sha256`)
	if err != nil {
		return fmt.Errorf("history: read the raw archive: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var sha string
		var body []byte
		if err := rows.Scan(&sha, &body); err != nil {
			return fmt.Errorf("history: read the raw archive: %w", err)
		}
		if err := visit(sha, body); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("history: read the raw archive: %w", err)
	}
	return nil
}

// LeaguePlayers lists the player ids a league file's own identity rows hold.
func LeaguePlayers(ctx context.Context, league *db.Pools) ([]string, error) {
	return readStrings(ctx, league, `SELECT id FROM main.league_player ORDER BY id`)
}

// MissingReference returns the player ids and archive keys a reference file lacks, in the order given.
func MissingReference(ctx context.Context, reference *db.Pools, players, bodies []string) (missingPlayers, missingBodies []string, err error) {
	held := func(query string, keys []string) ([]string, error) {
		var missing []string
		for _, key := range keys {
			var found bool
			if err := reference.Read().QueryRowContext(ctx, query, key).Scan(&found); err != nil {
				return nil, fmt.Errorf("history: read the reference file: %w", err)
			}
			if !found {
				missing = append(missing, key)
			}
		}
		return missing, nil
	}
	if missingPlayers, err = held(`SELECT EXISTS(SELECT 1 FROM main.canonical_player WHERE id=?)`, players); err != nil {
		return nil, nil, err
	}
	if missingBodies, err = held(`SELECT EXISTS(SELECT 1 FROM main.raw_archive WHERE sha256=?)`, bodies); err != nil {
		return nil, nil, err
	}
	return missingPlayers, missingBodies, nil
}

func readStrings(ctx context.Context, pools *db.Pools, query string) ([]string, error) {
	rows, err := pools.Read().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("history: read: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("history: read: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history: read: %w", err)
	}
	return out, nil
}
