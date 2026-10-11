package snapshot

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/eligibility"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// snapshotInjuries is a live injuries feed holding one IR designation, the case the rules
// turn on.
func snapshotInjuries() Injuries {
	return Injuries{
		Status: map[string]string{"2002": "IR-R"},
		Provenance: Provenance{Source: "injuries", Kind: "live",
			Freshness: domain.Freshness{State: domain.FreshLive, FetchedAt: "2026-10-05T20:00:00Z"}},
	}
}

type fakeSource struct {
	meta    Metadata
	rosters map[string][]state.PlayerState
	ids     []string
}

func (s fakeSource) Metadata() (Metadata, error) { return s.meta, nil }
func (s fakeSource) Franchises() []string        { return append([]string{}, s.ids...) }
func (s fakeSource) Roster(id string) ([]state.PlayerState, bool) {
	p, ok := s.rosters[id]
	return p, ok
}
func (s fakeSource) CapUsed(id string) (domain.Money, bool) {
	p, ok := s.rosters[id]
	var used domain.Money
	for _, row := range p {
		used += row.Salary
	}
	return used, ok
}

func (s fakeSource) Slots() Slots {
	return Slots{IR: eligibility.Capacity{Slots: 12, Held: true},
		Taxi: eligibility.Capacity{Slots: 8, Held: true}}
}

func testSource() fakeSource {
	capAmount := domain.Money(10000000)
	return fakeSource{
		meta: Metadata{Season: 2026, SalaryCap: &capAmount, Names: map[string]string{"0002": "Second", "0001": "First"}, MirrorAsOf: "2026-10-05T10:00:00Z"},
		ids:  []string{"0002", "0001"},
		rosters: map[string][]state.PlayerState{
			"0001": {{MFLID: "2002", Salary: 200000, RosterStatus: domain.RosterTaxi, ContractStatus: domain.CStatusRFA, ExpirationYear: 2027}, {MFLID: "0042", Salary: 300000, RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA, ExpirationYear: 2026}},
			"0002": {{MFLID: "2001", Salary: 400000, RosterStatus: domain.RosterIR, ContractStatus: domain.CStatusFT1}},
		},
	}
}

