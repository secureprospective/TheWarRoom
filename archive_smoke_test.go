package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

// fullArchive compiles the facts from the full league archive, read only.
// Opt-in: TWR_ARCHIVE_PARITY=1 TWR_FACTS_ARCHIVE=<raw dir>.
func fullArchive(t *testing.T) archiveCache {
	t.Helper()
	root := os.Getenv("TWR_FACTS_ARCHIVE")
	if os.Getenv("TWR_ARCHIVE_PARITY") != "1" || root == "" {
		t.Skip("opt-in")
	}
	raw, err := os.ReadFile("internal/lab/testdata/facts/inputs.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Inputs []struct {
			File   string `json:"file"`
			SHA256 string `json:"sha256"`
		} `json:"inputs"`
	}
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatal(err)
	}
	archive, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = archive.Close() }()
	read := func(file string) facts.ResolvedInput {
		b, err := archive.ReadFile(filepath.FromSlash(file))
		if err != nil {
			t.Fatal(err)
		}
		return facts.ResolvedInput{File: file, Bytes: int64(len(b)), Body: b, SHA256: hexSum(b)}
	}
	inputs := []facts.ResolvedInput{}
	for _, in := range fixture.Inputs {
		inputs = append(inputs, read(in.File))
	}
	curated, err := os.ReadFile("internal/lab/testdata/note_readings.csv")
	if err != nil {
		t.Fatal(err)
	}
	notes, err := facts.ParseCurated(curated)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := facts.Compile(t.Context(), inputs, notes)
	if err != nil {
		t.Fatal(err)
	}
	pins, err := poolPins()
	if err != nil {
		t.Fatal(err)
	}
	poolInputs := []facts.ResolvedInput{}
	for _, p := range pins {
		poolInputs = append(poolInputs, read(p.File))
	}
	pools, err := facts.CompilePools(poolInputs)
	if err != nil {
		t.Fatal(err)
	}
	return archiveCache{sha: compiled.Fingerprint(), facts: compiled, pools: pools, cutoff: "full archive"}
}

// TestArchiveReportsOnTheFullArchive compiles the facts from the full league archive (read only)
// and writes every Archive report as text, for reading real lines against MFL's own pages.
// Opt-in: TWR_ARCHIVE_PARITY=1 TWR_FACTS_ARCHIVE=<raw dir> TWR_ARCHIVE_SMOKE=<out file>.
func TestArchiveReportsOnTheFullArchive(t *testing.T) {
	out := os.Getenv("TWR_ARCHIVE_SMOKE")
	if out == "" {
		t.Skip("opt-in")
	}
	cache := fullArchive(t)
	var text strings.Builder
	for _, choice := range []struct {
		season      int
		team, other string
	}{{0, "", ""}, {2015, "0001", "0002"}} {
		board, err := buildArchive(cache, false, choice.season, choice.team, choice.other)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range board.Board.Reports {
			text.WriteString("\n== " + r.ID + " " + r.Title + " [" + r.Status + "] season " + strconv.Itoa(board.Season) + " team " + choice.team + "\n")
			text.WriteString(r.Source + " | " + r.Note + "\n" + strings.Join(r.Columns, " | ") + "\n")
			for _, row := range r.Summary {
				text.WriteString("  S: " + strings.Join(row.Cells, " | ") + "\n")
			}
			for _, row := range r.Rows {
				text.WriteString("  " + strings.Join(row.Cells, " | ") + "\n")
			}
		}
	}
	if err := os.WriteFile(out, []byte(text.String()), 0o600); err != nil { //nolint:gosec // opt-in report path the operator names
		t.Fatal(err)
	}
}

