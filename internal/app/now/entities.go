package now

import (
	"cmp"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

func (v nowView) inspectable(r *report.Report) {
	entities := []report.Entity{}
	for _, id := range v.in.Lookup.IDs() {
		name, _, ok := v.in.Lookup.DisplayFacts(id)
		if ok {
			entities = append(entities, report.Entity{Text: name.Name, Kind: "player", ID: id})
		}
	}
	for id, name := range v.names {
		entities = append(entities, report.Entity{Text: name, Kind: "franchise", ID: id})
	}
	AttachReportEntities(r, entities)
}

func AttachReportEntities(r *report.Report, entities []report.Entity) {
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
	slices.SortFunc(r.Entities, func(a, b report.Entity) int {
		return cmp.Or(cmp.Compare(len(b.Text), len(a.Text)), strings.Compare(a.Text, b.Text))
	})
}
