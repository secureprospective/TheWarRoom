package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type directoryDown struct{ calls int }

func (d *directoryDown) RoundTrip(*http.Request) (*http.Response, error) {
	d.calls++
	return nil, errors.New("directory offline")
}

func targetTestApp(t *testing.T) *App {
	t.Helper()
	ctx := context.Background()
	p, err := db.Open(ctx, filepath.Join(t.TempDir(), "league.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	rb := rulebook.New(p)
	mirror := state.NewMirror(p, rb)
	if err := mirror.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := mirror.Replace(ctx, state.MirrorSnapshot{Season: 2026, Players: []state.MirrorPlayer{{MFLID: "0042", FranchiseID: "0001"}}}); err != nil {
		t.Fatal(err)
	}
	hp, err := db.Open(ctx, filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := hp.Close(); err != nil {
			t.Error(err)
		}
	})
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	hs := history.New(hp, reg)
	if err := hs.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	a.ctx = ctx
	a.season = 2026
	a.league = mirror
	a.rulebook = rb
	a.history = hs
	close(a.started)
	return a
}

func TestTargetDirectoryReusesLiveCacheAndOriginalTime(t *testing.T) {
	a := targetTestApp(t)
	lk, err := normalize.NewLookup([]players.RawPlayer{{ID: "0042", Name: "Cached", Position: "QB", Team: "BUF"}})
	if err != nil {
		t.Fatal(err)
	}
	a.lookup, a.hasLookup, a.lookupAt = lk, true, time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	// A nil MFL client would panic if the live leg bypassed the directory cache.
	for range 2 {
		dir := a.targetDirectory(context.Background())
		f, ok := dir.Lookup.Facts("0042")
		if !ok || f.Name != "Cached" || dir.Provenance.Freshness.FetchedAt != "2026-10-05T12:00:00Z" || dir.Provenance.Freshness.State != domain.FreshLive {
			t.Fatalf("directory: %+v, %+v", dir.Provenance, f)
		}
	}
}

func recordTargetPlayers(t *testing.T, hs *history.Store) {
	t.Helper()
	body := []byte(`{"players":{"player":{"id":"0042","name":"Archived","position":"QB","team":"BUF"}}}`)
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	if _, err := zw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	err := hs.Record(context.Background(), archive.Fetch{URL: "/2026/export?TYPE=players&L=" + ingestion.LeagueID, Status: 200, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(body)), Gzip: gz.Bytes(), FetchedAt: time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTargetDirectoryFallsBackAfterCancelledLiveFetch(t *testing.T) {
	for _, archived := range []bool{true, false} {
		a := targetTestApp(t)
		down := &directoryDown{}
		client, err := mfl.New("", 100, mfl.WithTransport(down))
		if err != nil {
			t.Fatal(err)
		}
		a.mflClient = client
		if archived {
			recordTargetPlayers(t, a.history)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		dir := a.targetDirectory(ctx)
		if archived {
			f, ok := dir.Lookup.Facts("0042")
			if !ok || f.Name != "Archived" || dir.Provenance.Freshness.FetchedAt != "2026-10-05T10:00:00Z" || dir.Provenance.Source != "mfl-players-archive" || dir.Provenance.Freshness.State != domain.FreshStale {
				t.Fatalf("fallback: %+v, %+v", dir.Provenance, f)
			}
		} else if dir.Provenance.Freshness.State != domain.FreshFail || !strings.Contains(dir.Provenance.Freshness.Note, "no archived copy") {
			t.Fatal(dir.Provenance)
		}
		if a.hasLookup {
			t.Fatal("archive was promoted into the live cache")
		}
	}
}

func targetOfflineClient(t *testing.T, a *App) *directoryDown {
	t.Helper()
	down := &directoryDown{}
	client, err := mfl.New("", 100, mfl.WithTransport(down))
	if err != nil {
		t.Fatal(err)
	}
	a.mflClient = client
	return down
}

func TestTargetDraftIRAndMoves(t *testing.T) {
	a := targetTestApp(t)
	if _, err := a.TargetDraftIR("0001", "0042"); err == nil ||
		err.Error() != "target draft: no snapshot loaded" {
		t.Fatalf("before snapshot: %v", err)
	}
	moves, err := a.TargetMoves("0001")
	if err != nil || moves == nil || len(moves) != 0 {
		t.Fatalf("empty moves: %+v, %v", moves, err)
	}
	down := targetOfflineClient(t, a)
	if _, err := a.TargetSnapshot(); err != nil {
		t.Fatal(err)
	}
	calls := down.calls
	r, err := a.TargetDraftIR("0001", "0042")
	if err != nil {
		t.Fatal(err)
	}
	if r.State != envelope.Blocked || !regexp.MustCompile(`^ir-[0-9a-f]{32}$`).MatchString(r.CorrelationID) {
		t.Fatalf("receipt: %+v", r)
	}
	if r.Spec.Expected.Player.String() != "0042" || r.Spec.LeagueID != ingestion.LeagueID {
		t.Fatalf("scope: %+v", r.Spec)
	}
	for _, fid := range []string{"0001", "0002"} {
		moves, err := a.TargetMoves(fid)
		if err != nil || moves == nil {
			t.Fatalf("moves: %+v, %v", moves, err)
		}
		want := 0
		if fid == "0001" {
			want = 1
		}
		if len(moves) != want || (want == 1 && moves[0].CorrelationID != r.CorrelationID) {
			t.Fatalf("moves for %s: %+v", fid, moves)
		}
	}
	for _, bad := range []string{"abc", ""} {
		if _, err := a.TargetDraftIR("0001", bad); err == nil {
			t.Fatalf("accepted player %q", bad)
		}
	}
	moves, err = a.TargetMoves("0001")
	if err != nil || len(moves) != 1 || down.calls != calls {
		t.Fatalf("invalid input changed log or network calls: %+v, %v", moves, err)
	}
}

func TestTargetDraftIRConcurrent(t *testing.T) {
	a := targetTestApp(t)
	down := targetOfflineClient(t, a)
	if _, err := a.TargetSnapshot(); err != nil {
		t.Fatal(err)
	}
	calls := down.calls
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, err := a.TargetDraftIR("0001", "0042"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	moves, err := a.TargetMoves("0001")
	if err != nil || len(moves) != 8 {
		t.Fatalf("moves: %+v, %v", moves, err)
	}
	ids := map[string]bool{}
	for _, r := range moves {
		if ids[r.CorrelationID] || r.State != envelope.Blocked {
			t.Fatalf("duplicate ID or wrong state: %+v", r)
		}
		ids[r.CorrelationID] = true
	}
	if down.calls != calls {
		t.Fatal("drafts fetched the directory")
	}
}

func TestTargetSnapshotFailureKeepsCache(t *testing.T) {
	a := targetTestApp(t)
	targetOfflineClient(t, a)
	if _, err := a.TargetSnapshot(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a.ctx = ctx
	if _, err := a.TargetSnapshot(); err == nil {
		t.Fatal("cancelled build succeeded")
	}
	r, err := a.TargetDraftIR("0001", "0042")
	if err != nil || r.State != envelope.Blocked ||
		r.Audit[0].Note != "not verified (league rules not captured); MFL target not verified" {
		t.Fatalf("cached draft: %+v, %v", r, err)
	}
}
