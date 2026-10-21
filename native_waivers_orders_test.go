package main

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestNativeWaiverOrderCommandGovernsAwardRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	f.claim(t, cut.Entry, "owner.one", "franchise.01")
	f.claim(t, cut.Entry, "owner.three", "franchise.03")
	order := waiverOrder()
	slices.Reverse(order)
	change := f.command(t, commissioner.WaiverOrderSet,
		commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Order: order}}, "league")
	f.now = f.now.Add(24 * time.Hour)
	processed := f.process(t, cut.Entry)
	assert := func(winner string, expected []string) {
		t.Helper()
		r := f.reading(t, "owner.one", "franchise.01")
		if r.Players[0].Winner != winner || !slices.Equal(r.Order, expected) || r.OrderSource != "Set by the commissioner" {
			t.Fatal("order command did not govern award and priority reading", r)
		}
	}
	for range 2 {
		assert("franchise.03", order)
		f.restart(t)
	}
	f.revert(t, processed)
	f.revert(t, change)
	f.process(t, cut.Entry)
	for range 2 {
		assert("franchise.01", waiverOrder())
		f.restart(t)
	}
}

// recordedStandingsApp records the typed native source at the real starting-snapshot command boundary.
func recordedStandingsApp(t *testing.T) *waiverFixture {
	t.Helper()
	held := waiverApp(t)
	records, err := held.app.leagueRecords(context.Background(), legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	var seed *commissioner.RawNativeSnapshot
	for _, r := range records {
		if r.Command.Command == commissioner.NativeSeed {
			clone := *r.Command.Args.Native
			seed = &clone
		}
	}
	if seed == nil {
		t.Fatal("fixture has no starting assets")
	}
	seed.StandingsOrder = waiverOrder()
	a := commissionerTestApp(t)
	path := held.path + ".standings"
	store, pools := openDirectoryHistory(t, path)
	a.core.History = store
	a.core.Lookup, a.core.HasLookup = held.app.core.Lookup, held.app.core.HasLookup
	f := &waiverFixture{legalityFixture: &legalityFixture{app: a, path: path, pools: pools}, now: held.now}
	f.clock()
	createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	raw := f.raw(t, commissioner.NativeSeed, commissioner.RawArgs{Native: seed}, "league")
	if _, err := store.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
		t.Fatal(err)
	}
	f.set(t, "roster.limits", `{"min":0,"max":100}`, "now")
	f.set(t, "cap.amount", `100000000000`, "now")
	f.set(t, "calendar.boundary", `"start_of_day"`, "now")
	f.set(t, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim"]}}`, "now")
	for _, seat := range []struct{ franchise, principal string }{{"franchise.01", "owner.one"},
		{"franchise.02", "owner.two"}, {"franchise.03", "owner.three"}} {
		f.command(t, "commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{
			Franchise: seat.franchise, Principal: seat.principal}}, "league")
	}
	return f
}

func TestNativeWaiverAutomaticOrderGovernsRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := recordedStandingsApp(t)
	order := waiverOrder()
	slices.Reverse(order)
	f.command(t, commissioner.WaiverOrderSet, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Order: order}}, "league")
	cut := f.cut(t, "1000")
	f.claim(t, cut.Entry, "owner.one", "franchise.01")
	f.claim(t, cut.Entry, "owner.three", "franchise.03")
	auto := f.command(t, commissioner.WaiverOrderAuto, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{}}, "league")
	f.now = f.now.Add(24 * time.Hour)
	processed := f.process(t, cut.Entry)
	assert := func(winner string, expected []string, source string) {
		t.Helper()
		r := f.reading(t, "owner.one", "franchise.01")
		if r.Players[0].Winner != winner || !slices.Equal(r.Order, expected) || r.OrderSource != source {
			t.Fatal("automatic order consequence", r)
		}
		standings, err := f.app.core.History.StandingsOrder(t.Context(), legalityLeague, 2026, f.now)
		if err != nil || !slices.Equal(standings, waiverOrder()) {
			t.Fatal("durable standings seam", standings, err)
		}
	}
	for range 2 {
		assert("franchise.01", waiverOrder(), "From recorded standings, lowest standing first")
		f.restart(t)
	}
	f.revert(t, processed)
	f.revert(t, auto)
	f.process(t, cut.Entry)
	for range 2 {
		assert("franchise.03", order, "Set by the commissioner")
		f.restart(t)
	}
}

