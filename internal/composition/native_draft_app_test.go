package composition

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	// standingsWords is what round one reads in a league whose previous season has no schedule: the
	// standings are built, and say why they cannot rank anyone yet.
	standingsWords = "The draft order needs the final standings of season 2026. The league schedule is not " +
		"published yet, so there are no games to rank."
	laterRound = "Needs a ruling: later round draft pattern"
)

// governs is the survey's round trip for one setting of the draft: read, set through the real command,
// read the consequence, reopen the database for real, make the compensating revert and read again; then do
// it all again from a freshly reopened database.
func (f *draftFixture) governs(t *testing.T, read func() string, set func() commissioner.Receipt, before,
	after string) {
	t.Helper()
	for pass, name := range []string{"live", "after reopening the database"} {
		if pass == 1 {
			f.reopen(t)
		}
		if got := read(); got != before {
			t.Fatalf("%s: before the ruling: %q, want %q", name, got, before)
		}
		receipt := set()
		if got := read(); got != after {
			t.Fatalf("%s: after the ruling: %q, want %q", name, got, after)
		}
		f.reopen(t)
		if got := read(); got != after {
			t.Fatalf("%s: the ruling did not survive reopening: %q, want %q", name, got, after)
		}
		f.revert(t, receipt)
		if got := read(); got != before {
			t.Fatalf("%s: after the compensating revert: %q, want %q", name, got, before)
		}
	}
}

func deadlineOf(d commissioner.NativeDraft) string {
	if d.Draft.OnClock == nil {
		return d.Words
	}
	return d.Draft.OnClock.Deadline.Format(time.RFC3339)
}

func onClockOf(d commissioner.NativeDraft) string {
	if d.Draft.OnClock == nil {
		return d.Words
	}
	c := d.Draft.OnClock
	return fmt.Sprintf("round %d pick %d: %s (owned by %s)", c.Round, c.Slot, c.Original, c.Owner)
}

func draftRuleWords(d commissioner.NativeDraft, label string) string {
	for _, line := range d.Rules {
		if line.Label == label {
			return line.Words
		}
	}
	return "no such rule line: " + label
}

func TestDraftClockAndRoundsGovernTheDraft(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(time.Hour)
	read := func() string { return deadlineOf(f.read(t, "franchise.03")) }
	f.governs(t, read, func() commissioner.Receipt { return f.setRule(t, "draft.clock", `7200000000000`) },
		"2027-07-14T15:00:00-07:00", "2027-07-14T11:00:00-07:00")
	words := func() string { return draftRuleWords(f.read(t, "franchise.03"), "Draft pick clock") }
	f.governs(t, words, func() commissioner.Receipt { return f.setRule(t, "draft.clock", `5400000000000`) },
		"6 hours", "1 hour 30 minutes")
	rounds := func() string {
		d := f.read(t, "franchise.03")
		return fmt.Sprint(len(d.Order))
	}
	f.governs(t, rounds, func() commissioner.Receipt { return f.setRule(t, "draft.rounds", `3`) }, "5", "3")
}

func TestDraftOrderCommandsGovernWhoIsOnTheClock(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.draftCommand(t, commissioner.DraftStart, commissioner.RawDraftArgs{Season: draftSeason, At: draftStartAt()})
	round := func(n int) string {
		order := f.read(t, "franchise.01").Order
		if len(order[n-1].Picks) == 0 {
			return order[n-1].Words
		}
		return order[n-1].Words + ": " + order[n-1].Picks[0].Original
	}
	if got := round(1); got != standingsWords {
		t.Fatalf("round one invented an order: %q", got)
	}
	first := func() commissioner.Receipt {
		return f.draftCommand(t, commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason,
			Round: 1, Order: draftOrder(3, 2, 1)})
	}
	f.governs(t, func() string { return round(1) }, first, standingsWords, "Draft order set: franchise.03")
	first()
	if got := round(2); got != laterRound {
		t.Fatalf("the later-round pattern was defaulted instead of needing a ruling: %q", got)
	}
	pattern := func(p leaguerules.OrderPattern) func() commissioner.Receipt {
		return func() commissioner.Receipt {
			return f.draftCommand(t, commissioner.DraftOrderPattern, commissioner.RawDraftArgs{Pattern: p})
		}
	}
	f.governs(t, func() string { return round(2) }, pattern(leaguerules.Linear), laterRound,
		"Draft order set: franchise.03")
	f.governs(t, func() string { return round(2) }, pattern(leaguerules.Snake), laterRound,
		"Draft order set: franchise.32")
	pattern(leaguerules.Linear)()
	f.now = draftStartAt().Add(time.Hour)
	onClock := func() string { return onClockOf(f.read(t, "franchise.01")) }
	f.governs(t, onClock, func() commissioner.Receipt {
		return f.draftCommand(t, commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason,
			Order: draftOrder(5, 6), Running: true})
	}, "round 1 pick 1: franchise.03 (owned by franchise.03)",
		"round 1 pick 1: franchise.05 (owned by franchise.05)")
}

