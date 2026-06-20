package ingestion

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// This file holds the shared external-HTTP-CSV plumbing the non-MFL Layer-1
// fetchers reuse (crosswalk, nflproduction, …). These sources are static CSV files
// (nflverse / dynastyprocess GitHub releases), NOT the MFL API — so they do not use
// mfl.Client (no host discovery, no MFL error envelope, no MFL rate limit). They
// still inherit the ingestion boundary discipline: fail loud, validate the shape,
// bind columns by NAME not position. Extracted from the crosswalk fetcher (the
// first instance) before a second fetcher could copy-paste it (Codex M17).

// NACell is how R-generated nflverse / dynastyprocess CSVs encode a missing value:
// the literal string "NA", not an empty field. IsMissing treats both as absent.
const NACell = "NA"

// DefaultMaxCSVBytes bounds an external CSV read so a runaway or hostile source
// cannot exhaust memory at the boundary (Codex M15). Callers pass their own ceiling
// to FetchCSV; this is a sane default for the few-MB files. A fetcher reading a
// large file (e.g. play-by-play) must raise it deliberately.
const DefaultMaxCSVBytes = 64 << 20

// IsMissing reports whether a CSV cell is absent — empty or the "NA" sentinel.
func IsMissing(s string) bool { return s == "" || s == NACell }

// FetchCSV GETs an external CSV over plain HTTP, validates the status, and returns
// all records (header included). It reads at most maxBytes and FAILS LOUD if the
// source exceeds that ceiling rather than silently truncating — a silently
// truncated file would drop its tail of players with no error, which is worse than
// a clean failure. ctx bounds the request (and, via the request context, the body
// read); the response body is always closed.
func FetchCSV(ctx context.Context, client *http.Client, url string, maxBytes int64) ([][]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ingestion: build CSV request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ingestion: fetch CSV %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ingestion: CSV %s unexpected status %d", url, resp.StatusCode)
	}

	// Read one byte past the cap: if the LimitedReader is fully consumed (N == 0)
	// the source was over budget, so we error instead of parsing a truncated file.
	lr := &io.LimitedReader{R: resp.Body, N: maxBytes + 1}
	records, err := csv.NewReader(lr).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ingestion: read CSV %s: %w", url, err)
	}
	if lr.N == 0 {
		return nil, fmt.Errorf("ingestion: CSV %s exceeds %d-byte cap", url, maxBytes)
	}
	return records, nil
}

// CSVColumns binds each named column to its index in the CSV header, stripping a
// leading UTF-8 BOM from the first cell (an invisible byte some exporters emit that
// would otherwise defeat a name match — Technical pillar Debugging Discipline). It
// returns an error naming the first required column that is absent. Binding by name
// then asserting presence is the boundary shape check: it tolerates new or
// reordered columns but fails loud the moment a depended-on column disappears.
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
