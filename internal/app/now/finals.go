package now

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// HeldFinals keeps a final game's ESPN bodies in the history archive under the URLs they were
// fetched from, so a final is read once and survives a restart without the network.
type HeldFinals struct {
	Store     *history.Store
	Endpoints livegame.Endpoints
}

var finalEventID = regexp.MustCompile(`(?:event=|/events/)([0-9]{1,20})(?:$|[&/])`)

func (h HeldFinals) address(kind, id string) (string, error) {
	pattern := map[string]string{"summary": h.Endpoints.Summary, "plays": h.Endpoints.Plays}[kind]
	if pattern == "" {
		return "", fmt.Errorf("final details: unknown kind %q", kind)
	}
	return strings.ReplaceAll(pattern, "{id}", id), nil
}

func (h HeldFinals) Keep(ctx context.Context, kind, id string, body []byte, at time.Time) error {
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
	if err := h.Store.Record(ctx, archive.Fetch{URL: address, Status: 200, SHA256: hex.EncodeToString(sum[:]),
		Size: int64(len(body)), Gzip: gz.Bytes(), FetchedAt: at}); err != nil {
		return fmt.Errorf("final details: keep: %w", err)
	}
	return nil
}

func (h HeldFinals) Bodies(ctx context.Context, kind, id string, each func(string, []byte, time.Time) bool) error {
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
	_, err = h.Store.ArchivedBodies(ctx, part, func(src string, body []byte, at time.Time) bool {
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

//go:embed livegame_sources.json
var liveGameSources []byte

// LiveEndpoints is the checked-in ESPN endpoint set the live-game poller reads.
func LiveEndpoints() (livegame.Endpoints, error) {
	var endpoints livegame.Endpoints
	if err := json.Unmarshal(liveGameSources, &endpoints); err != nil {
		return livegame.Endpoints{}, fmt.Errorf("live game sources: %w", err)
	}
	if err := endpoints.Validate(); err != nil {
		return livegame.Endpoints{}, fmt.Errorf("live game sources: %w", err)
	}
	return endpoints, nil
}
