package leaguemigration

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	refreshRequestPrefix = "mfl-refresh."
	defaultRefreshReason = "Refresh from MFL"
	franchiseEditCommand = "commish.franchise.edit"
)

// RefreshRequest names an already-loaded app league to refresh from its MFL source.
type RefreshRequest struct {
	MFLLeague           string
	League              string
	Reason              string
	Principal           string
	DryRun              bool
	OverwriteLiveLineup bool
	DropAppAdjustments  bool
}

// RefreshOptions controls how PlanRefresh builds a refresh batch.
type RefreshOptions struct {
	MFLLeague           string
	Reason              string
	Principal           string
	OverwriteLiveLineup bool
	DropAppAdjustments  bool
	LineupFacts         func(season, week int) (commissioner.LineupFacts, error)
}

type lineupFactsReader interface {
	LineupFacts(ctx context.Context, season, week int) (commissioner.LineupFacts, error)
}

// RefreshSummary counts the changes planned or applied by a refresh.
type RefreshSummary struct {
	Transactions        int  `json:"transactions"`
	RosterAdds          int  `json:"rosterAdds"`
	RosterRemoves       int  `json:"rosterRemoves"`
	RosterStatusChanges int  `json:"rosterStatusChanges"`
	Contracts           int  `json:"contracts"`
	Adjustments         int  `json:"adjustments"`
	Picks               int  `json:"picks"`
	Franchises          int  `json:"franchises"`
	ScheduleUpdated     bool `json:"scheduleUpdated"`
	RecordedWeeks       int  `json:"recordedWeeks"`
	CutoverWeek         int  `json:"cutoverWeek"`
	Lineups             int  `json:"lineups"`
}

// RefreshReport is the verified result of planning or applying a refresh.
type RefreshReport struct {
	League    string                    `json:"league"`
	MFLLeague string                    `json:"mflLeague"`
	Season    int                       `json:"season"`
	DryRun    bool                      `json:"dryRun"`
	Commands  []commissioner.RawCommand `json:"-"`
	Receipts  []commissioner.Receipt    `json:"receipts,omitempty"`
	Summary   RefreshSummary            `json:"summary"`
}

// OpenTradeRefusal identifies one unsettled in-app trade blocking refresh.
type OpenTradeRefusal struct {
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	Franchises []string `json:"franchises"`
}

// OpenTradesError is returned when unsettled in-app trades block a refresh.
type OpenTradesError struct {
	Offers []OpenTradeRefusal
}

func (e *OpenTradesError) Error() string {
	parts := make([]string, len(e.Offers))
	for i, o := range e.Offers {
		parts[i] = fmt.Sprintf("%s (%s; franchises: %s)", o.ID, o.Status, strings.Join(o.Franchises, ", "))
	}
	return fmt.Sprintf("refresh refused: %d open in-app trade offer(s) must be settled first: %s",
		len(e.Offers), strings.Join(parts, "; "))
}

// Refresh reads MFL afresh, plans and verifies the atomic refresh batch, and commits unless DryRun is set.
func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (RefreshReport, error) {
	if strings.TrimSpace(req.League) == "" {
		return RefreshReport{}, fmt.Errorf("name the app league to refresh")
	}
	records, err := s.Ledger.Records(ctx, req.League)
	if err != nil {
		return RefreshReport{}, fmt.Errorf("read the app league: %w", err)
	}
	mflLeague := strings.TrimSpace(req.MFLLeague)
	if mflLeague == "" {
		if ref, ok := referenceMFLLeague(records); ok {
			mflLeague = ref
		}
	}
	export, err := s.read(ctx, mflLeague)
	if err != nil {
		return RefreshReport{}, err
	}
	plan, err := export.prepared()
	if err != nil {
		return RefreshReport{}, err
	}
	opts := RefreshOptions{
		MFLLeague:           mflLeague,
		Reason:              req.Reason,
		Principal:           req.Principal,
		OverwriteLiveLineup: req.OverwriteLiveLineup,
		DropAppAdjustments:  req.DropAppAdjustments,
	}
	if reader, ok := s.Ledger.(lineupFactsReader); ok {
		opts.LineupFacts = func(season, week int) (commissioner.LineupFacts, error) {
			return reader.LineupFacts(ctx, season, week)
		}
	}
	report, err := PlanRefresh(req.League, records, plan, s.Now(), opts)
	if err != nil {
		return RefreshReport{}, err
	}
	report.MFLLeague = mflLeague
	report.DryRun = req.DryRun
	if req.DryRun || len(report.Commands) == 0 {
		return report, nil
	}
	receipts, err := s.Ledger.Batch(ctx, report.Commands, req.Principal)
	if err != nil {
		return RefreshReport{}, fmt.Errorf("refresh the league from MFL (nothing was recorded): %w", err)
	}
	report.Receipts = receipts
	return report, nil
}

