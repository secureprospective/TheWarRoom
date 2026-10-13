package main

import (
	"cmp"
	"slices"
	"strings"
)

// ReportEntity ties a literal display name to its held identity, including names in prose.
type ReportEntity struct {
	Text string `json:"text"`
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func (v nowView) inspectable(r *Report) {
	entities := []ReportEntity{}
	for _, id := range v.in.lookup.IDs() {
		name, _, ok := v.in.lookup.DisplayFacts(id)
		if ok {
			entities = append(entities, ReportEntity{Text: name.Name, Kind: "player", ID: id})
		}
	}
	for id, name := range v.names {
		entities = append(entities, ReportEntity{Text: name, Kind: "franchise", ID: id})
	}
	attachReportEntities(r, entities)
}

func attachReportEntities(r *Report, entities []ReportEntity) {
	texts := append(slices.Clone(r.Columns), r.Title, r.Note, r.How)
	for _, row := range append(slices.Clone(r.Rows), r.Summary...) {
		texts = append(texts, row.Cells...)
		texts = append(texts, row.Group)
	}
	seen := map[string]int{}
	for _, e := range entities {
		seen[e.Text]++
	}
	for _, e := range entities {
		if e.Text == "" || seen[e.Text] != 1 {
			continue // An ambiguous display name is never assigned a guessed identity.
		}
		if slices.ContainsFunc(texts, func(s string) bool { return strings.Contains(s, e.Text) }) {
			r.Entities = append(r.Entities, e)
		}
	}
	slices.SortFunc(r.Entities, func(a, b ReportEntity) int {
		return cmp.Or(cmp.Compare(len(b.Text), len(a.Text)), strings.Compare(a.Text, b.Text))
	})
}
