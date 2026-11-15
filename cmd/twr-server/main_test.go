package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

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
