package commissioner

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func jobLedger(t *testing.T, teams int, players []NativePlayer, pendingDraft ...bool) *ledgerBuilder {
	t.Helper()
	create, err := (RawCommand{League: "native.test", Command: Create, Args: RawArgs{Name: "Ours",
		Preset: PresetLegacy, Source: SourceNative, Season: 2026, FranchiseCount: teams}, Request: "create",
		Effective: "now", Visibility: "league"}).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	b := &ledgerBuilder{t: t}
	b.add(create, seasonMoment(1))
	snapshot := &RawNativeSnapshot{Season: 2026, RosterAt: seasonMoment(1), DraftAt: seasonMoment(1), FutureAt: seasonMoment(1),
		Players: players}
	if len(pendingDraft) > 0 && pendingDraft[0] {
		snapshot.Picks = []NativePick{{Season: 2027, Round: 1, Original: "franchise.01", Owner: "franchise.01"}}
	}
	for i := 1; i <= teams; i++ {
		id := fmt.Sprintf("franchise.%02d", i)
		snapshot.Franchises = append(snapshot.Franchises, NativeFranchise{ID: id, HeldID: fmt.Sprintf("%04d", i), Name: id})
	}
	b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: snapshot}, Request: "seed",
		Effective: "now", Visibility: "league"}, seasonMoment(2))
	return b
}

func jobFor(jobs []Job, id string) (Job, bool) {
	for _, job := range jobs {
		if job.ID == id {
			return job, true
		}
	}
	return Job{}, false
}

func TestJobsCarryMatterIdentityAndSettlingCommands(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	reading, err := JobsReading("native.test", b.records, seasonMoment(5))
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range reading.Jobs {
		if job.Identity.League != reading.League || job.Identity.Season != b.first() || job.Identity.Matter != job.ID || job.Identity.Week < 0 {
			t.Errorf("job identity is incomplete: %+v", job)
		}
		if job.Controls == nil || job.Settles == nil {
			t.Errorf("job arrays must be empty, not null: %+v", job)
		}
		if job.State != JobUnbuilt && len(job.Settles) == 0 {
			t.Errorf("built job has no settling command: %+v", job)
		}
		for _, command := range job.Settles {
			if command == RulesSet {
				continue
			}
			if _, err := DefinitionFor(command); err != nil {
				t.Errorf("job %s names unknown settling command %s: %v", job.ID, command, err)
			}
		}
	}
	for _, check := range []struct{ id, command string }{
		{"bids", BidsOverride}, {"waivers", WaiverOverride}, {"seed", NativeSeed},
	} {
		job, ok := jobFor(reading.Jobs, check.id)
		if !ok || !slices.Contains(job.Settles, check.command) {
			t.Errorf("%s does not name %s as a settling command: %+v", check.id, check.command, job)
		}
	}
}

