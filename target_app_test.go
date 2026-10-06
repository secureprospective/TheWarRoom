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
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
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
	return &App{ctx: ctx, season: 2026, league: mirror, rulebook: rb, history: hs}
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
