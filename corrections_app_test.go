package main

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func assetCorrection(t *testing.T, f *draftFixture, command string, c commissioner.RawCorrection) commissioner.Receipt {
	t.Helper()
	return f.command(t, command, commissioner.RawArgs{Correction: &c}, "Repair the recorded asset")
}

func correctionAssets(t *testing.T, f *draftFixture) commissioner.NativeAssets {
	t.Helper()
	a, err := newCommissionerAPI(f.app).TargetNativeAssets(draftLeague)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestAssetCorrectionsGovernRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	for pass := range 2 {
		if pass == 1 {
			f.reopen(t)
		}
		before := correctionAssets(t, f)
		original, err := json.Marshal(before.Snapshot)
		if err != nil {
			t.Fatal(err)
		}
		p := before.Snapshot.Players[0]
		committedBefore := f.committed(t, p.Franchise)
		remove := assetCorrection(t, f, commissioner.CorrectRoster,
			commissioner.RawCorrection{Player: p.ID, Franchise: p.Franchise, Action: "remove"})
		after := correctionAssets(t, f)
		if len(after.Snapshot.Players) != len(before.Snapshot.Players)-1 || len(after.Corrections) == 0 {
			t.Fatal("roster correction did not remove one player with a public explanation")
		}
		checkCorrectionReadings(t, f, len(before.Snapshot.Players)-1)
		if got := f.committed(t, p.Franchise); got != committedBefore-int64(p.Salary) {
			t.Fatalf("cap still counts corrected-away salary: %d", got)
		}
		records, err := f.app.leagueRecords(t.Context(), draftLeague)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range records {
			if r.Command.Command != commissioner.NativeSeed {
				continue
			}
			held, err := json.Marshal(r.Command.Args.Native)
			if err != nil {
				t.Fatal(err)
			}
			if string(held) != string(original) {
				t.Fatal("correction rewrote the original snapshot")
			}
		}
		add := assetCorrection(t, f, commissioner.CorrectRoster,
			commissioner.RawCorrection{Player: p.ID, Franchise: p.Franchise, Action: "add"})
		if len(correctionAssets(t, f).Snapshot.Players) != len(before.Snapshot.Players) {
			t.Fatal("add lost the player")
		}
		f.revert(t, add)
		f.revert(t, remove)
		if !reflect.DeepEqual(correctionAssets(t, f).Snapshot, before.Snapshot) {
			t.Fatal("roster compensation did not restore")
		}
		provenance := assetCorrection(t, f, commissioner.CorrectProvenance,
			commissioner.RawCorrection{Player: p.ID, Via: "trade"})
		got := correctionAssets(t, f).Snapshot.Players[0]
		if got.AcquiredVia != "trade" || !strings.Contains(got.Correction, "Repair the recorded asset") {
			t.Fatal("provenance or public reason missing")
		}
		f.reopen(t)
		if correctionAssets(t, f).Snapshot.Players[0].AcquiredVia != "trade" {
			t.Fatal("provenance lost on replay")
		}
		f.revert(t, provenance)
		if !reflect.DeepEqual(correctionAssets(t, f).Snapshot, before.Snapshot) {
			t.Fatal("provenance compensation failed")
		}
	}
}

