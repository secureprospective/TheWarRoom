package commissioner

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestDefault32StartRefusesAssignedOrIncompleteCards(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	raw, err := NativeSeedCommand(b.records, "native.test", "start", "Start", 1, seasonMoment(2))
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
	if _, err := NativeStart("native.test", legacy.records, seasonMoment(2)); err == nil {
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
				b.set("lineup.positions", `{"OL":{"min":1,"max":1}}`, "now", seasonMoment(2))
				unit, err := json.Marshal(leaguerules.Default32().Scoring.LineUnit.Value)
				if err != nil {
					t.Fatal(err)
				}
				b.set("scoring.lineUnit", string(unit), "now", seasonMoment(2))
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

func TestNativeStartPoolsFollowSettings(t *testing.T) {
	baseline := newLedger(t, PresetDefault32)
	want, err := NativeStart("native.test", baseline.records, seasonMoment(3))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		setup     func(*ledgerBuilder)
		lots      int
		same      bool
		errorText string
	}{
		{"empty preset OL on", func(b *ledgerBuilder) {
			b.set("lineup.positions", `{"OL":{"min":1,"max":1}}`, "now", seasonMoment(2))
			b.set("roster.positionBounds", `{"OL":{"min":1,"max":1}}`, "now", seasonMoment(2))
			unit, e := json.Marshal(leaguerules.Default32().Scoring.LineUnit.Value)
			if e != nil {
				t.Fatal(e)
			}
			b.set("scoring.lineUnit", string(unit), "now", seasonMoment(2))
		}, 32, true, ""},
		{"off", func(b *ledgerBuilder) {
			b.set("lineup.positions", `{}`, "now", seasonMoment(2))
			b.set("roster.positionBounds", `{}`, "now", seasonMoment(2))
			b.set("scoring.lineUnit", `{"Used":false,"Weights":{},"Labels":{}}`, "now", seasonMoment(2))
		}, 0, false, ""},
		// Lots follow the roster bound alone; a half-changed O-Line setting must stay editable.
		{"roster bound only", func(b *ledgerBuilder) {
			b.set("lineup.positions", `{}`, "now", seasonMoment(2))
			b.set("roster.positionBounds", `{"OL":{"min":1,"max":1}}`, "now", seasonMoment(2))
			b.set("scoring.lineUnit", `{"Used":false,"Weights":{},"Labels":{}}`, "now", seasonMoment(2))
		}, 32, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := newLedger(t, PresetEmpty)
			tc.setup(b)
			got, err := NativeStart("native.test", b.records, seasonMoment(3))
			if tc.errorText != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errorText) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil || len(got.Pool) != tc.lots {
				t.Fatalf("pool %d: %v", len(got.Pool), err)
			}
			if tc.same && !reflect.DeepEqual(got.Pool, want.Pool) {
				t.Fatal("pool differs from Default32")
			}
		})
	}
}

func TestNativeStartTwelveTeamDefaultRules(t *testing.T) {
	create, err := (RawCommand{League: "native.test", Command: Create,
		Args: RawArgs{Name: "Twelve", Preset: PresetEmpty, FranchiseCount: 12, Season: 2026}}).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	b := &ledgerBuilder{t: t}
	b.add(create, seasonMoment(1))
	b.set("lineup.positions", `{"OL":{"min":1,"max":1}}`, "now", seasonMoment(2))
	b.set("roster.positionBounds", `{"OL":{"min":1,"max":1}}`, "now", seasonMoment(2))
	unit, err := json.Marshal(leaguerules.Default32().Scoring.LineUnit.Value)
	if err != nil {
		t.Fatal(err)
	}
	b.set("scoring.lineUnit", string(unit), "now", seasonMoment(2))
	seed, err := NativeStart("native.test", b.records, seasonMoment(3))
	if err != nil || len(seed.Franchises) != 12 || len(seed.Pool) != 32 {
		t.Fatalf("start: %+v, %v", seed, err)
	}
	raw := RawCommand{Command: NativeSeed, Args: RawArgs{Native: &seed}}
	if err := CheckNativeAppend(raw, b.records); err != nil {
		t.Fatal(err)
	}
}

func TestNativeStartDefaultPresetWithOlineOff(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	b.set("lineup.positions", `{}`, "now", seasonMoment(2))
	b.set("roster.positionBounds", `{}`, "now", seasonMoment(2))
	b.set("scoring.lineUnit", `{"Used":false,"Weights":{},"Labels":{}}`, "now", seasonMoment(2))
	seed, err := NativeStart("native.test", b.records, seasonMoment(3))
	if err != nil || len(seed.Pool) != 0 {
		t.Fatalf("pool %d, error %v", len(seed.Pool), err)
	}
}

func TestNativeStartHistoricalTwelveTeamDefaultPreset(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	b.records[0].Command.Args.Franchises = b.records[0].Command.Args.Franchises[:12]
	b.records[0].Command.Args.FranchiseCount = 12
	seed, err := NativeStart("native.test", b.records, seasonMoment(3))
	if err != nil || len(seed.Franchises) != 12 || len(seed.Pool) != 32 {
		t.Fatalf("start: %+v, %v", seed, err)
	}
	if err := CheckNativeAppend(RawCommand{Command: NativeSeed, Args: RawArgs{Native: &seed}}, b.records); err != nil {
		t.Fatal(err)
	}
}
