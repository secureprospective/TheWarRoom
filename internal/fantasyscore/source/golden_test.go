package source_test

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func heldWeeks(t *testing.T) []source.PlayerWeek {
	t.Helper()
	f, err := os.Open("testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	weeks, err := source.ReadWeeks(f)
	if err != nil {
		t.Fatal(err)
	}
	p, err := os.Open("testdata/plays.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := p.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	plays, err := source.ReadPlays(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.AttachPlays(weeks, plays); err != nil {
		t.Fatal(err)
	}
	return weeks
}

func TestHeldLegacyWeeks(t *testing.T) {
	weeks := heldWeeks(t)
	index := make(map[string]source.PlayerWeek)
	for _, w := range weeks {
		index[w.GSIS+"/"+strconv.Itoa(w.Week)] = w
	}
	body, err := os.ReadFile("testdata/comparison.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	matched := 0
	counts := make(map[string]int)
	positions := make(map[string]bool)
	for _, row := range rows[1:] {
		counts[row[0]]++
		positions[row[3]] = true
		w, found := index[row[2]+"/"+row[0]]
		if !found {
			t.Fatalf("missing rostered player week %v", row)
		}
		points, _, err := fantasyscore.Score(leaguerules.LegacyNFL().Scoring, leaguerules.Position(row[3]), w.Line)
		if err != nil || points.State != leaguerules.IsKnown {
			t.Fatalf("%v: points %+v error %v", row, points, err)
		}
		native := integer(t, row[6])
		parts := strings.Split(row[5], ".")
		if len(parts) != 2 || len(parts[1]) != 2 {
			t.Fatalf("held MFL total is not exact hundredths: %v", row)
		}
		want := integer(t, strings.ReplaceAll(row[5], ".", "")) * 100
		if want != integer(t, row[7]) {
			t.Fatalf("MFL oracle was altered: %v", row)
		}
		if points.Value != native {
			t.Errorf("week %s %s (%s): native %d, independent reference %d; MFL %d", row[0], row[4],
				row[1], points.Value, native, want)
		}
		if points.Value != want {
			if row[8] == "" {
				t.Errorf("unclassified MFL mismatch: %v", row)
			}
		} else {
			matched++
			if row[8] != "" {
				t.Errorf("stale mismatch classification: %v", row)
			}
		}
	}
	if counts["1"] != 879 || counts["2"] != 904 || len(positions) != 10 {
		t.Fatalf("two full offense/IDP weeks lost: weeks %v positions %v", counts, positions)
	}
	if len(rows)-1 != 1783 || matched != 1237 {
		t.Fatalf("coverage or match rate changed: %d/%d, want 1237/1783", matched, len(rows)-1)
	}
	t.Logf("MATCHED %d/%d; all 546 mismatches listed in testdata/comparison.csv", matched, len(rows)-1)
}

func integer(t *testing.T, raw string) leaguerules.Decimal {
	t.Helper()
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return leaguerules.Decimal(n)
}
