package composition

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"time"

	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"
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
	a.pruneBackups(ctx)
}

// errBackupFailed is all a person sees of a failed backup; the cause goes to the log.
var errBackupFailed = errors.New("the backup could not be saved; close and reopen the app, then try again")

// backupNow is Back up now: a manual matched set taken at once, after the local sign-in exists,
// because a set without it restores elsewhere with no one's roles. The five newest are kept.
func (a *App) backupNow() (commissionerapp.BackupResult, error) {
	status, err := a.commissionerSeat()
	if err != nil {
		return commissionerapp.BackupResult{}, err
	}
	if !status.HasAccount {
		return commissionerapp.BackupResult{}, commissionerapp.ErrBackupNeedsSignIn
	}
	if a.core.Backups == nil {
		log.Printf("the war room: backup now: backups are not configured")
		return commissionerapp.BackupResult{}, errBackupFailed
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, startupBudget)
	defer cancel()
	set, err := a.core.Backups.Capture(ctx, [3]*db.Pools{a.core.Pools, a.core.HistPools, a.core.WhatifPools}, "manual", time.Now().UTC())
	if err != nil {
		log.Printf("the war room: backup now: %s", err)
		return commissionerapp.BackupResult{}, errBackupFailed
	}
	log.Printf("the war room: backup now: saved %s", set)
	a.pruneBackups(ctx)
	name := filepath.Base(set)
	return commissionerapp.BackupResult{Set: name, Message: "Backup saved as " + name +
		". It holds your league and your sign-in; the app keeps your five newest."}, nil
}

func (a *App) pruneBackups(ctx context.Context) {
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