func checkCorrectionReadings(t *testing.T, f *draftFixture, count int) {
	t.Helper()
	roster, err := newCommissionerAPI(f.app).TargetNativeRosterCheck(draftLeague)
	if err != nil {
		t.Fatal(err)
	}
	capCheck, err := newCommissionerAPI(f.app).TargetNativeCapCheck(draftLeague, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if len(roster.Corrections) == 0 || len(capCheck.Corrections) == 0 {
		t.Fatal("native legality or cap omitted commissioner correction")
	}
	total := 0
	for _, r := range roster.Franchises {
		total += r.Counted
	}
	assets := correctionAssets(t, f)
	active := 0
	for _, p := range assets.Snapshot.Players {
		if p.Status == domain.RosterActive {
			active++
		}
	}
	if total != active || len(assets.Snapshot.Players) != count {
		t.Fatal("roster legality read the old roster")
	}
}

func TestCorrectedPickGovernsDraftOrderAndReplay(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t)
	for pass := range 2 {
		if pass == 1 {
			f.reopen(t)
		}
		before := f.read(t, draftTeam(1))
		c := commissioner.RawCorrection{Pick: &commissioner.CorrectionPick{Season: 2027, Round: 1, Original: draftTeam(2)}, Owner: draftTeam(3)}
		receipt := assetCorrection(t, f, commissioner.CorrectPick, c)
		after := f.read(t, draftTeam(1))
		if len(after.Corrections) == 0 {
			t.Fatal("draft omitted correction provenance")
		}
		found := false
		for _, r := range after.Order {
			for _, p := range r.Picks {
				if r.Round == 1 && p.Original == draftTeam(2) {
					found = true
					if p.Owner != draftTeam(3) {
						t.Fatal("corrected pick still has its old owner in draft order")
					}
				}
			}
		}
		if !found {
			t.Fatal("corrected pick disappeared")
		}
		f.reopen(t)
		if !reflect.DeepEqual(f.read(t, draftTeam(1)).Order, after.Order) {
			t.Fatal("corrected order lost on reopen")
		}
		f.revert(t, receipt)
		if !reflect.DeepEqual(f.read(t, draftTeam(1)).Order, before.Order) {
			t.Fatal("pick compensation did not restore order")
		}
	}
}

func TestCorrectionsRequireReasonAndPermission(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	p := correctionAssets(t, f).Snapshot.Players[0]
	c := commissioner.RawCorrection{Player: p.ID, Via: "draft"}
	before := f.revision(t)
	if _, err := f.tryCommand(commissioner.CorrectProvenance, commissioner.RawArgs{Correction: &c}, " "); err == nil {
		t.Fatal("correction accepted without typed reason")
	}
	f.actAs("outsider")
	if _, err := f.tryCommand(commissioner.CorrectProvenance, commissioner.RawArgs{Correction: &c}, "Correct provenance"); err == nil {
		t.Fatal("correction accepted without commissioner permission")
	}
	if f.revision(t) != before {
		t.Fatal("refusal changed the ledger")
	}
}

func TestRuleExceptionGovernsRealCalendarAndExpiresExactly(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.setRule(t, "calendar.windows", `{}`)
	until := f.now.Add(time.Hour)
	c := commissioner.RawCorrection{Rule: "calendar.lineup", Scope: &commissioner.ExceptionScope{Franchise: draftTeam(1)},
		Until: until.Format(time.RFC3339Nano)}
	for pass := range 2 {
		if pass == 1 {
			f.reopen(t)
		}
		records, err := f.app.leagueRecords(t.Context(), draftLeague)
		if err != nil {
			t.Fatal(err)
		}
		gate, err := commissioner.NativeActionGate(draftLeague, records, draftTeam(1), "lineup", f.now, time.UTC)
		if err != nil || gate.Open {
			t.Fatalf("baseline gate: %+v %v", gate, err)
		}
		receipt := assetCorrection(t, f, commissioner.RuleException, c)
		assertExceptionGate(t, f, f.now, true)
		assertExceptionGate(t, f, until.Add(-time.Nanosecond), true)
		assertExceptionGate(t, f, until, false)
		f.reopen(t)
		assertExceptionGate(t, f, f.now, true)
		f.revert(t, receipt)
		assertExceptionGate(t, f, f.now, false)
	}
}