// PlanRefresh builds and trial-verifies the atomic refresh batch without writing to the ledger.
func PlanRefresh(
	league string, records []commissioner.Record, plan Plan, now time.Time, opts RefreshOptions,
) (RefreshReport, error) {
	if len(records) == 0 {
		return RefreshReport{}, fmt.Errorf("import the MFL league before refreshing it")
	}
	app, err := ReadApp(league, records, now)
	if err != nil {
		return RefreshReport{}, err
	}
	if err := checkRefreshPreconditions(league, records, plan, app, now, opts); err != nil {
		return RefreshReport{}, err
	}
	reason := strings.TrimSpace(opts.Reason)
	if reason == "" {
		reason = defaultRefreshReason
	}
	p := &refreshPlanner{
		league:    league,
		plan:      plan,
		now:       now,
		opts:      opts,
		reason:    reason,
		principal: opts.Principal,
		baseRev:   int64(len(records)),
		trial:     slices.Clone(records),
	}
	if err := p.build(records, app); err != nil {
		return RefreshReport{}, err
	}
	return RefreshReport{
		League:   league,
		Season:   plan.Seed.Season,
		Commands: p.raws,
		Summary:  p.summary,
	}, nil
}

func checkRefreshPreconditions(
	league string, records []commissioner.Record, plan Plan, app AppLeague,
	now time.Time, opts RefreshOptions,
) error {
	if err := commissioner.RequireCommissionerOrCo(records, opts.Principal); err != nil {
		return fmt.Errorf("authorize refresh: %w", err)
	}
	if mflLeague := strings.TrimSpace(opts.MFLLeague); mflLeague != "" {
		if AppLeagueID(mflLeague, plan.Seed.Season) != league {
			return fmt.Errorf(
				"the MFL league %s (%d) does not match app league %s",
				mflLeague, plan.Seed.Season, league)
		}
		if ref, ok := referenceMFLLeague(records); ok && ref != mflLeague {
			return fmt.Errorf(
				"the MFL league %s does not match this league's recorded MFL reference %s",
				mflLeague, ref)
		}
	}
	if plan.Seed.Season != app.Snapshot.Season {
		return fmt.Errorf("the MFL source season (%d) does not match the league's current season (%d)",
			plan.Seed.Season, app.Snapshot.Season)
	}
	if len(plan.Seed.Franchises) != len(app.Snapshot.Franchises) {
		return fmt.Errorf("the MFL source franchise count (%d) does not match the app league (%d)",
			len(plan.Seed.Franchises), len(app.Snapshot.Franchises))
	}
	if err := checkScheduleMapping(records, plan.Mapping); err != nil {
		return err
	}
	return checkNoOpenTrades(league, records, now)
}

func checkNoOpenTrades(league string, records []commissioner.Record, now time.Time) error {
	open, err := commissioner.OpenTrades(league, records, now)
	if err != nil {
		return fmt.Errorf("check open trades: %w", err)
	}
	if len(open) == 0 {
		return nil
	}
	slices.SortFunc(open, func(a, b commissioner.TradeState) int { return strings.Compare(a.ID, b.ID) })
	offers := make([]OpenTradeRefusal, len(open))
	for i, t := range open {
		offers[i] = OpenTradeRefusal{ID: t.ID, Status: t.Status, Franchises: slices.Clone(t.Parties)}
	}
	return &OpenTradesError{Offers: offers}
}