func TestLineupHoleJobsNeedHeldWeekAndCloseAtNextKickoff(t *testing.T) {
	b := jobLedger(t, 32, nil)
	kickoff := time.Date(2026, 10, 11, 17, 0, 0, 0, time.UTC)
	week := leagueweek.Week{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"KC", "BUF"}, Kickoff: kickoff}}}
	inputs := JobInputs{Facts: nativecheck.Facts{"0001": {}}, Weeks: []leagueweek.Week{week}}
	reading, err := JobsReadingWithInputs("native.test", b.records, kickoff.Add(-time.Hour), inputs)
	if err != nil {
		t.Fatal(err)
	}
	job, ok := jobFor(reading.Jobs, "lineup.franchise.01")
	if !ok || job.State != JobLive || job.Closes != kickoff.Format(time.RFC3339) || job.Identity.Week != 1 ||
		!slices.Contains(job.Settles, LineupSet) || !slices.Contains(job.Settles, LineupCorrect) {
		t.Fatalf("missing or misdated lineup job: %+v", job)
	}
	withoutSchedule, err := JobsReadingWithInputs("native.test", b.records, kickoff.Add(-time.Hour), JobInputs{Facts: inputs.Facts})
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(withoutSchedule.Jobs, func(candidate Job) bool { return candidate.ID == job.ID }) {
		t.Fatal("lineup job guessed a lock without a held schedule")
	}
	afterLock, err := JobsReadingWithInputs("native.test", b.records, kickoff, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(afterLock.Jobs, func(candidate Job) bool { return candidate.ID == job.ID }) {
		t.Fatal("lineup job remained after its last lock")
	}

	nextKickoff := kickoff.Add(7 * 24 * time.Hour)
	nextWeek := leagueweek.Week{Number: 2, Games: []leagueweek.Game{{Teams: [2]string{"KC", "BUF"}, Kickoff: nextKickoff}}}
	following, err := JobsReadingWithInputs("native.test", b.records, kickoff.Add(time.Hour),
		JobInputs{Facts: inputs.Facts, Weeks: []leagueweek.Week{
			{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"KC", "BUF"}, Kickoff: kickoff, Final: true}}},
			nextWeek,
		}})
	if err != nil {
		t.Fatal(err)
	}
	nextJob, ok := jobFor(following.Jobs, job.ID)
	if !ok || nextJob.Identity.Season != job.Identity.Season || nextJob.Identity.Week != 2 ||
		nextJob.Identity.Matter != job.Identity.Matter {
		t.Fatalf("weekly recurrence did not receive a new week identity: before=%+v after=%+v", job, nextJob)
	}
}

func TestSoftCapJobsAppearOnlyOverTheHeldSoftCap(t *testing.T) {
	deadline := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		salary domain.Money
		want   bool
	}{{"over", 2_000_000, true}, {"within", 500_000, false}} {
		t.Run(tc.name, func(t *testing.T) {
			player := NativePlayer{ID: "0099", Name: "Held Player", Franchise: "franchise.01", Status: domain.RosterActive,
				Salary: tc.salary, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA}
			b := jobLedger(t, 32, []NativePlayer{player})
			b.set("cap.amount", `1000000`, "now", seasonMoment(3))
			b.set("cap.enforcement", `"soft"`, "now", seasonMoment(3))
			b.set("cap.complianceDeadline", `"2026-11-01T12:00:00Z"`, "now", seasonMoment(3))
			reading, err := JobsReading("native.test", b.records, seasonMoment(5))
			if err != nil {
				t.Fatal(err)
			}
			job, found := jobFor(reading.Jobs, "cap.franchise.01")
			if found != tc.want {
				t.Fatalf("over-cap job found=%t, want %t", found, tc.want)
			}
			if found && (job.Closes != deadline.Format(time.RFC3339) ||
				!slices.Contains(job.Settles, CapAdjust) || !slices.Contains(job.Settles, CapRelief) ||
				!slices.Contains(job.Settles, ReserveException)) {
				t.Fatalf("soft-cap job lost its due time or settling command: %+v", job)
			}
		})
	}
}

func TestTaxiThirdYearJobNeedsHeldExperience(t *testing.T) {
	player := NativePlayer{ID: "0099", Name: "Taxi Player", Franchise: "franchise.01", Status: domain.RosterTaxi,
		ExpirationYear: 2028, ContractStatus: domain.CStatusUFA}
	b := jobLedger(t, 32, []NativePlayer{player})
	b.set("taxi.maximumYears", `2`, "now", seasonMoment(3))
	withoutFacts, err := JobsReadingWithInputs("native.test", b.records, seasonMoment(5), JobInputs{})
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(withoutFacts.Jobs, func(job Job) bool { return job.ID == "taxi.franchise.01.0099" }) {
		t.Fatal("taxi job guessed a player's experience")
	}
	facts := nativecheck.Facts{"0099": {Name: "Taxi Player", DraftYear: 2024, DraftYearHeld: true}}
	withFacts, err := JobsReadingWithInputs("native.test", b.records, seasonMoment(5), JobInputs{Facts: facts})
	if err != nil {
		t.Fatal(err)
	}
	job, found := jobFor(withFacts.Jobs, "taxi.franchise.01.0099")
	if !found || !slices.Contains(job.Settles, ReserveException) {
		t.Fatalf("third-year taxi player job missing its ruling: %+v", job)
	}
	withinLimit := jobLedger(t, 32, []NativePlayer{player})
	withinLimit.set("taxi.maximumYears", `3`, "now", seasonMoment(3))
	allowed, err := JobsReadingWithInputs("native.test", withinLimit.records, seasonMoment(5), JobInputs{Facts: facts})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := jobFor(allowed.Jobs, "taxi.franchise.01.0099"); found {
		t.Fatal("taxi job appeared while held experience stayed within the league limit")
	}
}

