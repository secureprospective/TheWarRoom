package platform

import (
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
