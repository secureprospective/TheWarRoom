package commissioner

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestCorrectRosterContractAndAddWithContract(t *testing.T) {
	t.Parallel()
	b := rolloverLedger(t)

	// Add a brand-new player with explicit contract terms.
	addContract := &CorrectionContract{
		Salary:             domain.Money(45000000),
		ExpirationYear:     2029,
		ContractStatus:     domain.CStatusRFA,
		ContractInfo:       "45.0M / 3yr",
		HeldContractStatus: "RFA (2026)",
	}
	addCmd := RawCommand{
		League: rolloverLeague, Command: CorrectRoster, Request: "add-with-contract",
		Reason: "Refresh add with contract", Effective: "now", Visibility: "league",
		Args: RawArgs{Correction: &RawCorrection{
			Player: "0099", Franchise: "franchise.01", Action: "add", Contract: addContract,
		}},
	}
	if err := addCmd.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := CheckCorrectionAppend(addCmd, b.records, seasonMoment(5)); err != nil {
		t.Fatal(err)
	}
	b.add(addCmd, seasonMoment(5))

	// Update the added player's contract terms via Action: "contract".
	editContract := &CorrectionContract{
		Salary:             domain.Money(60000000),
		ExpirationYear:     2030,
		ContractStatus:     domain.CStatusUFA,
		ContractInfo:       "60.0M / EXT",
		HeldContractStatus: "EXT (2026)",
	}
	editCmd := RawCommand{
		League: rolloverLeague, Command: CorrectRoster, Request: "edit-contract",
		Reason: "Refresh contract", Effective: "now", Visibility: "league",
		Args: RawArgs{Correction: &RawCorrection{
			Player: "0099", Franchise: "franchise.01", Action: "contract", Contract: editContract,
		}},
	}
	if err := editCmd.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := CheckCorrectionAppend(editCmd, b.records, seasonMoment(6)); err != nil {
		t.Fatal(err)
	}
	b.add(editCmd, seasonMoment(6))

	assets, err := NativeAssetsReading(rolloverLeague, b.records)
	if err != nil {
		t.Fatal(err)
	}
	idx := assets.Snapshot.PlayerOf("franchise.01", "0099")
	if idx < 0 {
		t.Fatal("added player missing")
	}
	got := assets.Snapshot.Players[idx]
	if got.ContractMissing || got.Salary != 60000000 || got.ExpirationYear != 2030 ||
		got.ContractStatus != domain.CStatusUFA || got.ContractInfo != "60.0M / EXT" ||
		got.HeldContractStatus != "EXT (2026)" {
		t.Fatalf("unexpected contract projection: %+v", got)
	}

	// Reject missing contract payload on Action: "contract" and invalid missing contract terms.
	badNil := RawCorrection{Player: "0099", Franchise: "franchise.01", Action: "contract"}
	if err := badNil.Validate(CorrectRoster); err == nil {
		t.Fatal("accepted Action: contract with nil Contract")
	}
	badTerms := RawCorrection{
		Player: "0099", Franchise: "franchise.01", Action: "contract",
		Contract: &CorrectionContract{ContractMissing: true, Salary: 100},
	}
	if err := badTerms.Validate(CorrectRoster); err == nil {
		t.Fatal("accepted ContractMissing with non-zero Salary")
	}
}

