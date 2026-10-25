package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/serverstatus"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// The sentinel stands for private detail a backup error can carry, such as a bucket name.
const privateSentinel = "bucket warroom-private-SENTINEL unreachable"

type dialLog struct {
	mu    sync.Mutex
	calls []string
}

func (d *dialLog) dial(ctx context.Context, network, address string) (net.Conn, error) {
	d.mu.Lock()
	d.calls = append(d.calls, network+" "+address)
	d.mu.Unlock()
	return (&net.Dialer{}).DialContext(ctx, network, address)
}

// Every injected fault shows on the console dashboard; the Tunnel-facing handler shows none
// of it, and the dashboard dials nothing outside this container.
func TestFaultsShowOnPrivateDashboardOnly(t *testing.T) {
	dir := t.TempDir()
	pools, err := db.Open(t.Context(), filepath.Join(dir, "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	handler, err := hostedServices(t.Context(), pools, make([]byte, 32), "")
	if err != nil {
		t.Fatal(err)
	}
	// Faults: the off-site backup fails, the league history chain no longer verifies.
	manager := offsite.NewManager(nil, dir, func(context.Context, string) (string, time.Time, error) {
		return "", time.Time{}, errors.New(privateSentinel)
	})
	if err := manager.Backup(t.Context()); err == nil {
		t.Fatal("injected backup failure succeeded")
	}
	if _, err := pools.Write().ExecContext(t.Context(), `UPDATE audit_chain_head SET hash='changed' WHERE source='league_event'`); err != nil {
		t.Fatal(err)
	}
	handler.WriteAdmission = manager.CheckWrite
	handler.TrustedProxy = netip.MustParseAddr("127.0.0.1")

	socket := filepath.Join(dir, "status.sock")
	stopStatus, err := startStatusSocket(t.Context(), socket, statusSource(pools, handler.Store, func() (offsite.Status, bool) { return manager.Status(), true }))
	if err != nil {
		t.Fatal(err)
	}
	defer stopStatus()
	certPath, keyPath, _, roots := testCertificates(t)
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, stopPublic := context.WithCancel(t.Context())
	listener, err := serverhttp.ListenLoopback(ctx, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	served := make(chan error, 1)
	go func() { served <- serverhttp.ServeHosted(ctx, listener, certificate, handler) }()
	// Fault: cloudflared is up but has no edge connection.
	tunnel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"status":503,"readyConnections":0}`)
	}))
	defer tunnel.Close()

	dials := &dialLog{}
	console := serverstatus.Console{Socket: socket, Origin: listener.Addr().String(), OriginCA: roots,
		Tunnel: strings.TrimPrefix(tunnel.URL, "http://"), State: dir, Dial: dials.dial, Now: time.Now,
		Space: func(string) (uint64, uint64, error) { return 200e6, 4e9, nil },          // fault: disk nearly full
		Clock: func() error { return errors.New("host clock is not NTP synchronized") }} // fault: clock
	dashboard := func() string {
		t.Helper()
		checks, err := console.Checks(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if _, err := serverstatus.Render(&out, time.Now(), checks); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	shown := dashboard()
	for _, want := range []string{
		"PROBLEM  Backups: League changes and sign-ins are paused", privateSentinel,
		"PROBLEM  History: League history does not check out",
		"PROBLEM  Tunnel: Not connected to Cloudflare",
		"PROBLEM  Clock: Not synchronised (host clock is not NTP synchronized)",
		"PROBLEM  Disk: Only 0.2 GB free of 4.0 GB",
		// The test certificate lives an hour: that fault shows too, while the server answers.
		"PROBLEM  League server: Answering, but its private certificate for the Tunnel expires",
		"What to do: Change nothing and call Christopher today.",
		"6 problems.",
	} {
		if !strings.Contains(shown, want) {
			t.Errorf("dashboard lacks %q:\n%s", want, shown)
		}
	}

	// The public side: status-looking routes do not exist, and a paused write explains
	// itself in fixed words. None of the injected detail crosses the Tunnel.
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}}
	defer client.CloseIdleConnections()
	public := func(method, path string) (int, string) {
		t.Helper()
		request, err := http.NewRequestWithContext(t.Context(), method, "https://"+listener.Addr().String()+path, strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("CF-Connecting-IP", "203.0.113.9")
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, string(body)
	}
	for _, path := range []string{serverstatus.StatusPath, "/status", "/healthz", "/metrics", "/ready", "/"} {
		if code, body := public(http.MethodGet, path); code != http.StatusNotFound || body != "404 page not found\n" {
			t.Errorf("public %s answered %d %q", path, code, body)
		}
	}
	if code, body := public(http.MethodPost, serverhttp.LoginPath); code != http.StatusServiceUnavailable || body != manager.CheckWrite().Error()+"\n" ||
		strings.Contains(body, "SENTINEL") || strings.Contains(body, "history") || strings.Contains(body, "clock") {
		t.Errorf("paused write leaked or changed: %d %q", code, body)
	}

	// The league server stops: the dashboard says so instead of going blank.
	stopPublic()
	if err := <-served; err != nil {
		t.Fatal(err)
	}
	stopStatus()
	shown = dashboard()
	for _, want := range []string{"PROBLEM  League server: Not answering on this machine", "PROBLEM  Backups and history: Could not be read"} {
		if !strings.Contains(shown, want) {
			t.Errorf("stopped server not shown: lacks %q:\n%s", want, shown)
		}
	}
	// No outbound notification: only the private socket and loopback were ever dialled.
	for _, call := range dials.calls {
		if call != "unix "+socket && !strings.HasPrefix(call, "tcp 127.0.0.1:") {
			t.Errorf("dashboard dialled outside the container: %s", call)
		}
	}
	if len(dials.calls) == 0 {
		t.Fatal("dial log empty")
	}
	console.Tunnel = "1.1.1.1:443"
	if _, err := console.Checks(t.Context()); err == nil {
		t.Fatal("dashboard accepted an outside address")
	}
}
