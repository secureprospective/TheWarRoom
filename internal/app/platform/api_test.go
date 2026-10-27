package platform

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type fakeHost struct {
	names  map[string]string
	off    bool
	intent string
}

func (h *fakeHost) Snapshot() (snapshot.Snapshot, error) { return snapshot.Snapshot{}, nil }
func (h *fakeHost) Clock() (snapshot.Sourced[leagueclock.Reading], error) {
	return snapshot.Sourced[leagueclock.Reading]{}, nil
}
func (h *fakeHost) DraftRosterMove(intent, _, _ string) (envelope.Receipt, error) {
	h.intent = intent
	return envelope.Receipt{}, nil
}
func (h *fakeHost) Moves(string) ([]envelope.Receipt, error)        { return nil, nil }
func (h *fakeHost) HandOff(string) (envelope.Receipt, error)        { return envelope.Receipt{}, nil }
func (h *fakeHost) CheckMoves() error                               { return nil }
func (h *fakeHost) WithdrawRoster(string) (envelope.Receipt, error) { return envelope.Receipt{}, nil }
func (h *fakeHost) PlayerName(id string) (string, bool)             { n, ok := h.names[id]; return n, ok }
func (h *fakeHost) PhotosOff() bool                                 { return h.off }

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

func TestTargetPhotoNamesAnUnknownPlayerPlainly(t *testing.T) {
	t.Parallel()
	api := New(&fakeHost{names: map[string]string{}})
	if got := api.TargetPhoto("9999"); got.Alt != "Photo of this player" {
		t.Fatalf("TargetPhoto = %+v", got)
	}
	api = New(&fakeHost{names: map[string]string{}, off: true})
	if got := api.TargetPhoto("9999"); got.Src != "" {
		t.Fatalf("photos off still gave an address: %+v", got)
	}
}

func TestDraftActsNameTheirIntent(t *testing.T) {
	t.Parallel()
	host := &fakeHost{}
	api := New(host)
	if _, _ = api.TargetDraftIR("0001", "13130"); host.intent != "roster.ir" {
		t.Fatalf("TargetDraftIR intent = %q", host.intent)
	}
	if _, _ = api.TargetDraftTaxi("0001", "13130"); host.intent != "roster.taxi" {
		t.Fatalf("TargetDraftTaxi intent = %q", host.intent)
	}
}

func TestPhotoRuntimeInstallsBeforeSettingsAndReadsIDsLazily(t *testing.T) {
	available := false
	calls := []string{}
	service := PhotoRuntime{Available: func() bool { return available },
		SourceID: func(_ context.Context, id, source string) (string, error) {
			calls = append(calls, source)
			return id + source, nil
		},
		Install:  func(func(context.Context, string) (string, string, error)) { calls = append(calls, "install") },
		Settings: func() (bool, error) { calls = append(calls, "settings"); return true, nil }, SetOff: func(off bool) {
			if !off {
				t.Fatal("lost switch")
			}
			calls = append(calls, "off")
		},
	}
	service.Initialize()
	if !reflect.DeepEqual(calls, []string{"install", "settings", "off"}) {
		t.Fatal(calls)
	}
	calls = nil
	if _, _, err := service.IDs(context.Background(), "0099"); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	available = true
	espn, sleeper, err := service.IDs(context.Background(), "0099")
	if err != nil || espn != "0099espn" || sleeper != "0099sleeper" || !reflect.DeepEqual(calls, []string{"espn", "sleeper"}) {
		t.Fatal(espn, sleeper, err, calls)
	}
	calls = nil
	service.Settings = func() (bool, error) { return false, context.Canceled }
	service.Initialize()
	if !reflect.DeepEqual(calls, []string{"install"}) {
		t.Fatal("failed settings overwrote switch", calls)
	}
	service.SourceID = func(context.Context, string, string) (string, error) { return "", context.Canceled }
	if _, _, err := service.IDs(context.Background(), "0099"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