type refreshPlanner struct {
	league    string
	plan      Plan
	now       time.Time
	opts      RefreshOptions
	reason    string
	principal string
	baseRev   int64
	seq       int
	trial     []commissioner.Record
	raws      []commissioner.RawCommand
	summary   RefreshSummary
}

func (p *refreshPlanner) build(records []commissioner.Record, app AppLeague) error {
	serverCharges := serverChargeEntries(records)
	p.replayTransactions(records)
	if err := p.reconcileAssets(serverCharges); err != nil {
		return err
	}
	if err := p.reconcileFranchises(records, app); err != nil {
		return err
	}
	if err := p.reconcileSchedule(); err != nil {
		return err
	}
	if err := p.reconcileLineups(); err != nil {
		return err
	}
	if len(p.raws) > 0 {
		p.raws[0].Expected = p.baseRev
	}
	postApp, err := ReadApp(p.league, p.trial, p.now)
	if err != nil {
		return fmt.Errorf("verify refreshed league state: %w", err)
	}
	return verifyRefreshResult(p.plan, postApp, serverCharges, p.opts.DropAppAdjustments)
}

func (p *refreshPlanner) nextRequest(kind string) string {
	p.seq++
	return fmt.Sprintf("%s%d.%04d.%s", refreshRequestPrefix, p.baseRev, p.seq, kind)
}

func (p *refreshPlanner) detailReason(detail string) string {
	return p.reason + ": " + detail
}

func (p *refreshPlanner) emit(raw commissioner.RawCommand) error {
	next, err := appendTrialRecord(p.trial, raw, p.principal, p.now, p.opts.LineupFacts)
	if err != nil {
		return err
	}
	p.trial = next
	p.raws = append(p.raws, raw)
	return nil
}

func appendTrialRecord(
	trial []commissioner.Record,
	raw commissioner.RawCommand,
	principal string,
	now time.Time,
	lineupFacts func(season, week int) (commissioner.LineupFacts, error),
) ([]commissioner.Record, error) {
	prepared, err := raw.Prepared()
	if err != nil {
		return nil, fmt.Errorf("prepare %s: %w", raw.Command, err)
	}
	if err := prepared.Validate(); err != nil {
		return nil, fmt.Errorf("validate %s: %w", raw.Command, err)
	}
	if _, err := commissioner.AuthorizeRecords(prepared, principal, trial); err != nil {
		return nil, fmt.Errorf("authorize %s: %w", prepared.Command, err)
	}
	var generated []commissioner.ScheduleWeek
	switch prepared.Command {
	case commissioner.CorrectRoster, commissioner.CorrectPick, commissioner.CorrectProvenance,
		commissioner.CorrectTransaction, commissioner.RuleException:
		if err := commissioner.CheckCorrectionAppend(prepared, trial, now); err != nil {
			return nil, fmt.Errorf("check %s: %w", prepared.Command, err)
		}
	case commissioner.ScheduleGenerate, commissioner.SchedulePublish:
		if err := commissioner.CheckScheduleAppend(prepared, trial); err != nil {
			return nil, fmt.Errorf("check %s: %w", prepared.Command, err)
		}
		if prepared.Command == commissioner.ScheduleGenerate {
			generated, err = commissioner.FreezeSchedule(prepared, trial)
			if err != nil {
				return nil, fmt.Errorf("freeze schedule: %w", err)
			}
		}
	case commissioner.LineupCorrect:
		if err := checkTrialLineup(prepared, trial, now, lineupFacts); err != nil {
			return nil, err
		}
	case franchiseEditCommand:
		if err := commissioner.CheckGovernanceAppend(prepared, principal, trial, now); err != nil {
			return nil, fmt.Errorf("check %s: %w", prepared.Command, err)
		}
	}
	entry := fmt.Sprintf("proposed.refresh.%d", len(trial)+1)
	rec := commissioner.Record{
		Command:           prepared,
		Principal:         principal,
		GeneratedSchedule: generated,
		Receipt: commissioner.Receipt{
			Entry:      entry,
			League:     prepared.League,
			Revision:   int64(len(trial) + 1),
			RecordedAt: now,
		},
	}
	return append(slices.Clone(trial), rec), nil
}