func TestVacantSeatAndInvitationJobsUseGovernanceState(t *testing.T) {
	b := jobLedger(t, 32, nil, true)
	jobs, err := JobsReading("native.test", b.records, seasonMoment(5))
	if err != nil {
		t.Fatal(err)
	}
	vacancies := 0
	for _, job := range jobs.Jobs {
		if strings.HasPrefix(job.ID, "vacant.") {
			vacancies++
		}
	}
	if vacancies != 32 {
		t.Fatalf("32-team vacancy coverage = %d, want 32", vacancies)
	}
	withoutDraft := jobLedger(t, 32, nil)
	withoutDraftJobs, err := JobsReading("native.test", withoutDraft.records, seasonMoment(5))
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(withoutDraftJobs.Jobs, func(job Job) bool { return strings.HasPrefix(job.ID, "vacant.") }) {
		t.Fatal("vacant-seat jobs appeared without a pending rookie draft or lineup lock")
	}
	vacancy, found := jobFor(jobs.Jobs, "vacant.franchise.01")
	if !found || !slices.Contains(vacancy.Settles, CaretakerSeat) ||
		!slices.Contains(vacancy.Settles, "commish.seat.invite") || !slices.Contains(vacancy.Settles, "commish.seat.replace") {
		t.Fatalf("vacant-seat job does not name all settling commands: %+v", vacancy)
	}
	small := jobLedger(t, 12, nil)
	smallJobs, err := JobsReading("native.test", small.records, seasonMoment(5))
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(smallJobs.Jobs, func(job Job) bool { return strings.HasPrefix(job.ID, "vacant.") }) {
		t.Fatal("vacant-seat alerts appeared below the 14-franchise threshold")
	}
	draftStart := time.Date(2027, 4, 30, 17, 0, 0, 0, time.UTC)
	b.add(RawCommand{League: "native.test", Command: DraftStart, Args: RawArgs{Draft: &RawDraftArgs{
		Season: 2027, At: draftStart}}, Request: "draft.start", Effective: "now", Visibility: "league"}, seasonMoment(6))
	upcoming, err := JobsReading("native.test", b.records, seasonMoment(7))
	if err != nil {
		t.Fatal(err)
	}
	upcomingVacancy, found := jobFor(upcoming.Jobs, "vacant.franchise.01")
	if !found || upcomingVacancy.State != JobUpcoming || upcomingVacancy.Opens != draftStart.Format(time.RFC3339) ||
		upcomingVacancy.Identity.Season != 2027 {
		t.Fatalf("offseason vacancy does not block the dated rookie draft: %+v", upcomingVacancy)
	}
	invite := RawCommand{League: "native.test", Command: "commish.seat.invite", Args: RawArgs{People: &RawPeople{
		ID: "invite.one", Franchise: "franchise.01", Principal: "owner.one"}}, Request: "invite", Effective: "now", Visibility: "private"}
	b.add(invite, seasonMoment(6))
	pending, err := JobsReading("native.test", b.records, seasonMoment(6))
	if err != nil {
		t.Fatal(err)
	}
	invitation, found := jobFor(pending.Jobs, "invite.invite.one")
	if !found || !slices.Contains(invitation.Settles, "commish.seat.invite.revoke") {
		t.Fatalf("pending invite job missing: %+v", invitation)
	}
	b.add(RawCommand{League: "native.test", Command: "commish.seat.invite.redeem", Args: RawArgs{People: &RawPeople{
		ID: "invite.one", Principal: "owner.one"}}, Request: "redeem", Effective: "now", Visibility: "private"}, seasonMoment(7))
	settled, err := JobsReading("native.test", b.records, seasonMoment(7))
	if err != nil {
		t.Fatal(err)
	}
	if _, found := jobFor(settled.Jobs, "invite.invite.one"); found {
		t.Fatal("redeemed invite job remained")
	}
}

