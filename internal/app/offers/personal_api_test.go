package offers

import (
	"context"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

type personalHost struct {
	*fakeHost
	player snapshot.Player
	opened string
}

func (h *personalHost) OffersSnapshotNames() ([]snapshot.Player, []snapshot.Franchise, bool) {
	return []snapshot.Player{h.player}, []snapshot.Franchise{{ID: "0001", Name: "My franchise"}}, true
}
func (h *personalHost) OffersTradeBait(context.Context) report.HeldFeed[[]ingestion.RawTradeBait] {
	return report.HeldFeed[[]ingestion.RawTradeBait]{FetchedAt: time.Now(), Value: []ingestion.RawTradeBait{{Franchise: "0001", Give: h.player.ID.String(), Want: "Picks"}}}
}
func (h *personalHost) OffersInjuries() (snapshot.Provenance, map[string]string) {
	return snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshLive}}, map[string]string{}
}
func (h *personalHost) OffersRosterTarget(option string) (envelope.Target, []string) {
	return envelope.Target{URL: "https://fixture.invalid/?O=" + option}, nil
}
func (h *personalHost) OpenBrowser(target string) { h.opened = target }

func TestPersonalAPIReadsOwnListsAndKeepsHandOffReadOnly(t *testing.T) {
	base := bidTestApp(t)
	id, err := playerid.New("13593")
	if err != nil {
		t.Fatal(err)
	}
	h := &personalHost{fakeHost: base.core, player: snapshot.Player{ID: id, Name: "Smith"}}
	a := New(h)
	if err := personal.New(h.Pools).Initialize(h.Ctx); err != nil {
		t.Fatal(err)
	}
	change := personal.RawChange{Mine: "0001", Kind: "watch", Entity: "player", ID: id.String(), Action: "add"}
	if err := a.TargetPersonalChange(change); err != nil {
		t.Fatal(err)
	}
	reading, err := a.TargetPersonal("0001")
	if err != nil || len(reading.Entries) != 1 || len(reading.Players) != 1 || len(reading.Bait.Rows) != 1 {
		t.Fatalf("own list: %+v %v", reading, err)
	}
	if reading.Bait.Rows[0].Cells[0] != "My franchise" || reading.Bait.Rows[0].Cells[1] != "Smith" {
		t.Fatal(reading.Bait.Rows)
	}
	change.Mine = "0002"
	if err := a.TargetPersonalChange(change); err == nil {
		t.Fatal("unknown franchise accepted")
	}
	change.Mine, change.ID = "0001", "99999"
	if err := a.TargetPersonalChange(change); err == nil {
		t.Fatal("unknown entity accepted")
	}
	if err := a.TargetPersonalHandOff("watch"); err != nil || h.opened != "https://fixture.invalid/?O=178" {
		t.Fatal(h.opened, err)
	}
	if err := a.TargetPersonalHandOff("submit"); err == nil {
		t.Fatal("non-hand-off accepted")
	}
}

func TestPersonalNotesThreadIsPerPlayerAndFranchise(t *testing.T) {
	base := bidTestApp(t)
	id, err := playerid.New("13593")
	if err != nil {
		t.Fatal(err)
	}
	h := &personalHost{fakeHost: base.core, player: snapshot.Player{ID: id, Name: "Smith"}}
	a := New(h)
	store := personal.New(h.Pools)
	if err := store.Initialize(h.Ctx); err != nil {
		t.Fatal(err)
	}
	t0 := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	for i, text := range []string{"older", "newer"} {
		change := personal.RawChange{Mine: "0001", Kind: "note", Entity: "player", ID: id.String(), Action: "add", Text: text}
		if err := store.Change(h.Ctx, change, t0.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	thread, err := a.TargetPersonalNotes("0001", id.String())
	if err != nil || thread.Count != 2 || len(thread.Notes) != 2 ||
		thread.Notes[0].Text != "newer" || thread.Notes[1].Text != "older" {
		t.Fatalf("own thread: %+v %v", thread, err)
	}
	if _, err := a.TargetPersonalNotes("0002", id.String()); err == nil {
		t.Fatal("another franchise read the thread")
	}
	change := personal.RawChange{Mine: "0001", Kind: "note", Entity: "player", ID: id.String(),
		Action: "remove", At: t0.Format(time.RFC3339Nano)}
	if err := a.TargetPersonalChange(change); err != nil {
		t.Fatal(err)
	}
	thread, err = a.TargetPersonalNotes("0001", id.String())
	if err != nil || thread.Count != 1 || thread.Notes[0].Text != "newer" {
		t.Fatalf("delete one note: %+v %v", thread, err)
	}
}
