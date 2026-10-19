package history

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

// importStore is a store whose registry also holds "vault", a source that is imported, not fetched.
func importStore(t *testing.T) (*Store, *clock) {
	t.Helper()
	reg, err := measures.Load(fstest.MapFS{
		"measures.csv":      {Data: []byte(fixtureMeasures)},
		"sources.csv":       {Data: []byte(fixtureSources + "vault,Vault archive,active,30,,\n")},
		"source_fields.csv": {Data: []byte(fixtureFields)},
		"feeds.csv":         {Data: []byte("feed,source,url,first_season,id_column,id_type,season_column,week_column,filter\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s, c, _ := reopen(t, pools, reg)
	return s, c
}

func fetchOf(t *testing.T, url, source, body string, at time.Time) archive.Fetch {
	t.Helper()
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(body))
	return archive.Fetch{URL: url, Source: source, Status: 200, SHA256: hex.EncodeToString(sum[:]),
		Size: int64(len(body)), Gzip: gz.Bytes(), FetchedAt: at}
}

func TestImportedFetchesKeepTheirSourceAndStayOutOfFallbacks(t *testing.T) {
	t.Parallel()
	s, _ := importStore(t)
	ctx := context.Background()
	const u = "https://beta.example.com/2013/export?TYPE=rosters"
	if err := s.Record(ctx, fetchOf(t, u, "nowhere", "x", t0())); err == nil || !strings.Contains(err.Error(), `unregistered source "nowhere"`) {
		t.Fatalf("err = %v, want an unregistered named source refused", err)
	}
	if err := s.Record(ctx, fetchOf(t, u, "vault", "imported", t0().Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(ctx, fetchOf(t, u, "", "live", t0())); err != nil {
		t.Fatal(err)
	}
	got, err := s.SourceFetches(ctx, "vault")
	if err != nil || len(got) != 1 || got[0].URL != u || !got[0].FetchedAt.Equal(t0().Add(time.Hour)) {
		t.Fatalf("vault fetches = %+v, %v", got, err)
	}
	var offered []string
	if _, err := s.ArchivedBodies(ctx, "TYPE=rosters", func(_ string, body []byte, _ time.Time) bool {
		offered = append(offered, string(body))
		return false
	}); err != nil {
		t.Fatal(err)
	}
	if len(offered) != 1 || offered[0] != "live" {
		t.Errorf("fallback offered %q, want only the live body though the imported one is newer", offered)
	}
	held, err := s.HeldBodies(ctx, []string{got[0].SHA256, strings.Repeat("0", 64)})
	if err != nil || len(held) != 1 || !held[got[0].SHA256] {
		t.Errorf("held = %v, %v; want only the imported body", held, err)
	}
}

func TestManifestsPublishOncePerSHAAndSnapshotsAppend(t *testing.T) {
	t.Parallel()
	s, c := importStore(t)
	ctx := context.Background()
	m := ManifestRecord{SHA256: "m1", Name: "frozen", Entries: []byte(`[]`)}
	for i, want := range []bool{true, false} {
		published, err := s.PublishManifest(ctx, m)
		if err != nil || published != want {
			t.Fatalf("publish %d = %v, %v; want %v", i, published, err, want)
		}
	}
	c.now = t0().Add(time.Hour)
	if _, err := s.PublishManifest(ctx, ManifestRecord{SHA256: "m2", Name: "frozen", Entries: []byte(`[{}]`)}); err != nil {
		t.Fatal(err)
	}
	latest, ok, err := s.LatestManifest(ctx, "frozen")
	if err != nil || !ok || latest.SHA256 != "m2" || string(latest.Entries) != `[{}]` {
		t.Fatalf("latest = %+v, %v, %v; want m2", latest, ok, err)
	}
	if _, ok, _ := s.LatestManifest(ctx, "other"); ok {
		t.Error("a manifest of another name was found")
	}
	good := SnapshotRecord{ManifestSHA: "m1", Compiler: "c", RulesHash: "r", ResultSHA: "res", BuiltAt: t0()}
	failed := SnapshotRecord{ManifestSHA: "m2", Compiler: "c", RulesHash: "r", Error: "check failed", BuiltAt: t0().Add(time.Hour),
		Checks: `[{"name":"one"}]`}
	for _, r := range []SnapshotRecord{good, failed} {
		if err := s.RecordSnapshot(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RecordSnapshot(ctx, SnapshotRecord{ManifestSHA: "m1", ResultSHA: "x", Error: "both", BuiltAt: t0()}); err == nil {
		t.Error("an attempt with both a result and an error was stored")
	}
	got, err := s.Snapshots(ctx, "frozen")
	if err != nil || len(got) != 2 || got[0].Error != "check failed" || got[1].ResultSHA != "res" ||
		got[0].Checks != `[{"name":"one"}]` || got[1].Checks != "" {
		t.Fatalf("snapshots = %+v, %v; want the failure first, the good build kept", got, err)
	}
	for _, stmt := range []string{`UPDATE compiled_snapshots SET error = ''`, `DELETE FROM input_manifests`,
		`UPDATE snapshot_checks SET checks = '[]'`} {
		if _, err := s.pools.Write().Exec(stmt); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Errorf("%s: err = %v, want append-only abort", stmt, err)
		}
	}
}

func TestFetchHealthSaysWhenEachURLLastAnsweredAndFailed(t *testing.T) {
	t.Parallel()
	s, _ := importStore(t)
	ctx := context.Background()
	const u = "https://beta.example.com/2026/export?TYPE=rosters"
	if err := s.Record(ctx, fetchOf(t, u, "", "ok", t0())); err != nil {
		t.Fatal(err)
	}
	for _, f := range []archive.Fetch{
		{URL: u, Status: 503, FetchedAt: t0().Add(time.Hour)},
		{URL: u, Err: "dial tcp: timeout", FetchedAt: t0().Add(2 * time.Hour)},
	} {
		if err := s.Record(ctx, f); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.FetchHealth(ctx, "beta")
	if err != nil || len(got) != 1 {
		t.Fatalf("health = %+v, %v", got, err)
	}
	h := got[0]
	if !h.LastSuccess.Equal(t0()) || !h.LastFailure.Equal(t0().Add(2*time.Hour)) || h.LastError != "dial tcp: timeout" {
		t.Errorf("health = %+v, want good at t0 and the timeout last", h)
	}
}
