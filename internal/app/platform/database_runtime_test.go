package platform

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBackupAdmissionAndPrivateFailure(t *testing.T) {
	needsSignIn := errors.New("sign in first")
	configured, account := false, false
	logs := []string{}
	service := BackupWorkflow{Seat: func() (bool, error) { return account, nil }, NeedsSignIn: needsSignIn,
		Configured: func() bool { return configured }, Log: func(line string) { logs = append(logs, line) }}
	if _, err := service.Manual(); !errors.Is(err, needsSignIn) || len(logs) != 0 {
		t.Fatal(err, logs)
	}
	account = true
	if _, err := service.Manual(); !errors.Is(err, ErrBackupFailed) || len(logs) != 1 {
		t.Fatal(err, logs)
	}
	service.Daily(context.Background(), time.Now())
	if len(logs) != 1 {
		t.Fatal(logs)
	}
	configured = true
	service.Context = context.Background
	service.Budget = time.Minute
	service.Now = time.Now
	service.Capture = func(ctx context.Context, kind string, _ time.Time) (string, error) {
		if kind != "manual" {
			t.Fatal(kind)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded capture")
		}
		return "", errors.New("private-path-and-technical-cause")
	}
	if _, err := service.Manual(); !errors.Is(err, ErrBackupFailed) || strings.Contains(err.Error(), "private-path") || !strings.Contains(logs[1], "private-path") {
		t.Fatal(err, logs)
	}
}

func TestBackupCapturePrecedesBestEffortRetention(t *testing.T) {
	calls := []string{}
	service := BackupWorkflow{Seat: func() (bool, error) { return true, nil }, Configured: func() bool { return true }, Context: context.Background,
		Budget: time.Minute, Now: time.Now, Log: func(string) {},
		Capture: func(_ context.Context, kind string, _ time.Time) (string, error) {
			calls = append(calls, kind)
			return "/private/backups/set-1", nil
		},
		Prune: func(context.Context) ([]string, error) {
			calls = append(calls, "prune")
			return []string{"one finding"}, errors.New("retention failed")
		},
	}
	result, err := service.Manual()
	if err != nil || result.Set != "set-1" || !strings.Contains(result.Message, "five newest") || !reflect.DeepEqual(calls, []string{"manual", "prune"}) {
		t.Fatal(result, err, calls)
	}
	calls = nil
	service.Daily(context.Background(), time.Now())
	if !reflect.DeepEqual(calls, []string{"daily", "prune"}) {
		t.Fatal(calls)
	}
	calls = nil
	service.Capture = func(context.Context, string, time.Time) (string, error) {
		calls = append(calls, "capture failure")
		return "", context.Canceled
	}
	service.Daily(context.Background(), time.Now())
	if !reflect.DeepEqual(calls, []string{"capture failure"}) {
		t.Fatal(calls)
	}
}

func TestDatabaseWorkerReadyGateAndBoundedJoinedCycle(t *testing.T) {
	service := DatabaseWorker{Ready: func() error { return context.Canceled }}
	service.Run(context.Background(), time.Hour, time.Minute)
	parent, stop := context.WithCancel(context.Background())
	defer stop()
	calls := []string{}
	var checkContext context.Context
	service.Ready = func() error { return nil }
	service.Check = func(ctx context.Context, _ time.Time) {
		checkContext = ctx
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded checks")
		}
		calls = append(calls, "check")
	}
	service.Backup = func(ctx context.Context, _ time.Time) {
		if ctx != checkContext {
			t.Fatal("different maintenance cycle")
		}
		calls = append(calls, "backup")
		stop()
	}
	service.Run(parent, time.Hour, time.Minute)
	if !reflect.DeepEqual(calls, []string{"check", "backup"}) || checkContext.Err() == nil {
		t.Fatal(calls, checkContext.Err())
	}
	CheckDatabases(parent, time.Now(), []DatabaseFile{{Name: "absent"}}, func(string) { t.Fatal("logged absent pool") })
}
