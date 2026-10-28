// Package commissioner keeps the ledger clock over the rulebook in internal/leaguerules.
// This season fold is shared core, not a separate rules domain.
package commissioner

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	rolloverField    = "calendar.rollover"
	alignmentField   = "league.alignment"
	conferencesField = "league.divisionConferences"
	// carriedDivisions is the citation on a division alignment carried in from the starting snapshot.
	carriedDivisions = "Carried from the held Legacy NFL divisions when the league was seeded"
	// carriedConferences is the citation on division conferences carried in from the starting snapshot.
	carriedConferences = "Carried from the held Legacy NFL conferences when the league was seeded"
	// unsettledDivision is the word an older native start stored for a team with no division yet.
	unsettledDivision = "Needs a ruling"
)

// firstSeason numbers the season the ledger opens in: the starting snapshot's season once the league
// is seeded, else the NFL season the creation records. A league recorded before seasons were numbered,
// and not yet seeded, has no season to read; a year is never guessed. Later seasons follow the season
// rollover acts, or the league's rollover dates where no act exists.
func firstSeason(records []Record) (int, error) {
	for _, r := range records {
		if r.Command.Command == NativeSeed && r.Command.Args.Native != nil {
			return r.Command.Args.Native.Season, nil
		}
	}
	if len(records) > 0 && records[0].Command.Command == Create && records[0].Command.Args.Season != 0 {
		return records[0].Command.Args.Season, nil
	}
	return 0, fmt.Errorf("this league was started before its first season was recorded; " +
		"record its starting assets to name the season")
}

// seeded says whether the league's starting assets are recorded; native scores and standings need them.
func seeded(records []Record) bool {
	for _, r := range records {
		if r.Command.Command == NativeSeed && r.Command.Args.Native != nil {
			return true
		}
	}
	return false
}

// rolloverInstants lists, in order, the instants at which a new season begins. Each comes from an
// in-force "now" setting of the season rollover instant. A date set before the previous rollover has
// passed moves that rollover; one set after it schedules the next. A date that does not come after the
// one before it is not a new season.
func RolloverInstants(records []Record) ([]time.Time, error) {
	if starts, explicit := explicitRolloverInstants(records); explicit {
		return starts, nil
	}
	starts := []time.Time{}
	var pending time.Time
	index := newRolloverIndex()
	for _, r := range records {
		if !pending.IsZero() && !r.Receipt.RecordedAt.Before(pending) {
			starts = append(starts, pending)
			pending = time.Time{}
		}
		date, err := index.add(r)
		if err != nil {
			return nil, err
		}
		if r.Command.Command != Revert && (r.Command.Command != RulesSet || r.Command.Args.Field != rolloverField) {
			continue
		}
		pending = time.Time{}
		if len(starts) == 0 || date.After(starts[len(starts)-1]) {
			pending = date
		}
	}
	if !pending.IsZero() {
		starts = append(starts, pending)
	}
	return starts, nil
}

func rolloverDate(r Record) (time.Time, error) {
	setting, err := leaguerules.DecodeSetting(rolloverField, r.Command.Args.Value)
	if err != nil {
		return time.Time{}, fmt.Errorf("read recorded rollover: %w", err)
	}
	if date, ok := setting.(leaguerules.Value[time.Time]); ok && date.State == leaguerules.IsKnown {
		return date.Value, nil
	}
	return time.Time{}, nil
}

// nextRollover is the first rollover instant strictly after the moment a change was recorded.
func nextRollover(starts []time.Time, recorded time.Time) (time.Time, bool) {
	for _, s := range starts {
		if s.After(recorded) {
			return s, true
		}
	}
	return time.Time{}, false
}

// SeasonAt says which season the league is in at an instant.
func SeasonAt(records []Record, at time.Time) (int, error) {
	records = RecordsThrough(records, at)
	starts, err := RolloverInstants(records)
	if err != nil {
		return 0, err
	}
	season, err := firstSeason(records)
	if err != nil {
		return 0, err
	}
	for _, s := range starts {
		if !at.Before(s) {
			season++
		}
	}
	return season, nil
}

// ReadingInstant is the later of the clock and the ledger's last entry, so a slow clock never
// hides a recorded change.
func ReadingInstant(records []Record, now time.Time) time.Time {
	at := now
	for _, r := range records {
		if r.Receipt.RecordedAt.After(at) {
			at = r.Receipt.RecordedAt
		}
	}
	if at.IsZero() {
		return time.Unix(1, 0).UTC()
	}
	return at
}

