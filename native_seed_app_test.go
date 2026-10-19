package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func nativeSeedTestApp(t *testing.T) *App {
	t.Helper()
	a := commissionerTestApp(t)
	p, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "mirror.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	a.rulebook = rulebook.New(p)
	cfg := league.RawConfig{CurrentSeason: 2026}
	for i := 32; i >= 1; i-- {
		cfg.Franchises = append(cfg.Franchises, league.Franchise{ID: fmt.Sprintf("%04d", i),
			Name: fmt.Sprintf("Held team %d", i), Abbrev: fmt.Sprintf("H%d", i), Division: "division.1"})
	}
	if err := a.rulebook.Initialize(t.Context(), lineupRulesSource{cfg: cfg}); err != nil {
		t.Fatal(err)
	}
	a.league = state.NewMirror(p, a.rulebook)
	if err := a.league.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	a.season = 2026
	snap := state.MirrorSnapshot{Season: a.season, Adjustments: []state.Adjustment{
		{ID: "held-credit", FranchiseID: "0002", Amount: -12345, Description: "Held cap credit"}}}
	for i, status := range []domain.RosterStatus{domain.RosterActive, domain.RosterIR, domain.RosterTaxi} {
		snap.Players = append(snap.Players, state.MirrorPlayer{MFLID: fmt.Sprintf("%04d", 99+i),
			FranchiseID: "0002", RosterStatus: status, Salary: domain.Money(12345600 + i),
			ContractYear: 2028 + i, ContractStatus: domain.CStatusUFA, ContractInfo: "Held contract"})
	}
	if _, err := a.league.Replace(t.Context(), snap); err != nil {
		t.Fatal(err)
	}
	id, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	a.seasonDraftResults = heldSeasonFeed[[]draftreports.Pick]{fetchedAt: at,
		value: []draftreports.Pick{{Round: 1, Slot: 1, Franchise: "0002", Player: &id},
			{Round: 1, Slot: 2, Franchise: "0001", Skipped: true}}}
	a.seasonFuturePicks = heldSeasonFeed[[]draftreports.FuturePick]{fetchedAt: at,
		value: []draftreports.FuturePick{{Year: 2027, Round: 2, Original: "0001", Owner: "0002"}}}
	return a
}

func createNativeSeedLeague(t *testing.T, a *App, preset string) RawNativeSeedRequest {
	t.Helper()
	raw := commissioner.RawCommand{League: "native.seed", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Seed league", Preset: preset, Season: 2026, FranchiseCount: 32},
		Reason: "Create the league", Request: "create", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, a, raw)
	return RawNativeSeedRequest{League: raw.League, Expected: 1, Request: "held.start", Reason: "Start from held data"}
}

