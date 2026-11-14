package offsite

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// Status feeds the private console dashboard (internal/serverstatus), never a public
// route. Captured is the snapshot time, not upload completion; local disk timestamps
// cannot renew coverage. Failure is the newest attempt's error, kept while changes are
// paused and cleared by the next verified upload; it may name the bucket or endpoint.
type Status struct {
	Captured time.Time
	Verified time.Time
	Writable bool
	Failure  string
}
type Manager struct {
	mu            sync.Mutex
	backupMu      sync.Mutex
	coverageStart time.Time
	status        Status
	now           func() time.Time
	r2            *R2
	capture       func(context.Context, string) (string, time.Time, error)
	revision      func(context.Context) (int64, error)
	temp          string
	kind          SetKind
	// covered is when the live database was last known to match a verified set: the snapshot start,
	// or a later due tick that read the same revision. Writability is measured from it.
	covered     time.Time
	verifiedRev int64
	hasRev      bool // verifiedRev is the revision read at the start of the newest verified set
}

// NewManager takes revision as a reader of the database's commit revision (db.Pools.Revision): when it
// is unchanged since the newest verified set began, a due tick renews coverage without a new upload.
func NewManager(r2 *R2, temp string, capture func(context.Context, string) (string, time.Time, error), revision func(context.Context) (int64, error)) *Manager {
	return NewKindManager(LeagueSet, r2, temp, capture, revision)
}

// NewKindManager backs up one kind of set: its own uploads, coverage, skip-when-unchanged and retention.
func NewKindManager(kind SetKind, r2 *R2, temp string, capture func(context.Context, string) (string, time.Time, error), revision func(context.Context) (int64, error)) *Manager {
	return &Manager{r2: r2, temp: temp, capture: capture, revision: revision, now: time.Now, kind: kind}
}

// Kind is the kind of set this manager backs up.
func (m *Manager) Kind() SetKind { return m.kind }

// CheckWrite deliberately stops new writes one minute before the 15-minute limit,
// leaving room for already admitted bounded operations. Restart starts closed.
func (m *Manager) CheckWrite() error {
	if !m.Status().Writable {
		if m.kind == ReferenceSet {
			return fmt.Errorf("the football reference backup is not current; your commissioner needs to check backup settings and the internet connection")
		}
		if m.kind == SignInSet {
			return fmt.Errorf("sign-ins and account changes are paused because the sign-in backup is not current; your commissioner needs to check backup settings and the internet connection")
		}
		return fmt.Errorf("changes are paused because the off-site backup is not current; your commissioner needs to check backup settings and the internet connection")
	}
	return nil
}
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.status
	now := m.now()
	c := m.covered
	s.Writable = !c.IsZero() && !now.Before(c) && now.Sub(c) < 14*time.Minute && now.Sub(m.coverageStart) < 14*time.Minute
	return s
}
func (m *Manager) recordFailure(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.Failure = err.Error()
}

// Backup records its own failure for the dashboard; coverage changes only on success.
func (m *Manager) Backup(ctx context.Context) error {
	m.backupMu.Lock()
	defer m.backupMu.Unlock()
	if err := m.backup(ctx); err != nil {
		m.recordFailure(err)
		return err
	}
	return nil
}

func (m *Manager) backup(ctx context.Context) error {
	start := m.now()
	// Read before the snapshot begins: a commit at any later moment changes the revision, so the
	// next due tick captures again. A read error is treated as a change.
	revision, revErr := m.readRevision(ctx)
	stage, err := os.MkdirTemp(m.temp, ".offsite-")
	if err != nil {
		return fmt.Errorf("stage off-site backup: %w", err)
	}
	defer func() { _ = os.RemoveAll(stage) }()
	path, at, err := m.capture(ctx, stage)
	if err != nil {
		return err
	}
	name := m.kind.Prefix + at.UTC().Format(objectFormat) + ".tar.gz.age"
	if err := m.r2.upload(ctx, name, path); err != nil {
		return err
	}
	if err := m.r2.verify(ctx, name, path); err != nil {
		return err
	}
	m.mu.Lock()
	m.status = Status{Captured: at, Verified: m.now()}
	m.coverageStart, m.covered = start, at
	m.verifiedRev, m.hasRev = revision, revErr == nil
	m.mu.Unlock()
	return m.r2.prunePrefix(ctx, m.kind.Prefix, m.now())
}

func (m *Manager) readRevision(ctx context.Context) (int64, error) {
	if m.revision == nil {
		return 0, fmt.Errorf("no database revision reader")
	}
	return m.revision(ctx)
}

// renewUnchanged extends coverage without a new object when the database has not committed since the
// newest verified set began. It refuses (and the caller captures) on a read error, on any change,
// with no verified set since this process started, and when the clock reads earlier than the
// coverage it would extend.
func (m *Manager) renewUnchanged(ctx context.Context) bool {
	now := m.now()
	m.mu.Lock()
	known, want, covered, start := m.hasRev, m.verifiedRev, m.covered, m.coverageStart
	m.mu.Unlock()
	if !known || now.Before(covered) || now.Before(start) {
		return false
	}
	revision, err := m.readRevision(ctx)
	if err != nil || revision != want {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.verifiedRev != want || !m.hasRev {
		return false
	}
	m.covered, m.coverageStart = now, now
	return true
}

// Tick runs one due check: renew coverage when nothing changed, otherwise capture and upload.
func (m *Manager) Tick(ctx context.Context) error {
	m.backupMu.Lock()
	renewed := m.renewUnchanged(ctx)
	m.backupMu.Unlock()
	if renewed {
		return nil
	}
	return m.Backup(ctx)
}

// Run retries every minute, capturing every five minutes while healthy and the database changed. No persisted
// receipt is trusted on restart: a new upload must be read back before writes resume.
func (m *Manager) Run(ctx context.Context) { RunInOrder(ctx, m) }

// RunInOrder runs several managers on one clock, each due tick in the order given: the league set
// before the sign-in set, so an account always commits before any seat a captured league names. Each
// manager keeps its own failure and coverage; one failing never skips the next.
func RunInOrder(ctx context.Context, managers ...*Manager) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		for _, m := range managers {
			m.runDue(ctx)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Manager) runDue(ctx context.Context) {
	m.mu.Lock()
	start := m.coverageStart
	m.mu.Unlock()
	if start.IsZero() || m.now().Sub(start) >= 5*time.Minute {
		attempt, cancel := context.WithTimeout(ctx, 2*time.Minute)
		_ = m.Tick(attempt) // recorded in Status for the dashboard; the next tick retries
		cancel()
	}
}
