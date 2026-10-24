package serverhttp_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func fixtureStore(t *testing.T, path string) (*history.Store, *db.Pools) {
	t.Helper()
	pools, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	store := history.New(pools, &measures.Registry{})
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return store, pools
}

func fixtureClient(t *testing.T, store *history.Store) *serverclient.Client {
	t.Helper()
	certSource := httptest.NewTLSServer(nil)
	certificate := certSource.TLS.Certificates[0]
	roots := x509.NewCertPool()
	roots.AddCert(certSource.Certificate())
	certSource.Close()
	listener, err := serverhttp.ListenLoopback(t.Context(), "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	service := leagueservice.NewFixtureChanges(store)
	go func() { done <- serverhttp.ServeFixtureChanges(ctx, listener, tls.Certificate(certificate), service) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	client, err := serverclient.NewFixture("https://"+listener.Addr().String(), roots)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	return client
}

func seedFixture(t *testing.T, store *history.Store) {
	t.Helper()
	records, err := leagueservice.FixtureRecords()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ExecuteCommissioner(t.Context(), records[0].Command, "fixture.creator", nil); err != nil {
		t.Fatal(err)
	}
}

func note(request string, revision int64) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.https-fixture", Command: commissioner.NotePost,
		Args:   commissioner.RawArgs{Note: &commissioner.RawNote{Text: "Fixture change"}},
		Reason: "Exercise HTTPS append", Expected: revision, Request: request, Effective: "now", Visibility: "league"}
}

func TestHTTPSFixtureWriteReadReopenAndConcurrentRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	var finalToken int64
	t.Run("write", func(t *testing.T) {
		store, _ := fixtureStore(t, path)
		seedFixture(t, store)
		client := fixtureClient(t, store)
		at := time.Date(2026, 10, 24, 12, 0, 0, 123456789, time.UTC)
		store.SetClock(func() time.Time { return at })
		initial, err := client.FixtureLedger(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := client.FixtureChange(t.Context(), note("first", initial.Revision))
		if err != nil || !receipt.RecordedAt.Equal(at) {
			t.Fatalf("write: %+v %v", receipt, err)
		}
		retry, err := client.FixtureChange(t.Context(), note("first", initial.Revision))
		if err != nil || retry.Entry != receipt.Entry || retry.Revision != receipt.Revision {
			t.Fatalf("retry: %+v %v", retry, err)
		}
		var wg sync.WaitGroup
		results := make(chan error, 2)
		for i := range 2 {
			wg.Go(func() {
				_, err := client.FixtureChange(t.Context(), note(fmt.Sprintf("race.%d", i), receipt.Revision))
				results <- err
			})
		}
		wg.Wait()
		close(results)
		wins := 0
		for err := range results {
			if err == nil {
				wins++
			}
		}
		if wins != 1 {
			t.Fatalf("winners=%d", wins)
		}
		ledger, err := client.FixtureLedger(t.Context())
		if err != nil || len(ledger.Entries) != 3 || ledger.Revision == receipt.Revision {
			t.Fatalf("read: %+v %v", ledger, err)
		}
		finalToken = ledger.Revision
	})
	// The prior server and pools are closed: this is a cold disk reopen.
	reopened, _ := fixtureStore(t, path)
	after, err := fixtureClient(t, reopened).FixtureLedger(t.Context())
	if err != nil || after.Revision != finalToken || len(after.Entries) != 3 {
		t.Fatalf("reopen: %+v %v", after, err)
	}
}
