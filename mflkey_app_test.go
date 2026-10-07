package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/zalando/go-keyring"
)

type keySink struct{ records []archive.Fetch }

func (s *keySink) Record(_ context.Context, f archive.Fetch) error {
	s.records = append(s.records, f)
	return nil
}

func hasKey(text, key string) bool { return strings.Contains(text, key) }

// An app that started with no client or store dereferences nil: the binding must recover, or
// Wails would log the raw IPC message carrying the key.
func TestSetMFLKeyPanicNeverReachesWails(t *testing.T) {
	app := NewApp()
	close(app.started)
	status, err := app.SetMFLKey("panic-probe-" + strings.Repeat("x", 16))
	if err != nil || status.State != "unavailable" {
		t.Fatalf("state %q, err %v", status.State, err)
	}
}

func TestMFLKeyLeakGate(t *testing.T) {
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	sentinel := "SENTINEL-" + hex.EncodeToString(random)
	t.Run("absence checker detects deliberate leak", func(t *testing.T) {
		if !hasKey("deliberate "+sentinel, sentinel) {
			t.Fatal("leak gate cannot fail")
		}
	})
	for _, scenario := range []string{"success", "error", "redirect", "retry", "refused", "echo"} {
		t.Run(scenario, func(t *testing.T) {
			keyring.MockInit()
			var logs bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&logs)
			defer log.SetOutput(previous)
			var mu sync.Mutex
			received, attempts, followed := 0, 0, 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.URL.Path == "/redirect-target" {
					followed++
					return
				}
				if r.URL.Query().Get("TYPE") == "league" {
					// Synthetic documented league discovery shape; no real authenticated fixture exists.
					_, _ = fmt.Fprint(w, `{"league":{"baseURL":"https://www47.myfantasyleague.com"}}`)
					return
				}
				if r.URL.Query().Get("APIKEY") != sentinel {
					t.Error("server did not receive candidate")
				}
				if r.Header.Get("Cookie") != "" {
					t.Error("keyed request sent cookie")
				}
				received++
				attempts++
				switch scenario {
				case "error":
					_, _ = fmt.Fprintf(w, `{"error":{"$t":"rejected %s"}}`, sentinel)
				case "redirect":
					http.Redirect(w, r, "/redirect-target", http.StatusFound)
				case "retry":
					if attempts == 1 {
						w.WriteHeader(http.StatusTooManyRequests)
						return
					}
					_, _ = fmt.Fprint(w, `{"pendingTrades":{}}`)
				case "echo":
					_, _ = fmt.Fprintf(w, `{"pendingTrades":{"echo":"%s"}}`, sentinel)
				default:
					// Synthetic documented empty-success object.
					_, _ = fmt.Fprint(w, `{"pendingTrades":{}}`)
				}
			}))
			defer server.Close()
			base := server.Client().Transport.(*http.Transport).Clone()
			defer base.CloseIdleConnections()
			base.TLSClientConfig.ServerName = "example.com"
			address := server.Listener.Addr().String()
			base.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, network, address)
			}
			sink := &keySink{}
			store := mflkey.New(ingestion.LeagueID, "2026")
			client, err := mfl.New("api", 10000,
				mfl.WithTransport(&archive.Transport{Base: base, Sink: sink}), mfl.WithKeySource(store.Get))
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			if err := client.DiscoverHost(ctx, "2026", ingestion.LeagueID); err != nil {
				t.Fatal(err)
			}
			if scenario == "refused" {
				server.Close()
			}
			app := NewApp()
			app.ctx, app.season, app.mflClient = ctx, 2026, client
			app.keyStore = store
			app.seasonChanged = func(context.Context) {}
			close(app.started)
			want := map[string]string{
				"success": "connected", "retry": "connected", "echo": "connected",
				"error": "rejected", "redirect": "rejected", "refused": "unreachable",
			}[scenario]
			good := want == "connected"
			status, setErr := app.SetMFLKey(sentinel)
			app.weekWorkers.Wait()
			if setErr != nil || status.State != want {
				t.Fatalf("verification: state %q, want %q, err %v", status.State, want, setErr)
			}
			if good && status.VerifiedAt == "" {
				t.Fatal("connected without a verification time")
			}
			assertAbsent := func(text string) {
				t.Helper()
				if hasKey(text, sentinel) {
					t.Fatal("credential leak detected")
				}
			}
			body, err := json.Marshal(status)
			if err != nil {
				t.Fatal(err)
			}
			assertAbsent(string(body))
			state, err := app.MFLKeyStatus()
			if err != nil {
				t.Fatal(err)
			}
			body, err = json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			assertAbsent(string(body))
			if err := app.keyStore.Set(ctx, mflkey.Key(sentinel)); err != nil {
				t.Fatal(err)
			}
			// Failed replacement must not remove a previously stored key.
			if !good {
				replaced, err := app.SetMFLKey(sentinel)
				if err != nil || replaced.State != want {
					t.Fatalf("replacement: state %q, err %v", replaced.State, err)
				}
				body, err := json.Marshal(replaced)
				if err != nil {
					t.Fatal(err)
				}
				assertAbsent(string(body))
				kept, err := app.keyStore.Get(ctx)
				if err != nil || kept != mflkey.Key(sentinel) {
					t.Fatal("stored key changed")
				}
			}
			deleted, err := app.DeleteMFLKey()
			app.weekWorkers.Wait()
			if err != nil || deleted.State != "absent" {
				t.Fatalf("delete: %v %v", deleted, err)
			}
			body, err = json.Marshal(deleted)
			if err != nil {
				t.Fatal(err)
			}
			assertAbsent(string(body))
			for _, record := range sink.records {
				if (scenario == "echo" || scenario == "error") && strings.Contains(record.URL, "pendingTrades") {
					if record.Err == "" || len(record.Gzip) > 0 {
						t.Fatal("echo body not suppressed")
					}
				}
				assertAbsent(record.URL)
				assertAbsent(record.Err)
				if len(record.Gzip) > 0 {
					plain, err := archive.Gunzip(record.Gzip)
					if err != nil {
						t.Fatal(err)
					}
					assertAbsent(string(plain))
				}
			}
			assertAbsent(logs.String())
			key := mflkey.Key(sentinel)
			assertAbsent(fmt.Sprintf("%v %+v %#v %s", key, key, key, key))
			mu.Lock()
			defer mu.Unlock()
			if scenario != "refused" && received == 0 {
				t.Fatal("no authenticated request observed")
			}
			if followed != 0 {
				t.Fatal("redirect followed")
			}
			if scenario == "retry" && attempts != 3 {
				t.Fatalf("attempts: %d", attempts)
			}
		})
	}
}

