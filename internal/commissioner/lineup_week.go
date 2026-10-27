package commissioner

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// LineupFacts is what a lineup depends on and the ledger does not hold: who each player is, which NFL
// team he plays for, and when that team's game starts in the lineup's week. A lock is only as good as
// these, so a week whose kickoffs are not held locks nobody in or out rather than guessing.
type LineupFacts struct {
	Facts        nativecheck.Facts
	Locks        map[string]time.Time
	ScheduleHeld bool
	Zone         *time.Location
}

// lockKind says how a player's game bears on his place in a lineup.
type lockKind int

const (
	lockNoGame  lockKind = iota // no NFL team, or the team is on a bye
	lockUnknown                 // the week's kickoffs are not held
	lockAt
)

func (f LineupFacts) zone() *time.Location {
	if f.Zone == nil {
		return time.UTC
	}
	return f.Zone
}

func (f LineupFacts) lockOf(player string) (time.Time, lockKind) {
	team := f.Facts[player].Team
	if team == "" || team == "FA" {
		return time.Time{}, lockNoGame
	}
	if !f.ScheduleHeld {
		return time.Time{}, lockUnknown
	}
	kickoff, ok := f.Locks[team]
	if !ok {
		return time.Time{}, lockNoGame
	}
	return kickoff, lockAt
}

// localTime writes an instant in the owner's own zone.
func (f LineupFacts) localTime(at time.Time) string {
	return at.In(f.zone()).Format("Jan 2, 2006 at 3:04 PM MST")
}

func (f LineupFacts) nameOf(player string) string {
	if name := f.Facts[player].Name; name != "" {
		return name
	}
	return "player " + player
}

// RecordedLineup is the lineup standing for one franchise's week, with who put it there.
type RecordedLineup struct {
	Starters  []string
	Entry     string
	By        string
	Corrected bool
	Reason    string
	At        time.Time
}

// RecordedLineupFor folds the active lineup commands of a franchise's week: the latest one stands, and
// a reversed one stands for nothing, so compensation restores the lineup before it.
func RecordedLineupFor(records []Record, franchise string, season, week int) (RecordedLineup, bool, error) {
	var out RecordedLineup
	found := false
	var active map[string]bool
	for _, r := range records {
		if !isLineupCommand(r.Command.Command) {
			continue
		}
		if active == nil {
			active = Replay(records).Active
		}
		change, err := DecodeLineupChange(r.Command.Args, r.Command.Command)
		if err != nil {
			return RecordedLineup{}, false, err
		}
		if !active[r.Receipt.Entry] || change.Franchise != franchise || change.Season != season || change.Week != week {
			continue
		}
		out, found = RecordedLineup{Starters: slices.Clone(change.Starters), Entry: r.Receipt.Entry,
			By: r.Principal, Corrected: r.Command.Command == LineupCorrect, Reason: r.Command.Reason,
			At: r.Receipt.RecordedAt}, true
	}
	return out, found, nil
}

// LedgerWeekLineup is the lineup that counts for a franchise's week, read from the ledger alone: a franchise
// that recorded none has none, and nothing is borrowed from any outside league.
func LedgerWeekLineup(records []Record, franchise string, season, week int) (WeekLineup, error) {
	lineup, found, err := RecordedLineupFor(records, franchise, season, week)
	if err != nil || !found {
		return WeekLineup{}, err
	}
	out := WeekLineup{Set: true, Starters: make([]Starter, len(lineup.Starters))}
	for i, player := range lineup.Starters {
		out.Starters[i] = Starter{Player: player}
	}
	return out, nil
}

// CheckLineupAppend holds a lineup command to the league's rules before it is recorded. An owner's lineup
// must be legal under the lineup check and may not move a player whose game has started; a correction
// overrides the lock and is judged, not refused, for legality: an illegal one is penalised when scored.
func CheckLineupAppend(raw RawCommand, records []Record, now time.Time, env LineupFacts) error {
	if !isLineupCommand(raw.Command) {
		return nil
	}
	change, err := DecodeLineupChange(raw.Args, raw.Command)
	if err != nil {
		return Refused(err)
	}
	base, err := ReadNativeBase(raw.League, records, now)
	if err != nil {
		return err
	}
	if !base.Assets.Seeded {
		return Refused(fmt.Errorf("record the league's starting snapshot before setting a lineup"))
	}
	seat, ok := base.Directory.seat(change.Franchise)
	if !ok {
		return Refused(fmt.Errorf("choose one of this league's franchises"))
	}
	at := ReadingInstant(records, now)
	if _, err := RulesInForce(records, change.Season, at); err != nil {
		return Refused(err)
	}
	if err := checkLineupRoster(base, change, env); err != nil {
		return Refused(err)
	}
	if raw.Command == LineupCorrect {
		return nil
	}
	return checkOwnerLineup(records, now, env, change, seat)
}

