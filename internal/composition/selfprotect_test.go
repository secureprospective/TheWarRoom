package composition

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	"github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/app/comms"
	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/app/counterparties"
	"github.com/secureprospective/TheWarRoom/internal/app/draft"
	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/app/offers"
	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/research"
	"github.com/secureprospective/TheWarRoom/internal/app/sources"
	"github.com/secureprospective/TheWarRoom/internal/app/trades"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

func TestPackageSmokeRejectsInvalidDirectoryAndHeldLock(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	for _, path := range []string{"relative", filepath.Join(directory, "missing")} {
		var stderr bytes.Buffer
		if code := packageSmoke(path, Embedded{}, &stderr); code != 2 {
			t.Fatalf("invalid directory returned %d: %s", code, &stderr)
		}
	}
	lock, err := acquireInstanceLock(filepath.Join(directory, "smoke.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer releaseInstanceLock(lock)
	var stderr bytes.Buffer
	if code := packageSmoke(directory, Embedded{}, &stderr); code != 1 || !strings.Contains(stderr.String(), "another instance") {
		t.Fatalf("second smoke returned %d: %s", code, &stderr)
	}
	if _, err := os.Stat(filepath.Join(directory, "ready")); !os.IsNotExist(err) {
		t.Fatalf("rejected smoke wrote readiness: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "smoke.db")); !os.IsNotExist(err) {
		t.Fatalf("rejected smoke opened a database: %v", err)
	}
}

// TestInstanceLock_SecondAcquireFails proves the single-instance guard: once the
// lock is held, a second acquire on the same DB path is refused; after release a
// fresh acquire succeeds.
func TestInstanceLock_SecondAcquireFails(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "thewarroom.db")

	first, err := acquireInstanceLock(dbPath)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	if second, err := acquireInstanceLock(dbPath); err == nil {
		releaseInstanceLock(second)
		t.Fatal("second acquire succeeded while the lock was held — instance guard failed")
	}

	releaseInstanceLock(first)

	third, err := acquireInstanceLock(dbPath)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	releaseInstanceLock(third)
}

// TestInstanceLock_DevAndRealDoNotContend proves the lock is keyed to the DB path,
// so a dev build and a real build (different DB files) never falsely block.
func TestInstanceLock_DevAndRealDoNotContend(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	realLock, err := acquireInstanceLock(filepath.Join(dir, "thewarroom.db"))
	if err != nil {
		t.Fatalf("acquire real: %v", err)
	}
	defer releaseInstanceLock(realLock)

	dev, err := acquireInstanceLock(filepath.Join(dir, "thewarroom-dev.db"))
	if err != nil {
		t.Fatalf("dev build must not contend with the real ledger lock: %v", err)
	}
	releaseInstanceLock(dev)
}

// TestSetupLogging_PrunesToMax proves disk logging creates the logs dir and keeps
// at most maxLogFiles files.
func TestSetupLogging_PrunesToMax(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		t.Fatalf("mkdir logs: %v", err)
	}

	// Seed more than the cap of uniquely-named stale log files, then prune.
	for i := 0; i < maxLogFiles+5; i++ {
		name := filepath.Join(logDir, fmt.Sprintf("thewarroom-2020%06dT000000Z.log", i))
		if err := os.WriteFile(name, []byte("stub"), 0o600); err != nil {
			t.Fatalf("seed stub: %v", err)
		}
	}
	pruneLogs(logDir)

	matches, err := filepath.Glob(filepath.Join(logDir, "thewarroom-*.log"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) > maxLogFiles {
		t.Fatalf("pruneLogs kept %d files, want <= %d", len(matches), maxLogFiles)
	}
}

// TestRunFlag_UnknownArgumentNeverOpensAWindow: an unknown argument exits 2 with usage instead
// of falling through to a window on the real database; -version prints the build label.
func TestRunFlag_UnknownArgumentNeverOpensAWindow(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"-version", "x"}, {"--version"}, {"-probe", "-version"}, {"run"}} {
		var out, errOut bytes.Buffer
		if code := runFlag(args, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "usage") {
			t.Errorf("runFlag(%q) = %d, stderr %q; want 2 with usage", args, code, errOut.String())
		}
	}
	var out, errOut bytes.Buffer
	if code := runFlag([]string{"-version"}, &out, &errOut); code != 0 || strings.TrimSpace(out.String()) != buildLabel() {
		t.Errorf("-version = %d, stdout %q; want 0 and %q", code, out.String(), buildLabel())
	}
}

