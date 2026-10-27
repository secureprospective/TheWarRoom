package platform

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/offline"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type StartupWorkflow struct {
	Directory    func() (string, error)
	SetDirectory func(string)
	Logging      func(string) error
	Databases    func(context.Context, string) (string, *history.Store, error)
	Client       func() error
	Photos       func()
	Floor        func(context.Context, *history.Store) (bool, error)
	Keys         func(context.Context)
	LaunchDue    func(bool)
	Season       func() int
	Label        string
	Log          func(string)
}

func (s StartupWorkflow) Run(ctx context.Context) (bool, error) {
	dir, err := s.Directory()
	if err != nil {
		return false, fmt.Errorf("startup: resolve config dir: %w", err)
	}
	s.SetDirectory(dir)
	if err := s.Logging(dir); err != nil {
		s.Log(fmt.Sprintf("the war room: WARNING disk logging unavailable: %v", err))
	}
	path, hist, err := s.Databases(ctx, dir)
	if err != nil {
		return false, fmt.Errorf("startup: %w", err)
	}
	if err := s.Client(); err != nil {
		return false, fmt.Errorf("startup: mfl client: %w", err)
	}
	s.Photos()
	s.Log(fmt.Sprintf("the war room: starting %s, db %s", s.Label, path))
	began := time.Now()
	refreshed, err := s.Floor(ctx, hist)
	if err != nil {
		return false, err
	}
	s.Log(fmt.Sprintf("the war room: season %d ready in %s", s.Season(), time.Since(began).Round(time.Millisecond)))
	s.Keys(ctx)
	s.LaunchDue(!refreshed)
	return refreshed, nil
}

func StartupFailure(err error, logLine func(string)) error {
	if err != nil {
		logLine(fmt.Sprintf("the war room: startup failed: %v", err))
		return offline.Plain(err) //nolint:wrapcheck // Startup deliberately exposes the shared plain offline refusal.
	}
	return nil
}

func FinishStartup(failure error, refreshed bool, publish, revalidate func()) {
	publish()
	if failure != nil || !refreshed {
		return
	}
	revalidate()
}

type StoreFloor struct {
	History, Seat, Params, Moves, Personal, Rules, Mirror func(context.Context) error
	MirrorSeason                                          func() int
	Refresh, WhatIf, Coordinator                          func(context.Context) error
	MigrationFindings, Publish                            func()
	DraftReports, SeedClock                               func(context.Context)
	Log                                                   func(string)
}

func (s StoreFloor) Run(ctx context.Context) (bool, error) {
	refreshed := false
	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"history", s.History}, {"local sign-in", s.Seat}, {"params", s.Params}, {"moves", s.Moves},
		{"personal lists", s.Personal}, {"rulebook", s.Rules}, {"league mirror", s.Mirror},
		{"first MFL refresh", func(ctx context.Context) error {
			if s.MirrorSeason() != 0 {
				return nil
			}
			refreshed = true
			return s.Refresh(ctx)
		}},
		{"what-if league", s.WhatIf}, {"transaction coordinator", s.Coordinator},
	}
	for _, step := range steps {
		began := time.Now()
		if err := step.run(ctx); err != nil {
			return false, fmt.Errorf("startup: initialize %s: %w", step.name, err)
		}
		s.Log(fmt.Sprintf("the war room: %s ready in %s", step.name, time.Since(began).Round(time.Millisecond)))
	}
	s.MigrationFindings()
	s.Publish()
	if refreshed {
		s.DraftReports(ctx)
	}
	s.SeedClock(ctx)
	return refreshed, nil
}

type DatabaseOpening struct {
	Lock                  func(string) error
	Backups               func(string) error
	Recover               func(context.Context) error
	BeforeMigrations      func(context.Context, time.Time) error
	Main, History, WhatIf func(context.Context, string) error
	Registry              func() (*measures.Registry, error)
	Attach                func(*measures.Registry) *history.Store
	Archive               func(*history.Store)
	DevBuild              bool
}

func (s DatabaseOpening) Open(ctx context.Context, dir string) (string, *history.Store, error) {
	path := filepath.Join(dir, DatabaseFileName("thewarroom", s.DevBuild))
	if err := s.Lock(path); err != nil {
		return "", nil, err
	}
	if err := s.Backups(dir); err != nil {
		return "", nil, err
	}
	if err := s.Recover(ctx); err != nil {
		return "", nil, fmt.Errorf("recover databases: %w", err)
	}
	if err := s.BeforeMigrations(ctx, time.Now().UTC()); err != nil {
		return "", nil, fmt.Errorf("capture before migrations: %w", err)
	}
	if err := s.Main(ctx, path); err != nil {
		return "", nil, fmt.Errorf("open database: %w", err)
	}
	reg, err := s.Registry()
	if err != nil {
		return "", nil, err
	}
	if err := s.History(ctx, filepath.Join(dir, DatabaseFileName("history", s.DevBuild))); err != nil {
		return "", nil, fmt.Errorf("open history database: %w", err)
	}
	if err := s.WhatIf(ctx, filepath.Join(dir, DatabaseFileName("whatif", s.DevBuild))); err != nil {
		return "", nil, fmt.Errorf("open what-if database: %w", err)
	}
	hist := s.Attach(reg)
	s.Archive(hist)
	return path, hist, nil
}

func LogMigrationFindings(files []DatabaseFile, logLine func(string)) {
	for _, file := range files {
		if file.Pools != nil {
			findings := file.Pools.MigrationFindings()
			logLine(fmt.Sprintf("the war room: %s migration checks: before=%+v after=%+v", file.Name, findings.Before, findings.After))
		}
	}
}

// WorkerGroup operates on caller-owned admission state; cancellation and joining precede resource close.
type WorkerGroup struct {
	Mutex    *sync.Mutex
	Stopping *bool
	Workers  *sync.WaitGroup
}

func (s WorkerGroup) Start(work func()) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	if *s.Stopping {
		return
	}
	s.Workers.Add(1)
	go func() { defer s.Workers.Done(); work() }()
}

func (s WorkerGroup) Stop(cancels ...*context.CancelFunc) {
	s.Mutex.Lock()
	*s.Stopping = true
	for _, cancel := range cancels {
		if *cancel != nil {
			(*cancel)()
		}
	}
	s.Mutex.Unlock()
	s.Workers.Wait()
}

func CloseDatabases(files []*db.Pools) {
	for _, pool := range files {
		if pool != nil {
			_ = pool.Close()
		}
	}
}
