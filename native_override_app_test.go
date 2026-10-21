package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func overrideCommand(t *testing.T, f *legalityFixture, command, value string) commissioner.RawCommand {
	t.Helper()
	assets, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	var override commissioner.RawNativeOverride
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&override); err != nil {
		t.Fatal(err)
	}
	return commissioner.RawCommand{League: legalityLeague, Command: command,
		Args: commissioner.RawArgs{Override: &override}, Expected: assets.Revision,
		Request: fmt.Sprintf("override.%d", assets.Revision), Reason: "Correct the recorded injury decision",
		Effective: "now", Visibility: "league"}
}

func nativeContract(t *testing.T, f *legalityFixture, id string) commissioner.NativePlayer {
	t.Helper()
	assets, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, player := range assets.Snapshot.Players {
		if player.ID == id {
			return player
		}
	}
	t.Fatal("missing native player")
	return commissioner.NativePlayer{}
}

func TestNativeOverrideCommandsGovernCapAndReplay(t *testing.T) {
	t.Parallel()
	cases := []struct {
		command, value string
		committed      domain.Money
	}{
		{commissioner.ContractEdit, `{"player":"0099","years":4,"salaryCents":2000000000}`, 30_190_00000},
		{commissioner.CapAdjust, `{"franchise":"franchise.02","amountCents":-200000000,"season":2026}`,
			19_180_00000},
		{commissioner.CapRelief, `{"player":"0100","amountCents":300000000}`, 18_180_00000},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			t.Parallel()
			f := capApp(t)
			baseline := *capTeam(t, f).CommittedCents
			mirror, stamp, used := f.app.core.League.ContractsView()
			for pass := 0; pass < 2; pass++ {
				raw := overrideCommand(t, f, tc.command, tc.value)
				receipt := executeCommissionerTest(t, f.app, raw)
				if again := executeCommissionerTest(t, f.app, raw); again != receipt {
					t.Fatal("retry appended twice")
				}
				assertOverrideCap(t, f, tc.command, tc.committed)
				if pass == 0 {
					after, afterStamp, afterUsed := f.app.core.League.ContractsView()
					if !reflect.DeepEqual(mirror, after) || stamp != afterStamp || !reflect.DeepEqual(used, afterUsed) {
						t.Fatal("native correction wrote the MFL mirror")
					}
				}
				f.reopen(t)
				assertOverrideCap(t, f, tc.command, tc.committed)
				f.revert(t, receipt)
				if got := *capTeam(t, f).CommittedCents; got != baseline {
					t.Fatalf("reverted override counted: %d", got)
				}
				f.reopen(t)
				if got := *capTeam(t, f).CommittedCents; got != baseline {
					t.Fatalf("reverted replay counted: %d", got)
				}
			}
		})
	}
}

func assertOverrideCap(t *testing.T, f *legalityFixture, command string, expected domain.Money) {
	t.Helper()
	team := capTeam(t, f)
	if team.CommittedCents == nil || *team.CommittedCents != expected {
		t.Fatalf("%s cap consequence: %+v; want %d", command, team, expected)
	}
	if command == commissioner.ContractEdit {
		p := nativeContract(t, f, "0099")
		if p.Salary != 2000000000 || p.ExpirationYear != 2029 {
			t.Fatalf("contract consequence: %+v", p)
		}
	}
}

func reserveVerdict(t *testing.T, f *legalityFixture, id string) nativecheck.ReserveVerdict {
	t.Helper()
	for _, verdict := range f.reserves(t).Players {
		if verdict.Player == id {
			return verdict
		}
	}
	t.Fatal("missing reserve player")
	return nativecheck.ReserveVerdict{}
}

