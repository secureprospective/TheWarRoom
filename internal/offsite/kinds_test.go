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

// The league set and the sign-in set keep separate coverage: one kind's uploads failing pauses only
// that kind's writes at the 14-minute line, and an unchanged file renews without an upload while the
// other file uploads.
func TestKindsKeepSeparateCoverage(t *testing.T) {
	for _, failing := range []SetKind{LeagueSet, SignInSet} {
		t.Run(failing.Prefix, func(t *testing.T) {
			r2, fake := testR2(t)
			fake.failPrefix = failing.Prefix
			clock := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
			revisions := map[SetKind]int64{LeagueSet: 1, SignInSet: 1}
			captures := map[SetKind]int{}
			manager := func(kind SetKind) *Manager {
				capture := func(_ context.Context, stage string) (string, time.Time, error) {
					captures[kind]++
					path := filepath.Join(stage, "set.age")
					return path, clock, os.WriteFile(path, []byte(fmt.Sprintf("%s %d", kind.File, captures[kind])), 0600)
				}
				m := NewKindManager(kind, r2, t.TempDir(), capture, func(context.Context) (int64, error) { return revisions[kind], nil })
				m.now = func() time.Time { return clock }
				return m
			}
			managers := map[SetKind]*Manager{LeagueSet: manager(LeagueSet), SignInSet: manager(SignInSet)}
			other := SignInSet
			if failing == SignInSet {
				other = LeagueSet
			}
			tick := func() {
				for _, kind := range []SetKind{LeagueSet, SignInSet} {
					ctx, cancel := context.WithTimeout(t.Context(), time.Second)
					_ = managers[kind].Tick(ctx)
					cancel()
				}
			}
			// Restart begins paused for both; one healthy round opens both.
			for _, kind := range []SetKind{LeagueSet, SignInSet} {
				if managers[kind].CheckWrite() == nil {
					t.Fatalf("%s writable before its first verified set", kind.File)
				}
			}
			fake.failPrefix = ""
			tick()
			for _, kind := range []SetKind{LeagueSet, SignInSet} {
				if err := managers[kind].CheckWrite(); err != nil {
					t.Fatalf("%s after a healthy round: %v", kind.File, err)
				}
			}
			// Now only the failing kind's uploads fail while its file keeps changing; the other file is
			// unchanged and renews with no upload.
			fake.mu.Lock()
			fake.failPrefix = failing.Prefix
			fake.mu.Unlock()
			before := captures[other]
			for _, step := range []struct {
				at       time.Duration
				writable bool
			}{{5 * time.Minute, true}, {10 * time.Minute, true}, {14 * time.Minute, false}} {
				clock = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC).Add(step.at)
				revisions[failing]++
				tick()
				if (managers[failing].CheckWrite() == nil) != step.writable {
					t.Fatalf("failing %s at %v: writable=%v", failing.File, step.at, !step.writable)
				}
				if err := managers[other].CheckWrite(); err != nil {
					t.Fatalf("healthy %s at %v: %v", other.File, step.at, err)
				}
			}
			if captures[other] != before {
				t.Fatalf("unchanged %s uploaded %d times", other.File, captures[other]-before)
			}
			// The healthy file changes: it alone uploads, under its own prefix.
			revisions[other]++
			clock = clock.Add(5 * time.Minute)
			tick()
			if captures[other] != before+1 {
				t.Fatalf("changed %s did not upload", other.File)
			}
			fake.mu.Lock()
			defer fake.mu.Unlock()
			for name := range fake.objects {
				if strings.HasPrefix(name, failing.Prefix) {
					delete(fake.objects, name) // the one healthy-round set of the failing kind
					continue
				}
				if !strings.HasPrefix(name, other.Prefix) {
					t.Errorf("object %s outside both prefixes", name)
				}
			}
			if want := 2; len(fake.objects) != want {
				t.Errorf("objects %d, want %d under %s", len(fake.objects), want, other.Prefix)
			}
		})
	}
}
