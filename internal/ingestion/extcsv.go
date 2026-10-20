package ingestion

import (
	"compress/gzip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Shared plumbing for external static files (nflverse, DynastyProcess). CSV columns bind by name,
// bodies are byte-capped, and failures are loud.

// NACell is how R-generated CSVs write a missing value.
const NACell = "NA"

// DefaultMaxCSVBytes caps a CSV read; a fetcher reading a larger file raises it deliberately.
const DefaultMaxCSVBytes = 64 << 20

// IsMissing reports an empty or "NA" cell.
func IsMissing(s string) bool { return s == "" || s == NACell }

// openCappedCSV GETs a CSV, checks the status and returns a reader capped at maxBytes, the
// LimitedReader for the over-cap check, and a cleanup func. When gz, the cap applies to the
// decompressed bytes, which also bounds a gzip bomb; a non-gzip body fails on its header.
func openCappedCSV(ctx context.Context, client *http.Client, url string, maxBytes int64, gz bool,
) (*csv.Reader, *io.LimitedReader, func(), error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ingestion: build CSV request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ingestion: fetch CSV %s: %w", url, err)
	}
	cleanup := func() { _ = resp.Body.Close() }

	if resp.StatusCode != http.StatusOK {
		cleanup()
		return nil, nil, nil, fmt.Errorf("ingestion: CSV %s unexpected status %d", url, resp.StatusCode)
	}

	var src io.Reader = resp.Body
	if gz {
		zr, gzErr := gzip.NewReader(resp.Body)
		if gzErr != nil {
			cleanup()
			return nil, nil, nil, fmt.Errorf("ingestion: gunzip CSV %s: %w", url, gzErr)
		}
		base := cleanup
		cleanup = func() { _ = zr.Close(); base() }
		src = zr
	}

	// Read one byte past the cap; consuming it means the file was over budget.
	lr := &io.LimitedReader{R: src, N: maxBytes + 1}
	return csv.NewReader(lr), lr, cleanup, nil
}

// FetchCSV returns every record of an external CSV, header included. It fails over maxBytes
// rather than silently dropping a tail of players.
func FetchCSV(ctx context.Context, client *http.Client, url string, maxBytes int64) ([][]string, error) {
	return fetchCSV(ctx, client, url, maxBytes, false)
}

// FetchCSVGz is FetchCSV for a gzipped source; maxBytes caps the decompressed bytes.
func FetchCSVGz(ctx context.Context, client *http.Client, url string, maxBytes int64) ([][]string, error) {
	return fetchCSV(ctx, client, url, maxBytes, true)
}

func fetchCSV(ctx context.Context, client *http.Client, url string, maxBytes int64, gz bool) ([][]string, error) {
	r, lr, cleanup, err := openCappedCSV(ctx, client, url, maxBytes, gz)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ingestion: read CSV %s: %w", url, err)
	}
	if lr.N == 0 {
		return nil, fmt.Errorf("ingestion: CSV %s exceeds %d-byte cap", url, maxBytes)
	}
	return records, nil
}

// CSVColumns maps each required column name to its header index, stripping a leading UTF-8
// BOM. New or reordered columns are fine; a missing required one is an error.
func CSVColumns(header []string, names ...string) (map[string]int, error) {
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}

	pos := make(map[string]int, len(names))
	for _, name := range names {
		idx := -1
		for i, h := range header {
			if strings.TrimSpace(h) == name {
				idx = i
				break
			}
		}
		if idx < 0 {
			return nil, fmt.Errorf("ingestion: CSV missing required column %q", name)
		}
		pos[name] = idx
	}
	return pos, nil
}

// Get fetches url and returns its body, failing on a status other than 200 and on a body over
// maxBytes rather than reading a truncated file.
func Get(ctx context.Context, client *http.Client, url string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ingestion: request %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ingestion: fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ingestion: %s answered %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("ingestion: read %s: %w", url, err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("ingestion: %s is over %d bytes", url, maxBytes)
	}
	return body, nil
}
