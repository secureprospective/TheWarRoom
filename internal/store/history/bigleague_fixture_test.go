//go:build leaguebench

package history

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const bigLeagueID = "native.bigleague"

// The directory outlives benchmark calibration/count repetitions, but not the binary.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp(os.Getenv("BENCH_LEAGUE_SCRATCH"), "bigleague-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.Setenv("BENCH_LEAGUE_BINARY_DIR", dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root, err := os.OpenRoot(filepath.Dir(dir))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	if err := root.RemoveAll(filepath.Base(dir)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	if err := root.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}

type bigLeagueBuilder struct {
	t        testing.TB
	s        *Store
	now      time.Time
	revision int64
	nextID   int
	facts    nativecheck.Facts
	players  []leagueOwner
	offered  []commissioner.TradeAsset
}

type leagueOwner struct{ ID, Franchise string }

func bigTeam(n int) string  { return fmt.Sprintf("franchise.%02d", n%32+1) }
func bigOwner(n int) string { return fmt.Sprintf("owner.%02d", n%32+1) }

func openBigLeague(t testing.TB, path string) (*Store, *db.Pools) {
	t.Helper()
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	p, err := db.Open(context.Background(), path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	s := New(p, reg)
	if err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s, p
}

func (g *bigLeagueBuilder) command(who, command string, args commissioner.RawArgs, private bool) commissioner.Receipt {
	g.t.Helper()
	visibility := "league"
	if private {
		visibility = "private"
	}
	raw := commissioner.RawCommand{League: bigLeagueID, Command: command, Args: args,
		Expected: g.revision, Request: fmt.Sprintf("generated.%08d", g.revision),
		Reason: "Deterministic growth fixture", Effective: "now", Visibility: visibility}
	got, err := g.s.ExecuteCommissionerWithFacts(context.Background(), raw, who, nil, g.facts, time.UTC)
	if err != nil {
		g.t.Fatalf("year %d revision %d %s: %v", g.now.Year(), g.revision, command, err)
	}
	g.revision = got.Revision
	g.now = g.now.Add(time.Microsecond)
	return got
}

func (g *bigLeagueBuilder) set(field, value string) {
	g.command("creator", commissioner.RulesSet, commissioner.RawArgs{Field: field, Value: json.RawMessage(value)}, false)
}

func (g *bigLeagueBuilder) player(year int) string {
	g.nextID++
	id := fmt.Sprintf("%08d", g.nextID)
	g.facts[id] = nativecheck.PlayerFacts{Name: "Generated " + id, Position: leaguerules.LB,
		DraftYear: year, DraftYearHeld: true, DesignationHeld: true, CareerGamesHeld: true}
	return id
}

func (g *bigLeagueBuilder) assets() *commissioner.RawNativeSnapshot {
	g.t.Helper()
	got, err := g.s.NativeAssets(context.Background(), bigLeagueID)
	if err != nil {
		g.t.Fatal(err)
	}
	if got.Snapshot == nil {
		g.t.Fatal("generated league has no assets")
	}
	return got.Snapshot
}

func (g *bigLeagueBuilder) start(lastYear int) {
	g.command("creator", commissioner.Create, commissioner.RawArgs{Name: "Big league", Preset: commissioner.PresetLegacy,
		Season: 2013, FranchiseCount: 32}, false)
	seed := &commissioner.RawNativeSnapshot{Season: 2013, RosterAt: g.now, DraftAt: g.now, FutureAt: g.now}
	for n := range 32 {
		team := bigTeam(n)
		seed.Franchises = append(seed.Franchises, commissioner.NativeFranchise{ID: team, HeldID: fmt.Sprintf("%04d", n+1),
			Name: fmt.Sprintf("Team %02d", n+1), Abbreviation: fmt.Sprintf("T%02d", n+1), Division: "Generated"})
		seed.StandingsOrder = append(seed.StandingsOrder, team)
	}
	// Startup is a held snapshot: the native draft is a rookie draft, not a veteran startup.
	for n := range 1696 {
		id := g.player(2012)
		seed.Players = append(seed.Players, commissioner.NativePlayer{ID: id, Name: g.facts[id].Name,
			Franchise: bigTeam(n), Status: domain.RosterActive, Salary: 100000000,
			ExpirationYear: lastYear + 1, ContractStatus: domain.CStatusUFA, AcquiredVia: "held"})
	}
	for year := 2014; year <= lastYear+2; year++ {
		for round := 1; round <= 5; round++ {
			for n := range 32 {
				seed.Picks = append(seed.Picks, commissioner.NativePick{Season: year, Round: round, Original: bigTeam(n), Owner: bigTeam(n)})
			}
		}
	}
	for _, p := range seed.Players {
		g.players = append(g.players, leagueOwner{p.ID, p.Franchise})
	}
	g.command("creator", commissioner.NativeSeed, commissioner.RawArgs{Native: seed}, false)
	// Relax only workload legality; all moves still pass production authorization and append checks.
	g.set("roster.limits", `{"min":0,"max":10000}`)
	g.set("cap.amount", `100000000000000`)
	g.set("calendar.timezone", `"UTC"`)
	g.set("calendar.boundary", `"start_of_day"`)
	g.set("calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid","trade"]}}`)
	g.set("calendar.tradeDeadline", `"2013-12-31T00:00:00Z"`)
	g.set("trades.review", `"commissioner"`)
	g.set("draft.laterPattern", `"linear"`)
	g.set("draft.maximumPicks", `100`)
	for n := range 32 {
		g.command("creator", "commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{
			Franchise: bigTeam(n), Principal: bigOwner(n)}}, false)
	}
}

func (g *bigLeagueBuilder) moves(v leagueVolume) {
	players := slices.Clone(g.players)
	if len(players) < v.drops {
		g.t.Fatal("not enough players for measured drops")
	}
	// Separate WAIVER/FREE_AGENT counts were not supplied: an explicit synthetic half split,
	// bounded by real drops because a native waiver claim needs a preceding cut.
	waivers := min(v.adds/2, v.drops)
	windows := make([]string, v.drops)
	for n := range v.drops {
		p := players[n]
		who := ""
		for team := range 32 {
			if p.Franchise == bigTeam(team) {
				who = bigOwner(team)
			}
		}
		r := g.command(who, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
			Player: p.ID, Franchise: p.Franchise}}, false)
		windows[n] = r.Entry
		g.players = slices.DeleteFunc(g.players, func(p leagueOwner) bool { return p.ID == players[n].ID })
	}
	order := make([]string, 32)
	for n := range 32 {
		order[n] = bigTeam(n)
	}
	g.command("creator", commissioner.WaiverOrderSet, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Order: order}}, false)
	for n := range waivers {
		team := (n + 1) % 32
		if players[n].Franchise == bigTeam(team) {
			team = (team + 1) % 32
		}
		g.command(bigOwner(team), commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
			Window: windows[n], Franchise: bigTeam(team)}}, true)
	}
	bids := make([]string, v.adds-waivers)
	for n := range bids {
		id := g.player(v.year - 3)
		bids[n] = id
		g.command(bigOwner(n), commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{
			Player: id, Franchise: bigTeam(n), Salary: 100000000, Years: 1}}, true)
	}
	g.now = g.now.Add(25 * time.Hour)
	for n, window := range windows {
		g.command("creator", commissioner.WaiverProcess, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: window}}, false)
		if n < waivers {
			team := (n + 1) % 32
			if players[n].Franchise == bigTeam(team) {
				team = (team + 1) % 32
			}
			g.players = append(g.players, leagueOwner{players[n].ID, bigTeam(team)})
		}
	}
	for n, player := range bids {
		g.command("creator", commissioner.BidsProcess, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: player}}, false)
		g.players = append(g.players, leagueOwner{player, bigTeam(n)})
	}
}

func (g *bigLeagueBuilder) offer(count int) (commissioner.Receipt, string, string) {
	players := g.players
	if len(players) < count {
		g.t.Fatal("not enough trade assets")
	}
	from := players[0].Franchise
	to, proposer, receiver := bigTeam(0), "", bigOwner(0)
	if from == to {
		to, receiver = bigTeam(1), bigOwner(1)
	}
	for n := range 32 {
		if from == bigTeam(n) {
			proposer = bigOwner(n)
		}
	}
	assets := make([]commissioner.TradeAsset, 0, count)
	for _, p := range players {
		if p.Franchise != from {
			continue
		}
		assets = append(assets, commissioner.TradeAsset{From: from, To: to, Player: p.ID})
		if len(assets) == count {
			break
		}
	}
	if len(assets) != count {
		g.t.Fatal("not enough assets held by trade proposer")
	}
	g.offered = assets
	r := g.command(proposer, commissioner.TradeOffer, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{
		Franchise: from, Parties: []string{from, to}, Assets: assets}}, false)
	return r, to, receiver
}

func (g *bigLeagueBuilder) trades(v leagueVolume) {
	for n := range v.trades {
		count := v.assets / v.trades
		if n < v.assets%v.trades {
			count++
		}
		r, to, receiver := g.offer(count)
		g.command(receiver, commissioner.TradeAccept, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: r.Entry, Franchise: to}}, false)
		g.command("creator", commissioner.TradeReview, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: r.Entry, Decision: "approve"}}, false)
		for i := range g.players {
			for _, a := range g.offered {
				if g.players[i].ID == a.Player {
					g.players[i].Franchise = a.To
				}
			}
		}
	}
	// Lifecycle counts are not correlated by offer in the census. Independent offers avoid
	// inventing correlations; supplementary proposals make every observed response legal.
	responses := v.accepts + v.rejects + v.revokes + v.expired
	for n := range max(v.proposals, responses) {
		r, to, receiver := g.offer(1)
		switch {
		case n < v.accepts:
			g.command(receiver, commissioner.TradeAccept, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: r.Entry, Franchise: to}}, false)
		case n < v.accepts+v.rejects:
			g.command(receiver, commissioner.TradeDecline, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: r.Entry, Franchise: to}}, false)
		case n < v.accepts+v.rejects+v.revokes:
			from, who := g.players[0].Franchise, ""
			for team := range 32 {
				if from == bigTeam(team) {
					who = bigOwner(team)
				}
			}
			g.command(who, commissioner.TradeWithdraw, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: r.Entry, Franchise: from}}, false)
		}
	}
	// Expiry is a read-time status, not an appendable native command.
	g.now = g.now.Add(8 * 24 * time.Hour)
}

func (g *bigLeagueBuilder) draft(v leagueVolume) {
	if v.year == 2013 {
		return
	}
	order := make([]string, 32)
	for n := range 32 {
		order[n] = bigTeam(n)
	}
	g.command("creator", commissioner.DraftOrderSet, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: v.year, Order: order}}, false)
	g.command("creator", commissioner.DraftStart, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: v.year, At: g.now.Add(time.Second)}}, false)
	g.now = g.now.Add(2 * time.Second)
	for n := range v.picks {
		id := g.player(v.year)
		command := commissioner.DraftAssign
		act := &commissioner.RawDraftAct{Round: n/32 + 1, Original: bigTeam(n)}
		if n >= 160 {
			command, act.Round, act.Original, act.Franchise = commissioner.DraftInsert, 5, bigTeam(31), bigTeam(n)
		}
		act.Player = id
		g.command("creator", command, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: v.year, Act: act}}, false)
		g.players = append(g.players, leagueOwner{id, bigTeam(n)})
	}
	g.command("creator", commissioner.DraftArchive, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: v.year}}, false)
}

// Offseason environment is synthetic: bids are still judged by the production bid policy.
type bigLeagueEnvironment struct{}

func (bigLeagueEnvironment) BidEnvironment(context.Context, string, time.Time) (commissioner.BidEnvironment, error) {
	return commissioner.BidEnvironment{Season: nativecheck.BidSeason{Week: 1, WeekKnown: true}}, nil
}

func generateBigLeague(t testing.TB, s *Store, volumes []leagueVolume) {
	t.Helper()
	g := &bigLeagueBuilder{t: t, s: s, now: time.Date(2013, 4, 1, 12, 0, 0, 0, time.UTC), facts: nativecheck.Facts{}, nextID: 10000000}
	s.SetClock(func() time.Time { return g.now })
	s.SetBidEnvironment(bigLeagueEnvironment{})
	started := time.Now()
	g.start(volumes[len(volumes)-1].year)
	for i, v := range volumes {
		if i != 0 {
			g.now = time.Date(v.year, 4, 1, 12, 0, 0, 0, time.UTC)
			g.command("creator", commissioner.SeasonRollover, commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: v.year}}, false)
		}
		g.draft(v)
		g.moves(v)
		g.trades(v)
		fmt.Printf("BUILD season=%d cumulative-seconds=%.3f revision=%d\n", v.year, time.Since(started).Seconds(), g.revision)
	}
	got := g.assets()
	actual := make([]leagueOwner, 0, len(got.Players))
	for _, p := range got.Players {
		actual = append(actual, leagueOwner{p.ID, p.Franchise})
	}
	if !slices.Equal(g.players, actual) {
		t.Fatal("in-memory roster ownership/order differs from ledger")
	}
	if got.Season != volumes[len(volumes)-1].year || len(got.Franchises) != 32 {
		t.Fatal("generated league lost its season or franchises")
	}
}

func TestBigLeagueCommandSmoke(t *testing.T) {
	var first []byte
	for copyNo := range 2 {
		s, _ := openBigLeague(t, filepath.Join(t.TempDir(), "smoke.db"))
		volumes := []leagueVolume{{year: 2013, adds: 4, drops: 4, trades: 2, assets: 5, proposals: 4, accepts: 1, rejects: 1, revokes: 1, expired: 1},
			{year: 2014, adds: 4, drops: 4, trades: 2, assets: 5, picks: 170}}
		generateBigLeague(t, s, volumes)
		var records int
		if err := s.pools.Read().QueryRow("SELECT count(*) FROM league_event").Scan(&records); err != nil {
			t.Fatal(err)
		}
		if records != 266 {
			t.Fatalf("rows=%d, want 266", records)
		}
		events, err := s.Events(t.Context(), bigLeagueID, commissioner.Stream, 0)
		if err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal(events)
		if err != nil {
			t.Fatal(err)
		}
		if copyNo == 0 {
			first = body
		} else if !bytes.Equal(first, body) {
			t.Fatal("native ledger bytes differ between runs")
		}
		t.Logf("copy=%d rows=%d ledger-bytes=%d", copyNo, records, len(body))
	}
}
