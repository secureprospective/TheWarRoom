package platform

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestEntryNeverOpensDesktopForUnknownArguments(t *testing.T) {
	service := EntryPoints{}
	for _, args := range [][]string{{"--version"}, {"-probe", "extra"}, {"-native", "missing"}, {"-restore"}, {"-package-smoke"}} {
		var out, errOut bytes.Buffer
		if code := service.Run(args, &out, &errOut); code != 2 || out.Len() != 0 || errOut.Len() == 0 {
			t.Fatal(args, code, out.String(), errOut.String())
		}
	}
}

func TestEntryDispatchAndProbeShutdown(t *testing.T) {
	called := ""
	service := EntryPoints{Desktop: func() int { called = "desktop"; return 0 }, Smoke: func(dir string) int { called = dir; return 3 }, Native: func(address, league string) int { called = address + league; return 4 }, Label: "fixture"}
	var out bytes.Buffer
	if code := service.Run(nil, &out, &out); code != 0 || called != "desktop" {
		t.Fatal(code, called)
	}
	if code := service.Run([]string{"-package-smoke", "directory"}, &out, &out); code != 3 || called != "directory" {
		t.Fatal(code, called)
	}
	if code := service.Run([]string{"-native", "address", "league"}, &out, &out); code != 4 || called != "addressleague" {
		t.Fatal(code, called)
	}
	if code := service.Run([]string{"-version"}, &out, &out); code != 0 || out.String() != "fixture\n" {
		t.Fatal(code, out.String())
	}
	for _, failure := range []error{nil, context.Canceled} {
		calls := []string{}
		code := Probe(func(context.Context) { calls = append(calls, "start") }, func(context.Context) { calls = append(calls, "shutdown") }, func() error { calls = append(calls, "check"); return failure })
		want := 0
		if failure != nil {
			want = 1
		}
		if code != want || !reflect.DeepEqual(calls, []string{"start", "check", "shutdown"}) {
			t.Fatal(code, calls)
		}
	}
}

func TestSmokeWindowRefusesBeforeRunAndReleasesLock(t *testing.T) {
	var stderr bytes.Buffer
	if code := SmokeWindow("relative", &stderr, func() error { t.Fatal("ran invalid smoke"); return nil }); code != 2 {
		t.Fatal(code)
	}
	directory := t.TempDir()
	code := SmokeWindow(directory, &stderr, func() error {
		if file, err := AcquireInstanceLock(filepath.Join(directory, "smoke.db")); err == nil {
			ReleaseInstanceLock(file)
			t.Fatal("window ran without lock")
		}
		return context.Canceled
	})
	if code != 1 {
		t.Fatal(code)
	}
	file, err := AcquireInstanceLock(filepath.Join(directory, "smoke.db"))
	if err != nil {
		t.Fatal("smoke failure retained lock", err)
	}
	ReleaseInstanceLock(file)
}

type nativeWindowFixture struct{ calls *[]string }

func (f nativeWindowFixture) Close()                     { *f.calls = append(*f.calls, "close") }
func (f nativeWindowFixture) SetHelperCommands([]string) { *f.calls = append(*f.calls, "commands") }
func (f nativeWindowFixture) ConfigureUpdater(string, string, func()) error {
	*f.calls = append(*f.calls, "updater")
	return context.Canceled
}
func (f nativeWindowFixture) PracticeUpdate() { *f.calls = append(*f.calls, "practice") }

func TestNativeWindowAdmissionAndUnsignedTrustDoesNotBlockWindow(t *testing.T) {
	var stderr bytes.Buffer
	service := NativeWindow{Open: func() (NativeDesktop, error) { return nil, context.Canceled }}
	if code := service.RunWindow(&stderr); code != 2 {
		t.Fatal(code)
	}
	calls := []string{}
	service.Open = func() (NativeDesktop, error) { return nativeWindowFixture{calls: &calls}, nil }
	service.Commands = func() []string { return nil }
	service.Directory = func() (string, error) { return t.TempDir(), nil }
	service.Lock = func(directory string) (*os.File, error) {
		calls = append(calls, "lock")
		return AcquireInstanceLock(filepath.Join(directory, "desktop"))
	}
	service.Version = "dev"
	service.Practice = func() string { return "app" }
	service.Run = func() error { calls = append(calls, "run"); return nil }
	if code := service.RunWindow(&stderr); code != 0 || !reflect.DeepEqual(calls, []string{"commands", "lock", "updater", "practice", "run", "close"}) {
		t.Fatal(code, calls)
	}
	calls = nil
	service.Lock = func(string) (*os.File, error) { calls = append(calls, "refuse lock"); return nil, context.Canceled }
	if code := service.RunWindow(&stderr); code != 2 || !reflect.DeepEqual(calls, []string{"commands", "refuse lock", "close"}) {
		t.Fatal(code, calls)
	}
}

func TestNativeUpdateLockRetryBoundAndSuccess(t *testing.T) {
	failure := errors.New("held")
	attempts, pauses := 0, 0
	acquire := func(string) (*os.File, error) { attempts++; return nil, failure }
	pause := func(duration time.Duration) {
		if duration != 100*time.Millisecond {
			t.Fatal(duration)
		}
		pauses++
	}
	if _, err := NativeUpdateLock("directory", acquire, pause); !errors.Is(err, failure) || attempts != 100 || pauses != 100 {
		t.Fatal(err, attempts, pauses)
	}
	attempts, pauses = 0, 0
	acquire = func(string) (*os.File, error) {
		attempts++
		if attempts < 3 {
			return nil, failure
		}
		return nil, nil
	}
	if _, err := NativeUpdateLock("directory", acquire, pause); err != nil || attempts != 3 || pauses != 2 {
		t.Fatal(err, attempts, pauses)
	}
}