type seasonKeyFailure struct {
	key      string
	attempts int
}

func (s *seasonKeyFailure) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Query().Get("TYPE") == "league" {
		return seasonResponse(`{"league":{"baseURL":"https://www47.myfantasyleague.com"}}`), nil
	}
	s.attempts++
	response := seasonResponse(`{"error":{"$t":"rejected ` + s.key + `"}}`)
	if s.attempts > 1 {
		response.StatusCode = http.StatusInternalServerError
	}
	return response, nil
}

func TestMFLKeySeasonLeakGate(t *testing.T) {
	keyring.MockInit()
	sentinel := "SENTINEL-season-error-and-500"
	store := mflkey.New(ingestion.LeagueID, "2026")
	if err := store.Set(context.Background(), mflkey.Key(sentinel)); err != nil {
		t.Fatal(err)
	}
	transport := &seasonKeyFailure{key: sentinel}
	client, err := mfl.New("api", 10000, mfl.WithTransport(transport), mfl.WithKeySource(store.Get))
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.ctx, app.season, app.mflClient, app.keyStore = context.Background(), 2026, client, store
	app.seasonChanged = func(context.Context) {}
	close(app.started)
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	for range 2 {
		app.refreshPendingTradesInBackground()
		app.weekWorkers.Wait()
		reading, err := app.TargetSeason()
		if err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal(reading)
		if err != nil {
			t.Fatal(err)
		}
		if hasKey(string(body), sentinel) || hasKey(logs.String(), sentinel) {
			t.Fatal("season credential leak")
		}
		if reading.PendingTrades.Provenance.Freshness.State != FreshFail ||
			reading.PendingTrades.Provenance.Freshness.Note == "" {
			t.Fatal("failure not reported")
		}
	}
	if transport.attempts != 2 {
		t.Fatalf("requests %d, want error envelope then 500", transport.attempts)
	}
}
