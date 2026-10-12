package leaguearchive

import (
	"bytes"
	"cmp"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/archive"
)

// ErrorBody counts the files of one export and season that hold an MFL error instead of data.
type ErrorBody struct {
	Export string
	Season int
	Text   string
	Files  int
}

// Report is one import run.
type Report struct {
	Files    int // files in the archive
	Recorded int // fetches recorded by this run
	Held     int // files already recorded, with the same bytes, by an earlier run
	Errors   []ErrorBody
}

// HeldKey identifies an archived file already recorded: the request it stands for and its bytes.
func HeldKey(rawURL, sha256 string) string { return rawURL + "\x00" + sha256 }

// file is one archive file, read and checked, ready to record.
type file struct {
	rel    string
	export Export
	fetch  archive.Fetch
}

// Import records every file of the archive in fsys under source, skipping those held already.
// Every file is read and checked before any is recorded, so a malformed archive records nothing.
// A file's capture time is its modification time.
func Import(ctx context.Context, fsys fs.FS, source string, held map[string]bool, sink archive.Sink) (Report, error) {
	files, err := readArchive(fsys, source)
	if err != nil {
		return Report{}, err
	}
	rep := Report{Files: len(files)}
	errs := map[[3]string]int{}
	for _, f := range files {
		if f.fetch.Err != "" {
			errs[[3]string{f.export.Type, strconv.Itoa(f.export.Season), f.fetch.Err}]++
		}
		if held[HeldKey(f.fetch.URL, f.fetch.SHA256)] {
			rep.Held++
			continue
		}
		if err := sink.Record(ctx, f.fetch); err != nil {
			return rep, fmt.Errorf("leaguearchive: record %s: %w", f.rel, err)
		}
		rep.Recorded++
	}
	for k, n := range errs {
		season, _ := strconv.Atoi(k[1])
		rep.Errors = append(rep.Errors, ErrorBody{Export: k[0], Season: season, Text: k[2], Files: n})
	}
	slices.SortFunc(rep.Errors, func(a, b ErrorBody) int {
		return cmp.Or(strings.Compare(a.Export, b.Export), cmp.Compare(a.Season, b.Season), strings.Compare(a.Text, b.Text))
	})
	return rep, nil
}

// readArchive reads every file under fsys in path order, each season's league first.
func readArchive(fsys fs.FS, source string) ([]file, error) {
	leagues, err := readLeagues(fsys)
	if err != nil {
		return nil, err
	}
	seasons := slices.Sorted(maps.Keys(leagues))
	var files []file
	err = fs.WalkDir(fsys, ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("leaguearchive: read %s: %w", rel, err)
		}
		if d.IsDir() {
			return nil
		}
		f, err := readFile(fsys, rel, d, seasons[len(seasons)-1], leagues, source)
		if err != nil {
			return err
		}
		files = append(files, f)
		return nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // readFile's errors name the file
	}
	return files, nil
}

func readFile(fsys fs.FS, rel string, d fs.DirEntry, latest int, leagues map[int]League, source string) (file, error) {
	e, err := ParsePath(rel, latest)
	if err != nil {
		return file{}, err
	}
	lg, ok := leagues[e.Season]
	if !ok {
		return file{}, fmt.Errorf("leaguearchive: %s: season %d has no league.json", rel, e.Season)
	}
	body, err := fs.ReadFile(fsys, rel)
	if err != nil {
		return file{}, fmt.Errorf("leaguearchive: read %s: %w", rel, err)
	}
	mflErr, err := checkBody(body)
	if err != nil {
		return file{}, fmt.Errorf("leaguearchive: %s: %w", rel, err)
	}
	info, err := d.Info()
	if err != nil {
		return file{}, fmt.Errorf("leaguearchive: stat %s: %w", rel, err)
	}
	gz, err := compress(body)
	if err != nil {
		return file{}, fmt.Errorf("leaguearchive: compress %s: %w", rel, err)
	}
	sum := sha256.Sum256(body)
	return file{rel: rel, export: e, fetch: archive.Fetch{
		URL: e.URL(lg), Source: source, Status: http.StatusOK, SHA256: hex.EncodeToString(sum[:]),
		Size: int64(len(body)), Gzip: gz, Err: mflErr, FetchedAt: info.ModTime().UTC(),
	}}, nil
}

func compress(body []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(body); err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	return buf.Bytes(), nil
}

// rawExport is the part of any MFL export body the importer reads: its error envelope.
type rawExport struct {
	Error json.RawMessage `json:"error"`
}

// checkBody returns MFL's error text when body is an error envelope, and fails when body is not
// a JSON object.
func checkBody(body []byte) (string, error) {
	var raw rawExport
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", fmt.Errorf("not an MFL export: %w", err)
	}
	return raw.Validate(body)
}

func (raw rawExport) Validate(body []byte) (string, error) {
	if trimmed := bytes.TrimSpace(body); len(trimmed) == 0 || trimmed[0] != '{' {
		return "", errors.New("not an MFL export: expected a JSON object")
	}
	if len(raw.Error) == 0 {
		return "", nil
	}
	var text struct {
		T string `json:"$t"`
	}
	if json.Unmarshal(raw.Error, &text) != nil {
		if err := json.Unmarshal(raw.Error, &text.T); err != nil {
			return "", fmt.Errorf("unreadable MFL error: %s", raw.Error)
		}
	}
	return "MFL error: " + strings.TrimSpace(text.T), nil
}

// rawLeague is the part of a season's league.json that says which league it was.
type rawLeague struct {
	League struct {
		ID      string `json:"id"`
		BaseURL string `json:"baseURL"`
	} `json:"league"`
}

func (r rawLeague) Validate() (League, error) {
	if _, err := strconv.Atoi(r.League.ID); err != nil || r.League.ID == "" {
		return League{}, fmt.Errorf("league id %q is not a number", r.League.ID)
	}
	u, err := url.Parse(r.League.BaseURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return League{}, fmt.Errorf("baseURL %q is not an https host", r.League.BaseURL)
	}
	return League{ID: r.League.ID, Host: u.Hostname()}, nil
}

// readLeagues reads each season folder's league.json: historical seasons ran under other league
// ids and hosts.
func readLeagues(fsys fs.FS) (map[int]League, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("leaguearchive: read archive root: %w", err)
	}
	leagues := map[int]League{}
	for _, d := range entries {
		season, err := strconv.Atoi(d.Name())
		if !d.IsDir() || err != nil {
			continue
		}
		rel := d.Name() + "/league.json"
		body, err := fs.ReadFile(fsys, rel)
		if err != nil {
			return nil, fmt.Errorf("leaguearchive: read %s: %w", rel, err)
		}
		var raw rawLeague
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, fmt.Errorf("leaguearchive: %s: %w", rel, err)
		}
		if leagues[season], err = raw.Validate(); err != nil {
			return nil, fmt.Errorf("leaguearchive: %s: %w", rel, err)
		}
	}
	if len(leagues) == 0 {
		return nil, errors.New("leaguearchive: no season folders with a league.json")
	}
	return leagues, nil
}
