// Package commissionertest supplies offline ledger fixtures without importing the feature or shared state.
package commissionertest

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func History(t *testing.T) *history.Store {
	t.Helper()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, registry)
	store.SetClock(func() time.Time { return time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC) })
	if err := store.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func RepoPath(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(append([]string{filepath.Dir(file), "..", "..", "..", ".."}, parts...)...)
}

func Body(t *testing.T, raw commissioner.RawCommand) string {
	t.Helper()
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func Execute(t *testing.T, execute func(string) (commissioner.Receipt, error),
	raw commissioner.RawCommand) commissioner.Receipt {
	t.Helper()
	receipt, err := execute(Body(t, raw))
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func Preview(t *testing.T, preview func(string) (commissioner.Preview, error),
	raw commissioner.RawCommand) commissioner.Preview {
	t.Helper()
	reading, err := preview(Body(t, raw))
	if err != nil {
		t.Fatal(err)
	}
	return reading
}

func NoticeCommand(league, command string, expected int64, request, text string,
	pin bool) commissioner.RawCommand {
	return commissioner.RawCommand{League: league, Command: command, Expected: expected, Request: request,
		Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Note: &commissioner.RawNote{Text: text, Pin: pin}}}
}
