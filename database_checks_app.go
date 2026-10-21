package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func (a *App) databaseFiles() []struct {
	name  string
	pools *db.Pools
} {
	return []struct {
		name  string
		pools *db.Pools
	}{
		{"thewarroom", a.core.Pools}, {"history", a.core.HistPools}, {"whatif", a.core.WhatifPools},
	}
}

func (a *App) checkDatabases(ctx context.Context, now time.Time) {
	for _, file := range a.databaseFiles() {
		if file.pools == nil || ctx.Err() != nil {
			continue
		}
		file.pools.CheckDue(ctx, now)
		if result := file.pools.LastCheck(); result.Error != "" {
			log.Printf("the war room: %s routine checks: %s", file.name, result.Error)
		}
	}
}

func (a *App) startDatabaseChecks() {
	a.core.DatabaseCheckStart.Do(func() {
		parent := a.core.WorkCtx
		if parent == nil {
			parent = a.fallbackParent()
		}
		a.startWorker(func() {
			if a.ready() != nil {
				return
			}
			ticker := time.NewTicker(time.Hour)
			defer ticker.Stop()
			for {
				ctx, cancel := context.WithTimeout(parent, startupBudget)
				now := time.Now().UTC()
				a.checkDatabases(ctx, now)
				a.backupDatabases(ctx, now)
				cancel()
				select {
				case <-parent.Done():
					return
				case <-ticker.C:
				}
			}
		})
	})
}

func newDatabaseBackups(dir string) (*db.Backups, error) {
	b, err := db.NewBackups(dir, buildLabel(), [3]db.BackupFile{
		{Name: dbFileName("thewarroom", isDevBuild()), MaxVersion: db.MainVersion()},
		{Name: dbFileName("history", isDevBuild()), MaxVersion: history.SchemaVersion()},
		{Name: dbFileName("whatif", isDevBuild()), MaxVersion: state.SchemaVersion()},
	})
	if err != nil {
		return nil, fmt.Errorf("configure database backups: %w", err)
	}
	return b, nil
}

func (a *App) backupDatabases(ctx context.Context, now time.Time) {
	if a.core.Backups == nil {
		return
	}
	if _, err := a.core.Backups.Capture(ctx, [3]*db.Pools{a.core.Pools, a.core.HistPools, a.core.WhatifPools}, "daily", now); err != nil {
		log.Printf("the war room: daily backup: %s", err)
		return
	}
	reports, err := a.core.Backups.Prune(ctx, "")
	for _, report := range reports {
		log.Printf("the war room: backup retention: %s", report)
	}
	if err != nil {
		log.Printf("the war room: backup retention: %s", err)
	}
}

func (a *App) databaseChecksReport() *Report {
	r := wireReport(*a.databaseChecksReading())
	return &r
}

func (a *App) databaseChecksReading() *report.Report {
	r := &report.Report{ID: "database-checks", Title: "Local database checks", Status: "ok",
		Columns: []string{"File", "Last checked", "Result"}, Rows: []report.Row{},
		How: "SQLite integrity_check and foreign_key_check run after launch and weekly while open. The last attempt is held for this launch."}
	for _, file := range a.databaseFiles() {
		at, words := "Not checked this launch", "Waiting for checks"
		if file.pools != nil {
			result := file.pools.LastCheck()
			if !result.At.IsZero() {
				at = result.At.Format(time.RFC3339)
				words = databaseCheckWords(result)
				if words != "OK" {
					r.Status = "failed"
				}
			} else if r.Status == "ok" {
				r.Status = "empty"
			}
		} else if r.Status == "ok" {
			r.Status = "empty"
		}
		r.Rows = append(r.Rows, report.Row{Cells: []string{file.name, at, words}})
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