func TestOpenGovernanceVoteJobUsesItsRecordedCloseCommand(t *testing.T) {
	b := jobLedger(t, 32, nil)
	b.add(RawCommand{League: "native.test", Command: "commish.role.grant", Args: RawArgs{People: &RawPeople{
		Principal: "deputy.one", Role: "co-commissioner"}}, Request: "grant", Effective: "now", Visibility: "private"}, seasonMoment(3))
	b.add(RawCommand{League: "native.test", Command: "commish.succession.stepdown", Args: RawArgs{People: &RawPeople{Principal: "creator"}},
		Request: "stepdown", Effective: "now", Visibility: "league"}, seasonMoment(4))
	b.add(RawCommand{League: "native.test", Command: "commish.succession.decline", Args: RawArgs{People: &RawPeople{Principal: "deputy.one"}},
		Request: "decline", Effective: "now", Visibility: "league"}, seasonMoment(5))
	b.records = append(b.records, Record{Command: RawCommand{League: "native.test", Command: "commish.succession.election.open",
		Args: RawArgs{People: &RawPeople{ID: "vote.one", Principal: "deputy.one"}}}, Principal: "deputy.one",
		Receipt: Receipt{Entry: "election", League: "native.test", Revision: int64(len(b.records) + 1), RecordedAt: seasonMoment(6)}})
	jobs, err := JobsReading("native.test", b.records, seasonMoment(7))
	if err != nil {
		t.Fatal(err)
	}
	job, found := jobFor(jobs.Jobs, "vote.vote.one")
	if !found || job.Closes != seasonMoment(13).Format(time.RFC3339) ||
		!slices.Contains(job.Settles, "commish.vote.close") {
		t.Fatalf("open vote job missing or misdated: %+v", job)
	}
}

func TestRolloverJobUsesItsDatedSeasonInstantAndRolloverCommand(t *testing.T) {
	date := time.Date(2027, 3, 9, 5, 0, 0, 0, time.UTC)
	cal := ruleCalendar{rules: leaguerules.CalendarRules{Rollover: leaguerules.Known(date)}}
	jobs := []Job{rolloverJob(cal, date.Add(-time.Nanosecond))}
	identifyJobs("native.test", 2026, jobs)
	job := jobs[0]
	if job.State != JobUpcoming || job.Opens != date.Format(time.RFC3339) ||
		job.Identity.League != "native.test" || job.Identity.Season != 2026 || job.Identity.Matter != "rollover" ||
		!slices.Contains(job.Settles, SeasonRollover) {
		t.Fatalf("rollover job lost its date, identity or command: %+v", job)
	}
	if live := rolloverJob(cal, date); live.State != JobLive {
		t.Fatalf("rollover did not become live at its calendar instant: %+v", live)
	}
}