func TestNativeStartupRefusesUnsafeOriginBeforeWindow(t *testing.T) {
	var stderr bytes.Buffer
	if code := runNative("https://user:sentinel@localhost", "native.test", Embedded{}, &stderr); code != 2 {
		t.Fatalf("unsafe native startup = %d", code)
	}
	if strings.Contains(stderr.String(), "sentinel") || !strings.Contains(stderr.String(), "HTTPS origin") {
		t.Fatalf("unsafe startup diagnostic: %q", stderr.String())
	}
	// A missing native argument must not fall through to local app startup.
	stderr.Reset()
	var stdout bytes.Buffer
	if code := runFlag([]string{"-native", "https://localhost"}, &stdout, &stderr); code != 2 {
		t.Fatalf("incomplete native startup = %d", code)
	}
}

// Wails reflects the pointer method set without a signature filter; no exemptions are configured.
func TestBindingManifest(t *testing.T) {
	t.Parallel()
	got, problems := bindingManifest(bindings(NewApp()))
	if len(problems) > 0 {
		t.Fatal(strings.Join(problems, "\n"))
	}
	slices.Sort(got)
	want, err := os.ReadFile("testdata/bindings/manifest.txt")
	if err != nil {
		t.Fatal(err)
	}
	if actual := strings.Join(got, "\n") + "\n"; actual != string(want) {
		t.Fatalf("binding manifest changed; review the surface before updating the golden\ngot:\n%s", actual)
	}
}

func bindingManifest(bound []interface{}) ([]string, []string) {
	catalogs := boundCatalogs()
	seen := map[reflect.Type]bool{}
	var got, problems []string
	for _, value := range bound {
		typ := reflect.TypeOf(value)
		commands, feature := catalogs[typ]
		if !feature {
			problems = append(problems, fmt.Sprintf("unexpected bound type %v", typ))
			continue
		}
		if seen[typ] {
			problems = append(problems, fmt.Sprintf("duplicate bound type %v", typ))
		}
		seen[typ] = true
		methods := map[string]bool{}
		for _, command := range commands {
			if command.Role == catalog.Unset || command.Gravity == catalog.Unset {
				problems = append(problems, "catalog method "+command.Method+" has unset role or gravity")
			}
			if methods[command.Method] {
				problems = append(problems, "duplicate catalog method "+command.Method)
			}
			methods[command.Method] = true
		}
		prefix := typ.Elem().String() + "."
		for i := range typ.NumMethod() {
			name := typ.Method(i).Name
			got = append(got, prefix+name)
			if feature {
				if !methods[name] {
					problems = append(problems, "bound method "+prefix+name+" has no catalog entry")
				}
				delete(methods, name)
			}
		}
		for name := range methods {
			problems = append(problems, "catalog method "+name+" is not bound")
		}
	}
	for typ := range catalogs {
		if !seen[typ] {
			problems = append(problems, fmt.Sprintf("missing bound type %v", typ))
		}
	}
	return got, problems
}

func TestBindingManifestRejectsHeld(t *testing.T) {
	for _, held := range []interface{}{&research.Held{}, &sources.Held{}} {
		bound := append(bindings(NewApp()), held)
		_, problems := bindingManifest(bound)
		want := fmt.Sprintf("unexpected bound type %T", held)
		if len(problems) != 1 || problems[0] != want {
			t.Fatalf("directly bound holder accepted: %v", problems)
		}
	}
}

// boundCatalogs is each bound API's command catalog, by its type.
func boundCatalogs() map[reflect.Type][]catalog.Command {
	return map[reflect.Type][]catalog.Command{
		reflect.TypeFor[*archiveapp.API]():     archiveapp.Commands(),
		reflect.TypeFor[*commissioner.API]():   commissioner.Commands(),
		reflect.TypeFor[*comms.API]():          comms.Commands(),
		reflect.TypeFor[*controlapp.API]():     controlapp.Commands(),
		reflect.TypeFor[*contractsapp.API]():   contractsapp.Commands(),
		reflect.TypeFor[*counterparties.API](): counterparties.Commands(),
		reflect.TypeFor[*draft.API]():          draft.Commands(),
		reflect.TypeFor[*marketapp.API]():      marketapp.Commands(),
		reflect.TypeFor[*nowapp.API]():         nowapp.Commands(),
		reflect.TypeFor[*offers.API]():         offers.Commands(),
		reflect.TypeFor[*platformapp.API]():    platformapp.Commands(),
		reflect.TypeFor[*raceapp.API]():        raceapp.Commands(),
		reflect.TypeFor[*research.API]():       research.Commands(),
		reflect.TypeFor[*sources.API]():        sources.Commands(),
		reflect.TypeFor[*trades.API]():         trades.Commands(),
	}
}
