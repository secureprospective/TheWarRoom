package platform

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestPhotoIsGoWordedAndOffMeansNoAddress(t *testing.T) {
	t.Parallel()
	on := Photo("0501", "Kayden McDonald", false)
	if on.Src != "/headshot/0501" || on.Alt != "Photo of Kayden McDonald" {
		t.Fatalf("on = %+v", on)
	}
	if off := Photo("0501", "Kayden McDonald", true); off.Src != "" || off.Alt != "Kayden McDonald" {
		t.Fatalf("off = %+v", off)
	}
}

func TestDevBuildClassification(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		version string
		want    bool
	}{{"dev", true}, {"", false}, {"v1.0.0", false}, {"development", false}} {
		if got := IsDevBuild(tc.version); got != tc.want {
			t.Errorf("%q: %v", tc.version, got)
		}
	}
}

func TestEntryNeverOpensDesktopForUnknownArguments(t *testing.T) {
	service := EntryPoints{}
	for _, args := range [][]string{{"--version"}, {"-probe", "extra"}, {"-native", "missing"}, {"-restore"}, {"-package-smoke"}} {
		var out, errOut bytes.Buffer
		if code := service.Run(args, &out, &errOut); code != 2 || out.Len() != 0 || errOut.Len() == 0 {
			t.Fatal(args, code, out.String(), errOut.String())
		}
	}
}

func TestInstanceLock_SecondAcquireFails(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "thewarroom.db")

	first, err := AcquireInstanceLock(dbPath)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	if second, err := AcquireInstanceLock(dbPath); err == nil {
		ReleaseInstanceLock(second)
		t.Fatal("second acquire succeeded while the lock was held — instance guard failed")
	}

	ReleaseInstanceLock(first)

	third, err := AcquireInstanceLock(dbPath)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	ReleaseInstanceLock(third)
}

func TestInstanceLock_DevAndRealDoNotContend(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	realLock, err := AcquireInstanceLock(filepath.Join(dir, "thewarroom.db"))
	if err != nil {
		t.Fatalf("acquire real: %v", err)
	}
	defer ReleaseInstanceLock(realLock)

	dev, err := AcquireInstanceLock(filepath.Join(dir, "thewarroom-dev.db"))
	if err != nil {
		t.Fatalf("dev build must not contend with the real ledger lock: %v", err)
	}
	ReleaseInstanceLock(dev)
}
