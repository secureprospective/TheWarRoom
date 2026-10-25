package offsite

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type fakeS3 struct {
	mu                        sync.Mutex
	objects                   map[string][]byte
	corrupt, missing, failPut bool
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") == "" {
		http.Error(w, "signature missing", http.StatusForbidden)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/league/")
	switch r.Method {
	case http.MethodPut:
		if f.failPut {
			http.Error(w, "offline", http.StatusServiceUnavailable)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read", 400)
			return
		}
		f.objects[key] = body
		w.Header().Set("ETag", `"0123456789abcdef0123456789abcdef"`)
	case http.MethodGet:
		if r.URL.Query().Get("list-type") == "2" {
			type content struct {
				Key  string
				Size int
			}
			result := struct {
				XMLName     xml.Name `xml:"ListBucketResult"`
				Name        string
				Prefix      string
				KeyCount    int
				MaxKeys     int
				IsTruncated bool
				Contents    []content
			}{Name: "league", Prefix: "sets/", MaxKeys: 1000}
			for k, v := range f.objects {
				if strings.HasPrefix(k, "sets/") {
					result.Contents = append(result.Contents, content{k, len(v)})
				}
			}
			sort.Slice(result.Contents, func(i, j int) bool { return result.Contents[i].Key < result.Contents[j].Key })
			result.KeyCount = len(result.Contents)
			w.Header().Set("Content-Type", "application/xml")
			_ = xml.NewEncoder(w).Encode(result)
			return
		}
		body, ok := f.objects[key]
		if !ok || f.missing {
			w.WriteHeader(404)
			_, _ = io.WriteString(w, `<Error><Code>NoSuchKey</Code></Error>`)
			return
		}
		body = bytes.Clone(body)
		if f.corrupt && len(body) > 0 {
			body[len(body)/2] ^= 1
		}
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		_, _ = w.Write(body)
	case http.MethodDelete:
		delete(f.objects, key)
		w.WriteHeader(204)
	default:
		http.Error(w, "unsupported", 400)
	}
}
func testR2(t *testing.T) (*R2, *fakeS3) {
	t.Helper()
	fake := &fakeS3{objects: map[string][]byte{}}
	server := httptest.NewTLSServer(fake)
	t.Cleanup(server.Close)
	client, err := minio.New(strings.TrimPrefix(server.URL, "https://"), &minio.Options{Secure: true, Region: "auto", Creds: credentials.NewStaticV4(strings.Repeat("a", 32), strings.Repeat("b", 64), ""), BucketLookup: minio.BucketLookupPath, Transport: server.Client().Transport})
	if err != nil {
		t.Fatal(err)
	}
	return &R2{client, "league"}, fake
}