func TestNativeReserveExceptionGovernsAndReplay(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ player, bucket string }{{"2002", "ir"}, {"3003", "taxi"}} {
		t.Run(tc.bucket, func(t *testing.T) {
			t.Parallel()
			f := legalityApp(t)
			before := reserveVerdict(t, f, tc.player)
			if before.Standing != nativecheck.MustLeave {
				t.Fatalf("bad baseline: %+v", before)
			}
			value := fmt.Sprintf(`{"player":%q,"bucket":%q,"until":"2026-10-14T12:00:00Z"}`, tc.player, tc.bucket)
			for pass := 0; pass < 2; pass++ {
				receipt := executeCommissionerTest(t, f.app, overrideCommand(t, f, commissioner.ReserveException, value))
				assertException(t, f, tc.player)
				f.reopen(t)
				assertException(t, f, tc.player)
				f.revert(t, receipt)
				if got := reserveVerdict(t, f, tc.player); got != before {
					t.Fatalf("reverted exception counted: %+v", got)
				}
				f.reopen(t)
				if got := reserveVerdict(t, f, tc.player); got != before {
					t.Fatalf("exception survived compensation replay: %+v", got)
				}
			}
		})
	}
}

func assertException(t *testing.T, f *legalityFixture, player string) {
	t.Helper()
	got := reserveVerdict(t, f, player)
	if got.Standing != nativecheck.Stays || !strings.Contains(got.Words, "Correct the recorded injury decision") ||
		strings.Contains(got.Words, "2026-10-14T12:00:00Z") {
		t.Fatalf("exception not explained: %+v", got)
	}
}

func TestNativeReserveExceptionExpiryAndScope(t *testing.T) {
	t.Parallel()
	f := legalityApp(t)
	baseline := f.reserves(t)
	value := `{"player":"2002","bucket":"ir","until":"2026-10-14T12:00:00Z"}`
	executeCommissionerTest(t, f.app, overrideCommand(t, f, commissioner.ReserveException, value))
	assertException(t, f, "2002")
	for _, want := range baseline.Players {
		if want.Player != "2002" && reserveVerdict(t, f, want.Player) != want {
			t.Fatalf("exception covered another player: %s", want.Player)
		}
	}
	f.reopen(t)
	f.app.core.History.SetClock(func() time.Time { return time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) })
	if got := f.reserves(t).Players; !reflect.DeepEqual(got, baseline.Players) {
		t.Fatalf("exception outlived until: %+v", got)
	}
	f = legalityApp(t)
	taxi := reserveVerdict(t, f, "3003")
	executeCommissionerTest(t, f.app, overrideCommand(t, f, commissioner.ReserveException,
		`{"player":"3003","bucket":"ir"}`))
	if got := reserveVerdict(t, f, "3003"); got != taxi {
		t.Fatalf("IR exception covered taxi: %+v", got)
	}
}

