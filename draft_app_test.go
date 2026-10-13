package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func TestDraftReadingStatesAndParity(t *testing.T) {
	a := NewApp()
	raw := RawDraftScope{From: 2017, To: 2026, Seasons: 3, Group: "DL"}
	got := a.TargetDraftLab(raw)
	d, err := a.lab()
	if err != nil {
		t.Fatal(err)
	}
	picks := lab.DraftScope(d, draftScope(raw))
	p := lab.PositionDraft(picks, "DL")
	if got.Status != ReadingOK || got.Count <= 1000 || got.Threshold != 40 || got.InputSHA != lab.DataSHA || got.HowCounted == "" || got.Cutoff == "" {
		t.Fatalf("incomplete reading: %+v", got)
	}
	if len(got.Value.Sections) != 8 || !reflect.DeepEqual(got.Value.Sections, draftSections(d, draftScope(raw), picks, "DL")) {
		t.Fatal("surface diverged from engine")
	}
	for i, section := range got.Value.Sections {
		if section.ID != []string{"L-15", "L-16", "L-17", "L-18", "L-19", "L-20", "L-21", "L-22"}[i] {
			t.Fatal("registry surface missing")
		}
	}
	if got.Value.Sections[1].Rows[0].Range.Value != *lab.PositionDraft(picks, "").When[0].Ratio {
		t.Fatal("season range not computed by Go")
	}
	foundLike := false
	for _, row := range got.Value.Sections[4].Rows {
		foundLike = foundLike || strings.Contains(row.Detail, "plays like pick #")
	}
	if !foundLike {
		t.Fatal("plays-like missing")
	}
	if !strings.Contains(got.Value.Sections[5].Rows[0].Detail, percent(p.Hits.Rate)) || !strings.Contains(got.Value.Sections[6].Rows[0].Detail, percent(*p.Kept)) {
		t.Fatal("hit or kept estimate missing")
	}
	raw.From = 2100
	raw.To = 2101
	thin := a.TargetDraftLab(raw)
	if thin.Status != ReadingThin || thin.Count != 0 || thin.AbsenceReason == nil || len(thin.Value.Sections) != 0 {
		t.Fatal("empty scope fabricated")
	}
	a = &App{labData: func() (*lab.Data, error) { return nil, errors.New("fingerprint mismatch") }}
	failed := a.TargetDraftLab(raw)
	if failed.Status != ReadingFailed || failed.AbsenceReason == nil || !strings.Contains(*failed.AbsenceReason, "fingerprint mismatch") {
		t.Fatal("failed load hidden")
	}
}
func TestDraftBoundaryValidationAndFilters(t *testing.T) {
	valid := RawDraftScope{From: 2017, To: 2026, Seasons: 3, Group: "DB"}
	bads := []RawDraftScope{valid, valid, valid, valid, valid, valid, valid}
	bads[0].Seasons = 5
	bads[1].From = 2027
	bads[2].Group = "missing"
	bads[3].Segments = []string{"r1a", "r1a"}
	bads[4].NFL = []string{"n0"}
	bads[5].NFL = []string{"nx", "nx"}
	bads[6].Segments = []string{"r0"}
	for _, bad := range bads {
		if NewApp().TargetDraftLab(bad).Status != ReadingFailed {
			t.Fatal("invalid draft IPC accepted")
		}
	}
	a := NewApp()
	all := a.TargetDraftLab(valid)
	valid.Segments = []string{}
	valid.NFL = []string{}
	if !reflect.DeepEqual(a.TargetDraftLab(valid), all) {
		t.Fatal("empty wire arrays narrowed scope")
	}
	valid.Segments = []string{"r3"}
	narrow := a.TargetDraftLab(valid)
	if narrow.Status != ReadingOK || narrow.Count >= all.Count {
		t.Fatal("stretch filter ignored")
	}
	if !reflect.DeepEqual(narrow.Value.Sections[7], all.Value.Sections[7]) {
		// The market ranges stay the same; only the most common judged round can change its note.
		for i, row := range narrow.Value.Sections[7].Rows {
			if !reflect.DeepEqual(row.Range, all.Value.Sections[7].Rows[i].Range) {
				t.Fatal("draft filter changed trade-market yardstick")
			}
		}
	}
	valid.From = 1900
	valid.To = 2016
	if a.TargetDraftLab(valid).Status != ReadingThin {
		t.Fatal("pre-2017 facts shown as draft lab")
	}
}
func TestDraftUnavailableCellsAndKnownRetention(t *testing.T) {
	p := lab.DraftPosition{Group: "DB", All: []lab.JudgedPick{{DraftPick: lab.DraftPick{KeptYear2: nil}}, {DraftPick: lab.DraftPick{KeptYear2: boolPtr(true)}}}, Kept: floatPtr(1)}
	if draftKeptSection(p).Rows[0].Count != 1 {
		t.Fatal("unknown retention counted")
	}
	c := draftCellLine(lab.DraftCell{N: 7, Segment: "r3"})
	if c.Range != nil || !strings.Contains(c.Detail, "not a zero") {
		t.Fatal("thin cell fabricated")
	}
	if !strings.Contains(draftHitsSection(p).Rows[0].Detail, "Fewer than 8") {
		t.Fatal("thin hit rate fabricated")
	}
}
func boolPtr(v bool) *bool        { return &v }
func floatPtr(v float64) *float64 { return &v }

func TestDraftSummaryAndClosedBreakdownData(t *testing.T) {
	a := &App{}
	raw := RawDraftScope{From: 2017, To: 2026, Seasons: 3, Group: "DL"}
	r := a.TargetDraftLab(raw)
	d, err := a.lab()
	if err != nil {
		t.Fatal(err)
	}
	picks := lab.DraftScope(d, draftScope(raw))
	all := lab.PositionDraft(picks, "")
	if all.N != r.Count || all.Overall == nil {
		t.Fatal("all-position summary must count every judged pick")
	}
	for i, cells := range [][]lab.DraftCell{all.Where, all.When} {
		section := r.Value.Sections[i]
		if !strings.Contains(section.Title, "all positions") || len(section.Breakdowns) != 8 ||
			!reflect.DeepEqual(section.Rows, draftAxis("", "", cells).Rows) {
			t.Fatal("all-position summary missing")
		}
		for j, group := range marketAssets()[:8] {
			p := lab.PositionDraft(picks, group)
			b := section.Breakdowns[j]
			expected := p.Where
			if i == 1 {
				expected = p.When
			}
			if b.Title != marketName(group) || b.Count != p.N ||
				!reflect.DeepEqual(b.Rows, draftAxis("", "", expected).Rows) {
				t.Fatal("position counts or rows drifted")
			}
		}
	}
	p := lab.PositionDraft(picks, "DL")
	for i, cells := range [][]lab.DraftCell{p.NFL, p.Age} {
		if !reflect.DeepEqual(r.Value.Sections[i+2].Rows, draftAxis("", "", cells).Rows) {
			t.Fatal("NFL round or age lost frozen ranges and chance treatment")
		}
	}
}
