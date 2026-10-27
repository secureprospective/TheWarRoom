package commissioner

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func principalRecords(t *testing.T, records []Record, aliases ...string) ([]Record, string) {
	t.Helper()
	const account = "acct.12345678-1234-4234-8234-123456789abc"
	verified := make([]governance.VerifiedAlias, 0, len(aliases))
	for _, alias := range aliases {
		verified = append(verified, governance.VerifiedAlias{League: "native.test", Alias: governance.Principal(alias), Account: account})
	}
	principals, err := governance.NewPrincipals("native.test", verified)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolvePrincipals(WithPrincipals(t.Context(), principals), "native.test", records)
	if err != nil {
		t.Fatal(err)
	}
	return resolved, account
}

func TestAliasPrivateOwnerReadingsAndDraftRetry(t *testing.T) {
	t.Parallel()
	b := newLedger(t, PresetEmpty)
	records, account := principalRecords(t, b.records, "legacy-owner", "other-install")
	for _, command := range []string{BidPlace, BidRaise, BidWithdraw, WaiverClaim} {
		private := Record{Principal: "legacy-owner", Command: RawCommand{Command: command, Visibility: "private"}}
		if !ledgerEntryVisible(private, private, records, account, false) {
			t.Fatalf("own alias hidden for %s", command)
		}
		if ledgerEntryVisible(private, private, records, "unverified", false) {
			t.Fatalf("private alias leaked for %s", command)
		}
		if private.Principal != "legacy-owner" {
			t.Fatal("original author changed")
		}
	}
	if err := authorizeLineupReading("native.test", account, "legacy-owner", false, records); err != nil {
		t.Fatal(err)
	}
	if err := authorizeLineupReading("native.test", "unverified", "legacy-owner", false, records); err == nil {
		t.Fatal("private lineup leaked")
	}
	in := DraftPickInput{Request: "pick", Season: 2026, Franchise: "one", Player: "1234"}
	pick := Record{Principal: "other-install", Command: RawCommand{Command: DraftPick, Request: in.Request, Args: RawArgs{Draft: &RawDraftArgs{Season: 2026, Pick: &RawDraftSelection{Franchise: in.Franchise, Player: in.Player}}}}, Receipt: Receipt{Entry: "pick.receipt"}}
	records = append(records, pick)
	receipt, retried, err := RetriedDraftPick(in, account, records)
	if err != nil || !retried || receipt.Entry != "pick.receipt" {
		t.Fatalf("draft alias retry: %+v %t %v", receipt, retried, err)
	}
	if _, _, err := RetriedDraftPick(in, "unverified", records); err == nil {
		t.Fatal("draft retry stolen")
	}
}

func TestAliasesReplayAcrossClosedBooksWithoutRewritingThem(t *testing.T) {
	t.Parallel()
	b := rolloverLedger(t)
	b.rollover(2027, rolloverMoment())
	raw, err := CloseBooks(b.records)
	if err != nil {
		t.Fatal(err)
	}
	folded, ok := WithClosedBooks(b.records, raw)
	if !ok {
		t.Fatal("closed books not attached")
	}
	before, err := json.Marshal(folded)
	if err != nil {
		t.Fatal(err)
	}
	records, account := principalRecords(t, folded, "creator")
	state, err := GovernanceState(records)
	if err != nil || !state.HasRole(governance.Principal(account), governance.Commissioner) {
		t.Fatalf("alias lost at rollover: %v", err)
	}
	if usableClosedBooks(records) != nil || usableClosedBooks(folded) == nil {
		t.Fatal("alias view reused stale identities or changed shared books")
	}
	after, err := json.Marshal(folded)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("closed history changed")
	}
	original, err := GovernanceState(folded)
	if err != nil || !original.HasRole("creator", governance.Commissioner) || original.HasRole(governance.Principal(account), governance.Commissioner) {
		t.Fatal("identity view escaped into original replay")
	}
}

func TestAliasResolutionDoesNotChangeYesNoBallots(t *testing.T) {
	t.Parallel()
	b := newLedger(t, PresetEmpty)
	records, account := principalRecords(t, b.records, "yes", "candidate")
	raw := RawCommand{Command: "commish.vote.cast", Args: RawArgs{People: &RawPeople{ID: "vote", Principal: "yes", Choice: "yes"}}}
	got := policyCommand(raw, records)
	if got.Args.People.Principal != account || got.Args.People.Choice != "yes" || raw.Args.People.Principal != "yes" {
		t.Fatal("vote-out choice treated as identity or input mutated")
	}
	records = append(records, Record{Command: RawCommand{Command: "commish.succession.election.open", Args: RawArgs{People: &RawPeople{ID: "election"}}}})
	raw.Args.People.ID, raw.Args.People.Choice = "election", "candidate"
	if got := policyCommand(raw, records); got.Args.People.Choice != account {
		t.Fatal("election candidate alias unresolved")
	}
}
