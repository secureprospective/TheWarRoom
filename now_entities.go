package main

import (
	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

// ReportEntity ties a literal display name to its held identity, including names in prose.
type ReportEntity struct {
	Text string `json:"text"`
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func attachReportEntities(r *Report, entities []ReportEntity) {
	s := report.Report{Columns: r.Columns, Title: r.Title, Note: r.Note, How: r.How}
	for _, row := range r.Rows {
		s.Rows = append(s.Rows, report.Row{Cells: row.Cells, Group: row.Group})
	}
	for _, row := range r.Summary {
		s.Summary = append(s.Summary, report.Row{Cells: row.Cells, Group: row.Group})
	}
	for _, e := range r.Entities {
		s.Entities = append(s.Entities, report.Entity(e))
	}
	es := make([]report.Entity, len(entities))
	for i, e := range entities {
		es[i] = report.Entity(e)
	}
	nowapp.AttachReportEntities(&s, es)
	if s.Entities == nil {
		r.Entities = nil
		return
	}
	r.Entities = make([]ReportEntity, len(s.Entities))
	for i, e := range s.Entities {
		r.Entities[i] = ReportEntity(e)
	}
}