func TestNativeOverridesLedgerOrderAndCompensatedCompensation(t *testing.T) {
	t.Parallel()
	f := capApp(t)
	first := executeCommissionerTest(t, f.app, overrideCommand(t, f, commissioner.ContractEdit,
		`{"player":"0099","years":2,"salaryCents":1000000000}`))
	second := executeCommissionerTest(t, f.app, overrideCommand(t, f, commissioner.ContractEdit,
		`{"player":"0099","years":3,"salaryCents":1500000000}`))
	if got := nativeContract(t, f, "0099"); got.Salary != 1500000000 || got.ExpirationYear != 2028 {
		t.Fatalf("ledger order lost: %+v", got)
	}
	f.revert(t, second)
	if got := nativeContract(t, f, "0099"); got.Salary != 1000000000 {
		t.Fatalf("earlier correction lost: %+v", got)
	}
	assets, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := newCommissionerAPI(f.app).TargetCommissionerLedger(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	reversal := commissioner.Receipt{Entry: ledger.Entries[0].ID}
	f.revert(t, reversal)
	f.reopen(t)
	if got := nativeContract(t, f, "0099"); got.Salary != 1500000000 {
		t.Fatalf("reversal of reversal lost: %+v", got)
	}
	if assets.Revision != first.Revision+2 {
		t.Fatal("unexpected ledger revisions")
	}
}

func TestNativeOverrideWire(t *testing.T) {
	t.Parallel()
	f := capApp(t)
	executeCommissionerTest(t, f.app, overrideCommand(t, f, commissioner.ContractEdit,
		`{"player":"0099","years":4,"salaryCents":2000000000}`))
	executeCommissionerTest(t, f.app, overrideCommand(t, f, commissioner.CapAdjust,
		`{"franchise":"franchise.02","amountCents":-200000000,"season":2026}`))
	executeCommissionerTest(t, f.app, overrideCommand(t, f, commissioner.CapRelief,
		`{"player":"0100","amountCents":300000000}`))
	capReading := readCap(t, f)
	reserves := legalityApp(t)
	executeCommissionerTest(t, reserves.app, overrideCommand(t, reserves, commissioner.ReserveException,
		`{"player":"2002","bucket":"ir","until":"2026-10-14T12:00:00Z"}`))
	reserveReading, err := newCommissionerAPI(reserves.app).TargetNativeReserveCheck(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.MarshalIndent(struct {
		Cap      commissioner.NativeCapCheck     `json:"cap"`
		Reserves commissioner.NativeReserveCheck `json:"reserves"`
	}{capReading, reserveReading}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "internal/commissioner/testdata/native-overrides.wire.json", body, "native overrides")
}

func TestNativeOverrideRefusalsLeaveAssetsUntouched(t *testing.T) {
	t.Parallel()
	f := capApp(t)
	before := readCap(t, f)
	for _, tc := range []struct{ command, value string }{
		{commissioner.ContractEdit, `{"player":"9999","years":1,"salaryCents":100}`},
		{commissioner.ContractEdit, `{"player":"0099","years":9999,"salaryCents":100}`},
		{commissioner.CapAdjust, `{"franchise":"missing","amountCents":100,"season":2026}`},
		{commissioner.ReserveException, `{"player":"0099","bucket":"ir","until":"2026-10-13T12:00:00Z"}`},
	} {
		raw := overrideCommand(t, f, tc.command, tc.value)
		body, err := json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil {
			t.Fatal("invalid target accepted")
		}
		if after := readCap(t, f); !reflect.DeepEqual(before, after) {
			t.Fatal("refusal changed native assets")
		}
	}
	raw := overrideCommand(t, f, commissioner.CapRelief, `{"player":"0099","amountCents":100000000}`)
	raw.Expected--
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil {
		t.Fatal("stale correction accepted")
	}
	raw.Expected++
	receipt := executeCommissionerTest(t, f.app, raw)
	changed := *raw.Args.Override
	amount := domain.Money(200000000)
	changed.Amount = &amount
	raw.Args.Override = &changed
	body, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil {
		t.Fatal("request key reused")
	}
	f.revert(t, receipt)
}

func TestNativeCapAdjustmentSeasonGovernsActualReading(t *testing.T) {
	t.Parallel()
	f := capApp(t)
	f.set(t, "calendar.rollover", `"2027-03-01T05:00:00Z"`, "now")
	before := *capTeam(t, f).CommittedCents
	receipt := executeCommissionerTest(t, f.app, overrideCommand(t, f, commissioner.CapAdjust,
		`{"franchise":"franchise.02","amountCents":-200000000,"season":2027}`))
	if got := *capTeam(t, f).CommittedCents; got != before {
		t.Fatal("future adjustment counted now")
	}
	f.reopen(t)
	f.app.core.History.SetClock(func() time.Time { return time.Date(2027, 3, 1, 5, 0, 0, 0, time.UTC) })
	if got := *capTeam(t, f).CommittedCents; got != before-200000000 {
		t.Fatalf("future adjustment missing: %d", got)
	}
	f.revert(t, receipt)
	if got := *capTeam(t, f).CommittedCents; got != before {
		t.Fatal("future adjustment compensation lost")
	}
}
