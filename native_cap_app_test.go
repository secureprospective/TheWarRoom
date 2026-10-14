package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func capApp(t *testing.T) *legalityFixture {
	t.Helper()
	a := nativeSeedTestApp(t)
	path := filepath.Join(t.TempDir(), "cap.db")
	store, pools := openDirectoryHistory(t, path)
	a.history = store
	snap := state.MirrorSnapshot{Season: 2026, Adjustments: []state.Adjustment{
		{ID: "dead-player", FranchiseID: "0002", Amount: 2_500_00000, Description: "Held dead money"},
		{ID: "credit", FranchiseID: "0002", Amount: -1_200_00000, Description: "Held cap credit"}}}
	for i, status := range []domain.RosterStatus{domain.RosterActive, domain.RosterIR, domain.RosterTaxi} {
		salaries := []domain.Money{10_987_00000, 5_555_55555, 3_333_33333}
		snap.Players = append(snap.Players, state.MirrorPlayer{MFLID: fmt.Sprintf("%04d", 99+i),
			FranchiseID: "0002", RosterStatus: status, Salary: salaries[i], ContractYear: 2028,
			ContractStatus: domain.CStatusUFA})
	}
	if _, err := a.league.Replace(t.Context(), snap); err != nil {
		t.Fatal(err)
	}
	seedNativeTest(t, a, createNativeSeedLeague(t, a, commissioner.PresetLegacy))
	return &legalityFixture{app: a, path: path, pools: pools}
}