func TestTradeDeadlineAndOffseasonWindowJobsUseCalendarDates(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	rules.Season = 2026
	deadline := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	rules.Calendar.TradeDeadline = leaguerules.Known(deadline)
	due := deadline.Add(-tradeNoticeLead)
	rules.Calendar.TradeDeadlineWeek = leaguerules.Known(6)
	if _, ok := tradeDeadlineNotice(rules, due.Add(-time.Nanosecond)); ok {
		t.Fatal("trade notice appeared before its calendar lead time")
	}
	notice, ok := tradeDeadlineNotice(rules, due)
	if !ok || notice.Opens != due.Format(time.RFC3339) || notice.Closes != deadline.Format(time.RFC3339) ||
		!slices.Contains(notice.Settles, NoticePublish) || notice.Identity.Week != 6 {
		t.Fatalf("trade-deadline notice has wrong window or command: %+v", notice)
	}
	if _, ok := tradeDeadlineNotice(rules, deadline); ok {
		t.Fatal("trade notice remained after the deadline")
	}
	start, end := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC), time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	rollDate := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	rolled := newLedger(t, PresetLegacy)
	rolled.set("calendar.rollover", `"2026-11-01T00:00:00Z"`, "now", seasonMoment(5))
	rules.Calendar.Rollover = leaguerules.Known(rollDate)
	windows := rules.Calendar.Windows.Value
	windows["special_draft"] = leaguerules.Window{Opens: leaguerules.Known([]leaguerules.Action{"draft_pick"}),
		DatedStart: leaguerules.Known(start), DatedEnd: leaguerules.Known(end)}
	windows["rookie_draft"] = windows["special_draft"]
	rules.Calendar.Windows = leaguerules.Known(windows)
	if blocked := offseasonWindowJobs(rolled.records, rules, start.Add(-time.Second)); len(blocked) != 0 {
		t.Fatalf("offseason windows appeared before rollover: %+v", blocked)
	}
	rollover := Record{Command: RawCommand{Command: SeasonRollover}, Before: &RolloverBefore{},
		Receipt: Receipt{Entry: "season-rollover", RecordedAt: time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC),
			Plan: &RolloverPlan{}}}
	rolledRecords := append(slices.Clone(rolled.records), rollover)
	upcoming := offseasonWindowJobs(rolledRecords, rules, start.Add(-time.Second))
	window, found := jobFor(upcoming, "offseason_window.special_draft")
	if !found || window.State != JobUpcoming || window.Opens != start.Format(time.RFC3339) ||
		!slices.Contains(window.Settles, DraftPick) {
		t.Fatalf("upcoming window lost its date or command: %+v", window)
	}
	if _, found := jobFor(upcoming, "offseason_window.rookie_draft"); found {
		t.Fatal("dedicated rookie-draft job was duplicated by its calendar window")
	}
	live := offseasonWindowJobs(rolledRecords, rules, start)
	window, found = jobFor(live, "offseason_window.special_draft")
	if !found || window.State != JobLive || window.Closes != end.Format(time.RFC3339) {
		t.Fatalf("live window lost its close time: %+v", window)
	}
	if ended := offseasonWindowJobs(rolledRecords, rules, end); slices.ContainsFunc(ended, func(job Job) bool {
		return job.ID == "offseason_window.special_draft"
	}) {
		t.Fatal("closed calendar window remained in the feed")
	}
}

func TestBidTieJobAppearsUntilItsAwardOverride(t *testing.T) {
	book := BidBook{Auctions: []BidAuction{{Player: "0099", Season: 2026, Decided: true,
		Tied: []string{"bid.one", "bid.two"}}}}
	jobs := bidTieJobs(book, 2026, 8)
	job, found := jobFor(jobs, "bid_tie.2026.one-pool.0099")
	if !found || job.Identity.Season != 2026 || job.Identity.Week != 8 || !slices.Contains(job.Settles, BidsOverride) {
		t.Fatalf("unresolved tie lacks a dated identity or override: %+v", job)
	}
	book.Auctions[0].Tied = nil
	if settled := bidTieJobs(book, 2026, 8); len(settled) != 0 {
		t.Fatalf("overridden bid tie remained: %+v", settled)
	}
}
