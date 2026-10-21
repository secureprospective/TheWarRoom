package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

type waiverFixture struct {
	*legalityFixture
	now time.Time
}

func waiverApp(t *testing.T) *waiverFixture {
	t.Helper()
	f := &waiverFixture{legalityFixture: legalityApp(t), now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)}
	f.clock()
	f.set(t, "roster.limits", `{"min":0,"max":100}`, "now")
	f.set(t, "cap.amount", `100000000000`, "now")
	f.set(t, "calendar.boundary", `"start_of_day"`, "now")
	f.set(t, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim"]}}`, "now")
	for _, owner := range []struct{ franchise, principal string }{{"franchise.01", "owner.one"},
		{"franchise.02", "owner.two"}, {"franchise.03", "owner.three"}} {
		f.command(t, "commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{
			Franchise: owner.franchise, Principal: owner.principal}}, "league")
	}
	f.command(t, commissioner.WaiverOrderSet, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Order: waiverOrder()}}, "league")

	salary, years := domain.Money(100000000), 3
	for _, player := range []string{"1000", "1001"} {
		f.command(t, commissioner.ContractEdit, commissioner.RawArgs{Override: &commissioner.RawNativeOverride{
			Player: player, Salary: &salary, Years: &years}}, "league")
	}
	return f
}

func waiverOrder() []string {
	out := []string{}
	for i := 1; i <= 32; i++ {
		out = append(out, fmt.Sprintf("franchise.%02d", i))
	}
	return out
}

func (f *waiverFixture) clock() {
	f.app.core.ClockNow = func() time.Time { return f.now }
	f.app.core.History.SetClock(func() time.Time { return f.now })
	f.app.core.LocalZone = time.FixedZone("PDT", -7*60*60)
}

func (f *waiverFixture) restart(t *testing.T) { t.Helper(); f.reopen(t); f.clock() }

