package main

import (
	"path/filepath"
	"testing"
)

func TestTrustedProxyFlagAcceptsOnlyLoopback(t *testing.T) {
	if hop, err := trustedProxy(""); err != nil || hop.IsValid() {
		t.Fatalf("default trusts a proxy: %v %v", hop, err)
	}
	if hop, err := trustedProxy("127.0.0.1"); err != nil || hop.String() != "127.0.0.1" {
		t.Fatalf("loopback hop refused: %v %v", hop, err)
	}
	for _, bad := range []string{"192.0.2.9", "0.0.0.0", "localhost", "fe80::1%lo"} {
		if _, err := trustedProxy(bad); err == nil {
			t.Errorf("%q accepted as a trusted hop", bad)
		}
	}
}

func TestHostedBackupRequiresActualRuntimeSettings(t *testing.T) {
	dir := t.TempDir()
	files := []string{filepath.Join(dir, "authenticator.key"), filepath.Join(dir, "origin.pem"), filepath.Join(dir, "origin.key"), filepath.Join(dir, "endpoint-registry.csv")}
	if err := checkBackupFiles(dir, files); err != nil {
		t.Fatal(err)
	}
	files[0] = filepath.Join(t.TempDir(), "authenticator.key")
	if err := checkBackupFiles(dir, files); err == nil {
		t.Fatal("backup omitted the running authenticator key")
	}
}
