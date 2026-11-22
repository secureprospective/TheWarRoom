package history

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type handoutBidEnv struct{}

func (handoutBidEnv) BidEnvironment(context.Context, string, time.Time) (market.BidEnvironment, error) {
	return market.BidEnvironment{Season: nativecheck.BidSeason{Offseason: true}}, nil
}

type olineHandoutFixture struct {
	t      *testing.T
	s      *Store
	c      *clock
	league string
	facts  nativecheck.Facts
}

func newOLineHandoutFixture(t *testing.T, cardTeams []string) *olineHandoutFixture {
	t.Helper()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	s.SetBidEnvironment(handoutBidEnv{})
	at := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	c := &clock{now: at}
	s.SetClock(c.Now)
	facts := nativecheck.Facts{"0099": {Position: "QB"}}
	nativecheck.AddLineCards(facts)
	f := &olineHandoutFixture{t: t, s: s, c: c, league: "native.oline-handout", facts: facts}
	f.exec("creator", commissioner.Create, commissioner.RawArgs{
		Name: "OLine Handout", Preset: commissioner.PresetLegacy,
		Source: commissioner.SourceMFLImport, Season: 2026, FranchiseCount: 4,
	})
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
	for _, team := range cardTeams {
		id, ok := nativecheck.LineCardID(team)
		if !ok {
			t.Fatalf("unknown card team %s", team)
		}
		seed.Pool = append(seed.Pool, commissioner.NativePoolEntry{
			Player: id, Name: team + " O-Line", Season: 2026,
		})
	}
	f.exec("creator", commissioner.NativeSeed, commissioner.RawArgs{Native: seed})
	for _, r := range []struct{ field, value string }{
		{"cap.amount", `200000000`},
		{"roster.limits", `{"min":0,"max":100}`},
		{"roster.positionBounds", `{"OL":{"min":1,"max":1}}`},
		{"calendar.timezone", `"UTC"`},
		{"calendar.boundary", `"start_of_day"`},
		{"calendar.windows",
			`{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid"]}}`},
	} {
		f.exec("creator", commissioner.RulesSet,
			commissioner.RawArgs{Field: r.field, Value: json.RawMessage(r.value)})
	}
	f.exec("creator", "commish.seat.replace", commissioner.RawArgs{
		People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: "owner.one"},
	})
	return f
}

func (f *olineHandoutFixture) exec(principal, cmd string, args commissioner.RawArgs) {
	f.t.Helper()
	recs, err := f.s.nativeRecords(f.t.Context(), f.league)
	if err != nil {
		f.t.Fatal(err)
	}
	vis := "league"
	if cmd == commissioner.BidPlace {
		vis = "private"
	}
	raw := commissioner.RawCommand{
		League: f.league, Command: cmd, Args: args,
		Expected: int64(len(recs)), Request: fmt.Sprintf("req.%d.%s", len(recs), cmd),
		Reason: "Test setup", Effective: "now", Visibility: vis,
	}
	if _, err := f.s.ExecuteCommissionerWithFacts(f.t.Context(), raw, principal, nil,
		f.facts, time.UTC); err != nil {
		f.t.Fatalf("%s: %v", cmd, err)
	}
}

func (f *olineHandoutFixture) sellCardToSeated(team string, salary domain.Money) string {
	f.t.Helper()
	card, _ := nativecheck.LineCardID(team)
	f.exec("owner.one", commissioner.BidPlace, commissioner.RawArgs{
		Bid: &commissioner.RawBidArgs{
			Player: card, Franchise: "franchise.01", Salary: salary, Years: 1,
		},
	})
	f.c.now = f.c.now.Add(24 * time.Hour)
	if err := f.s.ProcessDueBids(f.t.Context(), f.league, f.facts); err != nil {
		f.t.Fatal(err)
	}
	return card
}

func (f *olineHandoutFixture) recordAuctionClosed() {
	f.t.Helper()
	recs, err := f.s.nativeRecords(f.t.Context(), f.league)
	if err != nil {
		f.t.Fatal(err)
	}
	at := f.c.Now()
	rev := int64(len(recs)) + 1
	entry := fmt.Sprintf("evt.oline.close.%d", rev)
	base := commissioner.RawCommand{
		League: f.league, Command: commissioner.OLineAuctionClose,
		Expected: int64(len(recs)), Request: "oline.close.empty",
		Reason: "Close auction", Effective: "now", Visibility: "league",
	}
	raw, err := base.Prepared()
	if err != nil {
		f.t.Fatal(err)
	}
	canonical, err := raw.Canonical("creator")
	if err != nil {
		f.t.Fatal(err)
	}
	def, err := raw.EffectiveDefinition()
	if err != nil {
		f.t.Fatal(err)
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
			RecordedAt: at, Message: "Closed O-Line auction",
		},
	}
	sum := sha256.Sum256([]byte(entry))
	ev := LeagueEvent{
		ID: entry, League: f.league, Stream: commissioner.Stream, Revision: rev,
		EffectiveAt: at, RecordedAt: at, Actor: "creator", Scope: "league",
		Kind: commissioner.OLineAuctionClose,
		Payload: EventPayload{
			Version: eventPayloadVersion, Commissioner: &rec,
		},
		InputHash: hex.EncodeToString(sum[:]), CorrelationID: raw.Request,
	}
	if _, err := f.s.AppendEvent(f.t.Context(), ev, int64(len(recs))); err != nil {
		f.t.Fatal(err)
	}
}