func checkTrialLineup(
	prepared commissioner.RawCommand,
	trial []commissioner.Record,
	now time.Time,
	lineupFacts func(season, week int) (commissioner.LineupFacts, error),
) error {
	var env commissioner.LineupFacts
	if lineupFacts != nil {
		change, err := commissioner.DecodeLineupChange(prepared.Args, prepared.Command)
		if err != nil {
			return fmt.Errorf("decode %s: %w", prepared.Command, err)
		}
		env, err = lineupFacts(change.Season, change.Week)
		if err != nil {
			return fmt.Errorf("load lineup facts for %s: %w", prepared.Command, err)
		}
	}
	if err := commissioner.CheckLineupAppend(prepared, trial, now, env); err != nil {
		return fmt.Errorf("check %s: %w", prepared.Command, err)
	}
	return nil
}

func (p *refreshPlanner) reconcileFranchises(records []commissioner.Record, app AppLeague) error {
	edited := inAppEditedFranchises(records)
	seats := app.Directory.Seats()
	for _, f := range p.plan.Seed.Franchises {
		if edited[f.ID] {
			continue
		}
		seat := seats.Of(f.ID)
		wantName := strings.TrimSpace(f.Name)
		rawAbbrev := strings.TrimSpace(f.Abbreviation)
		wantAbbrev := rawAbbrev
		if wantAbbrev == "" {
			wantAbbrev = strings.TrimSpace(seat.Abbreviation)
			if wantAbbrev == "" {
				wantAbbrev = "F" + strings.TrimPrefix(f.ID, "franchise.")
			}
		}
		if seat.Name == wantName && (rawAbbrev == "" || seat.Abbreviation == wantAbbrev) {
			continue
		}
		people := commissioner.RawPeople{Franchise: f.ID, Name: wantName, Abbreviation: wantAbbrev}
		raw := commissioner.RawCommand{
			League:     p.league,
			Command:    franchiseEditCommand,
			Args:       commissioner.RawArgs{People: &people},
			Request:    p.nextRequest("franchise." + f.ID),
			Reason:     p.detailReason("update franchise " + f.ID),
			Effective:  "now",
			Visibility: "league",
		}
		if err := p.emit(raw); err != nil {
			return err
		}
		p.summary.Franchises++
	}
	return nil
}

func (p *refreshPlanner) reconcileSchedule() error {
	held, err := commissioner.ScheduleReading(p.league, p.trial)
	if err != nil {
		return fmt.Errorf("read published schedule: %w", err)
	}
	previous := commissioner.RecordedResults(p.trial, p.plan.Seed.Season)
	recorded := mergeRecordedResults(previous, p.plan.Recorded)
	weeks := p.plan.Schedule
	if len(weeks) == 0 {
		weeks = held.Weeks
	}
	scheduleChanged := len(p.plan.Schedule) > 0 && !reflect.DeepEqual(held.Weeks, p.plan.Schedule)
	recordedChanged := recorded != nil && !reflect.DeepEqual(previous, recorded)
	if recorded != nil {
		p.summary.CutoverWeek = recorded.CutoverWeek
		p.summary.RecordedWeeks = countRecordedWeeks(recorded)
	}
	if len(weeks) == 0 || (!scheduleChanged && !recordedChanged) {
		return nil
	}
	cmds := ScheduleCommands(
		p.league, p.detailReason("update schedule and completed results"), weeks, p.baseRev, recorded)
	cmds[0].Request = p.nextRequest("schedule.draft")
	cmds[0].Expected = 0
	cmds[1].Request = p.nextRequest("schedule.publish")
	for _, cmd := range cmds {
		if err := p.emit(cmd); err != nil {
			return err
		}
	}
	p.summary.ScheduleUpdated = true
	return nil
}

