package offsite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Each league's file has its own coverage: one league's uploads failing pauses only that league at the
// 14-minute line, while the other league keeps writing and uploads under its own prefix.
func TestLeagueFilesKeepSeparateCoverage(t *testing.T) {
	r2, fake := testR2(t)
	start := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	clock := start
	failing, healthy := LeagueFileSet("native.alpha"), LeagueFileSet("native.beta")
	if failing.Prefix == healthy.Prefix || !strings.HasPrefix(failing.Prefix, LeagueFilePrefix) {
		t.Fatalf("league prefixes %q %q", failing.Prefix, healthy.Prefix)
	}
	revisions := map[string]int64{failing.Prefix: 1, healthy.Prefix: 1}
	uploads := map[string]int{}
	manager := func(kind SetKind) *Manager {
		capture := func(_ context.Context, stage string) (string, time.Time, error) {
			uploads[kind.Prefix]++
			path := filepath.Join(stage, "set.age")
			return path, clock, os.WriteFile(path, []byte(fmt.Sprintf("%s %d", kind.Prefix, uploads[kind.Prefix])), 0600)
		}
		m := NewKindManager(kind, r2, t.TempDir(), capture, func(context.Context) (int64, error) { return revisions[kind.Prefix], nil })
		m.now = func() time.Time { return clock }
		return m
	}
	managers := []*Manager{manager(failing), manager(healthy)}
	tick := func() {
		for _, m := range managers {
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			_ = m.Tick(ctx)
			cancel()
		}
	}
	tick()
	for _, m := range managers {
		if err := m.CheckWrite(); err != nil {
			t.Fatalf("%s after a healthy round: %v", m.Kind().Prefix, err)
		}
	}
	fake.mu.Lock()
	fake.failPrefix = failing.Prefix
	fake.mu.Unlock()
	for _, step := range []struct {
		at       time.Duration
		writable bool
	}{{5 * time.Minute, true}, {10 * time.Minute, true}, {14 * time.Minute, false}} {
		clock = start.Add(step.at)
		revisions[failing.Prefix]++
		revisions[healthy.Prefix]++
		tick()
		if (managers[0].CheckWrite() == nil) != step.writable {
			t.Fatalf("failing league at %v: writable=%v", step.at, !step.writable)
		}
		if err := managers[1].CheckWrite(); err != nil {
			t.Fatalf("healthy league at %v: %v", step.at, err)
		}
	}
	if uploads[healthy.Prefix] != 4 {
		t.Fatalf("healthy league uploaded %d sets, want 4", uploads[healthy.Prefix])
	}
	if !strings.Contains(managers[0].CheckWrite().Error(), "changes are paused") {
		t.Fatalf("failing league refusal: %v", managers[0].CheckWrite())
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	for name := range fake.objects {
		if !strings.HasPrefix(name, failing.Prefix) && !strings.HasPrefix(name, healthy.Prefix) {
			t.Errorf("object %s outside both leagues' prefixes", name)
		}
	}
}
