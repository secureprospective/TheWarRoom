package commissioner

import (
	"cmp"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// DraftKey names one draft: its season, and whether it is the league's startup draft rather than that
// season's rookie draft. A rookie draft's records carry no startup flag, so they key as they always did.
type DraftKey struct {
	Season  int
	Startup bool
}

func rookieDraft(season int) DraftKey { return DraftKey{Season: season} }

func (d *RawDraftArgs) key() DraftKey { return DraftKey{Season: d.Season, Startup: d.Startup} }

func (d *RawDraftArgs) noun() string { return d.key().noun() }

// noun is the draft as a sentence names it.
func (k DraftKey) noun() string {
	if k.Startup {
		return fmt.Sprintf("%d startup draft", k.Season)
	}
	return fmt.Sprintf("%d rookie draft", k.Season)
}

// ofDraft says whether a record is a command for one draft.
func ofDraft(r Record, k DraftKey) bool {
	d := r.Command.Args.Draft
	return d != nil && d.key() == k
}

// startupKey is the league's startup draft: it is run in the league's first season.
func startupKey(records []Record) DraftKey {
	return DraftKey{Season: records[0].Command.Args.Season, Startup: true}
}

// startupOpened is the open in force for the startup draft.
func startupOpened(records []Record, key DraftKey) (Record, bool) {
	active := Replay(records).Active
	for _, r := range records {
		if r.Command.Command == DraftStartupOpen && active[r.Receipt.Entry] && ofDraft(r, key) {
			return r, true
		}
	}
	return Record{}, false
}

// startedEmpty says the league recorded its start with no player on any roster.
func startedEmpty(records []Record) bool {
	active := Replay(records).Active
	i := slices.IndexFunc(records, func(r Record) bool { return r.Command.Command == NativeSeed && active[r.Receipt.Entry] })
	return i >= 0 && records[i].Command.Args.Native != nil && len(records[i].Command.Args.Native.Players) == 0
}

// draftsFirstRosters says the league's first rosters are drafted: an app league started here, with empty
// rosters, whose rules choose a draft for them.
func draftsFirstRosters(records []Record, rules leaguerules.RuleSet) bool {
	v := rules.League.InitialRosters
	return LeagueSource(records) == SourceNative && v.State == leaguerules.IsKnown && v.Value == leaguerules.ByDraft &&
		startedEmpty(records)
}

// readStartupDraft replays the startup draft. Its picks are its own, one a round for every franchise; its
// settings are read as they stood when it opened, with the rounds and seed the open recorded.
func readStartupDraft(records []Record, key DraftKey, at time.Time) (draftBasis, error) {
	rules, err := RulesNow(records, at)
	if err != nil {
		return draftBasis{}, err
	}
	basis := draftBasis{key: key, rules: rules, owners: draftOwners(records)}
	open, ok := startupOpened(records, key)
	if !ok {
		basis.words = "The startup draft has not been opened."
		return basis, nil
	}
	franchises := LeagueFranchises(records)
	draft, err := startupDraftRules(records, open, rules.Draft)
	if err != nil {
		basis.words = draftWords(err)
		return basis, nil
	}
	basis.rules.Draft, basis.start = draft, open.Receipt.RecordedAt
	basis.started = !at.Before(basis.start)
	basis.order, err = rookiedraft.Order(draft, franchises, rookiedraft.StartupPicks(franchises, draft.Rounds.Value), nil)
	if err != nil {
		basis.words = draftWords(err)
		return basis, nil
	}
	basis.draft, err = rookiedraft.ReplayActs(draft, basis.order, basis.start, at, draftActs(records, key))
	if err != nil {
		return basis, fmt.Errorf("replay the startup draft: %w", err)
	}
	if closed, ok := closeRecord(records, key); ok {
		basis.closed = &closed
	}
	return basis, nil
}

// startupDraftRules are the startup draft's rules: its settings as they stood when it opened, its recorded
// rounds and first round, and the league's pick clock as it stands.
func startupDraftRules(records []Record, open Record, now leaguerules.DraftRules) (leaguerules.DraftRules, error) {
	o := open.Command.Args.DraftOutcome
	if o == nil {
		return leaguerules.DraftRules{}, errors.New("the startup draft's opening recorded no seed")
	}
	seed, err := startupSeed(o.Seed)
	if err != nil {
		return leaguerules.DraftRules{}, err
	}
	then, err := RulesNow(records, open.Receipt.RecordedAt)
	if err != nil {
		return leaguerules.DraftRules{}, err
	}
	s, err := rookiedraft.ResolveStartup(then)
	if err != nil {
		//nolint:wrapcheck // The engine's words are the draft room's: they say which setting needs a ruling.
		return leaguerules.DraftRules{}, err
	}
	s.Rounds = o.Rounds
	first, err := rookiedraft.StartupFirstRound(s, LeagueFranchises(records), seed)
	if err != nil {
		//nolint:wrapcheck // The engine's words are the draft room's: they say what is wrong with the order.
		return leaguerules.DraftRules{}, err
	}
	return rookiedraft.StartupRules(now, s, first), nil
}

func startupSeed(hex string) (uint64, error) {
	seed, err := strconv.ParseUint(hex, 16, 64)
	if err != nil || len(hex) != 16 {
		return 0, errors.New("the startup draft's seed is not a recorded seed")
	}
	return seed, nil
}

// checkStartupOpen holds the opening to a league whose first rosters are drafted, once, in its first
// season, with settings the draft can run on.
func checkStartupOpen(raw RawCommand, records []Record, at time.Time) error {
	rules, err := RulesNow(records, at)
	if err != nil {
		return err
	}
	key := raw.Args.Draft.key()
	switch {
	case !draftsFirstRosters(records, rules):
		return errors.New("only an app league that started with empty rosters and drafts its first rosters runs a startup draft")
	case key != startupKey(records):
		return fmt.Errorf("open the startup draft for the league's first season, %d", startupKey(records).Season)
	}
	if _, open := startupOpened(records, key); open {
		return errors.New("the startup draft has already been opened")
	}
	// The check runs before Go fixes the seed, so it tries the opening with one of its own.
	trial := raw
	if err := prepareStartupOpen(&trial, records, at); err != nil {
		return err
	}
	basis, err := readStartupDraft(withProposed(trial, records, at), key, at)
	if err != nil {
		return err
	}
	if basis.words != "" {
		return errors.New(strings.TrimSuffix(basis.words, "."))
	}
	return nil
}

// prepareStartupOpen records the seed the random first round is drawn from, unpredictably, and the rounds.
func prepareStartupOpen(raw *RawCommand, records []Record, now time.Time) error {
	rules, err := RulesNow(records, now)
	if err != nil {
		return err
	}
	s, err := rookiedraft.ResolveStartup(rules)
	if err != nil {
		//nolint:wrapcheck // The engine's words are the refusal: they say which setting needs a ruling.
		return err
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Errorf("draw the startup draft's seed: %w", err)
	}
	raw.Args.DraftOutcome = &DraftOutcome{Seed: fmt.Sprintf("%016x", binary.BigEndian.Uint64(b[:])), Rounds: s.Rounds}
	return nil
}

// startupDraftJob is the startup draft until it closes, for a league whose first rosters are drafted.
func startupDraftJob(records []Record, rules leaguerules.RuleSet) (Job, bool) {
	if !draftsFirstRosters(records, rules) {
		return Job{}, false
	}
	key := startupKey(records)
	if _, closed := closeRecord(records, key); closed {
		return Job{}, false
	}
	job := Job{ID: "startup_draft", Name: "Run the startup draft", State: JobLive,
		Controls: []string{"CM-042", "CM-043", "CM-044", "CM-045"},
		Words:    "Open the startup draft to fill the empty rosters."}
	if _, open := startupOpened(records, key); open {
		job.Words = fmt.Sprintf("The %s is under way.", key.noun())
	}
	return job, true
}

// startupPool is every held player still on no roster and undrafted; an O-Line card is never drafted.
func startupPool(facts nativecheck.Facts, taken map[string]bool) []DraftPlayer {
	out := []DraftPlayer{}
	for id := range facts {
		if _, card := nativecheck.LineCardTeam(id); !card && !taken[id] {
			out = append(out, playerNamed(id, facts))
		}
	}
	slices.SortFunc(out, func(a, b DraftPlayer) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
	})
	return out
}

// checkStartupList holds a commissioner's startup order, once set, to every franchise exactly once.
func checkStartupList(trial []Record, at time.Time) error {
	rules, err := RulesNow(trial, at)
	if err != nil {
		return err
	}
	list := rules.Draft.StartupList
	if list.State != leaguerules.IsKnown || len(list.Value) == 0 {
		return nil
	}
	if err := listsEveryFranchise(list.Value, LeagueFranchises(trial)); err != nil {
		return fmt.Errorf("the startup draft order must %w", err)
	}
	return nil
}
