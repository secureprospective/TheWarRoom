package history

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

func openMoveTest(t *testing.T, path string) (*Store, *db.Pools) {
	t.Helper()
	p, err := db.Open(t.Context(), path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	s := New(p, registry)
	if err := s.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return s, p
}
func TestMoveCrashBoundaries(t *testing.T) {
	if dir := os.Getenv("TWR_MOVE_CRASH"); dir != "" {
		crashMove(t, dir, os.Getenv("TWR_MOVE_POINT"))
		return
	}
	for _, point := range []string{"before-seal", "sealed", "before-target-commit", "target-active", "source-moved", "target-cancelled", "source-unsealed"} {
		t.Run(point, func(t *testing.T) {
			dir := t.TempDir()
			source, sp := openMoveTest(t, filepath.Join(dir, "source.db"))
			target, tp := openMoveTest(t, filepath.Join(dir, "target.db"))
			if _, err := source.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
				t.Fatal(err)
			}
			events, _, _, err := source.CaptureTransfer(t.Context(), "native.test", nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := target.StageTransfer(t.Context(), ImportReceipt{League: "native.test", SourceEpoch: "epoch", Content: "content"}, []byte("document"), events, nil); err != nil {
				t.Fatal(err)
			}
			if err := sp.Close(); err != nil {
				t.Fatal(err)
			}
			if err := tp.Close(); err != nil {
				t.Fatal(err)
			}
			child := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestMoveCrashBoundaries$") //nolint:gosec // Fixed test binary and selector, no shell.
			child.Env = append(os.Environ(), "TWR_MOVE_CRASH="+dir, "TWR_MOVE_POINT="+point)
			output, err := child.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 73 {
				t.Fatalf("not a checkpoint crash: %v %s", err, output)
			}
			source, _ = openMoveTest(t, filepath.Join(dir, "source.db"))
			target, _ = openMoveTest(t, filepath.Join(dir, "target.db"))
			a, err := source.TransferStatus(t.Context(), "native.test")
			if err != nil {
				t.Fatal(err)
			}
			b, err := target.TransferStatus(t.Context(), "native.test")
			if err != nil {
				t.Fatal(err)
			}
			if a.Writable && b.Writable {
				t.Fatal("two writable authorities", a, b)
			}
			wantSource := point == "before-seal" || point == "source-unsealed"
			wantTarget := point == "target-active" || point == "source-moved"
			if a.Writable != wantSource || b.Writable != wantTarget {
				t.Fatal("wrong crash state", a, b)
			}
			f, err := source.TransferFence(t.Context(), "native.test")
			if err != nil {
				t.Fatal(err)
			}
			if f.State == "sealed" {
				activate := point != "target-cancelled"
				err = source.DecideTransfer(t.Context(), f, activate, func() error { return target.DecideDestination(t.Context(), f, activate, events, nil) })
				if err != nil {
					t.Fatal("crash recovery", err)
				}
				a, _ = source.TransferStatus(t.Context(), "native.test")
				b, _ = target.TransferStatus(t.Context(), "native.test")
				if a.Writable == b.Writable {
					t.Fatal("recovery must restore exactly one authority", a, b)
				}
			}
		})
	}
}
func crashMove(t *testing.T, dir, point string) {
	t.Helper()
	source, sp := openMoveTest(t, filepath.Join(dir, "source.db"))
	target, tp := openMoveTest(t, filepath.Join(dir, "target.db"))
	peer, err := target.TransferIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	f := TransferFence{League: "native.test", Epoch: "epoch", Content: "content", Peer: peer}
	events, _, anchor, err := source.CaptureTransfer(t.Context(), f.League, nil)
	if err != nil {
		t.Fatal(err)
	}
	if point == "before-seal" {
		tx, err := sp.Write().BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		f.State = "sealed"
		f.Move = "move"
		if err := transferTransition(t.Context(), tx, f); err != nil {
			t.Fatal(err)
		}
		moveCrashExit()
	}
	f, err = source.SealTransfer(t.Context(), f, events, anchor)
	if err != nil {
		t.Fatal(err)
	}
	if point == "sealed" {
		moveCrashExit()
	}
	if point == "before-target-commit" {
		tx, err := tp.Write().BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		f.State = "active"
		if err := transferTransition(t.Context(), tx, f); err != nil {
			t.Fatal(err)
		}
		moveCrashExit()
	}
	activate := point != "target-cancelled" && point != "source-unsealed"
	err = source.DecideTransfer(t.Context(), f, activate, func() error {
		if err := target.DecideDestination(t.Context(), f, activate, events, nil); err != nil {
			return err
		}
		if point == "target-active" || point == "target-cancelled" {
			moveCrashExit()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	moveCrashExit()
}
func moveCrashExit() { os.Exit(73) }
