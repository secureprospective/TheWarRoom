package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func readDirectoryTest(t *testing.T, a *App) commissioner.NativeDirectory {
	t.Helper()
	directory, err := a.TargetNativeDirectory("native.seed")
	if err != nil {
		t.Fatal(err)
	}
	people, err := a.TargetCommissionerPeople("native.seed")
	if err != nil {
		t.Fatal(err)
	}
	seats := make([]commissioner.PeopleSeat, 0, len(directory.Franchises))
	for _, f := range directory.Franchises {
		seats = append(seats, f.PeopleSeat)
	}
	if !reflect.DeepEqual(people.Seats, seats) {
		t.Fatalf("people and league disagree: %+v / %+v", people.Seats, seats)
	}
	return directory
}

func directoryCommand(t *testing.T, a *App, command, request string, args commissioner.RawArgs,
	effective string) commissioner.Receipt {
	t.Helper()
	before := readDirectoryTest(t, a)
	return executeCommissionerTest(t, a, commissioner.RawCommand{League: before.League,
		Command: command, Args: args, Request: request, Expected: before.Revision,
		Reason: "Record our league choice", Effective: effective, Visibility: "league"})
}

func revertDirectoryTest(t *testing.T, a *App, receipt commissioner.Receipt) {
	t.Helper()
	directoryCommand(t, a, commissioner.Revert, "undo."+receipt.Entry,
		commissioner.RawArgs{Entry: receipt.Entry}, "now")
}

func checkHeldDirectory(t *testing.T, a *App) {
	t.Helper()
	d := readDirectoryTest(t, a)
	if d.RosterSeason != 2026 || len(d.Franchises) != 32 || d.Franchises[1].Name != "Held team 2" ||
		d.Franchises[1].Abbreviation != "H2" || d.Franchises[1].Division != "division.1" ||
		d.Franchises[1].Owner != "Vacant" {
		t.Fatalf("starting names, alignment or seats lost: %+v", d)
	}
}

func TestNativeDirectoryNamesAndAlignmentRoundTrip(t *testing.T) {
	t.Parallel()
	a := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	before := readDirectoryTest(t, a)
	if before.Franchises[1].Name != "Franchise 2" || before.Franchises[1].Division != "Needs a ruling" {
		t.Fatalf("unseeded league borrowed mirror identities: %+v", before)
	}
	seedNativeTest(t, a, input)
	checkHeldDirectory(t, a)
	rename := directoryCommand(t, a, "commish.franchise.edit", "rename",
		commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.02",
			Name: "Native club", Abbreviation: "NC"}}, "now")
	if d := readDirectoryTest(t, a); d.Franchises[1].Name != "Native club" || d.Franchises[1].Abbreviation != "NC" {
		t.Fatalf("seed overwrote later rename: %+v", d)
	}
	revertDirectoryTest(t, a, rename)
	checkHeldDirectory(t, a)
	alignment := directoryCommand(t, a, commissioner.RulesSet, "align",
		commissioner.RawArgs{Field: "league.alignment", Value: json.RawMessage(`{"franchise.02":"native.division"}`)}, "now")
	if d := readDirectoryTest(t, a); d.Franchises[1].Division != "native.division" ||
		d.Franchises[0].Division != "Needs a ruling" {
		t.Fatalf("native alignment ignored or guessed: %+v", d)
	}
	revertDirectoryTest(t, a, alignment)
	checkHeldDirectory(t, a)
	unset := directoryCommand(t, a, commissioner.RulesSet, "clear",
		commissioner.RawArgs{Field: "league.alignment", Value: json.RawMessage(`{}`)}, "now")
	if d := readDirectoryTest(t, a); d.Franchises[1].Division != "Needs a ruling" {
		t.Fatalf("cleared alignment silently borrowed seed alignment: %+v", d)
	}
	revertDirectoryTest(t, a, unset)
	checkHeldDirectory(t, a)
	pending := directoryCommand(t, a, commissioner.RulesSet, "pending",
		commissioner.RawArgs{Field: "league.alignment", Value: json.RawMessage(`{"franchise.02":"future"}`)},
		"next_rollover")
	checkHeldDirectory(t, a)
	revertDirectoryTest(t, a, pending)
	checkHeldDirectory(t, a)
}

func openDirectoryHistory(t *testing.T, path string) (*history.Store, *db.Pools) {
	t.Helper()
	p, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(p, reg)
	store.SetClock(func() time.Time { return time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC) })
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return store, p
}

func TestNativeDirectoryReopenWithoutMirror(t *testing.T) {
	t.Parallel()
	a := nativeSeedTestApp(t)
	path := filepath.Join(t.TempDir(), "directory.db")
	store, pools := openDirectoryHistory(t, path)
	a.core.History = store
	input := createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	seedNativeTest(t, a, input)
	owner := directoryCommand(t, a, "commish.seat.replace", "owner",
		commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.02", Principal: "native.owner"}}, "now")
	rename := directoryCommand(t, a, "commish.franchise.edit", "rename",
		commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.02",
			Name: "Native club", Abbreviation: "NC"}}, "now")
	alignment := directoryCommand(t, a, commissioner.RulesSet, "align",
		commissioner.RawArgs{Field: "league.alignment", Value: json.RawMessage(`{"franchise.02":"native.division"}`)}, "now")
	before := readDirectoryTest(t, a)
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	store, _ = openDirectoryHistory(t, path)
	restarted := &App{raceHeld: &raceapp.Held{}, core: &core.State{Ctx: t.Context(), History: store, Started: make(chan struct{}),
		CommissionerSession: testCommissionerSession{"creator"}}}
	close(restarted.core.Started)
	if after := readDirectoryTest(t, restarted); !reflect.DeepEqual(before, after) {
		t.Fatalf("directory changed after physical reopen: %+v / %+v", before, after)
	}
	revertDirectoryTest(t, restarted, alignment)
	revertDirectoryTest(t, restarted, rename)
	revertDirectoryTest(t, restarted, owner)
	checkHeldDirectory(t, restarted)
}

func TestNativeDirectoryWire(t *testing.T) {
	t.Parallel()
	a := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	seedNativeTest(t, a, input)
	body, err := json.Marshal(readDirectoryTest(t, a))
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "internal/commissioner/testdata/native-directory.wire.json", body, "native directory")
}