func seedNativeTest(t *testing.T, a *App, input RawNativeSeedRequest) commissioner.Receipt {
	t.Helper()
	receipt, err := a.TargetNativeSeed(input)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func readNativeTest(t *testing.T, a *App, leagueID string) commissioner.NativeAssets {
	t.Helper()
	assets, err := a.TargetNativeAssets(leagueID)
	if err != nil {
		t.Fatal(err)
	}
	return assets
}

func nativeSeedWhatIf(t *testing.T, a *App) *db.Pools {
	t.Helper()
	p, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "whatif.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	a.whatif = state.New(p, "held-test", a.season, a.rulebook)
	if err := a.whatif.Initialize(t.Context(), a.league); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNativeSeedCapturesHeldAssetsWithoutChangingMirror(t *testing.T) {
	a := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	before, stamp, used := a.league.ContractsView()
	whatIfPools := nativeSeedWhatIf(t, a)
	simulation, ok := a.whatif.FranchiseState("0002")
	if !ok {
		t.Fatal("test has no what-if roster")
	}
	receipt := seedNativeTest(t, a, input)
	assets := readNativeTest(t, a, input.League)
	if !assets.Seeded || assets.Revision != 2 || assets.Entry != receipt.Entry || assets.Snapshot.Season != 2026 {
		t.Fatalf("not recorded as app-owned assets: %+v", assets)
	}
	seed := assets.Snapshot
	if len(seed.Franchises) != 32 || seed.Franchises[0].HeldID != "0001" ||
		seed.Franchises[1].ID != "franchise.02" || seed.Franchises[1].Name != "Held team 2" {
		t.Fatalf("held franchise identity lost: %+v", seed.Franchises)
	}
	for i, p := range seed.Players {
		held := before.Players[i]
		if p.ID != held.MFLID || p.Franchise != "franchise.02" || p.Status != held.RosterStatus ||
			p.Salary != held.Salary || p.ExpirationYear != held.ContractYear ||
			p.ContractStatus != held.ContractStatus || p.ContractInfo != held.ContractInfo {
			t.Fatalf("held contract or reserve status changed: %+v / %+v", p, held)
		}
	}
	if len(seed.Players) != 3 || len(seed.Picks) != 3 || seed.Picks[0].Player != "0099" ||
		!seed.Picks[1].Skipped || seed.Picks[2].Original != "franchise.01" ||
		seed.Picks[2].Owner != "franchise.02" || seed.Adjustments[0].Amount != -12345 {
		t.Fatalf("picks or cap adjustments lost: %+v", seed)
	}
	after, afterStamp, afterUsed := a.league.ContractsView()
	if !reflect.DeepEqual(before, after) || stamp != afterStamp || !reflect.DeepEqual(used, afterUsed) {
		t.Fatal("seeding wrote the MFL mirror")
	}
	replayed := state.New(whatIfPools, "held-test", a.season, a.rulebook)
	if err := replayed.Initialize(t.Context(), a.league); err != nil {
		t.Fatal(err)
	}
	if after, ok := replayed.FranchiseState("0002"); !ok || !reflect.DeepEqual(simulation, after) {
		t.Fatal("seeding changed the persisted what-if state")
	}
	ledger, err := a.TargetCommissionerLedger(input.League)
	if err != nil || len(ledger.Entries) != 2 || ledger.Entries[0].Undo != "" ||
		ledger.Entries[0].What != "Recorded the starting rosters, contracts, reserves, franchises and picks from held data" {
		t.Fatalf("seed act is missing or labelled as a reversal: %+v %v", ledger, err)
	}
}

func TestNativeSeedOnceAndRetryAfterMirrorRefresh(t *testing.T) {
	a := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	first := seedNativeTest(t, a, input)
	original := readNativeTest(t, a, input.League)
	fresh, _, _ := a.league.ContractsView()
	fresh.Players[0].Salary = 999999999
	fresh.Players[1].RosterStatus = domain.RosterActive
	if _, err := a.league.Replace(t.Context(), fresh); err != nil {
		t.Fatal(err)
	}
	if retry := seedNativeTest(t, a, input); retry != first {
		t.Fatal("retry did not return the recorded receipt")
	}
	second := input
	second.Request, second.Expected = "second.seed", 2
	if _, err := a.TargetNativeSeed(second); err == nil {
		t.Fatal("seed accepted twice")
	}
	second = input
	second.Reason = "different body"
	if _, err := a.TargetNativeSeed(second); err == nil {
		t.Fatal("same request with changed reason accepted")
	}
	a.commissionerSession = testCommissionerSession{"owner"}
	if _, err := a.TargetNativeSeed(input); err == nil {
		t.Fatal("unauthorized retry returned a receipt")
	}
	after := readNativeTest(t, a, input.League)
	if !reflect.DeepEqual(original, after) {
		t.Fatalf("refresh or second seed changed native assets: %+v / %+v", original, after)
	}
}

func TestNativeSeedRefusesMissingAndMixedHeldInputs(t *testing.T) {
	for name, breakInput := range map[string]func(*App){
		"rosters": func(a *App) { a.league = nil },
		"results": func(a *App) { a.seasonDraftResults.fetchedAt = time.Time{} },
		"future":  func(a *App) { a.seasonFuturePicks.fetchedAt = time.Time{} },
		"season":  func(a *App) { a.season++ },
		"owner":   func(a *App) { a.seasonFuturePicks.value[0].Owner = "missing" },
	} {
		t.Run(name, func(t *testing.T) {
			a := nativeSeedTestApp(t)
			input := createNativeSeedLeague(t, a, commissioner.PresetLegacy)
			breakInput(a)
			if _, err := a.TargetNativeSeed(input); err == nil {
				t.Fatal("incomplete or mixed snapshot accepted")
			}
			if assets := readNativeTest(t, a, input.League); assets.Seeded || assets.Revision != 1 {
				t.Fatalf("refused seed wrote assets: %+v", assets)
			}
		})
	}
}

func TestEmptyNativeLeagueHasNoSeed(t *testing.T) {
	a := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, a, commissioner.PresetEmpty)
	if _, err := a.TargetNativeSeed(input); err == nil {
		t.Fatal("empty league accepted a Legacy snapshot")
	}
	if assets := readNativeTest(t, a, input.League); assets.Seeded || assets.Snapshot != nil || assets.Revision != 1 {
		t.Fatalf("empty league invented a starting snapshot: %+v", assets)
	}
}

func TestNativeSeedCannotBeSuppliedByGenericIPC(t *testing.T) {
	a := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	raw, err := a.nativeSeedCommand(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.TargetCommissionerExecute(string(body)); err == nil {
		t.Fatal("caller-supplied starting assets accepted through generic IPC")
	}
	if assets := readNativeTest(t, a, input.League); assets.Seeded {
		t.Fatal("generic IPC wrote supplied assets")
	}
}

func TestNativeAssetsWire(t *testing.T) {
	a := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	seedNativeTest(t, a, input)
	assets := readNativeTest(t, a, input.League)
	// The mirror records its real import instant; the wire pins only that nondeterministic date.
	assets.Snapshot.RosterAt = time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)
	body, err := json.Marshal(assets)
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "internal/commissioner/testdata/native-assets.wire.json", body, "native assets")
}

func TestHeldConferencesNameEachDivisionsConference(t *testing.T) {
	config := league.RawConfig{
		Conferences: []league.Conference{{ID: "00", Name: "AFC"}, {ID: "01", Name: " "}},
		Divisions:   []league.Division{{ID: "00", Conference: "00"}, {ID: "01", Conference: "01"}, {ID: "02"}}}
	got := heldConferences(config)
	want := map[string]string{"00": "AFC", "01": "01"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("a division carries its conference's name, or its id when unnamed, and none when it has no conference: %v", got)
	}
	if len(heldConferences(league.RawConfig{})) != 0 {
		t.Fatal("a league without conferences held none")
	}
}