func (f *waiverFixture) raw(t *testing.T, command string, args commissioner.RawArgs, visibility string) commissioner.RawCommand {
	t.Helper()
	records, err := f.app.leagueRecords(context.Background(), legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	return commissioner.RawCommand{League: legalityLeague, Command: command, Args: args,
		Expected: int64(len(records)), Request: fmt.Sprintf("waiver.%d", len(records)),
		Effective: "now", Visibility: visibility, Reason: "Our league's recorded change"}
}

func (f *waiverFixture) command(t *testing.T, command string, args commissioner.RawArgs, visibility string) commissioner.Receipt {
	t.Helper()
	f.app.core.CommissionerSession = testCommissionerSession{"creator"}
	return executeCommissionerTest(t, f.app, f.raw(t, command, args, visibility))
}

func (f *waiverFixture) owner(t *testing.T, principal, command string, args commissioner.RawWaiverArgs) (commissioner.Receipt, error) {
	t.Helper()
	visibility := "league"
	if command == commissioner.WaiverClaim {
		visibility = "private"
	}
	f.app.core.CommissionerSession = testCommissionerSession{principal}
	return newCommissionerAPI(f.app).TargetNativeWaiverAction(f.raw(t, command, commissioner.RawArgs{Waiver: &args}, visibility))
}

func (f *waiverFixture) cut(t *testing.T, player string) commissioner.Receipt {
	t.Helper()
	receipt, err := f.owner(t, "owner.two", commissioner.RosterCut,
		commissioner.RawWaiverArgs{Player: player, Franchise: "franchise.02"})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func (f *waiverFixture) claim(t *testing.T, window, principal, franchise string) commissioner.Receipt {
	t.Helper()
	receipt, err := f.owner(t, principal, commissioner.WaiverClaim,
		commissioner.RawWaiverArgs{Window: window, Franchise: franchise})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func (f *waiverFixture) reading(t *testing.T, principal, franchise string) commissioner.NativeWaivers {
	t.Helper()
	f.app.core.CommissionerSession = testCommissionerSession{principal}
	reading, err := newCommissionerAPI(f.app).TargetNativeWaivers(legalityLeague, franchise, f.now)
	if err != nil {
		t.Fatal(err)
	}
	return reading
}

func (f *waiverFixture) process(t *testing.T, window string) commissioner.Receipt {
	t.Helper()
	return f.command(t, commissioner.WaiverProcess, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: window}}, "league")
}

func TestNativeCutOpensWaiversAndCountsDeadMoney(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	assert := func() {
		t.Helper()
		reading := f.reading(t, "owner.two", "franchise.02")
		if len(reading.Players) != 1 || reading.Players[0].ClosesAt.Sub(f.now) != 24*time.Hour ||
			reading.Players[0].Closes != "Oct 14, 2026 at 5:00 AM PDT" {
			t.Fatalf("waiver close: %+v", reading)
		}
		assets, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
		if err != nil {
			t.Fatal(err)
		}
		if slices.ContainsFunc(assets.Snapshot.Players, func(p commissioner.NativePlayer) bool { return p.ID == "1000" }) {
			t.Fatal("cut left player rostered")
		}
		found := false
		for _, c := range assets.Snapshot.Adjustments {
			if c.ID == cut.Entry {
				found = true
				if c.Amount != 70000000 || c.Season != 2026 {
					t.Fatalf("dead money: %+v", c)
				}
			}
		}
		if !found {
			t.Fatal("cut omitted dead money")
		}
	}
	assert()
	f.restart(t)
	assert()
}

func TestNativeClaimRefusesRosterAndCapBreaks(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	f.set(t, "roster.limits", `{"min":0,"max":40}`, "now")
	_, err := f.owner(t, "owner.one", commissioner.WaiverClaim,
		commissioner.RawWaiverArgs{Window: cut.Entry, Franchise: "franchise.01"})
	if err == nil || !strings.Contains(err.Error(), "roster would not be legal") {
		t.Fatalf("illegal roster claim accepted: %v", err)
	}
	if r := f.reading(t, "owner.one", "franchise.01"); len(r.Claims) != 0 {
		t.Fatal("refused claim was recorded")
	}
	f.set(t, "roster.limits", `{"min":0,"max":100}`, "now")
	f.set(t, "cap.amount", `100000000`, "now")
	_, err = f.owner(t, "owner.one", commissioner.WaiverClaim,
		commissioner.RawWaiverArgs{Window: cut.Entry, Franchise: "franchise.01"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "cap") {
		t.Fatalf("illegal cap claim accepted: %v", err)
	}
}

func TestNativeClaimsArePrivateAcrossReadingsAndLedger(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	claim := f.claim(t, cut.Entry, "owner.one", "franchise.01")
	for range 2 {
		own := f.reading(t, "owner.one", "franchise.01")
		if len(own.Claims) != 1 || own.Claims[0].ID != claim.Entry {
			t.Fatalf("own claim missing: %+v", own)
		}
		other := f.reading(t, "owner.three", "franchise.03")
		body, err := json.Marshal(other)
		if err != nil {
			t.Fatal(err)
		}
		if len(other.Claims) != 0 || strings.Contains(string(body), claim.Entry) {
			t.Fatal("another owner's claim visible before processing")
		}
		for _, principal := range []string{"creator", "owner.two", "owner.three"} {
			f.app.core.CommissionerSession = testCommissionerSession{principal}
			ledger, err := newCommissionerAPI(f.app).TargetCommissionerLedger(legalityLeague)
			if err != nil {
				t.Fatal(err)
			}
			if slices.ContainsFunc(ledger.Entries, func(e commissioner.Entry) bool { return e.ID == claim.Entry }) {
				t.Fatalf("private claim leaked to %s", principal)
			}
		}
		f.restart(t)
	}
}

func TestNativeWaiverPriorityAndRollingThroughProcessing(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"reverse_standings", "rolling"} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			f := waiverApp(t)
			if method != "reverse_standings" { // reverse standings is the Legacy method already in force
				f.set(t, "waivers.method", `"`+method+`"`, "now")
			}
			cut := f.cut(t, "1000")
			f.claim(t, cut.Entry, "owner.three", "franchise.03")
			f.claim(t, cut.Entry, "owner.one", "franchise.01")
			f.now = f.now.Add(24 * time.Hour)
			processed := f.process(t, cut.Entry)
			for range 2 {
				reading := f.reading(t, "owner.one", "franchise.01")
				if reading.Players[0].Winner != "franchise.01" {
					t.Fatal("lower-priority claim won", reading)
				}
				expected := waiverOrder()
				if method == "rolling" {
					expected = append(expected[1:], expected[0])
				}
				if !slices.Equal(reading.Order, expected) {
					t.Fatalf("waiver order did not update correctly: %v", reading.Order)
				}
				assets, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
				if err != nil {
					t.Fatal(err)
				}
				if !slices.ContainsFunc(assets.Snapshot.Players, func(p commissioner.NativePlayer) bool {
					return p.ID == "1000" && p.Franchise == "franchise.01" && p.Status == domain.RosterActive
				}) {
					t.Fatal("winner did not gain the player", assets.Snapshot.Players)
				}
				for _, c := range assets.Snapshot.Adjustments {
					if c.ID == cut.Entry && c.Amount != 0 {
						t.Fatal("claim did not clear dead money")
					}
				}
				f.restart(t)
			}
			f.app.core.CommissionerSession = testCommissionerSession{"creator"}
			f.revert(t, processed)
			reading := f.reading(t, "owner.one", "franchise.01")
			if reading.Players[0].Winner != "" || reading.Players[0].Status != "Closed; awaiting processing" ||
				!slices.Equal(reading.Order, waiverOrder()) {
				t.Fatal("award compensation failed", reading)
			}
			f.process(t, cut.Entry)
		})
	}
}