// RulesNow is the league's rules in force at the clock (never before its last recorded entry).
func RulesNow(records []Record, now time.Time) (leaguerules.RuleSet, error) {
	at := ReadingInstant(records, now)
	season, err := SeasonAt(records, at)
	if err != nil {
		return leaguerules.RuleSet{}, err
	}
	rules, err := RulesInForce(records, season, at)
	if err != nil {
		return leaguerules.RuleSet{}, err
	}
	countedShape(&rules, records)
	return rules, nil
}

func countedShape(rules *leaguerules.RuleSet, records []Record) {
	if len(records) != 0 && records[0].Command.Command == Create {
		rules.League.Teams = leaguerules.Value[int]{State: leaguerules.IsKnown,
			Value: len(LeagueFranchises(records)), Citation: "Counted from the league's franchises"}
	}
	if rules.League.Alignment.State == leaguerules.IsKnown {
		rules.League.Divisions = leaguerules.Value[int]{State: leaguerules.IsKnown,
			Value: leaguerules.DivisionCount(rules.League.Alignment.Value), Citation: "Counted from the division alignment"}
	}
	countSides(rules)
}

// RulesInForce is the one place a native league's rules are resolved. It returns the rules a season
// held at an instant: a change counts from the moment it starts (a next-rollover change from the
// rollover after it was recorded), and a season that has ended keeps the rules it ended with.
func RulesInForce(records []Record, season int, at time.Time) (leaguerules.RuleSet, error) {
	if len(records) == 0 {
		return leaguerules.RuleSet{}, fmt.Errorf("this app league has not been created")
	}
	records = RecordsThrough(records, at)
	if rules, explicit, err := explicitSeasonRules(records, season, at); explicit {
		return rules, err
	}
	starts, err := RolloverInstants(records)
	if err != nil {
		return leaguerules.RuleSet{}, err
	}
	first, err := firstSeason(records)
	if err != nil {
		return leaguerules.RuleSet{}, err
	}
	index := season - first
	if index < 0 || index > len(starts) {
		return leaguerules.RuleSet{}, fmt.Errorf("read league rules: season %d is not in this league's history", season)
	}
	if index > 0 && at.Before(starts[index-1]) {
		return leaguerules.RuleSet{}, fmt.Errorf("read league rules: season %d has not started", season)
	}
	if index < len(starts) && !at.Before(starts[index]) {
		at = starts[index].Add(-time.Nanosecond)
	}
	changes, err := seasonChanges(RecordsThrough(records, at), starts, season)
	if err != nil {
		return leaguerules.RuleSet{}, err
	}
	rules, err := leaguerules.Resolve(changes, season, at)
	if err != nil {
		return leaguerules.RuleSet{}, fmt.Errorf("read league rules: %w", err)
	}
	return rules, nil
}

// recordsThrough prevents later corrections from rewriting an earlier season or instant.
func RecordsThrough(records []Record, at time.Time) []Record {
	out := make([]Record, 0, len(records))
	for _, r := range records {
		if !r.Receipt.RecordedAt.After(at) {
			out = append(out, r)
		}
	}
	return out
}

type startedChange struct {
	change leaguerules.Change
	start  time.Time
}

// seasonChanges lists every change in force in the ledger, labelled with the season being read and
// ordered by when each starts (ledger order breaks ties).
func seasonChanges(records []Record, starts []time.Time, season int) ([]leaguerules.Change, error) {
	fold := seasonFold{starts: starts, active: Replay(records).Active, season: season}
	for _, r := range records {
		projected, err := ruleRecord(r)
		if err != nil {
			return nil, err
		}
		if err := fold.add(projected); err != nil {
			return nil, err
		}
	}
	slices.SortStableFunc(fold.timed, func(a, b startedChange) int { return a.start.Compare(b.start) })
	changes := make([]leaguerules.Change, len(fold.timed))
	for i, t := range fold.timed {
		changes[i] = t.change
		changes[i].Season = season
	}
	return changes, nil
}

// seasonFold collects, in ledger order, every change in force with the instant it starts.
type seasonFold struct {
	season       int
	starts       []time.Time
	active       map[string]bool
	timed        []startedChange
	alignmentSet bool
	// conferencesSet says the commissioner set division conferences before the starting snapshot.
	conferencesSet bool
	// franchises is the league's franchises as of the entry being folded; a Teams or Divisions change
	// carries it so its resolution keeps the alignment to them.
	franchises []string
}