func testCapture(t *testing.T) (*db.Pools, string, func(context.Context, string) ([]byte, error)) {
	t.Helper()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	ledger := history.New(pools, &measures.Registry{})
	if err := ledger.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	accounts := accountstore.New(pools)
	if err := accounts.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := pools.Write().ExecContext(t.Context(), `CREATE TABLE backup_evidence(body TEXT); INSERT INTO backup_evidence VALUES ('pinned evidence sentinel')`); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.New(accounts).LocalBootstrap(t.Context(), identity.RawCredentials{Username: "backup.owner", Password: "private-recovery-sentinel"}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, body := range map[string]string{"authenticator.key": strings.Repeat("k", 32), "origin.key": "TLS private sentinel", "origin.pem": "TLS certificate", "endpoint-registry.csv": "registry", "backup.json": "R2 credential sentinel"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	checkpoint := func(ctx context.Context, path string) ([]byte, error) {
		snapshot, err := db.Open(ctx, path, history.SchemaVersion())
		if err != nil {
			return nil, err
		}
		defer func() { _ = snapshot.Close() }()
		ledger := history.New(snapshot, &measures.Registry{})
		account := accountstore.New(snapshot)
		if _, err := ledger.VerifyAudit(ctx); err != nil {
			return nil, err
		}
		if _, err := account.VerifyAudit(ctx); err != nil {
			return nil, err
		}
		return []byte(`{"version":1,"verified":true}`), nil
	}
	return pools, dir, checkpoint
}

func TestEncryptUploadVerifyRestoreAfterHostLoss(t *testing.T) {
	pools, config, checkpoint := testCapture(t)
	// This identity belongs only to the restore side. Capture receives the public half.
	offlineKey, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	r2, fake := testR2(t)
	stage := t.TempDir()
	path, at, err := Capture(t.Context(), pools, config, stage, offlineKey.Recipient().String(), checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	name := "sets/" + at.Format(objectFormat) + ".tar.gz.age"
	if err := r2.upload(t.Context(), name, path); err != nil {
		t.Fatal(err)
	}
	if err := r2.verify(t.Context(), name, path); err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	encrypted := bytes.Clone(fake.objects[name])
	fake.mu.Unlock()
	for _, secret := range []string{"pinned evidence sentinel", "TLS private sentinel", "R2 credential sentinel", offlineKey.String()} {
		if bytes.Contains(encrypted, []byte(secret)) {
			t.Fatalf("R2 saw plaintext %q", secret)
		}
	}
	if err := os.RemoveAll(stage); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(config); err != nil {
		t.Fatal(err)
	}
	var originalPath string
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&originalPath); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Dir(originalPath)); err != nil {
		t.Fatal(err)
	}
	// Only the off-host object and the offline identity survive this point.
	dest := filepath.Join(t.TempDir(), "restore")
	m, err := Restore(bytes.NewReader(encrypted), offlineKey, dest)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Captured.Equal(at) {
		t.Fatal("capture age changed")
	}
	restored, err := db.Open(t.Context(), filepath.Join(dest, "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = restored.Close() }()
	var evidence string
	if err := restored.Read().QueryRowContext(t.Context(), `SELECT body FROM backup_evidence`).Scan(&evidence); err != nil || evidence != "pinned evidence sentinel" {
		t.Fatalf("lost evidence: %s %v", evidence, err)
	}
	if _, err := history.New(restored, &measures.Registry{}).VerifyAudit(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := accountstore.New(restored).Credential(t.Context(), "backup.owner"); err != nil {
		t.Fatal("lost restored account:", err)
	}
	if _, err := accountstore.New(restored).VerifyAudit(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"server.db", "checkpoint.json", "config/authenticator.key", "config/origin.key", "config/backup.json"} {
		info, err := os.Stat(filepath.Join(dest, name))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("restore mode %s: %v", name, err)
		}
	}
	for _, bad := range []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"tamper", func(b []byte) []byte { b[len(b)/2] ^= 1; return b }},
		{"missing tail", func(b []byte) []byte { return b[:len(b)-1] }},
	} {
		t.Run(bad.name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "bad")
			if _, err := Restore(bytes.NewReader(bad.mutate(bytes.Clone(encrypted))), offlineKey, target); err == nil {
				t.Fatal("damaged set restored")
			}
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Fatal("partial restore retained")
			}
		})
	}
}

func TestMissingTamperedAndFailedUploadNeverRenewCoverage(t *testing.T) {
	r2, fake := testR2(t)
	clock := time.Now().UTC()
	capture := func(_ context.Context, stage string) (string, time.Time, error) {
		path := filepath.Join(stage, "set.age")
		return path, clock, os.WriteFile(path, []byte("encrypted fixture"), 0600)
	}
	manager := NewManager(r2, t.TempDir(), capture)
	manager.now = func() time.Time { return clock }
	if manager.CheckWrite() == nil {
		t.Fatal("new process trusted coverage")
	}
	if err := manager.Backup(t.Context()); err != nil {
		t.Fatal(err)
	}
	original := manager.Status().Captured
	if manager.CheckWrite() != nil {
		t.Fatal("verified upload did not open writes")
	}
	for _, mode := range []string{"missing", "tamper", "upload"} {
		t.Run(mode, func(t *testing.T) {
			fake.mu.Lock()
			fake.missing = mode == "missing"
			fake.corrupt = mode == "tamper"
			fake.failPut = mode == "upload"
			fake.mu.Unlock()
			clock = clock.Add(time.Minute)
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			if err := manager.Backup(ctx); err == nil {
				t.Fatal("bad remote set verified")
			}
			if status := manager.Status(); !status.Captured.Equal(original) || status.Failure == "" {
				t.Fatalf("failure renewed coverage or left no reason for the dashboard: %+v", status)
			}
		})
	}
	clock = original.Add(14*time.Minute - time.Nanosecond)
	if manager.CheckWrite() != nil {
		t.Fatal("closed before margin")
	}
	clock = original.Add(14 * time.Minute)
	if manager.CheckWrite() == nil {
		t.Fatal("coverage breach admitted")
	}
	clock = original.Add(15 * time.Minute)
	if manager.CheckWrite() == nil {
		t.Fatal("RPO breach admitted")
	}
	clock = original.Add(-time.Second)
	if manager.CheckWrite() == nil {
		t.Fatal("backward clock admitted")
	}
	restarted := NewManager(r2, t.TempDir(), capture)
	if restarted.CheckWrite() == nil {
		t.Fatal("restart trusted old disk receipt")
	}
	fake.mu.Lock()
	fake.failPut = false
	fake.mu.Unlock()
	if err := manager.Backup(t.Context()); err != nil || manager.Status().Failure != "" {
		t.Fatalf("verified upload left a stale failure on the dashboard: %v %+v", err, manager.Status())
	}
}

