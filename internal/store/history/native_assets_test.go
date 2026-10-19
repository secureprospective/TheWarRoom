package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

func heldSeedCommand(t *testing.T) commissioner.RawCommand {
	t.Helper()
	body, err := os.ReadFile("../../commissioner/testdata/native-assets.wire.json")
	if err != nil {
		t.Fatal(err)
	}
	var wire commissioner.NativeAssets
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Snapshot == nil {
		t.Fatal("wire has no starting assets")
	}
	return commissioner.RawCommand{League: "native.test", Command: commissioner.NativeSeed,
		Args: commissioner.RawArgs{Native: wire.Snapshot}, Expected: 1, Request: "seed",
		Reason: "Record the held starting state", Effective: "now", Visibility: "league"}
}

func TestNativeSeedAtomicConcurrentAndReplay(t *testing.T) {
	s, _, p := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
		t.Fatal(err)
	}
	command := heldSeedCommand(t)
	var wait sync.WaitGroup
	receipts := make(chan commissioner.Receipt, 6)
	for range 6 {
		wait.Go(func() {
			receipt, err := s.ExecuteCommissioner(t.Context(), command, "creator", nil)
			if err != nil {
				t.Error(err)
				return
			}
			receipts <- receipt
		})
	}
	wait.Wait()
	close(receipts)
	var first commissioner.Receipt
	for receipt := range receipts {
		if first.Entry != "" && receipt != first {
			t.Fatalf("concurrent seed retries returned different receipts: %+v / %+v", receipt, first)
		}
		first = receipt
	}
	assets, err := s.NativeAssets(t.Context(), command.League)
	if err != nil || assets.Revision != 2 || assets.Entry != first.Entry || !reflect.DeepEqual(assets.Snapshot, command.Args.Native) {
		t.Fatalf("starting state is not exactly one ledger append: %+v %v", assets, err)
	}
	restarted, _, _ := reopen(t, p, registry(t, ""))
	again, err := restarted.NativeAssets(t.Context(), command.League)
	if err != nil || !reflect.DeepEqual(assets, again) {
		t.Fatalf("restart/replay lost starting assets: %+v %v", again, err)
	}
	if receipt, err := restarted.ExecuteCommissioner(t.Context(), command, "creator", nil); err != nil || receipt != first {
		t.Fatalf("restart lost retry identity: %+v %v", receipt, err)
	}
	if _, err := restarted.ExecuteCommissioner(t.Context(), reverseCommand(first.Entry, 2, "undo-seed"), "creator", nil); err == nil {
		t.Fatal("starting basis was reversed instead of requiring an individual correction")
	}
}

func TestNativeSeedSurvivesDatabaseCloseAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "durable.db")
	p, err := db.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	s, _, _ := reopen(t, p, registry(t, ""))
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
		t.Fatal(err)
	}
	command := heldSeedCommand(t)
	first, err := s.ExecuteCommissioner(t.Context(), command, "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	p, err = db.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	s, _, _ = reopen(t, p, registry(t, ""))
	assets, err := s.NativeAssets(t.Context(), command.League)
	if err != nil || assets.Entry != first.Entry || !reflect.DeepEqual(assets.Snapshot, command.Args.Native) {
		t.Fatalf("disk replay did not preserve starting state: %+v %v", assets, err)
	}
	if receipt, err := s.ExecuteCommissioner(t.Context(), command, "creator", nil); err != nil || receipt != first {
		t.Fatalf("disk replay did not preserve the original receipt: %+v %v", receipt, err)
	}
}

func TestNativeSeedValidationLeavesNoPartialAct(t *testing.T) {
	for name, breakInput := range map[string]func(*commissioner.RawNativeSnapshot){
		"no roster date":      func(s *commissioner.RawNativeSnapshot) { s.RosterAt = time.Time{} },
		"no draft date":       func(s *commissioner.RawNativeSnapshot) { s.DraftAt = time.Time{} },
		"no future date":      func(s *commissioner.RawNativeSnapshot) { s.FutureAt = time.Time{} },
		"no roster":           func(s *commissioner.RawNativeSnapshot) { s.Players = nil },
		"no franchises":       func(s *commissioner.RawNativeSnapshot) { s.Franchises = nil },
		"wrong franchise":     func(s *commissioner.RawNativeSnapshot) { s.Franchises[0].ID = "other" },
		"duplicate franchise": func(s *commissioner.RawNativeSnapshot) { s.Franchises[1] = s.Franchises[0] },
		"duplicate player":    func(s *commissioner.RawNativeSnapshot) { s.Players = append(s.Players, s.Players[0]) },
		"player owner":        func(s *commissioner.RawNativeSnapshot) { s.Players[0].Franchise = "unknown" },
		"player status":       func(s *commissioner.RawNativeSnapshot) { s.Players[0].Status = "unknown" },
		"negative salary":     func(s *commissioner.RawNativeSnapshot) { s.Players[0].Salary = -1 },
		"duplicate pick":      func(s *commissioner.RawNativeSnapshot) { s.Picks = append(s.Picks, s.Picks[0]) },
		"unknown pick owner":  func(s *commissioner.RawNativeSnapshot) { s.Picks[0].Owner = "unknown" },
		"unknown original":    func(s *commissioner.RawNativeSnapshot) { s.Picks[2].Original = "unknown" },
		"duplicate charge":    func(s *commissioner.RawNativeSnapshot) { s.Adjustments = append(s.Adjustments, s.Adjustments[0]) },
	} {
		t.Run(name, func(t *testing.T) {
			s, _, _ := newStore(t)
			if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
				t.Fatal(err)
			}
			command := heldSeedCommand(t)
			breakInput(command.Args.Native)
			if _, err := s.ExecuteCommissioner(t.Context(), command, "creator", nil); err == nil {
				t.Fatal("invalid starting snapshot accepted")
			}
			assets, err := s.NativeAssets(t.Context(), command.League)
			if err != nil || assets.Seeded || assets.Revision != 1 || assets.Snapshot != nil {
				t.Fatalf("invalid seed partially appended: %+v %v", assets, err)
			}
		})
	}
}