func (f *seasonFold) add(r Record) error {
	if standingRule(r.Command.Command) {
		f.addStanding(r)
		return nil
	}
	return f.addChange(r)
}

func (f *seasonFold) addChange(r Record) error {
	switch r.Command.Command {
	case Create:
		f.franchises = r.Command.Args.Franchises
		seeded, err := seedChanges(r, f.season)
		if err != nil {
			return err
		}
		for _, c := range seeded {
			f.timed = append(f.timed, startedChange{c, c.RecordedAt})
		}
	case NativeSeed:
		f.addCarried(r)
	case WindowSet, WindowMove, WindowRemove:
		if !f.active[r.Receipt.Entry] {
			return nil
		}
		c, err := windowChange(f.timed, r, f.season)
		if err != nil {
			return err
		}
		f.timed = append(f.timed, startedChange{c, c.RecordedAt})
	case RulesSet:
		if !f.active[r.Receipt.Entry] {
			return nil
		}
		c, start, ok, err := recordedChange(r, f.starts)
		if err != nil || !ok {
			return err
		}
		if n, ok := teamCount(r); ok {
			f.franchises = resizedFranchises(f.franchises, n)
		}
		if c.Field == teamsField || c.Field == divisionsField {
			c.Franchises = slices.Clone(f.franchises)
		}
		f.timed = append(f.timed, startedChange{c, start})
		f.noteSetNow(r)
	}
	return nil
}

// addCarried adds the divisions and conferences the starting snapshot carries into the rules.
func (f *seasonFold) addCarried(r Record) {
	if c, ok := carriedAlignment(r, f.alignmentSet); ok {
		f.timed = append(f.timed, startedChange{c, c.RecordedAt})
	}
	if c, ok := carriedConferenceNames(r, f.conferencesSet); ok {
		f.timed = append(f.timed, startedChange{c, c.RecordedAt})
	}
}

// noteSetNow remembers a commissioner setting of the divisions or conferences, which a later start keeps.
func (f *seasonFold) noteSetNow(r Record) {
	now := r.Command.Effective == "now"
	f.alignmentSet = f.alignmentSet || (now && (r.Command.Args.Field == alignmentField || r.Command.Args.Field == divisionsField))
	f.conferencesSet = f.conferencesSet || (now && r.Command.Args.Field == conferencesField)
}

// addStanding adds a draft pattern or tie rule that is in force.
func (f *seasonFold) addStanding(r Record) {
	if c, ok := standingChange(r); ok && f.active[r.Receipt.Entry] {
		f.timed = append(f.timed, startedChange{c, c.RecordedAt})
	}
}

// recordedChange turns an in-force rule change into a resolver change. ok is false while a
// next-rollover change has no rollover date to wait for.
func recordedChange(r Record, starts []time.Time) (leaguerules.Change, time.Time, bool, error) {
	setting, err := leaguerules.DecodeSetting(leaguerules.FieldPath(r.Command.Args.Field), r.Command.Args.Value)
	if err != nil {
		return leaguerules.Change{}, time.Time{}, false, fmt.Errorf("read recorded rule: %w", err)
	}
	c := leaguerules.Change{LeagueID: r.Command.League, Field: leaguerules.FieldPath(r.Command.Args.Field),
		New: setting, Effective: leaguerules.Effective{Kind: leaguerules.Now}, RecordedAt: r.Receipt.RecordedAt}
	if r.Command.Effective == "now" {
		return c, c.RecordedAt, true, nil
	}
	rollover, ok := nextRollover(starts, r.Receipt.RecordedAt)
	if !ok {
		return leaguerules.Change{}, time.Time{}, false, nil
	}
	c.Effective = leaguerules.Effective{Kind: leaguerules.NextRollover, At: rollover}
	return c, rollover, true, nil
}

// carriedAlignment gives the league the divisions of the starting snapshot, unless the commissioner
// had already set an alignment. Later changes and their reversals then act on it like any other value.
func carriedAlignment(r Record, alignmentSet bool) (leaguerules.Change, bool) {
	if alignmentSet || r.Command.Args.Native == nil {
		return leaguerules.Change{}, false
	}
	divisions := map[string]string{}
	for _, f := range r.Command.Args.Native.Franchises {
		if f.Division != "" && f.Division != unsettledDivision {
			divisions[f.ID] = f.Division
		}
	}
	if len(divisions) == 0 {
		return leaguerules.Change{}, false
	}
	return leaguerules.Change{LeagueID: r.Command.League, Field: alignmentField,
		New:       leaguerules.Value[map[string]string]{State: leaguerules.IsKnown, Value: divisions, Citation: carriedDivisions},
		Effective: leaguerules.Effective{Kind: leaguerules.Now}, RecordedAt: r.Receipt.RecordedAt}, true
}

