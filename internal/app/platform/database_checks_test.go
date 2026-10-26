package platform

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestDatabaseChecksReading(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	ok := db.RoutineCheck{At: at, Findings: db.CheckFindings{Integrity: []string{"ok"}}}
	failed := db.RoutineCheck{At: at, Error: "database closed"}
	for _, tc := range []struct {
		name, status string
		checks       []DatabaseCheck
		words        []string
	}{
		{"none", "ok", nil, []string{}},
		{"unchecked", "empty", []DatabaseCheck{{Name: "main"}}, []string{"Waiting for checks"}},
		{"healthy", "ok", []DatabaseCheck{{"main", ok}}, []string{"OK"}},
		{"failed then unchecked", "failed", []DatabaseCheck{{"history", failed}, {Name: "main"}}, []string{"database closed", "Waiting for checks"}},
		{"unchecked then failed", "failed", []DatabaseCheck{{Name: "main"}, {"history", failed}, {"whatif", ok}}, []string{"Waiting for checks", "database closed", "OK"}},
		{"integrity failure", "failed", []DatabaseCheck{{"main", db.RoutineCheck{At: at, Findings: db.CheckFindings{Integrity: []string{"damaged"}}}}}, []string{"Integrity: [damaged]; foreign key violations: 0"}},
		{"foreign key failure", "failed", []DatabaseCheck{{"main", db.RoutineCheck{At: at, Findings: db.CheckFindings{Integrity: []string{"ok"}, ForeignKeys: []db.ForeignKeyFinding{{Table: "players"}}}}}}, []string{"Integrity: [ok]; foreign key violations: 1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := DatabaseChecksReading(tc.checks)
			words := []string{}
			for i, row := range r.Rows {
				words = append(words, row.Cells[2])
				wantAt := "Not checked this launch"
				if !tc.checks[i].Result.At.IsZero() {
					wantAt = at.Format(time.RFC3339)
				}
				if row.Cells[0] != tc.checks[i].Name || row.Cells[1] != wantAt {
					t.Fatalf("row identity/date: %+v", row)
				}
			}
			if r.Status != tc.status || !reflect.DeepEqual(words, tc.words) {
				t.Fatalf("status %q words %v; want %q %v", r.Status, words, tc.status, tc.words)
			}
		})
	}
}