func TestCorrectRosterAdjustmentUpsertAndRemove(t *testing.T) {
	t.Parallel()
	b := rolloverLedger(t)

	charge := &NativeCharge{
		ID: "mfl-adj-1", Franchise: "franchise.01", Amount: 5000000, Description: "Trade dead cap",
	}
	upsert := RawCommand{
		League: rolloverLeague, Command: CorrectRoster, Request: "adj-add",
		Reason: "Refresh adjustment", Effective: "now", Visibility: "league",
		Args: RawArgs{Correction: &RawCorrection{Franchise: "franchise.01", Action: "adjustment", Charge: charge}},
	}
	if err := upsert.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := CheckCorrectionAppend(upsert, b.records, seasonMoment(5)); err != nil {
		t.Fatal(err)
	}
	b.add(upsert, seasonMoment(5))

	// Update the same MFL adjustment ID with a new amount and description.
	updated := &NativeCharge{
		ID: "mfl-adj-1", Franchise: "franchise.01", Amount: 7500000, Description: "Updated cap hit",
	}
	upsert2 := RawCommand{
		League: rolloverLeague, Command: CorrectRoster, Request: "adj-edit",
		Reason: "Refresh adjustment edit", Effective: "now", Visibility: "league",
		Args: RawArgs{Correction: &RawCorrection{Franchise: "franchise.01", Action: "adjustment", Charge: updated}},
	}
	if err := CheckCorrectionAppend(upsert2, b.records, seasonMoment(6)); err != nil {
		t.Fatal(err)
	}
	b.add(upsert2, seasonMoment(6))

	assets, err := NativeAssetsReading(rolloverLeague, b.records)
	if err != nil {
		t.Fatal(err)
	}
	isAdj1 := func(a NativeCharge) bool { return a.ID == "mfl-adj-1" }
	found := slices.IndexFunc(assets.Snapshot.Adjustments, isAdj1)
	if found < 0 || assets.Snapshot.Adjustments[found] != *updated {
		t.Fatalf("unexpected adjustments after upsert: %+v", assets.Snapshot.Adjustments)
	}

	// Remove the MFL adjustment.
	rm := RawCommand{
		League: rolloverLeague, Command: CorrectRoster, Request: "adj-remove",
		Reason: "Refresh adjustment removal", Effective: "now", Visibility: "league",
		Args: RawArgs{Correction: &RawCorrection{Franchise: "franchise.01", Action: "remove_adjustment",
			Charge: &NativeCharge{ID: "mfl-adj-1", Franchise: "franchise.01"}}},
	}
	if err := CheckCorrectionAppend(rm, b.records, seasonMoment(7)); err != nil {
		t.Fatal(err)
	}
	b.add(rm, seasonMoment(7))

	assets, err = NativeAssetsReading(rolloverLeague, b.records)
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(assets.Snapshot.Adjustments, isAdj1) {
		t.Fatalf("adjustment was not removed: %+v", assets.Snapshot.Adjustments)
	}

	// Reject mismatched franchise or player field on adjustment.
	bad := RawCorrection{Player: "0003", Franchise: "franchise.01", Action: "adjustment", Charge: charge}
	if err := bad.Validate(CorrectRoster); err == nil {
		t.Fatal("accepted adjustment with Player set")
	}
	badFranchise := RawCorrection{Franchise: "franchise.02", Action: "adjustment", Charge: charge}
	if err := badFranchise.Validate(CorrectRoster); err == nil {
		t.Fatal("accepted adjustment with mismatched franchise")
	}
	badRemoveExtra := RawCorrection{
		Franchise: "franchise.01", Action: "remove_adjustment",
		Charge: &NativeCharge{
			ID: "mfl-adj-1", Franchise: "franchise.01", Amount: 100, Description: "extra",
		},
	}
	if err := badRemoveExtra.Validate(CorrectRoster); err == nil {
		t.Fatal("accepted remove_adjustment with extraneous charge fields")
	}
}