func (f *olineHandoutFixture) assets() commissioner.NativeAssets {
	f.t.Helper()
	recs, err := f.s.nativeRecords(f.t.Context(), f.league)
	if err != nil {
		f.t.Fatal(err)
	}
	a, err := commissioner.NativeAssetsReading(f.league, recs, f.c.Now())
	if err != nil {
		f.t.Fatal(err)
	}
	return a
}

func (f *olineHandoutFixture) minPrice() domain.Money {
	f.t.Helper()
	recs, err := f.s.nativeRecords(f.t.Context(), f.league)
	if err != nil {
		f.t.Fatal(err)
	}
	rules, err := commissioner.RulesNow(recs, f.c.Now())
	if err != nil {
		f.t.Fatal(err)
	}
	card, _ := nativecheck.LineCardID("ARI")
	pf, _ := f.facts.Of(card)
	minSal, ok := nativecheck.MinimumSalary(rules.Contracts.MinimumSalaries.Value,
		nativecheck.ExperienceYears(rules.Season, pf))
	if !ok {
		f.t.Fatal("minimum salary missing")
	}
	return minSal
}

func TestOLineHandoutGivesUnsoldCardsToAutomaticTeamsInOrder(t *testing.T) {
	f := newOLineHandoutFixture(t, []string{"BAL", "ARI", "ATL"})
	sold := f.sellCardToSeated("ARI", 50_000_000)
	f.recordAuctionClosed()
	wantMin := f.minPrice()

	var logBuf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(prev) })

	if err := f.s.ProcessDueOLineHandouts(t.Context(), f.league, f.facts); err != nil {
		t.Fatal(err)
	}

	got := map[string]commissioner.NativePlayer{}
	for _, p := range f.assets().Snapshot.Players {
		if _, card := nativecheck.LineCardTeam(p.ID); card {
			got[p.Franchise] = p
		}
	}
	atl, _ := nativecheck.LineCardID("ATL")
	bal, _ := nativecheck.LineCardID("BAL")

	if p, ok := got["franchise.01"]; !ok || p.ID != sold || p.Salary != 50_000_000 {
		t.Fatalf("seated franchise altered: %+v", p)
	}
	if p, ok := got["franchise.02"]; !ok || p.ID != atl || p.Salary != wantMin {
		t.Fatalf("franchise.02 got %+v, want %s at %d", p, atl, wantMin)
	}
	if p, ok := got["franchise.03"]; !ok || p.ID != bal || p.Salary != wantMin {
		t.Fatalf("franchise.03 got %+v, want %s at %d", p, bal, wantMin)
	}
	if p, ok := got["franchise.04"]; ok {
		t.Fatalf("franchise.04 should have no card when cards ran out, got %+v", p)
	}
	logged := logBuf.String()
	if !strings.Contains(logged, "unfilled=1") {
		t.Fatalf("expected unfilled=1 log, got %q", logged)
	}
	if strings.Contains(logged, "Team 4") || strings.Contains(logged, "franchise.04") {
		t.Fatalf("log must not contain names: %q", logged)
	}

	recs, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	handouts := 0
	for _, r := range recs {
		if r.Principal == commissioner.OLineHandoutPrincipal {
			handouts++
			if r.Command.Command != commissioner.RosterAdd {
				t.Fatalf("expected roster.add, got %s", r.Command.Command)
			}
		}
	}
	if handouts != 2 {
		t.Fatalf("expected 2 handout records, got %d", handouts)
	}
}

func TestOLineHandoutDoesNothingWhileAuctionOpen(t *testing.T) {
	f := newOLineHandoutFixture(t, []string{"ARI", "ATL", "BAL"})
	f.sellCardToSeated("ARI", 50_000_000)

	before, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.s.ProcessDueOLineHandouts(t.Context(), f.league, f.facts); err != nil {
		t.Fatal(err)
	}
	after, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("open auction modified ledger: before=%d after=%d", len(before), len(after))
	}
	cards := 0
	for _, p := range f.assets().Snapshot.Players {
		if _, card := nativecheck.LineCardTeam(p.ID); card {
			cards++
		}
	}
	if cards != 1 {
		t.Fatalf("open auction assigned cards: %+v", f.assets().Snapshot.Players)
	}
}

