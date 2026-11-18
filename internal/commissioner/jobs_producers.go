package commissioner

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const tradeNoticeLead = 72 * time.Hour

func settlingCommands(id string) []string {
	if commands, ok := rosterSettles(id); ok {
		return commands
	}
	if commands, ok := peopleSettles(id); ok {
		return commands
	}
	if commands, ok := marketSettles(id); ok {
		return commands
	}
	if commands, ok := workflowSettles(id); ok {
		return commands
	}
	return seasonSettles(id)
}

func rosterSettles(id string) ([]string, bool) {
	switch {
	case strings.HasPrefix(id, "contract."):
		return []string{ContractEdit}, true
	case strings.HasPrefix(id, "pick."):
		return []string{CorrectPick}, true
	case strings.HasPrefix(id, "must_cut."):
		return []string{RosterCut, CorrectRoster, CapAdjust}, true
	case strings.HasPrefix(id, "oline"):
		return []string{OLineAuctionClose, BidsOverride, CapAdjust, RosterCut}, true
	case strings.HasPrefix(id, "startup_draft"):
		return []string{DraftStartupOpen, DraftPick, DraftSkip, DraftAssign, DraftChange, DraftArchive}, true
	case strings.HasPrefix(id, "lineup."):
		return []string{LineupSet, LineupCorrect}, true
	case strings.HasPrefix(id, "cap."):
		return []string{CapAdjust, CapRelief, ReserveException}, true
	case strings.HasPrefix(id, "taxi."):
		return []string{ReserveException}, true
	}
	return nil, false
}

func peopleSettles(id string) ([]string, bool) {
	switch {
	case strings.HasPrefix(id, "vacant."):
		return []string{CaretakerSeat, "commish.seat.invite", "commish.seat.replace"}, true
	case strings.HasPrefix(id, "invite."):
		return []string{"commish.seat.invite.redeem", "commish.seat.invite.revoke", "commish.seat.replace"}, true
	case strings.HasPrefix(id, "vote."):
		return []string{"commish.vote.close", "commish.vote.tiebreak"}, true
	}
	return nil, false
}

func marketSettles(id string) ([]string, bool) {
	switch {
	case strings.HasPrefix(id, "bid_tie."):
		return []string{BidsOverride}, true
	case strings.HasPrefix(id, "waiver_tie."):
		return []string{WaiverOverride}, true
	}
	return nil, false
}

func workflowSettles(id string) ([]string, bool) {
	switch id {
	case "setup":
		return []string{RulesSet}, true
	case "rollover":
		return []string{SeasonRollover}, true
	case "contract_season":
		return []string{ContractActionSet, ContractEdit}, true
	case "notifications":
		return []string{}, true
	case "waivers":
		return []string{WaiverProcess, WaiverOverride, WaiverOrderSet, WaiverOrderAuto}, true
	case "bids":
		return []string{BidsProcess, BidsOverride}, true
	case "seed":
		return []string{NativeSeed}, true
	}
	return nil, false
}

func seasonSettles(id string) []string {
	switch id {
	case "rookie_draft":
		return []string{DraftStart, DraftOrderSet, DraftOrderAuto, DraftOrderPattern, DraftOrderTiebreak,
			DraftPick, DraftSkip, DraftAssign, DraftChange, DraftArchive}
	case "stat_queue":
		return []string{StatApprove, StatReject}
	case "dot_review":
		return []string{TradeReview, TradeAppeal, TradeAppealRule, TradeOverride}
	case "playoffs":
		return []string{BracketBuild, BracketEdit, BracketSeed, BracketAdvance, TitleSet}
	case "in_season":
		return []string{CorrectRoster, CorrectTransaction, CorrectPick, CorrectProvenance, StatCorrect,
			LineupCorrect, PlayerPosition, PlayerPositionClear, ScorePlayer, ScoreFranchise,
			StandingsOverride, StandingsOverrideClear, StandingsRecordAdjust, RuleException}
	case "communications":
		return []string{NoticePublish, NotePost}
	case "trade_deadline_notice":
		return []string{NoticePublish}
	}
	return []string{}
}