func TestCorrectPickSelectionAddClearAndRemove(t *testing.T) {
	t.Parallel()
	seed := RawNativeSnapshot{
		Season:     2026,
		Franchises: []NativeFranchise{{ID: "franchise.01"}, {ID: "franchise.02"}},
		Picks: []NativePick{
			{Season: 2026, Round: 1, Slot: 1, Owner: "franchise.01"},
			{Season: 2026, Round: 1, Slot: 2, Owner: "franchise.02"},
		},
	}

	applyPick := func(c *RawCorrection, reason string) {
		t.Helper()
		if err := c.Validate(CorrectPick); err != nil {
			t.Fatal(err)
		}
		rec := Record{Command: RawCommand{
			Command: CorrectPick, Args: RawArgs{Correction: c}, Reason: reason,
		}}
		if err := correctPick(&seed, rec); err != nil {
			t.Fatal(err)
		}
	}

	// Record a draft selection (Player + owner update).
	applyPick(&RawCorrection{
		Owner: "franchise.02", Pick: &CorrectionPick{Season: 2026, Round: 1, Slot: 1, Player: "0088"},
	}, "Draft result")
	if seed.Picks[0].Owner != "franchise.02" || seed.Picks[0].Player != "0088" || seed.Picks[0].Skipped {
		t.Fatalf("selection not applied: %+v", seed.Picks[0])
	}

	// Mark slot 2 skipped.
	applyPick(&RawCorrection{
		Owner: "franchise.02", Pick: &CorrectionPick{Season: 2026, Round: 1, Slot: 2, Skipped: true},
	}, "Skipped")
	if !seed.Picks[1].Skipped || seed.Picks[1].Player != "" {
		t.Fatalf("skip not applied: %+v", seed.Picks[1])
	}

	// Clear slot 2 back to unselected.
	applyPick(&RawCorrection{
		Owner: "franchise.01", Pick: &CorrectionPick{Season: 2026, Round: 1, Slot: 2, Clear: true},
	}, "Clear")
	if seed.Picks[1].Skipped || seed.Picks[1].Player != "" || seed.Picks[1].Owner != "franchise.01" {
		t.Fatalf("clear not applied: %+v", seed.Picks[1])
	}

	// Add a new future draft pick.
	applyPick(&RawCorrection{
		Owner: "franchise.02",
		Pick:  &CorrectionPick{Season: 2028, Round: 1, Original: "franchise.01", Add: true},
	}, "Future pick")
	if len(seed.Picks) != 3 || seed.Picks[2].Season != 2028 ||
		seed.Picks[2].Original != "franchise.01" || seed.Picks[2].Owner != "franchise.02" {
		t.Fatalf("future pick not added: %+v", seed.Picks)
	}

	// Remove the added future pick.
	applyPick(&RawCorrection{
		Owner: "franchise.02",
		Pick:  &CorrectionPick{Season: 2028, Round: 1, Original: "franchise.01", Remove: true},
	}, "Remove pick")
	if len(seed.Picks) != 2 {
		t.Fatalf("future pick not removed: %+v", seed.Picks)
	}

	// Reject contradictory flags.
	for _, bad := range []CorrectionPick{
		{Season: 2026, Round: 1, Slot: 1, Player: "0088", Skipped: true},
		{Season: 2028, Round: 1, Original: "franchise.01", Player: "0088"},
		{Season: 2026, Round: 1, Slot: 1, Add: true, Remove: true},
		{Season: 2026, Round: 1, Slot: 1, Clear: true, Player: "0088"},
		{Season: 2028, Round: 101, Original: "franchise.01", Add: true},
	} {
		if err := (RawCorrection{Owner: "franchise.01", Pick: &bad}).Validate(CorrectPick); err == nil {
			t.Fatalf("accepted invalid pick correction %+v", bad)
		}
	}
}

