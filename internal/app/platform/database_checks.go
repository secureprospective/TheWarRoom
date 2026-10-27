package platform

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

// DatabaseCheck is the last routine check of one local database; zero means not checked.
type DatabaseCheck struct {
	Name   string
	Result db.RoutineCheck
}

// DatabaseChecksReading keeps failed files visible alongside files awaiting their first check.
func DatabaseChecksReading(files []DatabaseCheck) *report.Report {
	r := &report.Report{ID: "database-checks", Title: "Local database checks", Status: "ok",
		Columns: []string{"File", "Last checked", "Result"}, Rows: []report.Row{},
		How: "SQLite integrity_check and foreign_key_check run after launch and weekly while open. The last attempt is held for this launch."}
	for _, file := range files {
		at, words := "Not checked this launch", "Waiting for checks"
		result := file.Result
		if !result.At.IsZero() {
			at = result.At.Format(time.RFC3339)
			words = databaseCheckWords(result)
			if words != "OK" {
				r.Status = "failed"
			}
		} else if r.Status == "ok" {
			r.Status = "empty"
		}
		r.Rows = append(r.Rows, report.Row{Cells: []string{file.Name, at, words}})
	}
	return r
}

func databaseCheckWords(result db.RoutineCheck) string {
	if result.Error != "" {
		return result.Error
	}
	if len(result.Findings.Integrity) == 1 && result.Findings.Integrity[0] == "ok" && len(result.Findings.ForeignKeys) == 0 {
		return "OK"
	}
	return fmt.Sprintf("Integrity: %v; foreign key violations: %d", result.Findings.Integrity, len(result.Findings.ForeignKeys))
}
