package offsite

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// Status is a VPN/dashboard port, not a public health endpoint. Captured is the
// snapshot time, not upload completion; local disk timestamps cannot renew coverage.
type Status struct {
	Captured time.Time `json:"captured"`
	Verified time.Time `json:"verified"`
	Writable bool      `json:"writable"`
	Message  string    `json:"message"`
}
type Manager struct {
	mu            sync.Mutex
	backupMu      sync.Mutex
	coverageStart time.Time
	status        Status
	now           func() time.Time
	r2            *R2
	capture       func(context.Context, string) (string, time.Time, error)
	temp          string
}

func NewManager(r2 *R2, temp string, capture func(context.Context, string) (string, time.Time, error)) *Manager {
	return &Manager{r2: r2, temp: temp, capture: capture, now: time.Now}
}

// CheckWrite deliberately stops new writes one minute before the 15-minute limit,
// leaving room for already admitted bounded operations. Restart starts closed.
func (m *Manager) CheckWrite() error {
	if !m.Status().Writable {
		return fmt.Errorf("changes are paused because the off-site backup is not current; your commissioner needs to check backup settings and the internet connection")
	}
	return nil
}
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.status
	now := m.now()
	s.Writable = !s.Captured.IsZero() && !now.Before(s.Captured) && now.Sub(s.Captured) < 14*time.Minute && now.Sub(m.coverageStart) < 14*time.Minute
	if !s.Writable {
		s.Message = "Changes paused: no verified off-site backup less than 14 minutes old."
	}
	return s
}
func (m *Manager) recordFailure(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.Message = err.Error()
}

func (m *Manager) Backup(ctx context.Context) error {
	m.backupMu.Lock()
	defer m.backupMu.Unlock()
	start := m.now()
	stage, err := os.MkdirTemp(m.temp, ".offsite-")
	if err != nil {
		return fmt.Errorf("stage off-site backup: %w", err)
	}
	defer func() { _ = os.RemoveAll(stage) }()
	path, at, err := m.capture(ctx, stage)
	if err != nil {
		return err
	}
	name := "sets/" + at.UTC().Format(objectFormat) + ".tar.gz.age"
	if err := m.r2.upload(ctx, name, path); err != nil {
		return err
	}
	if err := m.r2.verify(ctx, name, path); err != nil {
		return err
	}
	m.mu.Lock()
	m.status = Status{Captured: at, Verified: m.now()}
	m.coverageStart = start
	m.mu.Unlock()
	return m.r2.prune(ctx, m.now())
}

// Run retries every minute, capturing every five minutes while healthy. No persisted
// receipt is trusted on restart: a new upload must be read back before writes resume.
func (m *Manager) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		m.mu.Lock()
		start := m.coverageStart
		m.mu.Unlock()
		if start.IsZero() || m.now().Sub(start) >= 5*time.Minute {
			attempt, cancel := context.WithTimeout(ctx, 2*time.Minute)
			err := m.Backup(attempt)
			cancel()
			if err != nil {
				m.recordFailure(err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