func TestCheckRecordedAppendMonotonicCutoverWeek(t *testing.T) {
	t.Parallel()
	b := rolloverLedger(t)
	weeks := []ScheduleWeek{{
		Week: 1, Games: []ScheduleGame{{ID: "w1g1", Home: "franchise.01", Away: "franchise.02"}},
	}}

	// Record initial boundary at CutoverWeek 3.
	gen1 := RawCommand{
		League: rolloverLeague, Command: ScheduleGenerate, Request: "sched-1",
		Reason: "Initial", Effective: "now", Visibility: "league",
		Args: RawArgs{Schedule: &RawScheduleArgs{
			Weeks: 1, Fixtures: weeks,
			Recorded: &RawRecordedResults{
				Season: 2026, CutoverWeek: 3,
				Teams: []RecordedTeamResult{
					{Franchise: "franchise.01", Week: 1, Points: leaguerules.Decimal(1000000), Result: "W"},
					{Franchise: "franchise.02", Week: 1, Points: leaguerules.Decimal(900000), Result: "L"},
				},
			},
		}},
	}
	if err := CheckScheduleAppend(gen1, b.records); err != nil {
		t.Fatal(err)
	}
	b.records = append(b.records, Record{
		Command: gen1, Principal: "creator", GeneratedSchedule: weeks,
		Receipt: Receipt{
			Entry: "e-sched-1", League: rolloverLeague,
			Revision: int64(len(b.records) + 1), RecordedAt: seasonMoment(5),
		},
	})

	// Moving backwards (3 -> 2) must be refused.
	backwards := RawRecordedResults{
		Season: 2026, CutoverWeek: 2,
		Teams: []RecordedTeamResult{
			{Franchise: "franchise.01", Week: 1, Points: leaguerules.Decimal(1000000), Result: "W"},
		},
	}
	err := checkRecordedAppend(backwards, b.records)
	if err == nil || !strings.Contains(err.Error(), "cannot move backwards") {
		t.Fatalf("expected backwards cutover refusal, got: %v", err)
	}

	// Staying at 3 or advancing monotonically (3 -> 5) must be allowed.
	for _, nextCutover := range []int{3, 5} {
		forward := RawRecordedResults{
			Season: 2026, CutoverWeek: nextCutover,
			Teams: []RecordedTeamResult{
				{Franchise: "franchise.01", Week: 1, Points: leaguerules.Decimal(1000000), Result: "W"},
				{Franchise: "franchise.02", Week: 1, Points: leaguerules.Decimal(900000), Result: "L"},
			},
		}
		if err := checkRecordedAppend(forward, b.records); err != nil {
			t.Fatalf("cutover %d refused: %v", nextCutover, err)
		}
	}
}

func TestCorrectionLabelsForRefreshActions(t *testing.T) {
	t.Parallel()
	b := rolloverLedger(t)
	names := newAssetNames(b.records)
	mkRec := func(cmd string, c RawCorrection) Record {
		return Record{Command: RawCommand{Command: cmd, Args: RawArgs{Correction: &c}}}
	}
	cases := []Record{
		mkRec(CorrectRoster, RawCorrection{
			Player: "0003", Franchise: "franchise.01", Action: "status", Status: domain.RosterIR,
		}),
		mkRec(CorrectRoster, RawCorrection{
			Player: "0003", Franchise: "franchise.01", Action: "contract",
			Contract: &CorrectionContract{Salary: 100},
		}),
		mkRec(CorrectRoster, RawCorrection{
			Franchise: "franchise.01", Action: "adjustment",
			Charge: &NativeCharge{ID: "adj-1", Franchise: "franchise.01"},
		}),
		mkRec(CorrectRoster, RawCorrection{
			Franchise: "franchise.01", Action: "remove_adjustment",
			Charge: &NativeCharge{ID: "adj-1", Franchise: "franchise.01"},
		}),
		mkRec(CorrectPick, RawCorrection{
			Owner: "franchise.01", Pick: &CorrectionPick{Season: 2026, Round: 1, Slot: 1, Player: "0003"},
		}),
		mkRec(CorrectPick, RawCorrection{
			Owner: "franchise.01", Pick: &CorrectionPick{Season: 2026, Round: 1, Slot: 1, Skipped: true},
		}),
		mkRec(CorrectPick, RawCorrection{
			Owner: "franchise.01",
			Pick:  &CorrectionPick{Season: 2028, Round: 1, Original: "franchise.02", Remove: true},
		}),
	}
	for i, r := range cases {
		if label := correctionLabel(r); label == "" || label == "Corrected a recorded asset" {
			t.Errorf("case %d: unexpected label %q", i, label)
		}
		desc := correctionDescription(names, r)
		badStatus := strings.Contains(desc, "Removed ") && r.Command.Args.Correction.Action == "status"
		if desc == "" || badStatus {
			t.Errorf("case %d: unexpected description %q", i, desc)
		}
	}
}

