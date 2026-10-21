package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const (
	draftLeague = "native.seed"
	draftSeason = 2027
)

// draftStartAt() is when the fixture's rookie draft is set to start: 9:00 AM in Los Angeles.
func draftStartAt() time.Time { return time.Date(2027, 7, 14, 16, 0, 0, 0, time.UTC) }

// draftFixture is a seeded Legacy league on a database that can be reopened, with 2027 picks for every
// franchise (franchise.02's first round pick belongs to franchise.01), a held rookie class and a clock the
// test moves.
type draftFixture struct {
	app     *App
	path    string
	pools   *db.Pools
	now     time.Time
	session string
}

func draftTeam(n int) string { return fmt.Sprintf("franchise.%02d", n) }

func draftOrder(first ...int) []string {
	out, seen := []string{}, map[int]bool{}
	for _, n := range first {
		out, seen[n] = append(out, draftTeam(n)), true
	}
	for n := 1; n <= 32; n++ {
		if !seen[n] {
			out = append(out, draftTeam(n))
		}
	}
	return out
}

func draftPicks() []draftreports.FuturePick {
	picks := []draftreports.FuturePick{}
	for round := 1; round <= 5; round++ {
		for n := 1; n <= 32; n++ {
			id := fmt.Sprintf("%04d", n)
			owner := id
			if round == 1 && n == 2 {
				owner = "0001"
			}
			if round == 1 && n == 4 {
				owner = "0002"
			}
			picks = append(picks, draftreports.FuturePick{Year: draftSeason, Round: round, Original: id, Owner: owner})
		}
	}
	return picks
}

func openClockedHistory(t *testing.T, path string, now func() time.Time) (*history.Store, *db.Pools) {
	t.Helper()
	p, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(p, reg)
	store.SetClock(now)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return store, p
}

func draftApp(t *testing.T) *draftFixture {
	t.Helper()
	f := &draftFixture{path: filepath.Join(t.TempDir(), "draft.db"), session: "creator",
		now: time.Date(2027, 6, 1, 12, 0, 0, 0, time.UTC)}
	a := commissionerTestApp(t)
	a.core.History, f.pools = openClockedHistory(t, f.path, func() time.Time { return f.now })
	a.core.ClockNow, a.core.LocalZone = func() time.Time { return f.now }, westCoast(t)
	legalityMirrorEnding(t, a, nil)
	a.core.Lookup, a.core.HasLookup = legalityLookup(t), true
	held := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	out, err := playerid.New("2001")
	if err != nil {
		t.Fatal(err)
	}
	a.core.SeasonInjuries = heldSeasonFeed[[]leaguefeed.Injury]{FetchedAt: held,
		Value: []leaguefeed.Injury{{ID: out, Status: "Out"}}}
	a.core.SeasonDraftResults = heldSeasonFeed[[]draftreports.Pick]{FetchedAt: held, Value: []draftreports.Pick{}}
	a.core.SeasonFuturePicks = heldSeasonFeed[[]draftreports.FuturePick]{FetchedAt: held, Value: draftPicks()}
	seedNativeTest(t, a, createNativeSeedLeague(t, a, commissioner.PresetLegacy))
	f.app = a
	f.setRule(t, "calendar.boundary", `"start_of_day"`)
	return f
}

// reopen closes the database for real and builds a fresh App on the same file, still acting as the same
// person: no mirror, no rulebook, nothing held in memory.
func (f *draftFixture) reopen(t *testing.T) {
	t.Helper()
	old := f.app
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	store, pools := openClockedHistory(t, f.path, func() time.Time { return f.now })
	f.pools = pools
	restarted := &App{marketHeld: &marketapp.Held{}, raceHeld: &raceapp.Held{}, core: &core.State{Ctx: t.Context(), History: store, Started: make(chan struct{}),
		ClockNow: func() time.Time { return f.now }, LocalZone: old.core.LocalZone}}
	close(restarted.core.Started)
	restarted.core.Lookup, restarted.core.HasLookup, restarted.core.SeasonInjuries = old.core.Lookup, old.core.HasLookup, old.core.SeasonInjuries
	f.app = restarted
	f.actAs(f.session)
}

