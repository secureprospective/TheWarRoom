package platform

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type DatabaseFile struct {
	Name  string
	Pools *db.Pools
}

func CheckDatabases(ctx context.Context, now time.Time, files []DatabaseFile, logLine func(string)) {
	for _, file := range files {
		if file.Pools == nil || ctx.Err() != nil {
			continue
		}
		file.Pools.CheckDue(ctx, now)
		if result := file.Pools.LastCheck(); result.Error != "" {
			logLine(fmt.Sprintf("the war room: %s routine checks: %s", file.Name, result.Error))
		}
	}
}

type DatabaseWorker struct {
	Ready         func() error
	Check, Backup func(context.Context, time.Time)
}

func (s DatabaseWorker) Run(parent context.Context, interval, budget time.Duration) {
	if s.Ready() != nil {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		ctx, cancel := context.WithTimeout(parent, budget)
		now := time.Now().UTC()
		s.Check(ctx, now)
		s.Backup(ctx, now)
		cancel()
		select {
		case <-parent.Done():
			return
		case <-ticker.C:
		}
	}
}

func NewDatabaseBackups(dir, label string, devBuild bool) (*db.Backups, error) {
	b, err := db.NewBackups(dir, label, [3]db.BackupFile{
		{Name: DatabaseFileName("thewarroom", devBuild), MaxVersion: db.MainVersion()},
		{Name: DatabaseFileName("history", devBuild), MaxVersion: history.SchemaVersion()},
		{Name: DatabaseFileName("whatif", devBuild), MaxVersion: state.SchemaVersion()},
	})
	if err != nil {
		return nil, fmt.Errorf("configure database backups: %w", err)
	}
	return b, nil
}

// ErrBackupFailed is public recovery wording; the technical cause stays in the private log.
var ErrBackupFailed = errors.New("the backup could not be saved; close and reopen the app, then try again")

type BackupReceipt struct{ Set, Message string }

type BackupWorkflow struct {
	Seat        func() (bool, error)
	NeedsSignIn error
	Configured  func() bool
	Context     func() context.Context
	Capture     func(context.Context, string, time.Time) (string, error)
	Prune       func(context.Context) ([]string, error)
	Log         func(string)
	Now         func() time.Time
	Budget      time.Duration
}

func (s BackupWorkflow) Daily(ctx context.Context, now time.Time) {
	if !s.Configured() {
		return
	}
	if _, err := s.Capture(ctx, "daily", now); err != nil {
		s.Log(fmt.Sprintf("the war room: daily backup: %s", err))
		return
	}
	s.PruneOld(ctx)
}

func (s BackupWorkflow) Manual() (BackupReceipt, error) {
	account, err := s.Seat()
	if err != nil {
		return BackupReceipt{}, err
	}
	if !account {
		return BackupReceipt{}, s.NeedsSignIn
	}
	if !s.Configured() {
		s.Log("the war room: backup now: backups are not configured")
		return BackupReceipt{}, ErrBackupFailed
	}
	ctx, cancel := context.WithTimeout(s.Context(), s.Budget)
	defer cancel()
	set, err := s.Capture(ctx, "manual", s.Now().UTC())
	if err != nil {
		s.Log(fmt.Sprintf("the war room: backup now: %s", err))
		return BackupReceipt{}, ErrBackupFailed
	}
	s.Log(fmt.Sprintf("the war room: backup now: saved %s", set))
	s.PruneOld(ctx)
	name := filepath.Base(set)
	return BackupReceipt{Set: name, Message: "Backup saved as " + name +
		". It holds your league and your sign-in; the app keeps your five newest."}, nil
}

func (s BackupWorkflow) PruneOld(ctx context.Context) {
	reports, err := s.Prune(ctx)
	for _, report := range reports {
		s.Log(fmt.Sprintf("the war room: backup retention: %s", report))
	}
	if err != nil {
		s.Log(fmt.Sprintf("the war room: backup retention: %s", err))
	}
}