func TestDraftTieRuleIsRecordedAndReverted(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	tie := func() string { return draftRuleWords(f.read(t, "franchise.01"), "Draft order tie breaker") }
	f.governs(t, tie, func() commissioner.Receipt {
		return f.draftCommand(t, commissioner.DraftOrderTiebreak, commissioner.RawDraftArgs{Tie: "Better record"})
	}, "Needs a ruling", "Better record")
}

func TestDraftOrderAutoAsksForTheFinalStandings(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.draftCommand(t, commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason,
		Round: 1, Order: draftOrder(3, 2, 1)})
	read := func() string { return f.read(t, "franchise.01").Order[0].Words }
	f.governs(t, read, func() commissioner.Receipt {
		return f.draftCommand(t, commissioner.DraftOrderAuto, commissioner.RawDraftArgs{Season: draftSeason,
			Source: leaguerules.WorstStandingsFirst})
	}, "Draft order set", standingsWords)
}

// calendarDraftStart reads the draft window's opening as the league calendar shows it.
func calendarDraftStart(t *testing.T, f *draftFixture) string {
	t.Helper()
	reading, err := newCommissionerAPI(f.app).TargetLeagueCalendar(RawLeagueCalendarScope{League: draftLeague, Season: draftSeason,
		LocalZone: "America/Los_Angeles", Franchise: "franchise.01", Action: "draft_pick"})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range reading.Clock.Windows {
		if w.ID == "rookie_draft" {
			return w.Start.Format(time.RFC3339)
		}
	}
	t.Fatal("the calendar has no rookie draft window")
	return ""
}

func TestDraftStartAndTheCalendarNeverDisagree(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.draftCommand(t, commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason,
		Order: draftOrder(3, 2, 1)})
	both := func() string {
		calendar, draft := calendarDraftStart(t, f), f.read(t, "franchise.01").Draft.Start.Format(time.RFC3339)
		if calendar != draft {
			t.Fatalf("the calendar opens the draft at %s but the draft starts at %s", calendar, draft)
		}
		return calendar
	}
	set := func(at time.Time) func() commissioner.Receipt {
		return func() commissioner.Receipt {
			return f.draftCommand(t, commissioner.DraftStart, commissioner.RawDraftArgs{Season: draftSeason, At: at})
		}
	}
	// With no start recorded the draft opens where the league's own window does: 14 July, midnight in New York.
	f.governs(t, both, set(draftStartAt()), "2027-07-13T21:00:00-07:00", "2027-07-14T09:00:00-07:00")
	first := set(draftStartAt())()
	moved := set(draftStartAt().Add(48 * time.Hour))()
	for pass := 0; pass < 2; pass++ {
		if got := both(); got != "2027-07-16T09:00:00-07:00" {
			t.Fatalf("the moved start: %s", got)
		}
		f.reopen(t)
	}
	f.revert(t, moved)
	if got := both(); got != "2027-07-14T09:00:00-07:00" {
		t.Fatalf("after reversing the move: %s", got)
	}
	f.revert(t, first)
	f.reopen(t)
	if got := both(); got != "2027-07-13T21:00:00-07:00" {
		t.Fatalf("after reversing every start: %s", got)
	}
}

