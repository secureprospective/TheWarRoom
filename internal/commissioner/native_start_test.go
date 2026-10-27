package commissioner

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestDefault32StartRefusesAssignedOrIncompleteCards(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	raw, err := Default32SeedCommand(b.records, "native.test", "start", "Start", 1, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(s *RawNativeSnapshot){
		"assigned card": func(s *RawNativeSnapshot) {
			s.Players = []NativePlayer{{ID: balCard, Franchise: "franchise.01", Status: domain.RosterActive, ExpirationYear: 2026}}
		},
		"missing lot":  func(s *RawNativeSnapshot) { s.Pool = s.Pool[:31] },
		"duplicate":    func(s *RawNativeSnapshot) { s.Pool[1] = s.Pool[0] },
		"not card":     func(s *RawNativeSnapshot) { s.Pool[0].Player = "1000" },
		"wrong season": func(s *RawNativeSnapshot) { s.Season = 2027 },
	} {
		snap := *raw.Args.Native
		snap.Pool = append([]NativePoolEntry{}, snap.Pool...)
		change(&snap)
		bad := raw
		bad.Args.Native = &snap
		if err := CheckNativeAppend(bad, b.records); err == nil {
			t.Errorf("%s admitted", name)
		}
	}
	legacy := newLedger(t, PresetLegacy)
	if _, err := Default32Start("native.test", legacy.records, seasonMoment(2)); err == nil {
		t.Fatal("Legacy accepted card start")
	}
}

func TestLeagueSourceCompatibility(t *testing.T) {
	for _, tc := range []struct{ preset, stored, want string }{
		{PresetLegacy, "", SourceMFLImport}, {PresetEmpty, "", SourceNative},
		{PresetDefault32, "", SourceNative}, {"", "", SourceNative},
		{PresetLegacy, SourceNative, SourceNative}, {PresetEmpty, SourceMFLImport, SourceMFLImport},
		{PresetDefault32, SourceMFLImport, SourceMFLImport},
	} {
		t.Run(tc.preset+"/"+tc.stored, func(t *testing.T) {
			records := []Record{{Command: RawCommand{Command: Create, Args: RawArgs{Preset: tc.preset, Source: tc.stored}}}}
			if got := LeagueSource(records); got != tc.want {
				t.Fatalf("source=%q, want %q", got, tc.want)
			}
		})
	}
	if LeagueSource(nil) != SourceNative {
		t.Fatal("no records must default to native")
	}
}

func TestNativeStartsFollowSourceAndOLBounds(t *testing.T) {
	for _, tc := range []struct {
		name, preset, source string
		bounded              bool
		lots                 int
	}{
		{"empty", PresetEmpty, SourceNative, false, 0},
		{"old empty", PresetEmpty, "", false, 0},
		{"empty with OL", PresetEmpty, SourceNative, true, 32},
		{"native Legacy rules", PresetLegacy, SourceNative, false, 0},
		{"default", PresetDefault32, SourceNative, false, 32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := newLedger(t, tc.preset)
			b.records[0].Command.Args.Source = tc.source
			if tc.bounded {
				b.set("roster.positionBounds", `{"OL":{"min":1,"max":1}}`, "now", seasonMoment(2))
			}
			seed, err := NativeStart("native.test", b.records, seasonMoment(3))
			if err != nil || len(seed.Pool) != tc.lots || len(seed.Players) != 0 || len(seed.Picks) != 0 {
				t.Fatalf("start=%+v, error=%v", seed, err)
			}
			raw := RawCommand{Command: NativeSeed, Args: RawArgs{Native: &seed}}
			if err := raw.Args.Validate(NativeSeed); err != nil {
				t.Fatal(err)
			}
			if err := CheckNativeAppend(raw, b.records); err != nil {
				t.Fatal(err)
			}
			if tc.lots != 0 {
				b.add(raw, seasonMoment(3))
				if !StartingCardLot(b.records, seed.Pool[0].Player) {
					t.Fatal("native OL lots did not open the starting auction")
				}
			}
		})
	}
}

func TestNativeStartRefusesHeldAssetsOutsideRecovery(t *testing.T) {
	for _, preset := range []string{PresetEmpty, PresetDefault32} {
		t.Run(preset, func(t *testing.T) {
			b := newLedger(t, preset)
			seed := *ledgerSnapshot()
			seed.Franchises = seed.Franchises[:len(b.records[0].Command.Args.Franchises)]
			seed.Picks = nil
			raw := RawCommand{Command: NativeSeed, Args: RawArgs{Native: &seed}, Request: "recovery-seed"}
			if err := raw.Args.Validate(NativeSeed); err != nil {
				t.Fatal(err)
			}
			if err := CheckNativeAppend(raw, b.records); err == nil {
				t.Fatal("normal command admitted played rosters using a recovery request name")
			}
			if err := CheckNativeRecoveryAppend(raw, b.records); err != nil {
				t.Fatal(err)
			}
		})
	}
}