// carriedConferenceNames gives the league the conference of each division held in the starting
// snapshot, unless the commissioner had already set them. Only an imported start holds conferences.
func carriedConferenceNames(r Record, conferencesSet bool) (leaguerules.Change, bool) {
	if conferencesSet || r.Command.Args.Native == nil || len(r.Command.Args.Native.Conferences) == 0 {
		return leaguerules.Change{}, false
	}
	return leaguerules.Change{LeagueID: r.Command.League, Field: conferencesField,
		New: leaguerules.Value[map[string]string]{State: leaguerules.IsKnown,
			Value: maps.Clone(r.Command.Args.Native.Conferences), Citation: carriedConferences},
		Effective: leaguerules.Effective{Kind: leaguerules.Now}, RecordedAt: r.Receipt.RecordedAt}, true
}

// RulesFunc resolves the rules a season held at an instant; RulesInForce is the one real resolver.
type RulesFunc func(records []Record, season int, at time.Time) (leaguerules.RuleSet, error)

// rulesKey is everything a resolution depends on: which ledger was read (the same entries, not merely the
// same length), the season, and the instant. Two readings of one ledger at different instants can see
// different rules when a change starts between them, so the instant is part of the key.
type rulesKey struct {
	ledger *Record
	length int
	season int
	at     time.Time
}

type rulesEntry struct {
	rules leaguerules.RuleSet
	err   error
}

// RulesMemo remembers resolved rules for the length of one reading, so scoring every starter of a season
// reads the ledger once, not once per starter. It is never shared between readings: a later reading starts
// from a new memo and so sees any change recorded since. The rule sets it returns are shared, read-only.
//
// It also keeps what scoring reads from the whole ledger whatever player is scored: which entries are
// active, the stat decisions, and each entry's place in the ledger.
type RulesMemo struct {
	entries  map[rulesKey]rulesEntry
	folds    map[ledgerKey]ledgerFold
	resolved int
}

// ledgerKey is one ledger: the same entries, not merely the same length.
type ledgerKey struct {
	ledger *Record
	length int
}

// ledgerFold is the player-independent part of scoring, read once per ledger. All of it is read-only.
type ledgerFold struct {
	active   map[string]bool
	position map[string]int
	effects  statEffects
	err      error
}

func NewRulesMemo() *RulesMemo {
	return &RulesMemo{entries: map[rulesKey]rulesEntry{}, folds: map[ledgerKey]ledgerFold{}}
}

func foldLedger(records []Record) ledgerFold {
	f := ledgerFold{active: Replay(records).Active, position: make(map[string]int, len(records))}
	for i, r := range records {
		f.position[r.Receipt.Entry] = i
	}
	f.effects, f.err = foldStatEffectsFrom(records, f.active)
	return f
}

// fold is foldLedger, answered from memory when this ledger was already folded.
func (m *RulesMemo) fold(records []Record) ledgerFold {
	if len(records) == 0 {
		return foldLedger(records)
	}
	key := ledgerKey{ledger: &records[0], length: len(records)}
	if hit, ok := m.folds[key]; ok {
		return hit
	}
	f := foldLedger(records)
	m.folds[key] = f
	return f
}

// InForce is RulesInForce, answered from memory when this ledger, season and instant were already resolved.
func (m *RulesMemo) InForce(records []Record, season int, at time.Time) (leaguerules.RuleSet, error) {
	if len(records) == 0 {
		return RulesInForce(records, season, at)
	}
	key := rulesKey{ledger: &records[0], length: len(records), season: season, at: at.Round(0).UTC()}
	if hit, ok := m.entries[key]; ok {
		return hit.rules, hit.err
	}
	m.resolved++
	rules, err := RulesInForce(records, season, at)
	m.entries[key] = rulesEntry{rules, err}
	return rules, err
}

// Resolved counts the times the ledger was actually read, which is what the memo exists to keep small.
func (m *RulesMemo) Resolved() int { return m.resolved }