func TestOpenTradesReportsUnsettledOnly(t *testing.T) {
	t.Parallel()
	b := rolloverLedger(t)
	now := seasonMoment(5)
	openOffer := TradeState{
		ID: "trade.open", Proposer: "franchise.01", Parties: []string{"franchise.01", "franchise.02"},
		Status: "offered", OfferedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour),
	}
	expiredOffer := TradeState{
		ID: "trade.expired", Proposer: "franchise.01", Parties: []string{"franchise.01", "franchise.02"},
		Status: "offered", OfferedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Minute),
	}
	reviewOffer := TradeState{
		ID: "trade.review", Proposer: "franchise.01", Parties: []string{"franchise.01", "franchise.02"},
		Status: "review", OfferedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour),
	}
	for _, tr := range []TradeState{openOffer, expiredOffer, reviewOffer} {
		trCopy := tr
		b.add(RawCommand{
			League: rolloverLeague, Command: TradeOffer, Request: "req-" + tr.ID,
			Reason: "Test trade", Effective: "now", Visibility: "league",
			Args: RawArgs{TradeOutcome: &trCopy},
		}, now)
	}
	open, err := OpenTrades(rolloverLeague, b.records, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 2 || open[0].Status != "offered" || open[1].Status != "review" {
		t.Fatalf("unexpected open trades: %+v", open)
	}
}

func TestSyncedLineupAllowsPostKickoffRosterAndOwnerOverride(t *testing.T) {
	t.Parallel()
	b := rolloverLedger(t)
	kickoff := seasonMoment(10)
	afterKickoff := kickoff.Add(time.Hour)

	// Add a player at afterKickoff (after Thursday kickoff).
	addCmd := RawCommand{
		League: rolloverLeague, Command: CorrectRoster, Request: "mfl-refresh.10.0001.roster-add",
		Reason: "Refresh add", Effective: "now", Visibility: "league",
		Args: RawArgs{Correction: &RawCorrection{
			Player: "0099", Franchise: "franchise.01", Action: "add",
		}},
	}
	b.add(addCmd, afterKickoff)

	env := LineupFacts{
		ScheduleHeld: true,
		Locks:        map[string]time.Time{"KC": kickoff},
		Zone:         time.UTC,
	}
	syncedLineup := RawCommand{
		League: rolloverLeague, Command: LineupCorrect,
		Request: "mfl-refresh.10.0002.lineup.1.franchise.01",
		Reason:  "Refresh from MFL: week 1 lineup", Effective: "now", Visibility: "league",
		Args: RawArgs{Lineup: &RawLineupChange{
			Season: 2026, Week: 1, Franchise: "franchise.01", Starters: []string{"0099"},
		}},
	}
	if err := CheckLineupAppend(syncedLineup, b.records, afterKickoff, env); err != nil {
		t.Fatalf("MFL-synced lineup refused after kickoff: %v", err)
	}
	b.add(syncedLineup, afterKickoff)
	gotSynced, foundSynced, err := RecordedLineupFor(b.records, "franchise.01", 2026, 1)
	if err != nil || !foundSynced || !gotSynced.Corrected || !gotSynced.Synced {
		t.Fatalf("expected Synced=true on MFL-synced lineup: %+v %t %v", gotSynced, foundSynced, err)
	}

	// Owner must still be able to submit LineupSet after an MFL-synced lineup.
	// Use a future kickoff so locks don't block the owner; the check under test is the
	// "a commissioner has set ... lineup" guard.
	futureEnv := LineupFacts{
		ScheduleHeld: true,
		Locks:        map[string]time.Time{"KC": afterKickoff.Add(24 * time.Hour)},
		Zone:         time.UTC,
	}
	ownerSet := RawCommand{
		League: rolloverLeague, Command: LineupSet, Request: "owner-override-synced",
		Reason: "Owner sets lineup", Effective: "now", Visibility: "private",
		Args: RawArgs{Lineup: &RawLineupChange{
			Season: 2026, Week: 1, Franchise: "franchise.01", Starters: []string{"0099"},
		}},
	}
	err = CheckLineupAppend(ownerSet, b.records, afterKickoff, futureEnv)
	if err != nil && strings.Contains(err.Error(), "a commissioner has set") {
		t.Fatalf("MFL-synced lineup blocked owner LineupSet: %v", err)
	}
}
