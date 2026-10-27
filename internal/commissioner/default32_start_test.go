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