func (p *refreshPlanner) reconcileLineups() error {
	if len(p.plan.Lineups) == 0 {
		return nil
	}
	lineups := slices.Clone(p.plan.Lineups)
	slices.SortFunc(lineups, func(a, b commissioner.RawLineupChange) int {
		if a.Week != b.Week {
			return cmp.Compare(a.Week, b.Week)
		}
		return strings.Compare(a.Franchise, b.Franchise)
	})
	cutover := 0
	if held := commissioner.RecordedResults(p.trial, p.plan.Seed.Season); held != nil {
		cutover = held.CutoverWeek
	}
	seen := map[string]bool{}
	for _, lineup := range lineups {
		key := fmt.Sprintf("%d.%s", lineup.Week, lineup.Franchise)
		if lineup.Season != p.plan.Seed.Season || seen[key] {
			return fmt.Errorf("lineups must name season %d and each franchise-week once", p.plan.Seed.Season)
		}
		seen[key] = true
		if err := p.reconcileOneLineup(lineup, cutover); err != nil {
			return err
		}
	}
	return nil
}

func (p *refreshPlanner) reconcileOneLineup(lineup commissioner.RawLineupChange, cutover int) error {
	previous, found, err := commissioner.RecordedLineupFor(
		p.trial, lineup.Franchise, p.plan.Seed.Season, lineup.Week)
	if err != nil {
		return fmt.Errorf("read lineup %s week %d: %w", lineup.Franchise, lineup.Week, err)
	}
	want, have := slices.Clone(lineup.Starters), slices.Clone(previous.Starters)
	slices.Sort(want)
	slices.Sort(have)
	if found && slices.Equal(want, have) {
		return nil
	}
	played := cutover > 0 && lineup.Week < cutover
	if !played && found && !previous.Synced && !p.opts.OverwriteLiveLineup {
		return nil
	}
	copyLineup := lineup
	copyLineup.Starters = slices.Clone(lineup.Starters)
	raw := commissioner.RawCommand{
		League:     p.league,
		Command:    commissioner.LineupCorrect,
		Args:       commissioner.RawArgs{Lineup: &copyLineup},
		Request:    p.nextRequest(fmt.Sprintf("lineup.%d.%s", lineup.Week, lineup.Franchise)),
		Reason:     p.detailReason(fmt.Sprintf("week %d lineup for %s", lineup.Week, lineup.Franchise)),
		Effective:  "now",
		Visibility: "league",
	}
	if err := p.emit(raw); err != nil {
		return err
	}
	p.summary.Lineups++
	return nil
}

func (p *refreshPlanner) correctionCmd(
	command, kind, detail string, c commissioner.RawCorrection,
) commissioner.RawCommand {
	return commissioner.RawCommand{
		League:     p.league,
		Command:    command,
		Args:       commissioner.RawArgs{Correction: &c},
		Request:    p.nextRequest(kind),
		Reason:     p.detailReason(detail),
		Effective:  "now",
		Visibility: "league",
	}
}

func mergeRecordedResults(
	previous, incoming *commissioner.RawRecordedResults,
) *commissioner.RawRecordedResults {
	if incoming == nil {
		return previous
	}
	if previous == nil {
		return incoming
	}
	cutover := max(previous.CutoverWeek, incoming.CutoverWeek)
	type weekTeam struct {
		week      int
		franchise string
	}
	merged := map[weekTeam]commissioner.RecordedTeamResult{}
	for _, t := range previous.Teams {
		if t.Week < cutover {
			merged[weekTeam{t.Week, t.Franchise}] = t
		}
	}
	for _, t := range incoming.Teams {
		if t.Week < cutover {
			merged[weekTeam{t.Week, t.Franchise}] = t
		}
	}
	teams := make([]commissioner.RecordedTeamResult, 0, len(merged))
	for _, t := range merged {
		teams = append(teams, t)
	}
	slices.SortFunc(teams, func(a, b commissioner.RecordedTeamResult) int {
		if a.Week != b.Week {
			return cmp.Compare(a.Week, b.Week)
		}
		return strings.Compare(a.Franchise, b.Franchise)
	})
	return &commissioner.RawRecordedResults{
		Season:      previous.Season,
		CutoverWeek: cutover,
		Teams:       teams,
	}
}