func TestDraftStartRefusals(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.draftCommand(t, commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason,
		Order: draftOrder(3, 2, 1)})
	tryStart := func(at time.Time) error {
		_, err := f.tryCommand(commissioner.DraftStart, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{
			Season: draftSeason, At: at}}, "")
		return err
	}
	if err := tryStart(f.now.Add(-time.Hour)); err == nil || !strings.Contains(err.Error(), "has not passed") {
		t.Fatalf("a start in the past: %v", err)
	}
	f.draftCommand(t, commissioner.DraftStart, commissioner.RawDraftArgs{Season: draftSeason, At: draftStartAt()})
	if _, err := f.tryCommand(commissioner.WindowMove, commissioner.RawArgs{Window: &commissioner.RawWindowArgs{
		ID: "rookie_draft", Start: date(8, 1)}}, ""); err == nil || !strings.Contains(err.Error(), "agree") {
		t.Fatalf("a window move beside the draft start: %v", err)
	}
	f.now = draftStartAt().Add(time.Minute)
	if err := tryStart(draftStartAt().Add(24 * time.Hour)); err == nil ||
		!strings.Contains(err.Error(), "can no longer move") {
		t.Fatalf("a start moved after the clock runs: %v", err)
	}
}

func onClockContract(d commissioner.NativeDraft) string {
	if d.Draft.OnClock == nil {
		return d.Words
	}
	c := d.Draft.OnClock.Contract
	return fmt.Sprintf("%d years at %d cents, restricted free agency %v", c.Years, c.Salary, c.RFAAfter)
}

func TestDraftPricingAndLengthsGovernTheContractOnTheClock(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(time.Hour)
	read := func() string { return onClockContract(f.read(t, "franchise.03")) }
	const legacy = "5 years at 300000000 cents, restricted free agency false"
	f.governs(t, read, func() commissioner.Receipt {
		return f.setRule(t, "draft.lengths", `{"1":4,"2":3,"3":3,"4":3,"5":3}`)
	}, legacy, "4 years at 300000000 cents, restricted free agency false")
	f.governs(t, read, func() commissioner.Receipt {
		return f.setRule(t, "draft.rfaAfter", `{"1":true,"2":true,"3":true,"4":true,"5":true}`)
	}, legacy, "5 years at 300000000 cents, restricted free agency true")
	flat := `{"1":[{"Slots":{"Min":1,"Max":32},"Salary":123400000}],"2":[{"Slots":{"Min":1,"Max":32},"Salary":1}],` +
		`"3":[{"Slots":{"Min":1,"Max":32},"Salary":1}],"4":[{"Slots":{"Min":1,"Max":32},"Salary":1}],` +
		`"5":[{"Slots":{"Min":1,"Max":32},"Salary":1}]}`
	f.governs(t, read, func() commissioner.Receipt { return f.setRule(t, "draft.pricing", flat) }, legacy,
		"5 years at 123400000 cents, restricted free agency false")
}

func TestDraftMaximumPicksForfeitsTheExtrasWithoutAClock(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(time.Hour)
	forfeited := func() string {
		count := 0
		for _, pick := range f.read(t, "franchise.01").Draft.Picks {
			if pick.Forfeited && pick.Owner == "franchise.01" {
				count++
			}
		}
		return fmt.Sprint(count)
	}
	// franchise.01 holds two first round picks and one in each later round: six, so one allowed forfeits five.
	f.governs(t, forfeited, func() commissioner.Receipt { return f.setRule(t, "draft.maximumPicks", `1`) }, "0", "5")
}

