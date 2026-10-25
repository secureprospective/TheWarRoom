package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/hostedfixture"
)

type rawState struct {
	Directory string
	PID       int
	Address   string
	Password  string
	Seed      string
}

func (s rawState) Validate(dir string) error {
	if s.Directory != dir || s.PID <= 1 || len(s.Password) < 12 {
		return fmt.Errorf("invalid sandbox ownership record")
	}
	if _, err := hostedfixture.Code(s.Seed, time.Now()); err != nil {
		return fmt.Errorf("invalid sandbox authenticator: %w", err)
	}
	return nil
}

func writePrivate(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0600); err != nil { //nolint:gosec // Paths are inside the explicitly claimed disposable directory.
		return fmt.Errorf("write sandbox file: %w", err)
	}
	return nil
}

func saveState(dir string, state rawState) error {
	data, err := json.Marshal(state) //nolint:gosec // Disposable credentials are intentionally retained in the owner-only sandbox record.
	if err != nil {
		return fmt.Errorf("encode sandbox record: %w", err)
	}
	return writePrivate(filepath.Join(dir, "sandbox.json"), data)
}

func readState(dir string) (rawState, error) {
	data, err := os.ReadFile(filepath.Join(dir, "sandbox.json")) //nolint:gosec // Explicit sandbox directory was canonicalized and checked against real profiles.
	if err != nil {
		return rawState{}, fmt.Errorf("read sandbox ownership record: %w", err)
	}
	var state rawState
	if err := json.Unmarshal(data, &state); err != nil {
		return rawState{}, fmt.Errorf("decode sandbox record: %w", err)
	}
	if err := state.Validate(dir); err != nil {
		return rawState{}, err
	}
	return state, nil
}

func down(ctx context.Context, dir string) error {
	state, err := readState(dir)
	if err != nil {
		return err
	}
	alive, err := sandboxProcess(dir, state.PID)
	if err != nil {
		return err
	}
	if alive {
		if err := syscall.Kill(state.PID, syscall.SIGTERM); err != nil && err != syscall.ESRCH {
			return fmt.Errorf("stop sandbox server: %w", err)
		}
		if err := awaitStop(ctx, dir, state.PID); err != nil {
			return err
		}
	}
	if err := os.RemoveAll(dir); err != nil { //nolint:gosec // Validated ownership record and stopped executable authorize only this sandbox directory.
		return fmt.Errorf("delete sandbox: %w", err)
	}
	return nil
}

func sandboxProcess(dir string, pid int) (bool, error) {
	executable, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("identify sandbox PID: %w", err)
	}
	if executable != filepath.Join(dir, "twr-server") {
		return false, fmt.Errorf("recorded PID belongs to another executable; refusing to signal or delete")
	}
	return true, nil
}

func awaitStop(ctx context.Context, dir string, pid int) error {
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		alive, err := sandboxProcess(dir, pid)
		if err != nil {
			return err
		}
		if !alive {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for sandbox shutdown: %w", ctx.Err())
		case <-timeout.C:
			return fmt.Errorf("server did not stop; sandbox retained")
		case <-ticker.C:
		}
	}
}