func testDirectory(t *testing.T) Directory {
	t.Helper()
	lk, err := normalize.NewLookup([]players.RawPlayer{
		{ID: "0042", Name: "Real, Name", Position: "QB", Team: "BUF", Status: "R", Birthdate: "1000000000"},
		{ID: "2001", Name: "Other, Name", Position: "LB", Team: "FA"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return Directory{Lookup: lk, Provenance: Provenance{Source: "mfl-players-archive", Kind: "live", Freshness: domain.Freshness{State: domain.FreshStale, FetchedAt: "2026-10-04T10:00:00Z", Note: "archived copy"}}}
}

func TestBuildJoinedFactsAndContracts(t *testing.T) {
	got, err := Build(context.Background(), testSource(), testDirectory(t), snapshotInjuries())
	if err != nil {
		t.Fatal(err)
	}
	if got.Franchises.Value[0].ID != "0001" || got.Rosters.Value[0].Players[0].ID.String() != "0042" || got.Players.Value[0].ID.String() != "0042" {
		t.Fatalf("ids/order: %+v", got)
	}
	p := got.Players.Value[0]
	if p.Name != "Real, Name" || p.Team != "BUF" || p.Position != domain.PosQB || p.Birthdate == nil || *p.Birthdate != 1000000000 || p.IsRookie == nil || !*p.IsRookie {
		t.Fatalf("facts: %+v", p)
	}
	if got.Players.Value[1].Birthdate != nil || got.Players.Value[2].Name != "" || got.Players.Value[2].IsRookie != nil {
		t.Fatalf("absent facts invented: %+v", got.Players)
	}
	if !strings.Contains(got.Players.Provenance.Freshness.Note, "1 unnamed rostered players") {
		t.Fatal(got.Players.Provenance)
	}
	r := got.Rosters.Value[0].Players[0]
	if r.YearsRemaining == nil || *r.YearsRemaining != 1 || r.Salary != 300000 || r.RosterStatus != domain.RosterActive || r.ContractStatus != domain.CStatusUFA {
		t.Fatal(r)
	}
	if got.Franchises.Value[0].CapRoom == nil || *got.Franchises.Value[0].CapRoom != 9500000 {
		t.Fatal(got.Franchises)
	}
	assertProvenance(t, got)
}

func assertProvenance(t *testing.T, got Snapshot) {
	t.Helper()
	for _, p := range []Provenance{got.League.Provenance, got.Franchises.Provenance, got.Rosters.Provenance, got.Players.Provenance} {
		if p.Source == "" || p.Kind == "" || p.Freshness.State == "" {
			t.Fatalf("provenance: %+v", p)
		}
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"value":null`) {
		t.Fatalf("nil section: %s", data)
	}
}

func TestBuildMissingSections(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		mirror, rules, directory bool
	}{
		{"empty mirror", false, true, true}, {"no rulebook", true, false, true}, {"no directory", true, true, false}, {"empty all", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src, dir := testSource(), testDirectory(t)
			if !tc.mirror {
				src.meta.Season = 0
				src.meta.MirrorAsOf = ""
				src.rosters = nil
				src.ids = nil
			}
			if !tc.rules {
				src.meta.Names = nil
				src.meta.SalaryCap = nil
			}
			if !tc.directory {
				dir = Directory{}
			}
			got, err := Build(context.Background(), src, dir, snapshotInjuries())
			if err != nil {
				t.Fatal(err)
			}
			assertProvenance(t, got)
			if !tc.mirror && (got.League.Value != (League{}) || got.League.Provenance.Freshness.State != domain.FreshFail || len(got.Rosters.Value) != 0 || got.Rosters.Provenance.Freshness.State != domain.FreshFail) {
				t.Fatal(got)
			}
			if !tc.rules && (len(got.Franchises.Value) != 0 || got.Franchises.Provenance.Freshness.State != domain.FreshFail) {
				t.Fatal(got.Franchises)
			}
			if (!tc.directory || !tc.mirror) && (len(got.Players.Value) != 0 || got.Players.Provenance.Freshness.State != domain.FreshFail) {
				t.Fatal(got.Players)
			}
		})
	}
}

func TestBuildStableOrderAndUnknownCap(t *testing.T) {
	src, dir := testSource(), testDirectory(t)
	first, err := Build(context.Background(), src, dir, snapshotInjuries())
	if err != nil {
		t.Fatal(err)
	}
	src.ids[0], src.ids[1] = src.ids[1], src.ids[0]
	src.rosters["0001"][0], src.rosters["0001"][1] = src.rosters["0001"][1], src.rosters["0001"][0]
	second, err := Build(context.Background(), src, dir, snapshotInjuries())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("unstable ordering: %+v vs %+v", first, second)
	}
	src.meta.SalaryCap = nil
	got, err := Build(context.Background(), src, dir, snapshotInjuries())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range got.Franchises.Value {
		if f.CapRoom != nil || f.CapUsed == nil {
			t.Fatal(f)
		}
	}
}

func TestBuildRejectsBadIDsDuplicatesAndCancellation(t *testing.T) {
	for _, id := range []string{"bad", "0042"} {
		src := testSource()
		src.rosters["0002"][0].MFLID = id
		if _, err := Build(context.Background(), src, testDirectory(t), snapshotInjuries()); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Build(ctx, testSource(), Directory{}, snapshotInjuries()); err == nil {
		t.Fatal("accepted cancelled build")
	}
}

func TestBuildEligibilityVerdictsAndFeeds(t *testing.T) {
	// The directory's provenance is stale in this fixture, so taxi verdicts must read as
	// not-checked rather than a guess; the live injuries feed carries the IR verdicts.
	dir := testDirectory(t)
	dir.Provenance = Provenance{Source: "mfl-players", Kind: "live",
		Freshness: domain.Freshness{State: domain.FreshLive, FetchedAt: "2026-10-05T20:00:00Z"}}
	lk, err := normalize.NewLookup([]players.RawPlayer{
		{ID: "0042", Name: "Rookie, Real", Position: "QB", Team: "BUF", Status: "R", DraftYear: "2026"},
		{ID: "2001", Name: "Old, Hand", Position: "LB", Team: "FA", DraftYear: "2019"},
	})
	if err != nil {
		t.Fatal(err)
	}
	dir.Lookup = lk
	got, err := Build(context.Background(), testSource(), dir, snapshotInjuries())
	if err != nil {
		t.Fatal(err)
	}
	// 0042: active, not injury-designated, drafted 2026.
	p42 := got.Eligibility.Players["0042"]
	if p42.IR.Verdict != eligibility.Ineligible || p42.IR.Reason != "not on NFL injured reserve" ||
		p42.IR.Source != eligibility.SourceInjuries || p42.IR.Direction != eligibility.ToIR {
		t.Fatalf("0042 IR = %+v", p42.IR)
	}
	if p42.Taxi.Verdict != eligibility.Eligible || p42.Taxi.Source != eligibility.SourcePlayers {
		t.Fatalf("0042 taxi = %+v", p42.Taxi)
	}
	// 2002: on taxi in the fixture roster, so its taxi slot is a promotion verdict and MFL
	// offers no IR deactivation from the taxi squad.
	p02 := got.Eligibility.Players["2002"]
	if p02.Taxi.Direction != eligibility.OffTaxi || p02.Taxi.Verdict != eligibility.Eligible ||
		p02.IR.Verdict != eligibility.Ineligible || p02.IR.Reason != "on the taxi squad" {
		t.Fatalf("2002 = %+v", p02)
	}
	// 2001: on IR in the fixture roster, so its IR slot is an activation verdict.
	p01 := got.Eligibility.Players["2001"]
	if p01.IR.Direction != eligibility.OffIR || p01.IR.Verdict != eligibility.Eligible ||
		p01.Taxi.Verdict != eligibility.Ineligible || p01.Taxi.Reason != "on injured reserve" {
		t.Fatalf("2001 = %+v", p01)
	}
	feeds := got.Eligibility.Feeds
	for _, source := range []string{eligibility.SourceRosters, eligibility.SourceInjuries, eligibility.SourcePlayers} {
		if _, ok := feeds[source]; !ok {
			t.Fatalf("feed provenance %s missing: %+v", source, feeds)
		}
	}
	if feeds[eligibility.SourceInjuries].Source != "injuries" ||
		feeds[eligibility.SourceInjuries].Freshness.State != domain.FreshLive {
		t.Fatalf("injuries provenance: %+v", feeds[eligibility.SourceInjuries])
	}
}

func TestBuildEligibilityStaleFeedsReadUnknown(t *testing.T) {
	stale := Injuries{Provenance: Provenance{Source: "injuries", Kind: "live",
		Freshness: domain.Freshness{State: domain.FreshStale, FetchedAt: "2026-10-01T20:00:00Z"}}}
	got, err := Build(context.Background(), testSource(), testDirectory(t), stale)
	if err != nil {
		t.Fatal(err)
	}
	// 0042 is an active-roster player, so both of its slots read the feeds.
	v := got.Eligibility.Players["0042"]
	if v.IR.Verdict != eligibility.Unknown || v.IR.Reason != "stale" {
		t.Fatalf("stale injuries = %+v, want unknown stale", v.IR)
	}
	if v.Taxi.Verdict != eligibility.Unknown || v.Taxi.Reason != "stale" {
		t.Fatalf("stale directory = %+v, want unknown stale", v.Taxi)
	}
	if got.Eligibility.Feeds[eligibility.SourceRosters].Freshness.State != domain.FreshLive {
		t.Fatalf("rosters feed must stay live: %+v", got.Eligibility.Feeds)
	}
	// The stale copy's own time rides the feeds section untouched: it is the as-of time the
	// eligibility lines carry, including not-checked ones.
	if got.Eligibility.Feeds[eligibility.SourceInjuries].Freshness.FetchedAt != "2026-10-01T20:00:00Z" ||
		got.Eligibility.Feeds[eligibility.SourceInjuries].Freshness.State != domain.FreshStale {
		t.Fatalf("stale injuries provenance lost its time: %+v",
			got.Eligibility.Feeds[eligibility.SourceInjuries])
	}
}