func assertExceptionGate(t *testing.T, f *draftFixture, at time.Time, open bool) {
	t.Helper()
	records, err := f.app.leagueRecords(t.Context(), draftLeague)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := commissioner.NativeActionGate(draftLeague, records, draftTeam(1), "lineup", at, time.UTC)
	if err != nil || gate.Open != open {
		t.Fatalf("exception gate at %s: %+v %v", at, gate, err)
	}
	reading, err := commissioner.CalendarReading(draftLeague, records, at, 2027, "lineup", draftTeam(1), time.UTC)
	if err != nil || reading.Decision.Open != open || reading.Clock.Decide(at, "lineup", draftTeam(1)).Open != open {
		t.Fatalf("calendar enforcement diverged: %+v %v", reading.Decision, err)
	}
	other, err := commissioner.NativeActionGate(draftLeague, records, draftTeam(2), "lineup", at, time.UTC)
	if err != nil || other.Open {
		t.Fatalf("exception escaped franchise scope: %+v %v", other, err)
	}
}

func TestCorrectionsWireMatchesGo(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	p := correctionAssets(t, f).Snapshot.Players[0]
	assetCorrection(t, f, commissioner.CorrectProvenance, commissioner.RawCorrection{Player: p.ID, Via: "waiver"})
	assetCorrection(t, f, commissioner.RuleException, commissioner.RawCorrection{Rule: "calendar.lineup",
		Scope: &commissioner.ExceptionScope{Franchise: draftTeam(1)}})
	reading, err := newCommissionerAPI(f.app).TargetCorrections(draftLeague)
	if err != nil {
		t.Fatal(err)
	}
	if len(reading.Items) != 2 || reading.Items[0].Rule != "calendar.lineup" || reading.Items[0].Why == "" {
		t.Fatal("reading is not newest-first with a public reason")
	}
	for i := range reading.Items {
		reading.Items[i].Entry = "entry." + reading.Items[i].What
	}
	body, err := json.MarshalIndent(reading, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := "internal/commissioner/testdata/corrections-wire.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(held, body) {
		t.Fatal("corrections wire differs from Go")
	}
}

func TestCorrectionUsesSharedVersionRetryAndCoPermission(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	p := correctionAssets(t, f).Snapshot.Players[0]
	executeCommissionerTest(t, f.app, commissioner.RawCommand{League: draftLeague, Command: "commish.role.grant",
		Args:     commissioner.RawArgs{People: &commissioner.RawPeople{Principal: "cc", Role: governance.CoCommissioner}},
		Expected: f.revision(t), Request: "grant.cc", Reason: "Seat a co-commissioner", Effective: "now", Visibility: "private"})
	f.actAs("cc")
	raw := commissioner.RawCommand{League: draftLeague, Command: commissioner.CorrectProvenance,
		Args:     commissioner.RawArgs{Correction: &commissioner.RawCorrection{Player: p.ID, Via: "draft"}},
		Expected: f.revision(t), Request: "correction.retry", Reason: "Repair the acquisition", Effective: "now", Visibility: "league"}
	send := func(r commissioner.RawCommand) (commissioner.Receipt, error) {
		body, err := json.Marshal(r)
		if err != nil {
			return commissioner.Receipt{}, err
		}
		return newCommissionerAPI(f.app).TargetCommissionerExecute(string(body))
	}
	receipt, err := send(raw)
	if err != nil {
		t.Fatalf("co-commissioner could not correct: %v", err)
	}
	again, err := send(raw)
	if err != nil || !reflect.DeepEqual(receipt, again) || f.revision(t) != raw.Expected+1 {
		t.Fatal("request-key retry appended twice or changed its receipt", err)
	}
	raw.Request = "new.request.old.version"
	if _, err := send(raw); err == nil || !strings.Contains(err.Error(), "reread") {
		t.Fatalf("stale correction needs a reread refusal, not a storage failure: %v", err)
	}
	f.revert(t, receipt)
	if correctionAssets(t, f).Snapshot.Players[0].AcquiredVia != p.AcquiredVia {
		t.Fatal("co-commissioner could not compensate")
	}
}
