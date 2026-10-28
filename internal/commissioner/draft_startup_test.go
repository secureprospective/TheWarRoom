package commissioner

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// startupLeague is a Default32 app league started with empty rosters, its startup draft two rounds long.
func startupLeague(t *testing.T) *ledgerBuilder {
	t.Helper()
	b := newLedger(t, PresetDefault32)
	b.set("draft.startupRounds", `2`, "now", seasonMoment(2))
	seed, err := NativeStart("native.test", b.records, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &seed}, Request: "seed",
		Effective: "now", Visibility: "league"}, seasonMoment(2))
	return b
}

// startupFacts are 80 held players at every drafted position, none of them this season's rookies.
func startupFacts() nativecheck.Facts {
	positions := []leaguerules.Position{leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.TE,
		leaguerules.PK, leaguerules.DT, leaguerules.DE, leaguerules.LB, leaguerules.CB, leaguerules.S}
	facts := nativecheck.Facts{}
	for i := range 80 {
		facts[fmt.Sprintf("%05d", 20000+i)] = nativecheck.PlayerFacts{Name: fmt.Sprintf("Player %02d", i),
			Position: positions[i%len(positions)], Team: "KC", DraftYear: 2018, DraftYearHeld: true}
	}
	return facts
}

func startupRaw(t *testing.T, command string, request string) RawCommand {
	t.Helper()
	raw, err := Decode(draftBody(t, command, RawDraftArgs{Season: 2026, Startup: true}, "Start the league"))
	if err != nil {
		t.Fatal(err)
	}
	raw.Request = request
	return raw
}

// record checks, prepares and appends a commissioner's draft change the way the store does.
func (b *ledgerBuilder) record(raw RawCommand, facts nativecheck.Facts, at time.Time) {
	b.t.Helper()
	if err := CheckDraftAppend(raw, b.records, at, nil); err != nil {
		b.t.Fatalf("%s refused: %v", raw.Command, err)
	}
	if err := PrepareDraftAppend(&raw, b.records, at, facts, nil); err != nil {
		b.t.Fatalf("%s not prepared: %v", raw.Command, err)
	}
	b.add(raw, at)
}

// runStartup opens the startup draft and makes every pick in turn, each with the first player left.
func runStartup(t *testing.T, b *ledgerBuilder, facts nativecheck.Facts) time.Time {
	t.Helper()
	at := seasonMoment(3)
	b.record(startupRaw(t, DraftStartupOpen, "open"), facts, at)
	for i := 0; ; i++ {
		at = at.Add(time.Minute)
		basis, err := readDraft(b.records, startupKey(b.records), at, nil)
		if err != nil || basis.words != "" {
			t.Fatalf("read: %v %q", err, basis.words)
		}
		clock := basis.draft.OnClock
		if clock == nil {
			return at
		}
		taken := map[string]bool{}
		for _, p := range basis.draft.Picks {
			taken[p.Player] = true
		}
		player := startupPool(facts, taken)[0].ID
		raw, err := PlanDraftPick(DraftPickInput{League: "native.test", Season: 2026, Startup: true,
			Franchise: clock.Owner, Player: player, Request: fmt.Sprintf("pick%d", i), Expected: int64(len(b.records))},
			b.records, facts, at, time.UTC, nil)
		if err != nil {
			t.Fatalf("pick %d: %v", i, err)
		}
		b.add(raw, at)
	}
}

func hasJob(t *testing.T, records []Record, id string, at time.Time) bool {
	t.Helper()
	jobs, err := JobsReading("native.test", records, at)
	if err != nil {
		t.Fatal(err)
	}
	return slices.ContainsFunc(jobs.Jobs, func(j Job) bool { return j.ID == id })
}

func TestStartupDraftJobFollowsFirstRostersSourceAndStart(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(b *ledgerBuilder)
		shown bool
	}{
		{"drafted first rosters", func(*ledgerBuilder) {}, true},
		{"free agency", func(b *ledgerBuilder) { b.set("league.initialRosters", `"free_agency"`, "now", seasonMoment(2)) }, false},
		{"imported from MFL", func(b *ledgerBuilder) { b.records[0].Command.Args.Source = SourceMFLImport }, false},
		{"rosters at the start", func(b *ledgerBuilder) {
			native := b.records[2].Command.Args.Native
			native.Players = []NativePlayer{{ID: "20000", Franchise: "franchise.01", Status: domain.RosterActive,
				ExpirationYear: 2026, ContractStatus: domain.CStatusUFA}}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := startupLeague(t)
			tc.setup(b)
			if got := hasJob(t, b.records, "startup_draft", seasonMoment(3)); got != tc.shown {
				t.Fatalf("startup draft job shown=%v, want %v", got, tc.shown)
			}
			err := CheckDraftAppend(startupRaw(t, DraftStartupOpen, "open"), b.records, seasonMoment(3), nil)
			if (err == nil) != tc.shown {
				t.Fatalf("open refused=%v, want refused=%v", err, !tc.shown)
			}
		})
	}
}

