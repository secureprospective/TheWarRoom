package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// The hosted server keeps football reference data in reference.db beside the league file. A one-file
// server moves it there on first start, through the same steps as the sign-in move (fileMove).
const referenceFileName = "reference.db"

func referencePath(leaguePath string) string {
	return filepath.Join(filepath.Dir(leaguePath), referenceFileName)
}

//nolint:gochecknoglobals // fixed by the code, read only
var referenceMove = fileMove{what: "football reference data", file: referenceFileName, held: "raw_archive",
	record: history.ReferenceMoveTable, build: buildReferenceMove, after: history.CompactLeague}

// openReference settles the reference move for the league file at leaguePath (opened on its own) and
// opens the reference file at this build's schema.
func openReference(ctx context.Context, league *db.Pools, leaguePath string) (*db.Pools, error) {
	if err := history.PrepareNewLeague(ctx, league); err != nil {
		return nil, err //nolint:wrapcheck // names the file
	}
	if err := referenceMove.settle(ctx, league, leaguePath); err != nil {
		return nil, err
	}
	pools, err := db.Open(ctx, referencePath(leaguePath), history.ReferenceSchemaVersion())
	if err != nil {
		return nil, fmt.Errorf("open the reference file: %w", err)
	}
	if err := history.PrepareReferenceFile(ctx, pools); err != nil {
		return nil, errors.Join(err, pools.Close())
	}
	return pools, nil
}

func buildReferenceMove(ctx context.Context, league *db.Pools, pending string, held bool) (func(context.Context) error, error) {
	record, err := buildReference(ctx, league, pending, held)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) error {
		return history.CommitReferenceMove(ctx, league, record)
	}, nil
}

// buildReference writes and proves the new file, then closes it so nothing of it is left in a WAL.
func buildReference(ctx context.Context, source *db.Pools, path string, held bool) (record history.ReferenceMove, err error) {
	record.File = referenceFileName
	pools, err := db.Open(ctx, path, history.ReferenceSchemaVersion())
	if err != nil {
		return record, fmt.Errorf("create the reference file: %w", err)
	}
	defer func() { err = errors.Join(err, pools.Close()) }()
	if err := history.PrepareReferenceFile(ctx, pools); err != nil {
		return record, err //nolint:wrapcheck // names the file
	}
	if held {
		if err := history.CopyReference(ctx, source, pools); err != nil {
			return record, fmt.Errorf("copy reference data: %w", err)
		}
		conversionStep("copied", path)
		want, err := history.ProveReference(ctx, source)
		if err != nil {
			return record, fmt.Errorf("prove the copy: %w", err)
		}
		got, err := history.ProveReference(ctx, pools)
		if err != nil {
			return record, fmt.Errorf("prove the copy: %w", err)
		}
		if err := history.SameReference(want, got); err != nil {
			return record, fmt.Errorf("prove the copy: %w", err)
		}
		record.Bodies, record.Archive = got.Archive.Rows, got.Archive.Digest
	}
	if _, err := pools.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return record, fmt.Errorf("finish the reference file: %w", err)
	}
	return record, nil
}

// captureReference captures one reference set: each raw capture goes to put (uploaded once as its own
// object), then is emptied in the staged copy, and the set's checkpoint names the archive by content.
func captureReference(ctx context.Context, put func(context.Context, string, []byte) error, pools *db.Pools, configDir, stage, recipient string) (string, time.Time, error) {
	var anchor history.ArchiveAnchor
	prepare := func(ctx context.Context, staged string) (err error) {
		anchor, err = history.StripArchiveBodies(ctx, staged, func(sha string, body []byte) error { return put(ctx, sha, body) })
		return err //nolint:wrapcheck // wrapped by the capture
	}
	checkpoint := func(context.Context, string) ([]byte, error) { return referenceSetCheckpoint(anchor) }
	return offsite.CapturePrepared(ctx, offsite.ReferenceSet, pools, configDir, stage, recipient, prepare, checkpoint) //nolint:wrapcheck // names the step
}
