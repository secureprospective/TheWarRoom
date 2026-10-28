package leagueservice

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// An imported league with O-Line enabled by settings, not the Default32 preset.
func newOLineDueFixture(t *testing.T) *dueFixture {
	t.Helper()
	f := &dueFixture{league: dueLeague, clock: &dueTestClock{}, path: filepath.Join(t.TempDir(), "history.db"), facts: nativecheck.Facts{"0099": {Position: "QB"}}}
	f.clock.instant.Store(time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC).UnixNano())
	f.open(t)
	f.execute(t, "creator", commissioner.Create, commissioner.RawArgs{Name: "Line league", Preset: commissioner.PresetLegacy, Season: 2026, FranchiseCount: 4})
	at := f.clock.Now()
	seed := &commissioner.RawNativeSnapshot{Season: 2026, RosterAt: at, DraftAt: at, FutureAt: at,
		Players: []commissioner.NativePlayer{{ID: "0099", Franchise: "franchise.02", Name: "Quarterback", Status: domain.RosterActive, Salary: 100_000_000, ExpirationYear: 2026, ContractStatus: domain.CStatusUFA}}}
	for i := 1; i <= 4; i++ {
		id := fmt.Sprintf("franchise.%02d", i)
		seed.Franchises = append(seed.Franchises, commissioner.NativeFranchise{ID: id, HeldID: id, Name: fmt.Sprintf("Team %d", i)})
	}
	for _, team := range nativecheck.LineCardTeams() {
		id, _ := nativecheck.LineCardID(team)
		seed.Pool = append(seed.Pool, commissioner.NativePoolEntry{Player: id, Name: team + " O-Line", Season: 2026})
	}
	f.execute(t, "creator", commissioner.NativeSeed, commissioner.RawArgs{Native: seed})
	for _, r := range []struct{ field, value string }{
		{"cap.amount", `200000000`},
		{"roster.limits", `{"min":0,"max":100}`}, {"roster.positionBounds", `{"OL":{"min":1,"max":1}}`},
		{"calendar.timezone", `"UTC"`}, {"calendar.boundary", `"start_of_day"`},
		{"calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid"]}}`},
	} {
		f.execute(t, "creator", commissioner.RulesSet, commissioner.RawArgs{Field: r.field, Value: json.RawMessage(r.value)})
	}
	f.execute(t, "creator", "commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.02", Principal: "owner.two"}})
	return f
}

func lineAssets(t *testing.T, f *dueFixture) commissioner.NativeAssets {
	t.Helper()
	records := []commissioner.Record{}
	for _, e := range f.events(t) {
		records = append(records, *e.Payload.Commissioner)
	}
	assets, err := commissioner.NativeAssetsReading(f.league, records)
	if err != nil {
		t.Fatal(err)
	}
	return assets
}

func TestOLineCloseFullCapCutAndRestart(t *testing.T) {
	f := newOLineDueFixture(t)
	// Only the receiver's franchise is at the cap; the other franchises have room.
	f.execute(t, "creator", commissioner.CapAdjust, commissioner.RawArgs{Override: &commissioner.RawNativeOverride{Franchise: "franchise.02", Amount: moneyPointer(100_000_000), Season: 2026}})
	raw := f.raw(t, commissioner.OLineAuctionClose, commissioner.RawArgs{})
	receipt, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil, f.facts, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	assets := lineAssets(t, f)
	if len(assets.Snapshot.Players) != 4 {
		t.Fatalf("full cap was overrun or open teams unfilled: %+v", assets.Snapshot.Players)
	}
	jobs, err := f.store.CommissionerJobs(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, job := range jobs.Jobs {
		if job.ID == "oline.must_cut.franchise.02" {
			found = strings.Contains(job.Words, "Team 2: cut a player")
		}
	}
	if !found {
		t.Fatalf("plain cut prompt missing: %+v", jobs)
	}
	count := len(f.events(t))
	_, bidErr := f.store.ExecuteCommissionerWithFacts(t.Context(), f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "990003", Franchise: "franchise.02", Salary: 33_000_000, Years: 1}}), "owner.two", nil, f.facts, time.UTC)
	if bidErr == nil || !strings.Contains(bidErr.Error(), "O-Line auction is closed") || len(f.events(t)) != count {
		t.Fatalf("reserved auction reopened: %v", bidErr)
	}
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.events(t)) != count {
		t.Fatal("blocked worker appended or assigned over cap")
	}
	retry, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil, f.facts, time.UTC)
	if err != nil || retry.Entry != receipt.Entry || len(f.events(t)) != count {
		t.Fatalf("retry changed close: %+v %v", retry, err)
	}
	f.execute(t, "owner.two", commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02"}})
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	f.open(t)
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	assets = lineAssets(t, f)
	cards := map[string]int{}
	for _, p := range assets.Snapshot.Players {
		if _, ok := nativecheck.LineCardTeam(p.ID); ok {
			cards[p.Franchise]++
			if p.Salary != 33_000_000 || p.ExpirationYear != 2026 {
				t.Fatalf("wrong minimum contract: %+v", p)
			}
		}
	}
	for _, franchise := range assets.Snapshot.Franchises {
		if cards[franchise.ID] != 1 {
			t.Fatalf("%s holds %d units", franchise.ID, cards[franchise.ID])
		}
	}
	last := f.events(t)[len(f.events(t))-1].Payload.Commissioner
	if last.Principal != commissioner.OLineClockPrincipal || !strings.Contains(strings.Join(last.Command.Args.OLineOutcome.Words, ". "), "Assigned the ATL O-Line to Team 2 at the minimum contract") {
		t.Fatalf("missing assignment ledger words: %+v", last)
	}
	count = len(f.events(t))
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.events(t)) != count {
		t.Fatal("worker assigned twice")
	}
	jobs, err = f.store.CommissionerJobs(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range jobs.Jobs {
		if strings.HasPrefix(job.ID, "oline.must_cut.") {
			t.Fatal("resolved prompt remained")
		}
	}
}

func moneyPointer(cents int64) *domain.Money { m := domain.Money(cents); return &m }

func TestOLineCloseOffAndUnauthorizedPreserveState(t *testing.T) {
	f := newOLineDueFixture(t)
	grant := f.raw(t, "commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{Principal: "deputy", Role: governance.CoCommissioner}})
	grant.Visibility = "private"
	if _, err := f.store.ExecuteCommissionerWithFacts(t.Context(), grant, "creator", nil, f.facts, time.UTC); err != nil {
		t.Fatal(err)
	}
	for _, principal := range []string{"owner.two", "deputy", "stranger", commissioner.OLineClockPrincipal} {
		raw := f.raw(t, commissioner.OLineAuctionClose, commissioner.RawArgs{})
		_, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, principal, nil, f.facts, time.UTC)
		if err == nil {
			t.Fatalf("%s closed auction", principal)
		}
	}
	f.execute(t, "creator", commissioner.RulesSet, commissioner.RawArgs{Field: "roster.positionBounds", Value: json.RawMessage(`{}`)})
	count := len(f.events(t))
	_, err := f.store.ExecuteCommissionerWithFacts(t.Context(), f.raw(t, commissioner.OLineAuctionClose, commissioner.RawArgs{}), "creator", nil, f.facts, time.UTC)
	if err == nil || !strings.Contains(err.Error(), "does not use O-Line") {
		t.Fatalf("off: %v", err)
	}
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.events(t)) != count || len(lineAssets(t, f).Snapshot.Players) != 1 {
		t.Fatal("O-Line off changed assets")
	}
}

func TestOLineWorkerCannotCloseAnOpenAuction(t *testing.T) {
	f := newOLineDueFixture(t)
	before := len(f.events(t))
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.events(t)) != before || len(lineAssets(t, f).Snapshot.Players) != 1 {
		t.Fatal("worker closed the auction without the commissioner")
	}
}

func TestOLineClosePreservesARealBidAward(t *testing.T) {
	f := newOLineDueFixture(t)
	nativecheck.AddLineCards(f.facts)
	f.execute(t, "owner.two", commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "990001", Franchise: "franchise.02", Salary: 50_000_000, Years: 1}})
	f.clock.instant.Add(int64(24 * time.Hour))
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	before := lineAssets(t, f)
	if len(before.Snapshot.Players) != 2 {
		t.Fatalf("bid did not award a unit: %+v", before.Snapshot.Players)
	}
	f.execute(t, "creator", commissioner.OLineAuctionClose, commissioner.RawArgs{})
	assets := lineAssets(t, f)
	cards := map[string]int{}
	for _, p := range assets.Snapshot.Players {
		if _, card := nativecheck.LineCardTeam(p.ID); !card {
			continue
		}
		cards[p.Franchise]++
		if p.ID == "990001" && (p.Franchise != "franchise.02" || p.Salary != 50_000_000) {
			t.Fatalf("winning contract rewritten: %+v", p)
		}
	}
	for _, team := range assets.Snapshot.Franchises {
		if cards[team.ID] != 1 {
			t.Fatalf("%s: %d O-Line units", team.ID, cards[team.ID])
		}
	}
	ledger, err := f.store.CommissionerLedger(t.Context(), f.league, "owner.two", nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range ledger.Entries {
		if strings.Contains(entry.What, "Assigned the ATL O-Line to Team 1 at the minimum contract") {
			found = true
		}
	}
	if !found {
		t.Fatal("owner cannot read the plain assignment ledger entry")
	}
}
