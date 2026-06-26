package ingestion

import (
	"compress/gzip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

// openCappedCSV GETs an external CSV, validates the status, and returns a csv.Reader
// over the CSV bytes capped at maxBytes (+1 sentinel byte for the over-cap check),
// the LimitedReader backing that check, and a cleanup func that closes the body (and
// the gzip reader when gz). It is the shared HTTP+cap plumbing under FetchCSV/StreamCSV
// and their *Gz siblings, so the request/status/cap discipline is defined once, not
// copy-pasted across four entry points (Codex M17).
//
// When gz, the body is gunzipped BEFORE the cap, so maxBytes bounds the DECOMPRESSED
// CSV bytes — the same quantity the plain path caps and the quantity that actually
// lands in memory (this also bounds a gzip bomb's expansion; the compressed download
// is bounded by ctx). gzip.NewReader reads the gzip header eagerly, so a body that is
// not gzip FAILS LOUD here ("gunzip ... invalid header") rather than mis-parsing.
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

	// Read one byte past the cap: if the LimitedReader is fully consumed (N == 0)
	// the source was over budget, so callers error instead of using a truncated file.
	lr := &io.LimitedReader{R: src, N: maxBytes + 1}
	return csv.NewReader(lr), lr, cleanup, nil
}

// FetchCSV GETs an external CSV over plain HTTP, validates the status, and returns
// all records (header included). It reads at most maxBytes and FAILS LOUD if the
// source exceeds that ceiling rather than silently truncating — a silently
// truncated file would drop its tail of players with no error, which is worse than
// a clean failure. ctx bounds the request (and, via the request context, the body
// read); the response body is always closed.
func FetchCSV(ctx context.Context, client *http.Client, url string, maxBytes int64) ([][]string, error) {
	return fetchCSV(ctx, client, url, maxBytes, false)
}

// FetchCSVGz is FetchCSV for a gzip-compressed source (e.g. nflverse next_gen_stats,
// served as ngs_*.csv.gz). maxBytes bounds the DECOMPRESSED bytes; a non-gzip body
// fails loud. See openCappedCSV for the cap and gzip semantics.
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

// StreamCSV GETs an external CSV over plain HTTP and invokes fn once per DATA row
// (header excluded), reading the body row-by-row instead of buffering it whole. It is
// the streaming sibling of FetchCSV for sources too large to hold in memory at once:
// play-by-play is hundreds of MB, so FetchCSV's csv.ReadAll would buffer the entire
// body and risk OOM on a small host (CT105 is 2 GB). Peak memory here is one row,
// regardless of file size — the caller accumulates only the aggregates it needs and
// the row is discarded each iteration.
//
// It keeps every FetchCSV guarantee: status-checked, byte-capped via io.LimitedReader
// (fails loud over the cap rather than silently truncating a tail of rows), and the
// header bound BY NAME (required columns asserted present, BOM stripped). The bound
// column map is built once from the header and passed into every fn call; returning an
// error from fn aborts the stream with that error (the per-row fail-loud hook). The
// response body is always closed.
//
// The record slice passed to fn is REUSED between calls (csv ReuseRecord) to avoid a
// per-row allocation across tens of thousands of rows: fn may read/copy cell values
// (each cell string is freshly allocated) but MUST NOT retain the rec slice itself.
func StreamCSV(ctx context.Context, client *http.Client, url string, maxBytes int64,
	required []string, fn func(cols map[string]int, rec []string) error) error {
	return streamCSV(ctx, client, url, maxBytes, false, required, fn)
}

// StreamCSVGz is StreamCSV for a gzip-compressed source. maxBytes bounds the
// DECOMPRESSED bytes; a non-gzip body fails loud. This is the variant for a large
// gzipped file (e.g. a gzipped play-by-play or next_gen_stats export) where buffering
// the whole decompressed body would risk OOM. See openCappedCSV for cap/gzip semantics.
func StreamCSVGz(ctx context.Context, client *http.Client, url string, maxBytes int64,
	required []string, fn func(cols map[string]int, rec []string) error) error {
	return streamCSV(ctx, client, url, maxBytes, true, required, fn)
}

func streamCSV(ctx context.Context, client *http.Client, url string, maxBytes int64, gz bool,
	required []string, fn func(cols map[string]int, rec []string) error) error {
	r, lr, cleanup, err := openCappedCSV(ctx, client, url, maxBytes, gz)
	if err != nil {
		return err
	}
	defer cleanup()
	r.ReuseRecord = true

	header, err := r.Read()
	if err != nil {
		return fmt.Errorf("ingestion: read CSV %s header: %w", url, err)
	}
	cols, err := CSVColumns(header, required...)
	if err != nil {
		return fmt.Errorf("ingestion: %w", err)
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ingestion: read CSV %s: %w", url, err)
		}
		if err := fn(cols, rec); err != nil {
			return err
		}
	}

	if lr.N == 0 {
		return fmt.Errorf("ingestion: CSV %s exceeds %d-byte cap", url, maxBytes)
	}
	return nil
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

// IntCell parses a counting-stat cell at rec[idx]: a missing/NA cell is a legitimate
// 0 (the entity recorded none), while a present-but-unparseable cell fails loud — a
// value that is present yet won't parse is corruption, not zero, and silently
// zeroing it would hide the corruption. label names the column in the error so a
// failure points at the offending source field. Shared by the numeric fetchers
// (nflproduction, touchshare, …) so the missing-is-zero / malformed-fails-loud rule
// is defined once, not copy-pasted (Codex M17).
func IntCell(rec []string, idx int, label string) (int, error) {
	v := strings.TrimSpace(rec[idx])
	if IsMissing(v) {
		return 0, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("ingestion: column %q value %q: %w", label, v, err)
	}
	return n, nil
}

// FloatCell parses a numeric cell, applying the same missing-is-zero /
// malformed-fails-loud rule as IntCell for fractional values (yardage, share/pct).
func FloatCell(rec []string, idx int, label string) (float64, error) {
	v := strings.TrimSpace(rec[idx])
	if IsMissing(v) {
		return 0, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("ingestion: column %q value %q: %w", label, v, err)
	}
	return f, nil
}