func countRecordedWeeks(r *commissioner.RawRecordedResults) int {
	if r == nil {
		return 0
	}
	weeks := map[int]bool{}
	for _, t := range r.Teams {
		weeks[t.Week] = true
	}
	return len(weeks)
}

func inAppEditedFranchises(records []commissioner.Record) map[string]bool {
	active := commissioner.Replay(records).Active
	out := map[string]bool{}
	for _, r := range records {
		if r.Command.Command != franchiseEditCommand || !active[r.Receipt.Entry] || r.Command.Args.People == nil {
			continue
		}
		if !strings.HasPrefix(r.Command.Request, refreshRequestPrefix) {
			out[r.Command.Args.People.Franchise] = true
		}
	}
	return out
}

func serverChargeEntries(records []commissioner.Record) map[string]bool {
	active := commissioner.Replay(records).Active
	out := map[string]bool{}
	for _, r := range records {
		if !active[r.Receipt.Entry] {
			continue
		}
		if r.Command.Command == commissioner.CapAdjust || r.Command.Command == commissioner.CapRelief {
			out[r.Receipt.Entry] = true
		}
	}
	return out
}

func requestSet(records []commissioner.Record) map[string]bool {
	out := make(map[string]bool, len(records))
	for _, r := range records {
		out[r.Command.Request] = true
	}
	return out
}

func initialSeedSnapshot(records []commissioner.Record) (commissioner.RawNativeSnapshot, bool) {
	for _, r := range records {
		if r.Command.Command == commissioner.NativeSeed && r.Command.Args.Native != nil {
			return *r.Command.Args.Native, true
		}
	}
	return commissioner.RawNativeSnapshot{}, false
}

func referenceMFLLeague(records []commissioner.Record) (string, bool) {
	active := commissioner.Replay(records).Active
	for i := len(records); i > 0; i-- {
		r := records[i-1]
		if !active[r.Receipt.Entry] {
			continue
		}
		isFollow := r.Command.Command == commissioner.RulesSet &&
			r.Command.Args.Field == "schedule.follow"
		if r.Command.Command != commissioner.Create && !isFollow {
			continue
		}
		rules, err := commissioner.RulesNow(records[:i], time.Time{})
		if err != nil {
			continue
		}
		follow := rules.Schedule.Follow
		if follow.State == leaguerules.IsKnown && follow.Value.Source == "mfl" &&
			strings.TrimSpace(follow.Value.League) != "" {
			return strings.TrimSpace(follow.Value.League), true
		}
	}
	return "", false
}

func verifyRefreshResult(
	plan Plan, postApp AppLeague, serverCharges map[string]bool, dropAppCharges bool,
) error {
	diffs := Compare(plan, postApp)
	var unexpected []string
	for _, d := range diffs {
		switch group(d.Field) {
		case "roster", "contract", "pick":
			unexpected = append(unexpected, fmt.Sprintf("%s %s (MFL=%s, app=%s)", d.Franchise, d.Field, d.MFL, d.App))
		case "salary adjustment":
			id := strings.TrimPrefix(d.Field, "salary adjustment ")
			if d.MFL == "not on MFL" && serverCharges[id] && !dropAppCharges {
				continue
			}
			unexpected = append(unexpected, fmt.Sprintf("%s %s (MFL=%s, app=%s)", d.Franchise, d.Field, d.MFL, d.App))
		}
	}
	if len(unexpected) > 0 {
		return fmt.Errorf("refresh verification failed with %d remaining asset diff(s): %s",
			len(unexpected), strings.Join(unexpected, "; "))
	}
	return nil
}
