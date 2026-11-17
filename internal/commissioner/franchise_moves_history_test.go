package commissioner

import (
	"slices"
	"testing"
	"time"
)

func TestFranchiseHistorySeasonPagesPreserveCorrectionsAndAudience(t *testing.T) {
	b := seededLedger(t)
	cut := b.add(RawCommand{League: "native.test", Command: RosterCut, Visibility: "league",
		Args: RawArgs{Waiver: &RawWaiverArgs{Player: "0099", Franchise: "franchise.01"}}}, seasonMoment(3))
	b.records[len(b.records)-1].TokenAuditID = "test-helper"
	correction := b.add(RawCommand{League: "native.test", Command: CorrectTransaction, Visibility: "league",
		Reason: "Wrong player", Args: RawArgs{Correction: &RawCorrection{Entry: cut}}}, seasonMoment(4))
	secret := b.add(RawCommand{League: "native.test", Command: BidPlace, Visibility: "private",
		Args: RawArgs{Bid: &RawBidArgs{Franchise: "franchise.02", Player: "0100"}}}, seasonMoment(5))
	b.records[len(b.records)-1].Principal = "other"
	b.add(RawCommand{League: "native.test", Command: SeasonRollover, Visibility: "league",
		Args: RawArgs{Rollover: &RawRolloverArgs{To: 2027}}}, time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC))
	b.records[len(b.records)-1].Receipt.Plan = &RolloverPlan{From: 2026, To: 2027}
	b.records[len(b.records)-1].Before = &RolloverBefore{}
	next := b.add(RawCommand{League: "native.test", Command: RosterCut, Visibility: "league",
		Args: RawArgs{Waiver: &RawWaiverArgs{Player: "0100", Franchise: "franchise.01"}}}, time.Date(2027, 3, 2, 0, 0, 0, 0, time.UTC))
	read := func(franchise, principal string, season int) Ledger {
		t.Helper()
		got, err := FranchiseMovesHistoryReading(RawFranchiseMovesHistory{"native.test", franchise, season}, principal, b.records, CreatorAuthorizer{Creator: "creator"})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	ids := func(l Ledger) []string {
		out := []string{}
		for _, e := range l.Entries {
			out = append(out, e.ID)
		}
		return out
	}
	page := read("franchise.01", "member", 2026)
	if !slices.Contains(ids(page), cut) || !slices.Contains(ids(page), correction) || slices.Contains(ids(page), secret) || slices.Contains(ids(page), next) {
		t.Fatalf("original/correction/season privacy: %+v", page)
	}
	if !slices.ContainsFunc(page.Entries, func(e Entry) bool { return e.ID == cut && e.Who == "via AI/CLI, acting for creator" }) {
		t.Fatal("actor stamp lost")
	}
	if slices.Contains(ids(read("franchise.02", "member", 2026)), cut) {
		t.Fatal("unrelated franchise move included")
	}
	if slices.Contains(ids(read("franchise.02", "member", 2026)), secret) {
		t.Fatal("sealed bid leaked to member")
	}
	if !slices.Contains(ids(read("franchise.02", "other", 2026)), secret) {
		t.Fatal("owner cannot see own bid")
	}
	if slices.Contains(ids(read("franchise.01", "member", 2027)), cut) || !slices.Contains(ids(read("franchise.01", "member", 2027)), next) {
		t.Fatal("season pages overlap or omit move")
	}
	if len(read("franchise.01", "member", 2025).Entries) != 0 {
		t.Fatal("empty season not empty")
	}
}

func TestFranchiseHistoryRejectsInvalidScope(t *testing.T) {
	for _, scope := range []RawFranchiseMovesHistory{
		{"", "franchise.01", 2026}, {"native.test", " ", 2026}, {"native.test", "franchise.01", 0}, {"native.test", "franchise.01", 10000},
	} {
		if err := scope.Validate(); err == nil {
			t.Fatalf("invalid scope: %+v", scope)
		}
	}
	b := newLedger(t, PresetEmpty)
	if _, err := FranchiseMovesHistoryReading(RawFranchiseMovesHistory{"native.test", "franchise.missing", 2026}, "creator", b.records, CreatorAuthorizer{Creator: "creator"}); err == nil {
		t.Fatal("unknown franchise accepted")
	}
}

func TestFranchiseHistoryPickCorrectionKeepsBothOwnersInRecordedSeason(t *testing.T) {
	b := seededLedger(t)
	pick := b.records[1].Command.Args.Native.Picks[0]
	owner := "franchise.02"
	if owner == pick.Owner {
		owner = "franchise.01"
	}
	id := b.add(RawCommand{League: "native.test", Command: CorrectPick, Visibility: "league",
		Args: RawArgs{Correction: &RawCorrection{Owner: owner, Pick: &CorrectionPick{
			Season: pick.Season, Round: pick.Round, Original: pick.Original, Slot: pick.Slot}}}}, seasonMoment(3))
	for _, franchise := range []string{pick.Owner, owner} {
		page, err := FranchiseMovesHistoryReading(RawFranchiseMovesHistory{"native.test", franchise, 2026}, "creator", b.records, CreatorAuthorizer{Creator: "creator"})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(page.Entries, func(e Entry) bool { return e.ID == id }) {
			t.Fatalf("pick correction absent for %s", franchise)
		}
	}
}
