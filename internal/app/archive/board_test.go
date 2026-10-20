package archive

import (
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/archive/archivetest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

func reduced(t *testing.T) Snapshot {
	t.Helper()
	f := archivetest.Facts(t)
	return Snapshot{SHA: f.Fingerprint(), Facts: f, Cutoff: archivetest.Cutoff}
}

func archiveReport(t *testing.T, b Board, id string) report.Report {
	t.Helper()
	for _, r := range b.Board.Reports {
		if r.ID == id {
			return r
		}
	}
	t.Fatal("missing report", id)
	return report.Report{}
}

func TestArchiveBeforeImportSaysHowToImport(t *testing.T) {
	t.Parallel()
	b, err := Build(Snapshot{}, false, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range b.Board.Reports {
		if r.Status != "failed" || !strings.Contains(r.Note, "Import the league archive in Control Room › Data and sources") || len(r.Rows) != 0 {
			t.Fatal(r.ID, r.Status, r.Note)
		}
		if strings.Contains(strings.ToLower(r.Note), "refresh from mfl") {
			t.Fatal("before import, a report says to refresh from MFL", r.ID)
		}
	}
}

func TestArchiveReportsOnTheReducedArchive(t *testing.T) {
	t.Parallel()
	cache := reduced(t)
	b, err := Build(cache, false, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	history := archiveReport(t, b, "M-080")
	if history.Status != "ok" || len(history.Rows) == 0 {
		t.Fatal(history)
	}
	last := history.Rows[len(history.Rows)-1].Cells
	if last[1] != "Season in progress" {
		t.Fatal("the newest season is not called in progress", last)
	}
	rules := archiveReport(t, b, "M-182")
	for _, row := range rules.Rows {
		if strings.ContainsAny(strings.Join(row.Cells, " "), "{\"") {
			t.Fatal("raw MFL setting text in the rules report", row.Cells)
		}
	}
	if champions := archiveReport(t, b, "M-114"); champions.Status == "failed" || champions.Title != "League champions" {
		t.Fatal(champions)
	}
	for _, id := range []string{"M-104", "M-105", "M-023", "M-110", "M-019"} {
		if r := archiveReport(t, b, id); r.Status != "empty" || !strings.Contains(r.Note, "Choose") {
			t.Fatal(id, "without a franchise", r.Status, r.Note)
		}
	}
	team := b.Teams[0].Value
	other := b.Teams[1].Value
	chosen, err := Build(cache, false, 0, team, other)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, id := range []string{"M-104", "M-105", "M-023", "M-110"} {
		r := archiveReport(t, chosen, id)
		shape := strings.Join(r.Columns, "|")
		if prior, ok := seen[shape]; ok {
			t.Fatal(id, "repeats", prior)
		}
		seen[shape] = id
	}
	for _, row := range archiveReport(t, chosen, "M-084").Rows {
		if strings.Contains(row.Cells[0], "wins") && strings.Contains(row.Cells[3], ".") {
			t.Fatal("a win count written with decimals", row.Cells)
		}
	}
	series := archiveReport(t, chosen, "M-019")
	for _, row := range series.Rows {
		if !strings.Contains(row.Cells[4], "–") {
			t.Fatal("a meeting without both scores", row.Cells)
		}
	}
}

func TestArchiveLineupInWords(t *testing.T) {
	t.Parallel()
	raw := `{"count":"21","iop_starters":"8","position":[{"name":"QB","limit":"1"},{"limit":"1-3","name":"RB"}]}`
	if got := archiveLineup(raw); got != "QB 1 · RB 1–3 · 21 starters · 8 offensive" {
		t.Fatal(got)
	}
	if archiveLineup("") != "Not recorded" || archiveLineup("{") != "Not recorded" || archiveUnquote(`"8"`) != "8" {
		t.Fatal("unreadable settings")
	}
}

func TestTrophyTitleNamesATrophyOnce(t *testing.T) {
	t.Parallel()
	for _, c := range [][3]string{
		{"NFC", "NFC Conference Champ", "NFC Conference Champ"},
		{"2015 Legacy Superbowl III", "2015 Legacy Champions", "2015 Legacy Superbowl III · 2015 Legacy Champions"},
		{"Division Champions", "", "Division Champions"},
	} {
		if got := trophyTitle(c[0], c[1]); got != c[2] {
			t.Errorf("trophyTitle(%q, %q) = %q, want %q", c[0], c[1], got, c[2])
		}
	}
}

func TestArchiveRetainsEveryNonPoolReport(t *testing.T) {
	t.Parallel()
	b, err := Build(Snapshot{}, false, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if b.Board.Title != "Archive" {
		t.Fatal(b.Board.Title)
	}
	ids := []string{}
	for _, r := range b.Board.Reports {
		ids = append(ids, r.ID)
	}
	want := []string{"M-080", "M-114", "M-182", "M-104", "M-105", "M-023", "M-110", "M-019",
		"M-083", "M-084", "M-085", "M-086", "M-082", "M-073", "M-079", "M-081", "M-089"}
	if !slices.Equal(ids, want) {
		t.Fatalf("report ids = %v, want %v", ids, want)
	}
}

func TestChampionsLatestUnplayedAndOldMissing(t *testing.T) {
	t.Parallel()
	cache := reduced(t)
	cache.Facts.Season.Year = []int64{2013, 2026}
	r := (archiveView{cache: cache}).champions()
	if len(r.Rows) != 2 || r.Rows[0].Cells[1] != "not in MFL's archive" || r.Rows[0].Cells[2] != "No scored championship final held" || !strings.Contains(r.Rows[1].Cells[2], "final has not been played") {
		t.Fatal(r.Rows)
	}
}

func TestPulseArchiveNamesTotalsAndFirstChampion(t *testing.T) {
	t.Parallel()
	cache := reduced(t)
	board, err := Build(cache, false, 2026, "", "")
	if err != nil {
		t.Fatal(err)
	}
	odd := archiveReport(t, board, "M-089")
	if strings.Contains(strings.ToLower(odd.Note+odd.How), "ring") || len(odd.Rows) == 0 {
		t.Fatal(odd.Note)
	}
	if !strings.Contains(odd.Rows[0].Cells[1], " · ") || !strings.Contains(odd.Rows[0].Cells[2], ": ") {
		t.Fatal("oddities lack count or franchise grouping", odd.Rows[0])
	}
	scoring := archiveReport(t, board, "M-073")
	for _, row := range scoring.Summary {
		if !strings.Contains(row.Cells[0], "all scored weeks") || (row.Cells[1] != "0.00" && !strings.Contains(row.Cells[1], ",")) {
			t.Fatal("unlabelled or unformatted era total", row.Cells)
		}
	}
	points := 51484.67
	if archivePoints(&points) != "51,484.67" || archiveRecordValue(archiveRecordEntry{value: points}) != "51,484.67" {
		t.Fatal("archive points formatter")
	}
	champ := archiveReport(t, board, "M-114")
	if len(champ.Rows) == 0 || champ.Rows[0].Cells[0] != "2013" || champ.Rows[0].Cells[1] != "not in MFL's archive" {
		t.Fatal("2013 silently omitted", champ.Rows)
	}
	if trophyTitle("2015 Legacy Superbowl III", "2015 Legacy Champions") !=
		"2015 Legacy Superbowl III · 2015 Legacy Champions" {
		t.Fatal("league's written title changed")
	}
}
