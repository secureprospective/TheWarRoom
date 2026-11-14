package main

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

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
