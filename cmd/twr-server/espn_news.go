package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/headshot"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/espnnews"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type espnNewsArchive struct {
	store *history.Store
}

func (a espnNewsArchive) LoadESPNBody(
	ctx context.Context, key string,
) (body []byte, fetchedAt time.Time, ok bool, err error) {
	found, err := a.store.ExactArchivedBodies(
		ctx, key, func(_ string, raw []byte, at time.Time) bool {
			body = raw
			fetchedAt = at
			return true
		},
	)
	if err != nil {
		return nil, time.Time{}, false, fmt.Errorf("load espn archive %s: %w", key, err)
	}
	return body, fetchedAt, found, nil
}

func (a espnNewsArchive) SaveESPNBody(
	ctx context.Context, key string, body []byte, fetchedAt time.Time,
) error {
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return fmt.Errorf("gzip espn archive %s: %w", key, err)
	}
	if _, err := gz.Write(body); err != nil {
		_ = gz.Close()
		return fmt.Errorf("write espn archive %s: %w", key, err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("close espn archive %s: %w", key, err)
	}
	sum := sha256.Sum256(body)
	rec := archive.Fetch{
		URL:       key,
		Status:    http.StatusOK,
		SHA256:    hex.EncodeToString(sum[:]),
		Size:      int64(len(body)),
		Gzip:      buf.Bytes(),
		FetchedAt: fetchedAt.UTC(),
	}
	if err := a.store.Record(ctx, rec); err != nil {
		return fmt.Errorf("record espn archive %s: %w", key, err)
	}
	return nil
}

func newHostedNewsService(referenceStore *history.Store) *leagueservice.NewsService {
	client := &http.Client{Timeout: 10 * time.Second}
	return leagueservice.NewNewsService(
		client, espnnews.DefaultEndpoints(), espnNewsArchive{store: referenceStore}, referenceStore,
	)
}

// hostedPhotoTimeout bounds one request to a photo host; a dead host never holds a card.
const hostedPhotoTimeout = 10 * time.Second

// newHostedPhotos is the desktop photo cache on the server: /headshot/<mfl id> is fetched once
// from ESPN (Sleeper as fallback) and kept next to the database, so the web app shows photos
// without loosening its img-src self policy. Photos are personal use only, never in history.db.
func newHostedPhotos(databasePath string, reference *history.Store) http.Handler {
	ids := func(ctx context.Context, mflID string) (string, string, error) {
		espn, err := reference.SourceID(ctx, mflID, "espn")
		if err != nil {
			return "", "", fmt.Errorf("server photo ids: %w", err)
		}
		sleeper, err := reference.SourceID(ctx, mflID, "sleeper")
		if err != nil {
			return "", "", fmt.Errorf("server photo ids: %w", err)
		}
		return espn, sleeper, nil
	}
	dir := filepath.Join(filepath.Dir(databasePath), headshot.CacheDir)
	photos := headshot.New(dir, ids, &http.Client{Timeout: hostedPhotoTimeout})
	return photos.Handler(func() bool { return true })
}