func TestCaretakerPicksOnlyForVacantFranchiseAndStopsAtOwnerSeating(t *testing.T) {
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(time.Minute)
	f.command(t, commissioner.CaretakerSeat, commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.03", Principal: "creator"}}, "Care for a vacant team")
	f.pick(t, "franchise.03", "5001")
	f.seat(t, "franchise.03", "new-owner")
	before := f.revision(t)
	if _, err := f.tryPick("franchise.03", "5002", "closed-caretaker", before); err == nil {
		t.Fatal("caretaker acted after owner was seated")
	}
	if f.revision(t) != before {
		t.Fatal("refused caretaker appended a pick")
	}
}

type agentOwnerHost struct {
	leagueservice.StoredHost
	at    func() time.Time
	facts nativecheck.Facts
}

func (h agentOwnerHost) CommissionerNow() time.Time     { return h.at() }
func (h agentOwnerHost) NativeFacts() nativecheck.Facts { return h.facts }

func ownerAgent(t *testing.T, pools *db.Pools, app *App, league, franchise, command string, at func() time.Time) (*leagueservice.Service, string, string) {
	t.Helper()
	credentials := accountstore.New(pools)
	if err := credentials.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	passwords := accounts.New(credentials)
	login := accounts.RawCredentials{Username: "agent-commissioner", Password: "disposable-agent-password"}
	account, err := passwords.LocalBootstrap(t.Context(), login)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	for i, change := range []struct {
		command string
		people  commissioner.RawPeople
	}{
		{"commish.role.grant", commissioner.RawPeople{Principal: string(principal), Role: governance.Commissioner}},
		{commissioner.CaretakerSeat, commissioner.RawPeople{Principal: string(principal), Franchise: franchise}},
	} {
		events, err := app.core.History.Events(t.Context(), league, commissioner.Stream, 0)
		if err != nil {
			t.Fatal(err)
		}
		actor := "creator"
		if i == 1 {
			actor = string(principal)
		}
		raw := commissioner.RawCommand{League: league, Command: change.command, Args: commissioner.RawArgs{People: &change.people}, Request: fmt.Sprintf("agent-setup-%d", i), Expected: int64(len(events)), Reason: "Prepare caretaker", Effective: "now", Visibility: "league"}
		if i == 0 {
			raw.Visibility = "private"
		}
		if _, err := app.core.History.ExecuteCommissioner(t.Context(), raw, actor, nil); err != nil {
			t.Fatal(err)
		}
	}
	sessions := accounts.NewSessions(passwords, credentials)
	session, err := sessions.Login(t.Context(), login, "agent-test")
	if err != nil {
		t.Fatal(err)
	}
	tokens := accounts.NewTokens(credentials, serverhttp.TokenPolicy{Store: app.core.History})
	token, err := tokens.Issue(t.Context(), session.RawSession, accounts.RawTokenScope{League: league, Franchise: franchise, Commands: []string{command, "commish.ledger.read"}, Modes: []string{"execute", "read"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := commissioner.WithTokenAudit(t.Context(), token.ID)
	host := agentOwnerHost{StoredHost: leagueservice.StoredHost{Store: app.core.History, Principal: string(principal), Context: func() context.Context { return ctx }}, at: at, facts: app.nativeFacts()}
	return leagueservice.NewHosted(host, tokens.Admission(token.RawSession)), string(principal), token.ID
}

func TestTokenCaretakerDraftUsesDraftRoomAndCurrentSeat(t *testing.T) {
	f := draftApp(t)
	f.scheduled(t, 3, 4, 1)
	f.now = draftStartAt().Add(time.Minute)
	service, principal, auditID := ownerAgent(t, f.pools, f.app, draftLeague, "franchise.03", commissioner.DraftPick, func() time.Time { return f.now })
	in := commissioner.DraftPickInput{League: draftLeague, Season: draftSeason, Franchise: "franchise.04", Player: "5001", Request: "wrong-team", Expected: ownerAgentRevision(t, service, draftLeague)}
	if _, err := service.Pick(commissioner.WithTokenAudit(t.Context(), auditID), in); err == nil {
		t.Fatal("token escaped caretaker franchise")
	}
	in.Franchise, in.Request = "franchise.03", "agent-pick"
	if _, err := service.Pick(commissioner.WithTokenAudit(t.Context(), auditID), in); err != nil {
		t.Fatal(err)
	}
	events, err := f.app.core.History.Events(t.Context(), draftLeague, commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	last := events[len(events)-1].Payload.Commissioner
	if last.TokenAuditID == "" || last.Principal != principal {
		t.Fatal("draft lost token audit")
	}
	in.Expected, in.Player, in.Request = ownerAgentRevision(t, service, draftLeague), "5002", "off-clock"
	if _, err := service.Pick(commissioner.WithTokenAudit(t.Context(), auditID), in); err == nil || !strings.Contains(err.Error(), "turn") {
		t.Fatalf("off-clock token pick: %v", err)
	}
	f.seat(t, "franchise.03", "new-owner")
	in.Expected = ownerAgentRevision(t, service, draftLeague)
	if _, err := service.Pick(commissioner.WithTokenAudit(t.Context(), auditID), in); err == nil {
		t.Fatal("former caretaker token acted")
	}
}

func ownerAgentRevision(t *testing.T, service *leagueservice.Service, league string) int64 {
	t.Helper()
	reading, err := service.TargetCommissionerLedger(league)
	if err != nil {
		t.Fatal(err)
	}
	return reading.Revision
}