func (f *draftFixture) actAs(principal string) {
	f.session = principal
	f.app.core.CommissionerSession = testCommissionerSession{principal}
}

func (f *draftFixture) revisionOf() (int64, error) {
	assets, err := f.app.TargetNativeAssets(draftLeague)
	if err != nil {
		return 0, fmt.Errorf("read league version: %w", err)
	}
	return assets.Revision, nil
}

func (f *draftFixture) revision(t *testing.T) int64 {
	t.Helper()
	rev, err := f.revisionOf()
	if err != nil {
		t.Fatal(err)
	}
	return rev
}

// tryCommand sends a commissioner command with the right league version and returns what came back.
func (f *draftFixture) tryCommand(command string, args commissioner.RawArgs,
	reason string) (commissioner.Receipt, error) {
	rev, err := f.revisionOf()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	raw := commissioner.RawCommand{League: draftLeague, Command: command, Args: args, Reason: reason,
		Expected: rev, Request: fmt.Sprintf("draft.%d", rev), Effective: "now", Visibility: "league"}
	body, err := json.Marshal(raw)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("encode test command: %w", err)
	}
	return f.app.TargetCommissionerExecute(string(body))
}

func (f *draftFixture) command(t *testing.T, command string, args commissioner.RawArgs,
	reason string) commissioner.Receipt {
	t.Helper()
	receipt, err := f.tryCommand(command, args, reason)
	if err != nil {
		t.Fatalf("%s: %v", command, err)
	}
	return receipt
}

func (f *draftFixture) draftCommand(t *testing.T, command string, d commissioner.RawDraftArgs) commissioner.Receipt {
	t.Helper()
	reason := ""
	if d.Running {
		reason = "Correct the order while the clock runs"
	}
	return f.command(t, command, commissioner.RawArgs{Draft: &d}, reason)
}

func (f *draftFixture) setRule(t *testing.T, field, value string) commissioner.Receipt {
	t.Helper()
	return f.command(t, commissioner.RulesSet, commissioner.RawArgs{Field: field, Value: json.RawMessage(value)},
		"Our league's ruling")
}

func (f *draftFixture) revert(t *testing.T, receipt commissioner.Receipt) {
	t.Helper()
	f.command(t, commissioner.Revert, commissioner.RawArgs{Entry: receipt.Entry}, "Undo that ruling")
}

func (f *draftFixture) seat(t *testing.T, franchise, principal string) {
	t.Helper()
	f.command(t, "commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{
		Franchise: franchise, Principal: principal}}, "Seat the owner")
}

func (f *draftFixture) read(t *testing.T, franchise string) commissioner.NativeDraft {
	t.Helper()
	reading, err := f.app.TargetNativeDraft(commissioner.RawNativeDraftScope{League: draftLeague,
		Season: draftSeason, Franchise: franchise, At: f.now})
	if err != nil {
		t.Fatal(err)
	}
	return reading
}

func (f *draftFixture) tryPick(franchise, player, request string, expected int64) (commissioner.Receipt, error) {
	return f.app.TargetNativeDraftPick(commissioner.DraftPickInput{League: draftLeague, Season: draftSeason,
		Franchise: franchise, Player: player, Request: request, Expected: expected})
}

func (f *draftFixture) pick(t *testing.T, franchise, player string) commissioner.Receipt {
	t.Helper()
	rev := f.revision(t)
	receipt, err := f.tryPick(franchise, player, fmt.Sprintf("pick.%d", rev), rev)
	if err != nil {
		t.Fatalf("pick %s for %s: %v", player, franchise, err)
	}
	return receipt
}

// scheduled orders every round by the given first teams, seats the owners and starts the draft at draftStartAt().
func (f *draftFixture) scheduled(t *testing.T, first ...int) (order, start commissioner.Receipt) {
	t.Helper()
	order = f.draftCommand(t, commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason,
		Order: draftOrder(first...)})
	start = f.draftCommand(t, commissioner.DraftStart, commissioner.RawDraftArgs{Season: draftSeason,
		At: draftStartAt()})
	return order, start
}