func TestRetentionPrunesOnlyExpiredRecoveryObjects(t *testing.T) {
	r2, fake := testR2(t)
	now := time.Now().UTC()
	for day := 0; day < 40; day++ {
		for minute := 0; minute < 15; minute++ {
			at := now.Add(-time.Duration(day)*24*time.Hour - time.Duration(minute)*time.Minute)
			fake.objects["sets/"+at.Format(objectFormat)+".tar.gz.age"] = []byte("encrypted")
		}
	}
	fake.objects["unrelated"] = []byte("keep")
	fake.objects["sets/foreign"] = []byte("keep")
	if err := r2.prune(t.Context(), now); err != nil {
		t.Fatal(err)
	}
	if len(fake.objects) > 44 || len(fake.objects) < 31 {
		t.Fatalf("retained %d objects", len(fake.objects))
	}
	if _, ok := fake.objects["unrelated"]; !ok {
		t.Fatal("foreign object pruned")
	}
	if _, ok := fake.objects["sets/foreign"]; !ok {
		t.Fatal("foreign prefix object pruned")
	}
	for minute := 0; minute < 12; minute++ {
		name := "sets/" + now.Add(-time.Duration(minute)*time.Minute).Format(objectFormat) + ".tar.gz.age"
		if _, ok := fake.objects[name]; !ok {
			t.Fatalf("recent set removed %s", name)
		}
	}
	for name := range fake.objects {
		if at, err := objectTime(name); err == nil && at.Before(now.Add(-30*24*time.Hour)) {
			t.Fatal("expired set retained")
		}
	}
}

func TestSpareHostListsNewestSetAndNamesWrongKey(t *testing.T) {
	pools, config, checkpoint := testCapture(t)
	offlineKey, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	r2, fake := testR2(t)
	path, at, err := Capture(t.Context(), pools, config, t.TempDir(), offlineKey.Recipient().String(), checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	newest := "sets/" + at.Format(objectFormat) + ".tar.gz.age"
	older := "sets/" + at.Add(-5*time.Minute).Format(objectFormat) + ".tar.gz.age"
	for _, name := range []string{older, newest} {
		if err := r2.upload(t.Context(), name, path); err != nil {
			t.Fatal(err)
		}
	}
	fake.objects["sets/zz-foreign"] = []byte("keep")
	sets, err := r2.Sets(t.Context())
	if err != nil || len(sets) != 2 || sets[0] != newest || sets[1] != older {
		t.Fatalf("sets %v: %v", sets, err)
	}
	if _, err := r2.Open(t.Context(), "sets/zz-foreign"); err == nil {
		t.Fatal("opened a foreign object")
	}
	wrong, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[*age.X25519Identity]error{wrong: ErrRestoreKey, offlineKey: nil} {
		body, err := r2.Open(t.Context(), newest)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Restore(body, key, filepath.Join(t.TempDir(), "restore"))
		_ = body.Close()
		if !errors.Is(err, want) {
			t.Fatalf("restore with key: got %v want %v", err, want)
		}
	}
}
