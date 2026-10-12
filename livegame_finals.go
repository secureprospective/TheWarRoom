package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// heldFinals keeps a final game's ESPN bodies in the history archive under the URLs they were
// fetched from, so a final is read once and survives a restart without the network.
type heldFinals struct {
	store     *history.Store
	endpoints livegame.Endpoints
}

var finalEventID = regexp.MustCompile(`(?:event=|/events/)([0-9]{1,20})(?:$|[&/])`)

func (h heldFinals) address(kind, id string) (string, error) {
	pattern := map[string]string{"summary": h.endpoints.Summary, "plays": h.endpoints.Plays}[kind]
	if pattern == "" {
		return "", fmt.Errorf("final details: unknown kind %q", kind)
	}
	return strings.ReplaceAll(pattern, "{id}", id), nil
}

func (h heldFinals) Keep(ctx context.Context, kind, id string, body []byte, at time.Time) error {
	address, err := h.address(kind, id)
	if err != nil {
		return err
	}
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("final details: compress: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("final details: compress: %w", err)
	}
	sum := sha256.Sum256(body)
	if err := h.store.Record(ctx, archive.Fetch{URL: address, Status: 200, SHA256: hex.EncodeToString(sum[:]),
		Size: int64(len(body)), Gzip: gz.Bytes(), FetchedAt: at}); err != nil {
		return fmt.Errorf("final details: keep: %w", err)
	}
	return nil
}

func (h heldFinals) Bodies(ctx context.Context, kind, id string, each func(string, []byte, time.Time) bool) error {
	address, err := h.address(kind, "0")
	if err != nil {
		return err
	}
	u, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("final details: %w", err)
	}
	part := u.Host + strings.TrimSuffix(u.Path, "/0/competitions/0/plays")
	if id != "" {
		full, err := h.address(kind, id)
		if err != nil {
			return err
		}
		part = strings.TrimPrefix(full, u.Scheme+"://")
	}
	_, err = h.store.ArchivedBodies(ctx, part, func(src string, body []byte, at time.Time) bool {
		m := finalEventID.FindStringSubmatch(src)
		if m == nil || (id != "" && m[1] != id) {
			return false
		}
		return !each(m[1], body, at)
	})
	if err != nil {
		return fmt.Errorf("final details: %w", err)
	}
	return nil
}