func TestNativeWaiverRecordedStandingsPrioritySurvivesProcessing(t *testing.T) {
	t.Parallel()
	f := recordedStandingsApp(t)
	for _, player := range []string{"1000", "1001"} {
		cut := f.cut(t, player)
		f.claim(t, cut.Entry, "owner.one", "franchise.01")
		f.now = f.now.Add(24 * time.Hour)
		f.process(t, cut.Entry)
		r := f.reading(t, "owner.one", "franchise.01")
		if !slices.Equal(r.Order, waiverOrder()) {
			t.Fatal("processing discarded recorded standings priority", r)
		}
		f.restart(t)
	}
}

func TestNativeWaiverRollingOverrideRestoresPriorityAndReplays(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	f.set(t, "waivers.method", `"rolling"`, "now")
	cut := f.cut(t, "1000")
	f.claim(t, cut.Entry, "owner.one", "franchise.01")
	claim := f.claim(t, cut.Entry, "owner.three", "franchise.03")
	f.now = f.now.Add(24 * time.Hour)
	f.process(t, cut.Entry)
	change := f.command(t, commissioner.WaiverOverride, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Claim: claim.Entry, Franchise: "franchise.03"}}, "league")
	order := waiverOrder()
	order = append(slices.Delete(order, 2, 3), "franchise.03")
	for range 2 {
		r := f.reading(t, "owner.one", "franchise.01")
		if r.Players[0].Winner != "franchise.03" || !slices.Equal(r.Order, order) {
			t.Fatal("rolling override wrongly penalized original winner", r)
		}
		f.restart(t)
	}
	f.revert(t, change)
	expected := waiverOrder()
	expected = append(expected[1:], expected[0])
	r := f.reading(t, "owner.one", "franchise.01")
	if r.Players[0].Winner != "franchise.01" || !slices.Equal(r.Order, expected) {
		t.Fatal("rolling override compensation", r)
	}
}

func TestNativeWaiverAllCommissionerCommandsRetryAfterReopen(t *testing.T) {
	t.Parallel()
	f := recordedStandingsApp(t)
	cut := f.cut(t, "1000")
	claim := f.claim(t, cut.Entry, "owner.one", "franchise.01")
	f.now = f.now.Add(24 * time.Hour)
	commands := []struct {
		command string
		args    commissioner.RawWaiverArgs
	}{
		{commissioner.WaiverOrderSet, commissioner.RawWaiverArgs{Order: waiverOrder()}},
		{commissioner.WaiverOrderAuto, commissioner.RawWaiverArgs{}},
		{commissioner.WaiverProcess, commissioner.RawWaiverArgs{Window: cut.Entry}},
		{commissioner.WaiverOverride, commissioner.RawWaiverArgs{Claim: claim.Entry, Franchise: "franchise.01"}},
	}
	for _, command := range commands {
		raw := f.raw(t, command.command, commissioner.RawArgs{Waiver: &command.args}, "league")
		body, err := json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
		if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil {
			t.Fatalf("owner ran %s", command.command)
		}
		f.app.core.CommissionerSession = testCommissionerSession{"creator"}
		receipt, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body))
		if err != nil {
			t.Fatal(err)
		}
		f.restart(t)
		again, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body))
		if err != nil || !reflect.DeepEqual(receipt, again) {
			t.Fatalf("%s changed on replay: %+v %v", command.command, again, err)
		}
		raw.Request += ".different"
		body, err = json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil || !strings.Contains(err.Error(), "reread") {
			t.Fatalf("%s accepted a stale version: %v", command.command, err)
		}
	}
}

