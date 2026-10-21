package offers

import (
	"context"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
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