func hexSum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// TestArchiveMatchesMFLStandingsPages is the fact exam for the archive: every franchise's regular
// record and points in M-105 must equal what MFL's own standings page showed a GM that season.
// The pages are saved copies (no fetch). Four franchise seasons differ by one game that no recorded
// game accounts for (the reference's checks.py leaves the same four open: possibly a commissioner
// adjustment); they are listed, so a fifth fails. Opt-in as fullArchive, plus TWR_MFL_PAGES=<dir> holding
// standings_<year>.html.
func TestArchiveMatchesMFLStandingsPages(t *testing.T) {
	dir := os.Getenv("TWR_MFL_PAGES")
	if dir == "" {
		t.Skip("opt-in")
	}
	cache := fullArchive(t)
	pages, err := filepath.Glob(filepath.Join(dir, "standings_*.html"))
	if err != nil || len(pages) == 0 {
		t.Fatal("no standings pages", err)
	}
	board, err := buildArchive(cache, false, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	ours := map[string][2]string{} // "year name" → regular W-L-T, points
	for _, team := range board.Teams {
		b, err := buildArchive(cache, false, 0, team.Value, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range b.Board.Reports {
			if r.ID == "M-105" {
				for _, row := range r.Rows {
					ours[row.Cells[0]+" "+row.Cells[1]] = [2]string{row.Cells[2], row.Cells[4]}
				}
			}
		}
	}
	open := map[string]bool{"2014 Houston Texans": true, "2014 New York Giants": true,
		"2021 Tennessee Titans": true, "2021 New York Giants": true}
	checked := 0
	for _, page := range pages {
		year := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(page), "standings_"), ".html")
		body, err := os.ReadFile(page) //nolint:gosec // opt-in page copies the operator names
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range mflStandingsPage(string(body)) {
			got, ok := ours[year+" "+row[0]]
			switch {
			case !ok:
				t.Errorf("%s %s: on MFL's page, not in the archive", year, row[0])
			case got[1] != row[2] || (got[0] != row[1] && !open[year+" "+row[0]]):
				t.Errorf("%s %s: MFL %s %s, archive %s %s", year, row[0], row[1], row[2], got[0], got[1])
			}
			checked++
		}
	}
	if checked < 32*len(pages) {
		t.Fatalf("checked %d franchise seasons on %d pages", checked, len(pages))
	}
	t.Logf("%d franchise seasons checked against MFL's standings pages", checked)
}

var (
	pageRow  = regexp.MustCompile(`(?s)<tr[^>]*>(.*?)</tr>`)
	pageCell = regexp.MustCompile(`(?s)<t[hd][^>]*>(.*?)</t[hd]>`)
	pageTag  = regexp.MustCompile(`<[^>]+>`)
	pageAlt  = regexp.MustCompile(`alt="([^"]+)"`)
)

// mflStandingsPage reads MFL's standings page into franchise, W-L-T and PF, locating the columns by
// their headers (the page's layout changed over the years).
func mflStandingsPage(page string) [][3]string {
	out := [][3]string{}
	name, record, points := -1, -1, -1
	for _, tr := range pageRow.FindAllStringSubmatch(page, -1) {
		cells := []string{}
		for _, td := range pageCell.FindAllStringSubmatch(tr[1], -1) {
			text := html.UnescapeString(pageTag.ReplaceAllString(td[1], " "))
			if alt := pageAlt.FindStringSubmatch(td[1]); strings.TrimSpace(text) == "" && alt != nil {
				text = html.UnescapeString(alt[1]) // later pages name the franchise only in its logo
			}
			cells = append(cells, strings.Join(strings.Fields(strings.ReplaceAll(text, "\u2011", "-")), " "))
		}
		if i := slices.Index(cells, "Franchise"); i >= 0 {
			name, record, points = i, slices.Index(cells, "W-L-T"), slices.Index(cells, "PF")
			continue
		}
		if name < 0 || record < 0 || points < 0 || len(cells) <= max(name, record, points) {
			continue
		}
		if wlt := cells[record]; strings.Count(wlt, "-") == 2 && cells[name] != "" {
			out = append(out, [3]string{cells[name], wlt, cells[points]})
		}
	}
	return out
}