func TestNativeOwnerActionsCannotSkipGateOrIdentity(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	f.set(t, "calendar.windows", `{}`, "now")
	actions := []struct {
		command, principal string
		args               commissioner.RawWaiverArgs
	}{
		{commissioner.RosterCut, "owner.two", commissioner.RawWaiverArgs{Player: "1001", Franchise: "franchise.02"}},
		{commissioner.WaiverClaim, "owner.one", commissioner.RawWaiverArgs{Window: cut.Entry, Franchise: "franchise.01"}},
	}
	for _, action := range actions {
		visibility := "league"
		if action.command == commissioner.WaiverClaim {
			visibility = "private"
		}
		f.app.core.CommissionerSession = testCommissionerSession{action.principal}
		raw := f.raw(t, action.command, commissioner.RawArgs{Waiver: &action.args}, visibility)
		if _, err := newCommissionerAPI(f.app).TargetNativeWaiverAction(raw); err == nil {
			t.Fatalf("%s skipped native gate", action.command)
		}
		if _, err := f.app.core.History.ExecuteCommissionerWithFacts(t.Context(), raw, action.principal, nil,
			f.app.nativeFacts(), f.app.ownerZone()); err == nil {
			t.Fatalf("%s skipped transactional gate", action.command)
		}
		body, err := json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil {
			t.Fatal("owner action used commissioner door")
		}
	}
	f.set(t, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim"]}}`, "now")
	if _, err := f.owner(t, "owner.one", commissioner.RosterCut,
		commissioner.RawWaiverArgs{Player: "1001", Franchise: "franchise.02"}); err == nil {
		t.Fatal("owner cut another roster")
	}
	if _, err := f.owner(t, "creator", commissioner.WaiverClaim,
		commissioner.RawWaiverArgs{Window: cut.Entry, Franchise: "franchise.01"}); err == nil {
		t.Fatal("commissioner impersonated owner")
	}
}

func TestNativeWaiverDurationGovernsWindowRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	f.cut(t, "1000")
	change := f.set(t, "waivers.duration", `172800000000000`, "now")
	assert := func(hours time.Duration) {
		t.Helper()
		reading := f.reading(t, "owner.one", "franchise.01")
		words := "1 day"
		if hours == 48 {
			words = "2 days"
		}
		assertWaiverRule(t, f, "waivers.duration", words)
		if reading.Players[0].ClosesAt.Sub(f.now) != hours*time.Hour {
			t.Fatal("duration did not govern persisted window", reading)
		}
	}
	for range 2 {
		assert(48)
		f.restart(t)
	}
	f.revert(t, change)
	for range 2 {
		assert(24)
		f.restart(t)
	}
}

func TestNativeWaiverMethodGovernsAwardRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	f.set(t, "waivers.method", `"rolling"`, "now")
	first := f.cut(t, "1000")
	f.claim(t, first.Entry, "owner.one", "franchise.01")
	f.now = f.now.Add(24 * time.Hour)
	f.process(t, first.Entry)
	second := f.cut(t, "1001")
	f.claim(t, second.Entry, "owner.three", "franchise.03")
	f.claim(t, second.Entry, "owner.one", "franchise.01")
	method := f.set(t, "waivers.method", `"reverse_standings"`, "now")
	f.now = f.now.Add(24 * time.Hour)
	processed := f.process(t, second.Entry)
	assert := func(winner string) {
		t.Helper()
		reading := f.reading(t, "owner.one", "franchise.01")
		method := "Reverse standings"
		if winner == "franchise.03" {
			method = "Rolling"
		}
		assertWaiverRule(t, f, "waivers.method", method)
		if reading.Players[1].Winner != winner {
			t.Fatalf("method did not govern award: %+v", reading)
		}
	}
	for range 2 {
		assert("franchise.01")
		f.restart(t)
	}
	f.revert(t, processed)
	f.revert(t, method)
	f.process(t, second.Entry)
	for range 2 {
		assert("franchise.03")
		f.restart(t)
	}
}

func TestNativeWaiverOverrideGovernsCompensationAndReplay(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	f.claim(t, cut.Entry, "owner.one", "franchise.01")
	third := f.claim(t, cut.Entry, "owner.three", "franchise.03")
	f.now = f.now.Add(24 * time.Hour)
	f.process(t, cut.Entry)
	change := f.command(t, commissioner.WaiverOverride, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Claim: third.Entry, Franchise: "franchise.03"}}, "league")
	assert := func(winner string) {
		t.Helper()
		reading := f.reading(t, "owner.one", "franchise.01")
		if reading.Players[0].Winner != winner {
			t.Fatal("override did not govern award", reading)
		}
	}
	for range 2 {
		assert("franchise.03")
		f.restart(t)
	}
	f.revert(t, change)
	for range 2 {
		assert("franchise.01")
		f.restart(t)
	}
}