func readCap(t *testing.T, f *legalityFixture) commissioner.NativeCapCheck {
	t.Helper()
	reading, err := f.app.TargetNativeCapCheck(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	return reading
}

func capTeam(t *testing.T, f *legalityFixture) commissioner.NativeCapFranchise {
	t.Helper()
	reading := readCap(t, f)
	for _, team := range reading.Franchises {
		if team.Franchise == "franchise.02" {
			return team
		}
	}
	t.Fatal("native franchise missing")
	return commissioner.NativeCapFranchise{}
}

func TestNativeCapReadingUsesRecordedAssetsAndSignedDeadMoney(t *testing.T) {
	f := capApp(t)
	before := readCap(t, f)
	team := capTeam(t, f)
	if !before.Seeded || before.RosterSeason != 2026 || before.RulesSeason != 2026 ||
		len(before.Franchises) != 32 || team.Name != "Held team 2" || team.Abbreviation != "H2" {
		t.Fatalf("native identity or season lost: %+v", before)
	}
	if team.CommittedCents == nil || *team.CommittedCents != 21_180_00000 ||
		team.AdjustmentCents == nil || *team.AdjustmentCents != 1_300_00000 || len(team.Charges) != 2 {
		t.Fatalf("signed adjustments or dead money lost: %+v", team)
	}
	for _, other := range before.Franchises {
		if other.Franchise != team.Franchise && (other.CommittedCents == nil || *other.CommittedCents != 0) {
			t.Fatalf("another franchise's salary was counted: %+v", other)
		}
	}
	refresh := state.MirrorSnapshot{Season: 2027, Players: []state.MirrorPlayer{
		{MFLID: "0099", FranchiseID: "0001", RosterStatus: domain.RosterActive, Salary: 999_000_00000}}}
	if _, err := f.app.league.Replace(t.Context(), refresh); err != nil {
		t.Fatal(err)
	}
	if got := readCap(t, f); !reflect.DeepEqual(got, before) {
		t.Fatal("native cap borrowed a refreshed MFL mirror")
	}
	f.reopen(t)
	if got := readCap(t, f); !reflect.DeepEqual(got, before) {
		t.Fatal("native cap changed after physical reopen without a mirror")
	}
}

func TestNativeCapAmountAndSharesGovernActualReading(t *testing.T) {
	f := capApp(t)
	f.governs(t, "cap.amount", `2000000000`,
		func(t *testing.T, f *legalityFixture) string { return capTeam(t, f).Words },
		"Within the salary cap; $103,820,000.00 cap room", "Over the salary cap by $1,180,000.00")
	read := func(t *testing.T, f *legalityFixture) string {
		team := capTeam(t, f)
		return fmt.Sprintf("%d", *team.CommittedCents)
	}
	f.governs(t, "cap.irShare", `5000`, read, "2118000000", "1840000000")
	f.governs(t, "cap.taxiShare", `0`, read, "2118000000", "1785000000")
	f.governs(t, "cap.quantum", `100`, read, "2118000000", "2117588900")
}

func TestNativeCapEnforcementAndDeadlineGovernActualReading(t *testing.T) {
	f := capApp(t)
	f.set(t, "cap.amount", `2000000000`, "now")
	read := func(t *testing.T, f *legalityFixture) string {
		team := capTeam(t, f)
		allow, words := nativecheck.CapAllows(team.CapState, 0)
		if words != team.Enforcement {
			t.Fatalf("reading and pure move decision diverged: %s / %s", words, team.Enforcement)
		}
		return fmt.Sprintf("%v %s", allow, team.Enforcement)
	}
	f.governs(t, "cap.enforcement", `"soft"`, read,
		"false A move that ends over the salary cap is refused",
		"true Over the salary cap: flagged; compliance deadline needs a ruling")
	f.set(t, "cap.enforcement", `"soft"`, "now")
	f.governs(t, "cap.complianceDeadline", `"2026-10-20T00:00:00Z"`, read,
		"true Over the salary cap: flagged; compliance deadline needs a ruling",
		"true Over the salary cap: flagged; comply by 2026-10-20T00:00:00Z")
}

func TestNativeCapRulesAndCompensationSurviveReopen(t *testing.T) {
	f := capApp(t)
	before := capTeam(t, f).Words
	receipt := f.set(t, "cap.amount", `2000000000`, "now")
	f.reopen(t)
	if got := capTeam(t, f).Words; got != "Over the salary cap by $1,180,000.00" {
		t.Fatal("cap ruling was not replayed: " + got)
	}
	f.revert(t, receipt)
	f.reopen(t)
	if got := capTeam(t, f).Words; got != before {
		t.Fatal("cap compensation was not replayed: " + got)
	}
}

func TestNativeCapWaitsForSeasonEffectiveRules(t *testing.T) {
	f := capApp(t)
	f.set(t, "calendar.rollover", `"2027-03-01T05:00:00Z"`, "now")
	f.set(t, "cap.amount", `2000000000`, "next_rollover")
	f.reopen(t)
	if team := capTeam(t, f); *team.CapCents != 125_000_00000 {
		t.Fatal("pending cap counted early")
	}
	f.app.history.SetClock(func() time.Time { return time.Date(2027, 3, 1, 5, 0, 0, 0, time.UTC) })
	reading := readCap(t, f)
	if reading.RulesSeason != 2027 || reading.RosterSeason != 2026 || *capTeam(t, f).CapCents != 20_000_00000 {
		t.Fatal("cap did not use season-effective rules")
	}
}

func TestNativeCapUnseededAndInvalidScope(t *testing.T) {
	a := nativeSeedTestApp(t)
	createNativeSeedLeague(t, a, commissioner.PresetEmpty)
	reading, err := a.TargetNativeCapCheck(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	if reading.Seeded || len(reading.Franchises) != 0 || !strings.Contains(reading.Source, "no starting snapshot") {
		t.Fatalf("unseeded cap invented rosters: %+v", reading)
	}
	for _, scope := range []string{"", "14432", "native.missing", " native.seed"} {
		if _, err := a.TargetNativeCapCheck(scope); err == nil {
			t.Fatalf("invalid scope accepted: %q", scope)
		}
	}
}

func TestNativeCapWire(t *testing.T) {
	f := capApp(t)
	f.set(t, "cap.amount", `2000000000`, "now")
	f.set(t, "cap.enforcement", `"soft"`, "now")
	body, err := json.Marshal(readCap(t, f))
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "internal/commissioner/testdata/native-cap-check.wire.json", body, "native cap")
}
