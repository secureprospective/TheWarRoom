package composition

import (
	"context"
	"log"
	"time"

	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"
	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/db"
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
	files := []platformapp.DatabaseFile{}
	for _, file := range a.databaseFiles() {
		files = append(files, platformapp.DatabaseFile{Name: file.name, Pools: file.pools})
	}
	platformapp.CheckDatabases(ctx, now, files, func(line string) { log.Print(line) })
}

func (a *App) startDatabaseChecks() {
	a.core.DatabaseCheckStart.Do(func() {
		parent := a.core.WorkCtx
		if parent == nil {
			parent = a.fallbackParent()
		}
		a.startWorker(func() {
			(platformapp.DatabaseWorker{Ready: a.ready, Check: a.checkDatabases, Backup: a.backupDatabases}).Run(parent, time.Hour, startupBudget)
		})
	})
}

func newDatabaseBackups(dir string) (*db.Backups, error) {
	return platformapp.NewDatabaseBackups(dir, buildLabel(), isDevBuild()) //nolint:wrapcheck // Preserve backup configuration errors.
}

func (a *App) backupDatabases(ctx context.Context, now time.Time) {
	a.backupWorkflow().Daily(ctx, now) //nolint:contextcheck // Daily never calls the manual Seat/Context ports; capture and retention use ctx.
}

func (a *App) backupNow() (commissionerapp.BackupResult, error) {
	result, err := a.backupWorkflow().Manual()
	return commissionerapp.BackupResult(result), err
}

func (a *App) backupWorkflow() platformapp.BackupWorkflow {
	return platformapp.BackupWorkflow{
		Seat: func() (bool, error) { status, err := a.commissionerSeat(); return status.HasAccount, err }, NeedsSignIn: commissionerapp.ErrBackupNeedsSignIn,
		Configured: func() bool { return a.core.Backups != nil }, Context: func() context.Context { return a.core.Ctx },
		Capture: func(ctx context.Context, kind string, now time.Time) (string, error) {
			return a.core.Backups.Capture(ctx, [3]*db.Pools{a.core.Pools, a.core.HistPools, a.core.WhatifPools}, kind, now)
		},
		Prune: func(ctx context.Context) ([]string, error) { return a.core.Backups.Prune(ctx, "") },
		Log:   func(line string) { log.Print(line) }, Now: time.Now, Budget: startupBudget,
	}
}

func (a *App) databaseChecksReport() *Report {
	r := wireReport(*a.databaseChecksReading())
	return &r
}

func (a *App) databaseChecksReading() *report.Report {
	checks := []platformapp.DatabaseCheck{}
	for _, file := range a.databaseFiles() {
		check := platformapp.DatabaseCheck{Name: file.name}
		if file.pools != nil {
			check.Result = file.pools.LastCheck()
		}
		checks = append(checks, check)
	}
	return platformapp.DatabaseChecksReading(checks)
}