func d60LedgerJobs(league string, records []Record, rules leaguerules.RuleSet, at time.Time, inputs JobInputs,
	assets NativeAssets) ([]Job, error) {
	jobs := []Job{}
	week, lock, hasLock := nextLineupLock(at, inputs.Weeks)
	if assets.Seeded {
		lineups, err := lineupHoleJobs(league, records, rules, at, inputs, week, lock, hasLock)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, lineups...)
	}
	peopleJobs, err := governanceJobs(league, records, rules, at, week, lock, hasLock, assets)
	if err != nil {
		return nil, err
	}
	jobs = append(jobs, peopleJobs...)
	if notice, ok := tradeDeadlineNotice(rules, at); ok {
		jobs = append(jobs, notice)
	}
	if !assets.Seeded {
		return jobs, nil
	}
	capJobs, err := softCapJobs(league, records, rules, at, week)
	if err != nil {
		return nil, err
	}
	jobs = append(jobs, capJobs...)
	taxiRows, err := taxiJobs(league, records, rules, at, inputs.Facts, week, lock, hasLock)
	if err != nil {
		return nil, err
	}
	jobs = append(jobs, taxiRows...)
	return jobs, nil
}

func nextLineupLock(at time.Time, weeks []leagueweek.Week) (leagueweek.Week, time.Time, bool) {
	week, ok := leagueweek.LineupWeek(at, weeks...)
	if !ok {
		return leagueweek.Week{}, time.Time{}, false
	}
	lock, ok := leagueweek.NextLock(at, week)
	return week, lock, ok
}

func lineupHoleJobs(league string, records []Record, rules leaguerules.RuleSet, at time.Time, inputs JobInputs,
	week leagueweek.Week, lock time.Time, hasLock bool) ([]Job, error) {
	if !hasLock || len(inputs.Facts) == 0 {
		return nil, nil
	}
	facts := LineupFacts{Facts: inputs.Facts, Locks: leagueweek.Locks(week), ScheduleHeld: true, Zone: time.UTC}
	franchises := LeagueFranchises(records)
	lineups, err := NativeLineupsReading(league, Replay(records).Creator, franchises, week.Number, records, at, facts)
	if err != nil {
		return nil, fmt.Errorf("read lineup jobs: %w", err)
	}
	jobs := []Job{}
	for _, lineup := range lineups {
		short := slices.ContainsFunc(lineup.Verdict.Problems, func(problem nativecheck.LineupProblem) bool {
			return problem.Kind == nativecheck.ProblemShort
		})
		if lineup.Verdict.Flagged || lineup.Verdict.Status == nativecheck.LineupInvalid && short {
			jobs = append(jobs, Job{ID: "lineup." + lineup.Franchise, Name: "Lineup with a hole — " + lineup.FranchiseName,
				State: JobLive, Closes: lock.UTC().Format(time.RFC3339), Controls: []string{"CM-028", "CM-029"},
				Words: lineup.Verdict.Words, Identity: JobIdentity{Season: rules.Season, Week: week.Number},
				Settles: []string{LineupSet, LineupCorrect}})
		}
	}
	return jobs, nil
}

