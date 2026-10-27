package sources

import (
	"errors"
	"testing"
)

func TestCompileAvailableRefusesBusyAndReleasesAfterFailure(t *testing.T) {
	if result, err := CompileAvailable(func() bool { return false }, func() { t.Fatal("unlock without lock") }, func() (string, error) { t.Fatal("busy compile"); return "", nil }); err != nil || result != "The checked facts are already compiling." {
		t.Fatal(result, err)
	}
	locked := false
	failure := errors.New("bad source")
	result, err := CompileAvailable(func() bool { locked = true; return true }, func() { locked = false }, func() (string, error) {
		if !locked {
			t.Fatal("compile without lock")
		}
		return "partial", failure
	})
	if result != "partial" || !errors.Is(err, failure) || locked {
		t.Fatal(result, err, locked)
	}
}
