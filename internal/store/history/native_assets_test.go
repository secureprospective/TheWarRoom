package history

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	t.Parallel()
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
	t.Parallel()
	path := filepath.Join(t.TempDir(), "durable.db")
	p, err := db.Open(t.Context(), path, SchemaVersion())
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
	p, err = db.Open(t.Context(), path, SchemaVersion())
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
	t.Parallel()
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

func TestEmptySeedAdmissionBySource(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ preset, source, want string }{
		{commissioner.PresetEmpty, commissioner.SourceNative, ""},
		{commissioner.PresetLegacy, commissioner.SourceMFLImport, "hold the Legacy NFL rosters, franchises and both draft reports before seeding"},
	} {
		t.Run(tc.preset, func(t *testing.T) {
			s, _, _ := newStore(t)
			if _, err := s.ExecuteCommissioner(t.Context(), createWith(tc.preset, 12), "creator", nil); err != nil {
				t.Fatal(err)
			}
			records, err := s.nativeRecords(t.Context(), "native.test")
			if err != nil || commissioner.LeagueSource(records) != tc.source || records[0].Command.Args.Source != tc.source {
				t.Fatalf("stored source: %v", err)
			}
			seed := heldSeedCommand(t)
			seed.Args.Native.Franchises = seed.Args.Native.Franchises[:12]
			seed.Args.Native.Players, seed.Args.Native.Picks, seed.Args.Native.Pool, seed.Args.Native.Adjustments = nil, nil, nil, nil
			before := historyJSON(t, s.pools.Read(), `SELECT json_group_array(payload) FROM league_event`)
			_, err = s.ExecuteCommissioner(t.Context(), seed, "creator", nil)
			if tc.want != "" {
				var refusal *commissioner.Refusal
				if !errors.As(err, &refusal) || refusal.Error() != tc.want {
					t.Fatalf("empty import seed error=%v, want %q", err, tc.want)
				}
				if after := historyJSON(t, s.pools.Read(), `SELECT json_group_array(payload) FROM league_event`); after != before {
					t.Fatal("refused seed changed the ledger")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assets, err := s.NativeAssets(t.Context(), seed.League)
			if err != nil || !assets.Seeded || len(assets.Snapshot.Players) != 0 || assets.Revision != 2 {
				t.Fatalf("empty native start not recorded: %+v %v", assets, err)
			}
		})
	}
}

func TestNativeCreationRollsBackWhenStartFails(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	if _, err := s.pools.Write().ExecContext(t.Context(), `CREATE TRIGGER fail_start BEFORE INSERT ON league_event WHEN NEW.kind='commish.league.seed_held' BEGIN SELECT RAISE(ABORT,'start unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	raw := createCommand()
	if _, err := s.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil, nil, time.UTC); err == nil || !strings.Contains(err.Error(), "start unavailable") {
		t.Fatalf("start failure lost: %v", err)
	}
	events, err := s.Events(t.Context(), raw.League, commissioner.Stream, 0)
	if err != nil || len(events) != 0 {
		t.Fatalf("partial league after failed start: %+v %v", events, err)
	}
	if _, err := s.pools.Write().ExecContext(t.Context(), `DROP TRIGGER fail_start`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil, nil, time.UTC); err != nil {
		t.Fatal(err)
	}
	assets, err := s.NativeAssets(t.Context(), raw.League)
	if err != nil || !assets.Seeded || assets.Revision != 2 {
		t.Fatalf("retry after rollback: %+v %v", assets, err)
	}
}