// checkOwnerLineup holds an owner's lineup to this season, to the commissioner's ruling if one stands, to the
// league's lineup check and to the locks.
func checkOwnerLineup(records []Record, now time.Time, env LineupFacts, change RawLineupChange,
	seat NativeDirectorySeat) error {
	at := ReadingInstant(records, now)
	if current, err := SeasonAt(records, at); err != nil || current != change.Season {
		return Refused(fmt.Errorf("owners set this season's lineups; ask a commissioner to correct an earlier one"))
	}
	held, found, err := RecordedLineupFor(records, change.Franchise, change.Season, change.Week)
	if err != nil {
		return err
	}
	if found && held.Corrected {
		return Refused(fmt.Errorf("a commissioner has set %s's week %d lineup; ask them to change it", seat.Name, change.Week))
	}
	rules, err := RulesInForce(records, change.Season, at)
	if err != nil {
		return err
	}
	if verdict, _ := judgeStarters(rules.Lineup, true, change.Starters, env.Facts); verdict.Status != nativecheck.LineupValid {
		return Refused(fmt.Errorf("%s", verdict.Words))
	}
	return checkLineupLocks(held.Starters, change, now, env)
}

// checkLineupRoster requires every starter to be on the franchise's active roster: IR and taxi players
// cannot start.
func checkLineupRoster(base NativeBase, change RawLineupChange, env LineupFacts) error {
	active := map[string]bool{}
	for _, p := range base.playersOf()[change.Franchise] {
		active[p.ID] = p.Status == domain.RosterActive
	}
	for _, id := range change.Starters {
		if !active[id] {
			return fmt.Errorf("%s is not on this franchise's active roster, so cannot start", env.nameOf(id))
		}
	}
	return nil
}

// checkLineupLocks refuses any move of a player whose game has started, in or out of the lineup.
func checkLineupLocks(was []string, change RawLineupChange, now time.Time, env LineupFacts) error {
	moved := []string{}
	for _, player := range change.Starters {
		if !slices.Contains(was, player) {
			moved = append(moved, player)
		}
	}
	for _, player := range was {
		if !slices.Contains(change.Starters, player) {
			moved = append(moved, player)
		}
	}
	locked, unknown := []string{}, []string{}
	for _, player := range moved {
		kickoff, kind := env.lockOf(player)
		switch {
		case kind == lockUnknown:
			unknown = append(unknown, fmt.Sprintf("We don't hold the week %d NFL kickoff times, so we can't tell when %s locks",
				change.Week, env.nameOf(player)))
		case kind == lockAt && !now.Before(kickoff):
			locked = append(locked, fmt.Sprintf("%s locked at kickoff (%s)", env.nameOf(player), env.localTime(kickoff)))
		}
	}
	sentences := slices.Clone(unknown)
	if len(locked) != 0 {
		sentences = append(sentences, strings.Join(locked, "; ")+
			"; a lineup can't add or drop a player once that player's game has started")
	}
	if len(sentences) != 0 {
		return Refused(fmt.Errorf("%s", strings.Join(sentences, ". ")))
	}
	return nil
}

// judgeStarters runs the league's lineup check on starters named by id, using the positions held for them.
// A position the app does not hold leaves the lineup unjudged rather than guessed.
func judgeStarters(rules leaguerules.LineupRules, submitted bool, ids []string,
	facts nativecheck.Facts) (nativecheck.LineupVerdict, []NativeStarter) {
	starters, listed, unplaced := []nativecheck.Starter{}, []NativeStarter{}, 0
	for _, id := range ids {
		f, _ := facts.Of(id)
		if f.Position == "" {
			unplaced++
		}
		starters = append(starters, nativecheck.Starter{ID: id, Position: f.Position})
		listed = append(listed, NativeStarter{Player: id, Name: f.Name, Position: string(f.Position)})
	}
	verdict := nativecheck.JudgeLineup(rules, submitted, starters)
	if unplaced > 0 {
		verdict = unjudgedLineup(unplaced)
	}
	return verdict, listed
}
