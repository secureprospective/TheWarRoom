package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func testCertificates(t *testing.T) (string, string, string, *x509.CertPool) {
	t.Helper()
	now := time.Now()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Fixture test CA"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(2), NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage: x509.KeyUsageDigitalSignature}
	certDER, err := x509.CreateCertificate(rand.Reader, leaf, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certPath, keyPath, caPath := filepath.Join(dir, "server.pem"), filepath.Join(dir, "key.pem"), filepath.Join(dir, "ca.pem")
	for path, block := range map[string]*pem.Block{
		certPath: {Type: "CERTIFICATE", Bytes: certDER}, keyPath: {Type: "PRIVATE KEY", Bytes: keyDER}, caPath: {Type: "CERTIFICATE", Bytes: caDER},
	} {
		if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
			t.Fatal(err)
		}
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})) {
		t.Fatal("CA rejected")
	}
	return certPath, keyPath, caPath, roots
}

func startBinary(ctx context.Context, t *testing.T, binary, cert, key string) string {
	t.Helper()
	command := exec.CommandContext(ctx, binary, "-fixture", "-listen", "127.0.0.1:0", "-tls-cert", cert, "-tls-key", key)
	// An actual headless child process, not the desktop composition or a test HTTP server.
	command.Env = append(os.Environ(), "DISPLAY=", "WAYLAND_DISPLAY=")
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := command.Process.Signal(syscall.SIGTERM); err != nil {
			t.Error(err)
		}
		if err := command.Wait(); err != nil {
			t.Errorf("server exit: %v: %s", err, stderr.String())
		}
	})
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		t.Fatal("server did not report its address")
	}
	return scanner.Text()
}

func TestRealBinaryFixtureHTTPS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)
	binary := filepath.Join(t.TempDir(), "twr-server")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("headless build: %v: %s", err, output)
	}
	cert, key, ca, roots := testCertificates(t)
	origin := startBinary(ctx, t, binary, cert, key)
	client, err := serverclient.NewFixture(origin, roots)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	got, err := client.FixtureLedger(ctx)
	if err != nil {
		t.Fatal(err)
	}
	records, err := leagueservice.FixtureRecords()
	if err != nil {
		t.Fatal(err)
	}
	direct, err := commissioner.Reading("native.https-fixture", "fixture.observer", records, observer{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, direct) || len(got.Entries) != 1 || got.Entries[0].What == "" {
		t.Fatalf("HTTPS != engine: %+v / %+v", got, direct)
	}
	read := exec.CommandContext(ctx, binary, "-fixture", "-read", origin, "-ca", ca)
	output, err := read.CombinedOutput()
	if err != nil {
		t.Fatalf("local Go client: %v: %s", err, output)
	}
	wantJSON, err := json.Marshal(direct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(output), wantJSON) {
		t.Fatalf("client JSON mismatch: %s", output)
	}
	bad, err := serverclient.NewFixture(origin, x509.NewCertPool())
	if err != nil {
		t.Fatal(err)
	}
	defer bad.Close()
	if _, err := bad.FixtureLedger(ctx); err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("untrusted cert accepted: %v", err)
	}
	assertReadOnlyProtocol(ctx, t, origin, roots)
	after, err := client.FixtureLedger(ctx)
	if err != nil || !reflect.DeepEqual(after, got) {
		t.Fatalf("rejection changed fixture: %+v %v", after, err)
	}
}

func assertReadOnlyProtocol(ctx context.Context, t *testing.T, origin string, roots *x509.CertPool) {
	t.Helper()
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/v2/fixture/ledger", 426}, {"POST", serverhttp.FixturePath, 405},
		{"GET", serverhttp.FixturePath + "?principal=fixture.creator", 400},
	} {
		request, err := http.NewRequestWithContext(ctx, tc.method, origin+tc.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != tc.status {
			t.Fatalf("%s %s: HTTP %d", tc.method, tc.path, response.StatusCode)
		}
	}
}

func TestFixtureStartupRefusesUnsafeModes(t *testing.T) {
	for _, args := range [][]string{nil, {"-fixture", "unexpected"}} {
		if err := run(t.Context(), args, io.Discard); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	for _, address := range []string{"0.0.0.0:0", "[::]:0", "192.0.2.1:0", "localhost:0", ":0", "invalid"} {
		listener, err := serverhttp.ListenLoopback(t.Context(), address)
		if err == nil {
			if err := listener.Close(); err != nil {
				t.Error(err)
			}
			t.Fatalf("unsafe listen accepted: %s", address)
		}
	}
}

type observer struct{}

func (observer) Allow(_, _, _ string) (string, error) {
	return "", fmt.Errorf("fixture observer has no permissions")
}