func TestStartupDraftSignsEveryPickAndClosesWithoutIssuingPicks(t *testing.T) {
	b, facts := startupLeague(t), startupFacts()
	at := runStartup(t, b, facts)
	if err := CheckDraftAppend(startupRaw(t, DraftStartupOpen, "again"), b.records, at, nil); err == nil {
		t.Fatal("a second opening was admitted")
	}
	due := DueDrafts(b.records, at, nil)
	if !reflect.DeepEqual(due, []DraftKey{startupKey(b.records)}) {
		t.Fatalf("due drafts %v", due)
	}
	b.record(startupRaw(t, DraftArchive, "close"), facts, at)
	assets, err := NativeAssetsReading("native.test", b.records)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets.Snapshot.Picks) != 0 || len(assets.Snapshot.Pool) != 32 {
		t.Fatalf("close issued %d picks, pool %d", len(assets.Snapshot.Picks), len(assets.Snapshot.Pool))
	}
	held := map[string]int{}
	for _, p := range assets.Snapshot.Players {
		held[p.Franchise]++
		if p.Salary != 33000000 || p.ExpirationYear != 2026 || p.RookieRound != 0 ||
			!strings.HasSuffix(p.ContractInfo, "of the 2026 startup draft") {
			t.Fatalf("player signed %+v", p)
		}
	}
	if len(assets.Snapshot.Players) != 64 || len(held) != 32 || held["franchise.07"] != 2 {
		t.Fatalf("rosters %v", held)
	}
	if hasJob(t, b.records, "startup_draft", at) {
		t.Fatal("a closed startup draft still has a job")
	}
	if label := recordLabel(b.records[len(b.records)-1]); !strings.Contains(label, "no picks were issued") {
		t.Fatalf("close label %q", label)
	}
}

func TestStartupDraftOpensOwnersPicksUntilItCloses(t *testing.T) {
	b, facts := startupLeague(t), startupFacts()
	pickOpen := func(at time.Time) bool {
		var clock leagueclock.ActionCalendar
		projectStartupCalendar(&clock, b.records, 2026, at)
		return clock.Decide(at, draftPickAction, "franchise.01").Open
	}
	if pickOpen(seasonMoment(3)) {
		t.Fatal("the startup draft opened picks before it opened")
	}
	at := runStartup(t, b, facts)
	if !pickOpen(at) || !pickOpen(seasonMoment(3).Add(time.Second)) {
		t.Fatal("owners' picks are closed while the startup draft runs")
	}
	b.record(startupRaw(t, DraftArchive, "close"), facts, at)
	if pickOpen(at.Add(time.Minute)) {
		t.Fatal("owners' picks stayed open after the startup draft closed")
	}
}

func TestStartupDraftReplaysItsRecordedSeed(t *testing.T) {
	b, facts := startupLeague(t), startupFacts()
	at := runStartup(t, b, facts)
	body, err := json.Marshal(b.records)
	if err != nil {
		t.Fatal(err)
	}
	var again []Record
	if err := json.Unmarshal(body, &again); err != nil {
		t.Fatal(err)
	}
	first, err := NativeAssetsReading("native.test", b.records)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NativeAssetsReading("native.test", again)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Snapshot.Players) != 64 || !reflect.DeepEqual(first.Snapshot.Players, second.Snapshot.Players) {
		t.Fatal("the replayed startup draft gave different rosters")
	}
	open, _ := startupOpened(b.records, startupKey(b.records))
	seed, err := startupSeed(open.Command.Args.DraftOutcome.Seed)
	if err != nil {
		t.Fatal(err)
	}
	s := rookiedraft.Startup{Order: leaguerules.RandomOrder}
	drawn, err := rookiedraft.StartupFirstRound(s, b.records[0].Command.Args.Franchises, seed)
	if err != nil {
		t.Fatal(err)
	}
	basis, err := readDraft(again, startupKey(again), at, nil)
	if err != nil {
		t.Fatal(err)
	}
	owners := []string{}
	for _, p := range basis.order[0].Picks {
		owners = append(owners, p.Owner)
	}
	if !reflect.DeepEqual(owners, drawn) {
		t.Fatalf("first round %v, the recorded seed draws %v", owners, drawn)
	}
}

func TestStartupListMustNameEveryFranchiseOnce(t *testing.T) {
	b := startupLeague(t)
	every, err := json.Marshal(b.records[0].Command.Args.Franchises)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		value string
		ok    bool
	}{
		{`[]`, true}, {string(every), true}, {`["franchise.01"]`, false},
		{strings.Replace(string(every), "franchise.02", "franchise.01", 1), false},
		{strings.Replace(string(every), "franchise.02", "franchise.99", 1), false},
	} {
		raw := RawCommand{League: "native.test", Command: RulesSet, Request: "list",
			Args: RawArgs{Field: "draft.startupList", Value: json.RawMessage(tc.value)}, Effective: "now", Visibility: "league"}
		err := CheckDraftAppend(raw, b.records, seasonMoment(3), nil)
		if (err == nil) != tc.ok {
			t.Fatalf("%.40s: error %v, want ok=%v", tc.value, err, tc.ok)
		}
	}
}

func TestStartupListOffersTheLeaguesFranchises(t *testing.T) {
	b := startupLeague(t)
	rules, err := RulesReading("native.test", b.records, seasonMoment(3))
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range rules.Sections {
		for _, row := range section.Rows {
			if row.Field != "draft.startupList" {
				continue
			}
			choices := row.Schema.Element.Choices
			if len(choices) != 32 || choices[0] != (leaguerules.Choice{Value: "franchise.01", Label: "Franchise 1"}) {
				t.Fatalf("startup list choices %v", choices)
			}
			for _, f := range leaguerules.Catalogue() {
				if f.Path == startupListField && len(f.Schema.Element.Choices) != 0 {
					t.Fatal("the catalogue's own schema was changed")
				}
			}
			return
		}
	}
	t.Fatal("no startup list row")
}