func TestNativeWaiverOwnerRetriesAfterCalendarCloses(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	f.app.core.CommissionerSession = testCommissionerSession{"owner.two"}
	raw := f.raw(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "1000", Franchise: "franchise.02"}}, "league")
	receipt, err := newCommissionerAPI(f.app).TargetNativeWaiverAction(raw)
	if err != nil {
		t.Fatal(err)
	}
	f.set(t, "calendar.windows", `{}`, "now")
	for range 2 {
		f.app.core.CommissionerSession = testCommissionerSession{"owner.two"}
		again, err := newCommissionerAPI(f.app).TargetNativeWaiverAction(raw)
		if err != nil || !reflect.DeepEqual(receipt, again) {
			t.Fatal("closed calendar lost original cut receipt", again, err)
		}
		f.restart(t)
	}
}

func TestNativeWaiverClockAwardsClaimsAtExactClose(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	f.set(t, "waivers.method", `"rolling"`, "now")
	cut := f.cut(t, "1000")
	f.claim(t, cut.Entry, "owner.three", "franchise.03")
	f.claim(t, cut.Entry, "owner.one", "franchise.01")
	f.now = f.now.Add(24*time.Hour - time.Nanosecond)
	if err := f.app.core.History.ProcessDueWaivers(t.Context(), legalityLeague, f.app.nativeFacts()); err != nil {
		t.Fatal(err)
	}
	if r := f.reading(t, "owner.one", "franchise.01"); r.Players[0].Winner != "" {
		t.Fatal("clock awarded before close", r)
	}
	f.now = f.now.Add(time.Nanosecond)
	if _, err := f.owner(t, "owner.three", commissioner.WaiverClaim,
		commissioner.RawWaiverArgs{Window: cut.Entry, Franchise: "franchise.03"}); err == nil {
		t.Fatal("claim accepted at close")
	}
	if err := f.app.core.History.ProcessDueWaivers(t.Context(), legalityLeague, f.app.nativeFacts()); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		r := f.reading(t, "owner.one", "franchise.01")
		if r.Players[0].Winner != "franchise.01" || r.Order[len(r.Order)-1] != "franchise.01" {
			t.Fatal("clock did not award by rolling priority", r)
		}
		f.app.core.CommissionerSession = testCommissionerSession{"creator"}
		ledger, err := newCommissionerAPI(f.app).TargetCommissionerLedger(legalityLeague)
		if err != nil {
			t.Fatal(err)
		}
		claims := 0
		for _, e := range ledger.Entries {
			if e.What == "Recorded a private waiver claim" {
				claims++
			}
		}
		if claims != 2 {
			t.Fatal("processed claims unavailable for commissioner corrections", ledger)
		}
		f.restart(t)
	}
}

func TestNativeWaiverCompensationCannotStrandWinningClaimOrCut(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	claim := f.claim(t, cut.Entry, "owner.one", "franchise.01")
	f.now = f.now.Add(24 * time.Hour)
	processed := f.process(t, cut.Entry)
	for _, target := range []commissioner.Receipt{cut, claim} {
		f.app.core.CommissionerSession = testCommissionerSession{"creator"}
		visibility := "league"
		if target.Entry == claim.Entry {
			visibility = "private"
		}
		raw := f.raw(t, commissioner.Revert, commissioner.RawArgs{Entry: target.Entry}, visibility)
		body, err := json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil {
			t.Fatal("compensation stranded live award", target.Entry)
		}
	}
	f.revert(t, processed)
	f.app.core.CommissionerSession = testCommissionerSession{"creator"}
	f.command(t, commissioner.Revert, commissioner.RawArgs{Entry: claim.Entry}, "private")
	f.revert(t, cut)
	f.restart(t)
	r := f.reading(t, "owner.one", "franchise.01")
	if len(r.Players) != 0 || len(r.Claims) != 0 {
		t.Fatal("compensation did not restore pre-cut state", r)
	}
}