func TestNativeWaiverClockProcessesUnclaimedAtClose(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	if err := f.app.core.History.ProcessDueWaivers(t.Context(), legalityLeague, f.app.nativeFacts()); err != nil {
		t.Fatal(err)
	}
	if r := f.reading(t, "owner.one", "franchise.01"); r.Players[0].Status != "On waivers" {
		t.Fatal("clock closed early", r)
	}
	f.now = f.now.Add(24 * time.Hour)
	for range 2 {
		if err := f.app.core.History.ProcessDueWaivers(t.Context(), legalityLeague, f.app.nativeFacts()); err != nil {
			t.Fatal(err)
		}
		r := f.reading(t, "owner.one", "franchise.01")
		if r.Players[0].Status != "Available as a free agent" || r.Players[0].Winner != "" {
			t.Fatal("unclaimed player not free", r)
		}
		records, err := f.app.leagueRecords(context.Background(), legalityLeague)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, record := range records {
			if record.Command.Command == commissioner.WaiverProcess {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("clock processed %d times for %s", count, cut.Entry)
		}
		f.restart(t)
	}
}

func TestNativeWaiverProcessingRechecksEligibility(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	f.claim(t, cut.Entry, "owner.one", "franchise.01")
	f.claim(t, cut.Entry, "owner.three", "franchise.03")
	f.set(t, "roster.limits", `{"min":0,"max":40}`, "now")
	f.now = f.now.Add(24 * time.Hour)
	f.process(t, cut.Entry)
	r := f.reading(t, "owner.one", "franchise.01")
	if r.Players[0].Winner != "franchise.03" || len(r.Claims) != 1 ||
		!strings.Contains(r.Claims[0].Status, "roster would not be legal") {
		t.Fatal("processing did not recheck changed roster rule or explain refusal", r)
	}
}

func TestNativeWaiverCommandsPermissionVersionRetry(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	f.now = f.now.Add(24 * time.Hour)
	raw := f.raw(t, commissioner.WaiverProcess, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: cut.Entry}}, "league")
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
	if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil {
		t.Fatal("owner processed waivers")
	}
	f.command(t, "commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: "owner.three", Role: governance.CoCommissioner}}, "private")
	raw.Expected++
	raw.Request = "co.process"
	body, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	f.app.core.CommissionerSession = testCommissionerSession{"owner.three"}
	receipt, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body))
	if err != nil {
		t.Fatal(err)
	}
	again, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body))
	if err != nil || !reflect.DeepEqual(receipt, again) {
		t.Fatal("processing retry changed receipt", again, err)
	}
	raw.Request = "stale"
	body, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil {
		t.Fatal("stale version accepted")
	}
}

func TestNativeWaiversWire(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	f.claim(t, cut.Entry, "owner.one", "franchise.01")
	reading := f.reading(t, "owner.one", "franchise.01")
	body, err := json.MarshalIndent(reading, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("internal", "commissioner", "testdata", "native-waivers.wire.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(body) {
		t.Fatal("native waiver wire differs; regenerate with Go")
	}
}

func TestNativeWaiverStandingsSeamDoesNotInventPriority(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	if _, err := f.app.core.History.StandingsOrder(t.Context(), legalityLeague, 2026, f.now); err == nil ||
		!strings.Contains(err.Error(), "Waiver order needs the standings, which are not built yet.") {
		t.Fatal("standings seam invented priority", err)
	}
	raw := f.raw(t, commissioner.WaiverOrderAuto, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{}}, "league")
	f.app.core.CommissionerSession = testCommissionerSession{"creator"}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil {
		t.Fatal("automatic order invented standings")
	}
	records, err := f.app.leagueRecords(context.Background(), legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	for i := range records {
		if records[i].Command.Command == commissioner.NativeSeed {
			clone := *records[i].Command.Args.Native
			clone.StandingsOrder = waiverOrder()
			records[i].Command.Args.Native = &clone
		}
	}
	order, err := commissioner.RecordedStandingsOrder(legalityLeague, records, 2026, f.now)
	if err != nil || !slices.Equal(order, waiverOrder()) {
		t.Fatal("recorded standings seam did not read typed order", order, err)
	}
	if _, err := commissioner.RecordedStandingsOrder(legalityLeague, records, 2027, f.now); err == nil {
		t.Fatal("standings order leaked across seasons")
	}
}

func (f *waiverFixture) set(t *testing.T, field, value, effective string) commissioner.Receipt {
	t.Helper()
	f.app.core.CommissionerSession = testCommissionerSession{"creator"}
	return f.legalityFixture.set(t, field, value, effective)
}

func (f *waiverFixture) revert(t *testing.T, receipt commissioner.Receipt) {
	t.Helper()
	f.app.core.CommissionerSession = testCommissionerSession{"creator"}
	f.legalityFixture.revert(t, receipt)
}

func assertWaiverRule(t *testing.T, f *waiverFixture, field, words string) {
	t.Helper()
	reading, err := newCommissionerAPI(f.app).TargetCommissionerRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range reading.Sections {
		for _, row := range section.Rows {
			if row.Field == field {
				if row.Value != words {
					t.Fatalf("%s = %s, want %s", field, row.Value, words)
				}
				return
			}
		}
	}
	t.Fatal("missing waiver rule", field)
}
