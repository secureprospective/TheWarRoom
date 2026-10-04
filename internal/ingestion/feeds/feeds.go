// Package feeds is the table-driven loader. It reads any file listed in the registry's feeds.csv
// and turns each row into facts through source_fields: which column holds the player id and of
// what type, which hold the period, which rows to keep, and which column feeds which measure.
// Adding a signal from such a file is registry rows, never code.
package feeds

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

// maxBytes caps one file. The largest nflverse file the registry lists is under 10 MB.
const maxBytes = 128 << 20

// Result is one file read: the batch to ingest, and the mapped columns the file did not have.
// A missing column leaves its measures empty for the file; the rest still loads.
type Result struct {
	Batch   measures.Batch
	Rows    int // rows kept by the feed's filter
	Missing []string

	numeric []string // the numeric measures the file has columns for
}

// Read fetches a feed's file for season (ignored for a single-file feed) and maps it. A
// week-grain number of zero is not emitted: the batch's scope says the file is the whole report
// for its season, so the store reads an absent week count as zero.
func Read(ctx context.Context, client *http.Client, reg *measures.Registry, f measures.Feed, season int) (Result, error) {
	body, err := get(ctx, client, f.URLFor(season))
	if err != nil {
		return Result{}, err
	}
	sum := sha256.Sum256(body)
	res, err := parse(reg, f, body)
	if err != nil {
		return Result{}, fmt.Errorf("feeds: %s %s: %w", f.Name, f.URLFor(season), err)
	}
	res.Batch.Source, res.Batch.BodySHA256 = f.Source, hex.EncodeToString(sum[:])
	if f.Grain() == measures.GrainWeek {
		res.Batch.Scope = &measures.Scope{Seasons: []int{season}, Measures: res.numeric}
	}
	return res, nil
}

func get(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("feeds: request %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("feeds: fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feeds: %s answered %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("feeds: read %s: %w", url, err)
	}
	if len(body) > maxBytes {
		return nil, fmt.Errorf("feeds: %s is over %d bytes", url, maxBytes)
	}
	return body, nil
}

// column is one mapped field and where it sits in the file.
type column struct {
	field   string
	index   int
	numeric bool
}

// layout is where a file keeps what the feed reads.
type layout struct {
	at   map[string]int
	cols []column
}

// newLayout finds the feed's columns in a header. The id, period and filter columns must be
// there; a missing field column is recorded on res and its measures stay empty.
func newLayout(reg *measures.Registry, f measures.Feed, header []string, res *Result) (layout, error) {
	l := layout{at: map[string]int{}}
	for i, h := range header {
		l.at[strings.TrimSpace(h)] = i
	}
	need := []string{f.IDColumn, f.SeasonColumn, f.WeekColumn}
	for _, c := range f.Filter {
		need = append(need, c.Column)
	}
	for _, c := range need {
		if _, ok := l.at[c]; c != "" && !ok {
			return layout{}, fmt.Errorf("no %q column", c)
		}
	}
	for _, sf := range reg.FeedFields(f) {
		i, ok := l.at[measures.FeedColumn(sf)]
		if !ok {
			res.Missing = append(res.Missing, measures.FeedColumn(sf))
			continue
		}
		m, _ := reg.Measure(sf.Measure)
		l.cols = append(l.cols, column{field: sf.Field, index: i, numeric: !m.IsText()})
		if !m.IsText() {
			res.numeric = append(res.numeric, m.Name)
		}
	}
	return l, nil
}

// parse maps every kept row of a CSV body. Rows without a player id are skipped. When a key
// repeats, the later row wins: the combine lists a player who tested twice under both years.
func parse(reg *measures.Registry, f measures.Feed, body []byte) (Result, error) {
	r := csv.NewReader(bytes.NewReader(body))
	r.ReuseRecord = true
	header, err := r.Read()
	if err != nil {
		return Result{}, fmt.Errorf("read header: %w", err)
	}
	var res Result
	l, err := newLayout(reg, f, header, &res)
	if err != nil {
		return Result{}, err
	}
	order := map[factKey]int{}
	for line := 2; ; line++ {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return res, nil
		}
		if err != nil {
			return Result{}, fmt.Errorf("line %d: %w", line, err)
		}
		if err := l.row(f, rec, &res, order); err != nil {
			return Result{}, fmt.Errorf("line %d: %w", line, err)
		}
	}
}

// row adds one record's facts to res. A week-grain zero is left out.
func (l layout) row(f measures.Feed, rec []string, res *Result, order map[factKey]int) error {
	cell := func(col string) string { return strings.TrimSpace(rec[l.at[col]]) }
	id := cell(f.IDColumn)
	if !f.Keep(cell) || ingestion.IsMissing(id) {
		return nil
	}
	res.Rows++
	season, wk, err := period(f, cell)
	if err != nil {
		return err
	}
	week := f.Grain() == measures.GrainWeek
	for _, c := range l.cols {
		raw := strings.TrimSpace(rec[c.index])
		if ingestion.IsMissing(raw) || (week && c.numeric && isZero(raw)) {
			continue
		}
		fact := measures.Fact{IDType: f.IDType, ID: strings.Clone(id), Season: season, Week: wk,
			Field: c.field, Raw: strings.Clone(raw)}
		k := factKey{fact.ID, season, wk, c.field}
		if i, dup := order[k]; dup {
			res.Batch.Facts[i] = fact
			continue
		}
		order[k] = len(res.Batch.Facts)
		res.Batch.Facts = append(res.Batch.Facts, fact)
	}
	return nil
}

type factKey struct {
	id           string
	season, week int
	field        string
}

// period reads a row's season and week; a single-period feed reports 0 for what it lacks.
func period(f measures.Feed, cell func(string) string) (season, week int, err error) {
	if f.SeasonColumn != "" {
		if season, err = strconv.Atoi(cell(f.SeasonColumn)); err != nil {
			return 0, 0, fmt.Errorf("season %q: %w", cell(f.SeasonColumn), err)
		}
	}
	if f.WeekColumn != "" {
		if week, err = strconv.Atoi(cell(f.WeekColumn)); err != nil {
			return 0, 0, fmt.Errorf("week %q: %w", cell(f.WeekColumn), err)
		}
	}
	return season, week, nil
}

func isZero(raw string) bool {
	v, err := strconv.ParseFloat(raw, 64)
	return err == nil && v == 0
}