func TestOLineHandoutIsIdempotent(t *testing.T) {
	f := newOLineHandoutFixture(t, []string{"ARI", "ATL", "BAL", "BUF", "CAR"})
	f.sellCardToSeated("ARI", 50_000_000)
	f.recordAuctionClosed()

	if err := f.s.ProcessDueOLineHandouts(t.Context(), f.league, f.facts); err != nil {
		t.Fatal(err)
	}
	firstPass, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.s.ProcessDueOLineHandouts(t.Context(), f.league, f.facts); err != nil {
		t.Fatal(err)
	}
	secondPass, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondPass) != len(firstPass) {
		t.Fatalf("second pass appended records: first=%d second=%d",
			len(firstPass), len(secondPass))
	}
	// Deterministic request key replay lands once.
	atl, _ := nativecheck.LineCardID("ATL")
	var firstHandout commissioner.Record
	for _, r := range firstPass {
		if r.Principal == commissioner.OLineHandoutPrincipal &&
			r.Command.Args.Waiver != nil && r.Command.Args.Waiver.Player == atl {
			firstHandout = r
			break
		}
	}
	retry := firstHandout.Command
	retry.Args.WaiverOutcome = nil
	auth := olineHandoutAuthorizer{
		records:   firstPass[:firstHandout.Receipt.Revision-1],
		franchise: "franchise.02", player: atl, now: f.c.Now(),
	}
	got, err := f.s.ExecuteCommissionerWithFacts(t.Context(), retry,
		commissioner.OLineHandoutPrincipal, auth, f.facts, time.UTC)
	if err != nil || got.Entry != firstHandout.Receipt.Entry {
		t.Fatalf("deterministic retry did not replay receipt: got=%+v err=%v", got, err)
	}
	afterRetry, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterRetry) != len(firstPass) {
		t.Fatalf("retry appended duplicate record: before=%d after=%d",
			len(firstPass), len(afterRetry))
	}
}

func TestOLineHandoutAuthorizerRefusesSeatedAndSold(t *testing.T) {
	f := newOLineHandoutFixture(t, []string{"ARI", "ATL", "BAL"})
	sold := f.sellCardToSeated("ARI", 50_000_000)
	unsold, _ := nativecheck.LineCardID("ATL")

	openRecs, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	openAuth := olineHandoutAuthorizer{
		records: openRecs, franchise: "franchise.02", player: unsold, now: f.c.Now(),
	}
	if _, err := openAuth.Allow(f.league, commissioner.OLineHandoutPrincipal,
		commissioner.RosterAdd); err == nil {
		t.Fatal("authorizer allowed handout while auction was open")
	}

	f.recordAuctionClosed()
	recs, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	seatedAuth := olineHandoutAuthorizer{
		records: recs, franchise: "franchise.01", player: unsold, now: f.c.Now(),
	}
	if _, err := seatedAuth.Allow(f.league, commissioner.OLineHandoutPrincipal,
		commissioner.RosterAdd); err == nil {
		t.Fatal("authorizer allowed handout for a seated franchise")
	}
	soldAuth := olineHandoutAuthorizer{
		records: recs, franchise: "franchise.02", player: sold, now: f.c.Now(),
	}
	if _, err := soldAuth.Allow(f.league, commissioner.OLineHandoutPrincipal,
		commissioner.RosterAdd); err == nil {
		t.Fatal("authorizer allowed handout for a sold card")
	}
	goodAuth := olineHandoutAuthorizer{
		records: recs, franchise: "franchise.02", player: unsold, now: f.c.Now(),
	}
	if _, err := goodAuth.Allow(f.league, commissioner.OLineHandoutPrincipal,
		commissioner.OLineAuctionClose); err == nil {
		t.Fatal("authorizer allowed wrong command")
	}
	role, err := goodAuth.Allow(f.league, commissioner.OLineHandoutPrincipal,
		commissioner.RosterAdd)
	if err != nil || role != "automatic" {
		t.Fatalf("authorizer refused legal handout: role=%q err=%v", role, err)
	}
}

func TestOLineHandoutSkipsCapRefusalToNextFranchise(t *testing.T) {
	f := newOLineHandoutFixture(t, []string{"ARI", "ATL"})
	f.sellCardToSeated("ARI", 50_000_000)
	fullCap := domain.Money(200_000_000)
	f.exec("creator", commissioner.CapAdjust, commissioner.RawArgs{
		Override: &commissioner.RawNativeOverride{
			Franchise: "franchise.02", Amount: &fullCap, Season: 2026,
		},
	})
	f.recordAuctionClosed()

	if err := f.s.ProcessDueOLineHandouts(t.Context(), f.league, f.facts); err != nil {
		t.Fatal(err)
	}
	atl, _ := nativecheck.LineCardID("ATL")
	got := map[string]string{}
	for _, p := range f.assets().Snapshot.Players {
		if _, card := nativecheck.LineCardTeam(p.ID); card {
			got[p.Franchise] = p.ID
		}
	}
	if _, ok := got["franchise.02"]; ok {
		t.Fatalf("over-cap franchise.02 should have been skipped, got %s", got["franchise.02"])
	}
	if got["franchise.03"] != atl {
		t.Fatalf("franchise.03 got %q, want %s", got["franchise.03"], atl)
	}
}