func softCapJobs(league string, records []Record, rules leaguerules.RuleSet, at time.Time, week leagueweek.Week) ([]Job, error) {
	reading, err := NativeCapCheckReading(RawNativeCapScope{League: league, ViewerZone: "UTC"}, records, at)
	if err != nil {
		return nil, fmt.Errorf("read soft-cap jobs: %w", err)
	}
	jobs := []Job{}
	for _, franchise := range reading.Franchises {
		if franchise.Mode != leaguerules.SoftCap || franchise.Exception != "" || franchise.RoomCents == nil || *franchise.RoomCents >= 0 {
			continue
		}
		job := Job{ID: "cap." + franchise.Franchise, Name: "Over the soft cap — " + franchise.Name,
			State: JobLive, Controls: []string{"CM-008", "CM-034", "CM-038"}, Words: franchise.Enforcement,
			Identity: JobIdentity{Season: rules.Season}, Settles: []string{CapAdjust, CapRelief, ReserveException}}
		if franchise.Deadline != nil {
			job.Closes = franchise.Deadline.UTC().Format(time.RFC3339)
		}
		if week.Number > 0 {
			job.Identity.Week = week.Number
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func taxiJobs(league string, records []Record, rules leaguerules.RuleSet, at time.Time, facts nativecheck.Facts,
	week leagueweek.Week, lock time.Time, hasLock bool) ([]Job, error) {
	reading, err := NativeReserveCheckReading(league, records, at, facts)
	if err != nil {
		return nil, fmt.Errorf("read taxi jobs: %w", err)
	}
	jobs := []Job{}
	for _, franchise := range reading.Franchises {
		for _, player := range franchise.Players {
			if player.Squad != "Taxi" || player.Standing != nativecheck.MustLeave {
				continue
			}
			job := Job{ID: "taxi." + franchise.Franchise + "." + player.Player,
				Name: "Taxi player must leave — " + player.Name, State: JobLive,
				Controls: []string{"CM-038"}, Words: player.Words,
				Identity: JobIdentity{Season: rules.Season, Week: week.Number}, Settles: []string{ReserveException}}
			if hasLock {
				job.Closes = lock.UTC().Format(time.RFC3339)
			}
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func governanceJobs(league string, records []Record, rules leaguerules.RuleSet, at time.Time,
	week leagueweek.Week, lock time.Time, hasLock bool, assets NativeAssets) ([]Job, error) {
	state, err := GovernanceState(records)
	if err != nil {
		return nil, fmt.Errorf("read governance jobs: %w", err)
	}
	vacancies, err := vacantSeatJobs(league, records, rules, at, week, lock, hasLock, assets)
	if err != nil {
		return nil, err
	}
	jobs := vacancies
	jobs = append(jobs, pendingInviteJobs(records, state, rules)...)
	return append(jobs, openVoteJobs(records, state, rules, at)...), nil
}

func vacantSeatJobs(league string, records []Record, rules leaguerules.RuleSet, at time.Time,
	week leagueweek.Week, lock time.Time, hasLock bool, assets NativeAssets) ([]Job, error) {
	if len(LeagueFranchises(records)) < 14 {
		return nil, nil
	}
	directory, err := NativeDirectoryReading(league, records, at)
	if err != nil {
		return nil, fmt.Errorf("read vacant-seat jobs: %w", err)
	}
	season := rules.Season
	if !hasLock {
		if !assets.Seeded {
			return nil, nil
		}
		var pending bool
		season, pending = draftSeasonPending(assets, RecordsThrough(records, at))
		if !pending {
			return nil, nil
		}
	}
	start, dated := draftStartFor(records, season)
	jobs := []Job{}
	for _, seat := range directory.Franchises {
		if seat.Owner == "Vacant" {
			jobs = append(jobs, vacantSeatJob(seat, rules.Season, season, week, lock, hasLock, start, dated, at))
		}
	}
	return jobs, nil
}

func vacantSeatJob(seat NativeDirectorySeat, currentSeason, draftSeason int, week leagueweek.Week, lock time.Time,
	hasLock bool, draftStart time.Time, draftDated bool, at time.Time) Job {
	job := Job{ID: "vacant." + seat.Franchise, Name: "Vacant seat — " + seat.Name, State: JobLive,
		Controls: []string{"CM-019"}, Words: "Assign an owner or caretaker to this vacant seat.",
		Identity: JobIdentity{Season: currentSeason},
		Settles:  []string{CaretakerSeat, "commish.seat.invite", "commish.seat.replace"}}
	if hasLock {
		job.Identity.Week = week.Number
		job.Closes = lock.UTC().Format(time.RFC3339)
		return job
	}
	job.Identity.Season = draftSeason
	if draftDated && at.Before(draftStart) {
		job.State, job.Opens = JobUpcoming, draftStart.UTC().Format(time.RFC3339)
		job.Words = "This vacant seat blocks the upcoming rookie draft; appoint an owner or caretaker."
	} else {
		job.Words = "This vacant seat blocks the rookie draft; appoint an owner or caretaker."
	}
	return job
}

func pendingInviteJobs(records []Record, state governance.State, rules leaguerules.RuleSet) []Job {
	_, invites := PeopleInventory(records)
	jobs := []Job{}
	for _, id := range invites {
		invite, ok := state.Invite(id)
		if ok && !invite.Revoked && !invite.Redeemed {
			jobs = append(jobs, Job{ID: "invite." + id, Name: "Owner invitation awaiting acceptance", State: JobLive,
				Controls: []string{"CM-019"}, Words: "An owner invitation is still available.",
				Identity: JobIdentity{Season: rules.Season},
				Settles:  []string{"commish.seat.invite.redeem", "commish.seat.invite.revoke", "commish.seat.replace"}})
		}
	}
	return jobs
}

func openVoteJobs(records []Record, state governance.State, rules leaguerules.RuleSet, at time.Time) []Job {
	ids := make(map[string]bool)
	for _, record := range records {
		if people := record.Command.Args.People; people != nil && people.ID != "" {
			ids[people.ID] = true
		}
	}
	jobs := []Job{}
	for id := range ids {
		vote, ok := state.Vote(id)
		if !ok || vote.Closed {
			continue
		}
		view := readVote(state, vote, len(LeagueFranchises(records)), true, at)
		job := Job{ID: "vote." + id, Name: "Governance vote — " + id, State: JobLive,
			Controls: []string{"CM-022"}, Words: view.Status, Identity: JobIdentity{Season: rules.Season},
			Settles: []string{"commish.vote.close", "commish.vote.tiebreak"}}
		if !vote.EndsAt.IsZero() {
			job.Closes = vote.EndsAt.UTC().Format(time.RFC3339)
		}
		jobs = append(jobs, job)
	}
	return jobs
}

func tradeDeadlineNotice(rules leaguerules.RuleSet, at time.Time) (Job, bool) {
	if rules.Calendar.TradeDeadline.State != leaguerules.IsKnown {
		return Job{}, false
	}
	deadline := rules.Calendar.TradeDeadline.Value.UTC()
	due := deadline.Add(-tradeNoticeLead)
	week := 0
	if rules.Calendar.TradeDeadlineWeek.State == leaguerules.IsKnown {
		week = rules.Calendar.TradeDeadlineWeek.Value
	}
	if at.Before(due) || !at.Before(deadline) {
		return Job{}, false
	}
	return Job{ID: "trade_deadline_notice", Name: "Trade deadline notice", State: JobLive,
		Words: "Publish the in-app notice before the trade deadline.", Opens: due.Format(time.RFC3339),
		Closes: deadline.Format(time.RFC3339), Controls: []string{"CM-052"},
		Identity: JobIdentity{Season: rules.Season, Week: week}, Settles: []string{NoticePublish}}, true
}

func offseasonWindowJobs(records []Record, rules leaguerules.RuleSet, at time.Time) []Job {
	if !offseasonWindowsOpen(records, rules, at) {
		return nil
	}
	ids := make([]string, 0, len(rules.Calendar.Windows.Value))
	for id := range rules.Calendar.Windows.Value {
		if excludedOffseasonWindow(id) {
			continue
		}
		ids = append(ids, id)
	}
	slices.Sort(ids)
	jobs := []Job{}
	cal := ruleCalendar{rules: rules.Calendar}
	for _, id := range ids {
		dates, ok := cal.Window(id)
		if !ok || !dates.Close.After(dates.Start) || !at.Before(dates.Close) {
			continue
		}
		job := Job{ID: "offseason_window." + id, Name: "League window — " + id, State: JobLive,
			Closes: dates.Close.UTC().Format(time.RFC3339), Controls: []string{"CM-023"},
			Words:    "Complete the league work opened by this calendar window.",
			Identity: JobIdentity{Season: rules.Season}}
		if at.Before(dates.Start) {
			job.State, job.Opens = JobUpcoming, dates.Start.UTC().Format(time.RFC3339)
		}
		if windows, ok := rules.Calendar.Windows.Value[id]; ok && windows.Opens.State == leaguerules.IsKnown {
			job.Settles = windowSettlementCommands(windows.Opens.Value)
		}
		if len(job.Settles) == 0 {
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs
}

func offseasonWindowsOpen(records []Record, rules leaguerules.RuleSet, at time.Time) bool {
	return rules.Calendar.Rollover.State == leaguerules.IsKnown &&
		rolloverRunSinceDated(RecordsThrough(records, at)) && rules.Calendar.Windows.State == leaguerules.IsKnown
}

func excludedOffseasonWindow(id string) bool {
	if id == "lineups" || id == "waivers" || id == "rookie_draft" {
		return true
	}
	return slices.ContainsFunc(contractWindows(), func(row struct{ id, name string }) bool { return row.id == id })
}

func windowSettlementCommands(actions []leaguerules.Action) []string {
	commands := []string{}
	add := func(values ...string) {
		for _, command := range values {
			if !slices.Contains(commands, command) {
				commands = append(commands, command)
			}
		}
	}
	for _, action := range actions {
		switch action {
		case "tag", "extension", "restructure", "buyout", "rfa_tender", "resign":
			add(ContractActionSet, ContractEdit)
		case "draft_pick":
			add(DraftStart, DraftOrderSet, DraftOrderAuto, DraftOrderPattern, DraftOrderTiebreak, DraftPick)
		case "cut":
			add(RosterCut, ContractActionSet)
		case "ufa_bid":
			add(BidPlace, BidRaise, BidWithdraw, BidsProcess, BidsOverride)
		case "waiver_claim":
			add(WaiverClaim, WaiverOrderSet, WaiverOrderAuto, WaiverProcess, WaiverOverride)
		case "lineup":
			add(LineupSet, LineupCorrect)
		case "trade":
			add(TradeOffer, TradeAccept)
		}
	}
	return commands
}
