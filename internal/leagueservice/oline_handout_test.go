package leagueservice

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestDueWorkerHandsOutUnsoldOLineCardsAfterClose(t *testing.T) {
	f := &dueFixture{
		league: dueLeague, clock: &dueTestClock{},
		path:  filepath.Join(t.TempDir(), "history.db"),
		facts: nativecheck.Facts{"0099": {Position: "QB"}},
	}
	nativecheck.AddLineCards(f.facts)
	f.clock.instant.Store(time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC).UnixNano())
	f.open(t)
	f.execute(t, "creator", commissioner.Create, commissioner.RawArgs{
		Name: "Handout league", Preset: commissioner.PresetLegacy,
		Source: commissioner.SourceMFLImport, Season: 2026, FranchiseCount: 4,
	})
	at := f.clock.Now()
	seed := &commissioner.RawNativeSnapshot{
		Season: 2026, RosterAt: at, DraftAt: at, FutureAt: at,
		Players: []commissioner.NativePlayer{{
			ID: "0099", Franchise: "franchise.01", Name: "Quarterback",
			Status: domain.RosterActive, Salary: 10_000_000, ExpirationYear: 2026,
			ContractStatus: domain.CStatusUFA,
		}},
	}
	for i := 1; i <= 4; i++ {
		id := fmt.Sprintf("franchise.%02d", i)
		seed.Franchises = append(seed.Franchises, commissioner.NativeFranchise{
			ID: id, HeldID: id, Name: fmt.Sprintf("Team %d", i),
		})
	}
	for _, team := range []string{"ARI", "ATL", "BAL"} {
		id, _ := nativecheck.LineCardID(team)
		seed.Pool = append(seed.Pool, commissioner.NativePoolEntry{
			Player: id, Name: team + " O-Line", Season: 2026,
		})
	}
	f.execute(t, "creator", commissioner.NativeSeed, commissioner.RawArgs{Native: seed})
	for _, r := range []struct{ field, value string }{
		{"cap.amount", `200000000`},
		{"roster.limits", `{"min":0,"max":100}`},
		{"roster.positionBounds", `{"OL":{"min":1,"max":1}}`},
		{"calendar.timezone", `"UTC"`},
		{"calendar.boundary", `"start_of_day"`},
		{"calendar.windows",
			`{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid"]}}`},
	} {
		f.execute(t, "creator", commissioner.RulesSet,
			commissioner.RawArgs{Field: r.field, Value: json.RawMessage(r.value)})
	}
	f.execute(t, "creator", "commish.seat.replace", commissioner.RawArgs{
		People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: "owner.one"},
	})
	ari, _ := nativecheck.LineCardID("ARI")
	f.execute(t, "owner.one", commissioner.BidPlace, commissioner.RawArgs{
		Bid: &commissioner.RawBidArgs{
			Player: ari, Franchise: "franchise.01", Salary: 50_000_000, Years: 1,
		},
	})
	f.clock.instant.Add(int64(24 * time.Hour))
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	// While auction is open, only the seated winner holds a card (plus the seeded QB).
	if len(lineAssets(t, f).Snapshot.Players) != 2 {
		t.Fatalf("open auction handed out cards: %+v", lineAssets(t, f).Snapshot.Players)
	}
	// Record the auction as closed without pre-assigning automatic teams.
	evs := f.events(t)
	rev := int64(len(evs)) + 1
	entry := fmt.Sprintf("evt.oline.close.%d", rev)
	closeAt := f.clock.Now()
	base := commissioner.RawCommand{
		League: f.league, Command: commissioner.OLineAuctionClose,
		Expected: int64(len(evs)), Request: "oline.close.worker",
		Reason: "Close auction", Effective: "now", Visibility: "league",
	}
	raw, err := base.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := raw.Canonical("creator")
	if err != nil {
		t.Fatal(err)
	}
	def, err := raw.EffectiveDefinition()
	if err != nil {
		t.Fatal(err)
	}
	raw.Gravity = def.Gravity
	raw.Args.OLineOutcome = &commissioner.OLineOutcome{
		Assigned: []commissioner.NativePlayer{},
		Pending:  []commissioner.NativePlayer{},
		Words:    []string{"Closed O-Line auction"},
	}
	rec := commissioner.Record{
		CommandVersion: commissioner.DurableCommandVersion,
		Command:        raw, Principal: "creator", Role: "commissioner",
		Gravity: raw.Gravity, Canonical: canonical,
		Receipt: commissioner.Receipt{
			League: f.league, Entry: entry, Revision: rev,
			RecordedAt: closeAt, Message: "Closed O-Line auction",
		},
	}
	sum := sha256.Sum256([]byte(entry))
	if _, err := f.store.AppendEvent(t.Context(), history.LeagueEvent{
		ID: entry, League: f.league, Stream: commissioner.Stream, Revision: rev,
		EffectiveAt: closeAt, RecordedAt: closeAt, Actor: "creator", Scope: "league",
		Kind: commissioner.OLineAuctionClose,
		Payload: history.EventPayload{
			Version: 1, Commissioner: &rec,
		},
		InputHash: hex.EncodeToString(sum[:]), CorrelationID: raw.Request,
	}, int64(len(evs))); err != nil {
		t.Fatal(err)
	}
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, p := range lineAssets(t, f).Snapshot.Players {
		if _, card := nativecheck.LineCardTeam(p.ID); card {
			got[p.Franchise] = p.ID
		}
	}
	atl, _ := nativecheck.LineCardID("ATL")
	bal, _ := nativecheck.LineCardID("BAL")
	if got["franchise.01"] != ari || got["franchise.02"] != atl ||
		got["franchise.03"] != bal || got["franchise.04"] != "" {
		t.Fatalf("unexpected worker handout assignments: %+v", got)
	}
	// Second worker pass is idempotent.
	count := len(f.events(t))
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.events(t)) != count {
		t.Fatalf("second worker pass changed ledger: %d -> %d", count, len(f.events(t)))
	}
}
