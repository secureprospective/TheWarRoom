package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/zalando/go-keyring"
)

// Disk snapshots avoid package globals and give each caller its own decoded, mutable copy.
func TestMain(m *testing.M) {
	// The wire goldens were recorded in the league's zone; pin it so CI (UTC) renders the same text.
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		log.Printf("load test time zone: %v", err)
		os.Exit(1)
	}
	time.Local = loc
	dir, err := os.MkdirTemp("", "warroom-test-fixtures-")
	if err != nil {
		log.Printf("create test fixtures: %v", err)
		os.Exit(1)
	}
	if err := os.Setenv("TWR_ROOT_TEST_FIXTURES", dir); err != nil {
		log.Printf("set test fixtures: %v", err)
		os.Exit(1)
	}
	if err := seedHistoryFixture(context.Background(), filepath.Join(dir, "history.db")); err != nil {
		log.Printf("seed history fixture: %v", err)
		os.Exit(1)
	}
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("remove test fixtures: %v", err)
		code = 1
	}
	os.Exit(code)
}

func suiteFixturePath(t *testing.T, name string) string {
	t.Helper()
	dir := os.Getenv("TWR_ROOT_TEST_FIXTURES")
	if dir == "" {
		t.Fatal("test fixture directory not initialized")
	}
	return filepath.Join(dir, name)
}

func TestReducedArchiveCopiesAreIndependent(t *testing.T) {
	t.Parallel()
	first := reducedArchive(t)
	if first.Facts.Fingerprint() != first.SHA {
		t.Fatal("archive snapshot serialization changed its facts")
	}
	year := first.Facts.Season.Year[0]
	if first.Facts.Season.Name[0] == nil {
		t.Fatal("fixture must include a league name")
	}
	name := *first.Facts.Season.Name[0]
	first.Facts.Season.Year[0] = -1
	*first.Facts.Season.Name[0] = "mutated by this test"
	second := reducedArchive(t)
	if second.SHA != first.SHA || second.Facts.Season.Year[0] != year ||
		second.Facts.Season.Name[0] == nil || *second.Facts.Season.Name[0] != name {
		t.Fatal("mutating a test's archive changed the shared fixture")
	}
}

func TestLineupFixtureDoesNotResetKeyring(t *testing.T) {
	want := errors.New("keyring owned by a different test")
	keyring.MockInitWithError(want)
	t.Cleanup(keyring.MockInit)
	lineupDraftApp(t)
	if _, err := keyring.Get("TheWarRoom", "fixture-guard"); !errors.Is(err, want) {
		t.Fatal("lineup fixture changed the process-global keyring mock", err)
	}
}

func seedHistoryFixture(ctx context.Context, path string) (err error) {
	pools, err := db.Open(ctx, path, history.SchemaVersion())
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, pools.Close()) }()
	reg, err := measures.Embedded()
	if err != nil {
		return err
	}
	return history.New(pools, reg).Initialize(ctx)
}

func TestHistoryFixturesAreIndependent(t *testing.T) {
	t.Parallel()
	first := targetTestApp(t)
	recordTargetPlayers(t, first.core.History)
	body := []byte(`{"players":{"player":{"id":"0042","name":"Archived","position":"QB","team":"BUF"}}}`)
	sum := sha256.Sum256(body)
	sha := hex.EncodeToString(sum[:])
	if _, err := first.core.History.Body(first.core.Ctx, sha); err != nil {
		t.Fatal("first fixture did not retain its write", err)
	}
	second := targetTestApp(t)
	if _, err := second.core.History.Body(second.core.Ctx, sha); !errors.Is(err, history.ErrNoBody) {
		t.Fatal("history fixtures share writes", err)
	}
}
