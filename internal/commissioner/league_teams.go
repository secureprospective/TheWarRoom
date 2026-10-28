package commissioner

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	teamsField     = "league.teams"
	divisionsField = "league.divisions"
)

// errTeamsInPlay refuses a team count change once the league has played or moved a player.
var errTeamsInPlay = errors.New("teams can't change once the season has games or player moves; " +
	"an offseason change is not available yet")

// teamCount reads the count a Teams change sets; ok is false for any other entry.
func teamCount(r Record) (int, bool) {
	if r.Command.Command != RulesSet || r.Command.Args.Field != teamsField || r.Command.Effective != "now" {
		return 0, false
	}
	setting, err := leaguerules.DecodeSetting(teamsField, r.Command.Args.Value)
	if err != nil {
		return 0, false
	}
	value, ok := setting.(leaguerules.Value[int])
	if !ok || value.State != leaguerules.IsKnown {
		return 0, false
	}
	return value.Value, true
}

// LeagueFranchises is the league's franchises in order: the creation's list, resized by each Teams change
// in force. The Teams change is the record of the list; every reader of the franchises reads it here.
func LeagueFranchises(records []Record) []string {
	if len(records) == 0 {
		return nil
	}
	ids := records[0].Command.Args.Franchises
	var active map[string]bool
	for _, r := range records[1:] {
		n, ok := teamCount(r)
		if !ok {
			continue
		}
		if active == nil {
			active = Replay(records).Active
		}
		if active[r.Receipt.Entry] {
			ids = resizedFranchises(ids, n)
		}
	}
	return ids
}

// resizedFranchises adds the next ids in the creation's pattern, or drops the highest.
func resizedFranchises(ids []string, n int) []string {
	if n <= len(ids) {
		return slices.Clone(ids[:n])
	}
	out := slices.Clone(ids)
	for i := len(ids); i < n; i++ {
		out = append(out, franchiseIDs(i + 1)[i])
	}
	return out
}

// defaultFranchiseName is the name a franchise starts with until it is renamed.
func defaultFranchiseName(id string) (string, string) {
	n := franchiseNumber(id)
	return fmt.Sprintf("Franchise %d", n), fmt.Sprintf("F%d", n)
}

// franchiseNumber is a franchise's seat number from its id, 0 for an id the app did not number.
func franchiseNumber(id string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "franchise."))
	if err != nil || !strings.HasPrefix(id, "franchise.") || n < 1 || fmt.Sprintf("franchise.%02d", n) != id {
		return 0
	}
	return n
}

// teamsEvents seats the franchises a Teams change lists and removes the rest.
func teamsEvents(n int) []governance.Event {
	listed := make([]governance.FranchiseIdentitySet, n)
	for i, id := range franchiseIDs(n) {
		name, abbreviation := defaultFranchiseName(id)
		listed[i] = governance.FranchiseIdentitySet{Franchise: governance.FranchiseID(id), Name: name, Abbreviation: abbreviation}
	}
	return []governance.Event{governance.FranchisesResized{Franchises: listed}}
}

// projectTeamsChange resizes the native assets when a rule change sets the number of teams.
func projectTeamsChange(assets *NativeAssets, record Record) error {
	if n, ok := teamCount(record); ok && assets.Seeded {
		return resizeNativeFranchises(assets.Snapshot, n)
	}
	return nil
}

// resizeNativeFranchises applies a Teams change to the native assets: new franchises start empty and
// undivided; a removed franchise must hold nothing.
func resizeNativeFranchises(seed *RawNativeSnapshot, n int) error {
	ids := make([]string, len(seed.Franchises))
	for i, f := range seed.Franchises {
		ids[i] = f.ID
	}
	want := resizedFranchises(ids, n)
	for _, f := range seed.Franchises[min(n, len(seed.Franchises)):] {
		if err := emptyFranchise(seed, f); err != nil {
			return err
		}
	}
	kept := seed.Franchises[:min(n, len(seed.Franchises))]
	for _, id := range want[len(kept):] {
		name, abbreviation := defaultFranchiseName(id)
		kept = append(kept, NativeFranchise{ID: id, HeldID: id, Name: name, Abbreviation: abbreviation})
	}
	seed.Franchises = kept
	return nil
}

// emptyFranchise refuses to remove a franchise that still holds players, picks or cap charges.
func emptyFranchise(seed *RawNativeSnapshot, f NativeFranchise) error {
	switch {
	case slices.ContainsFunc(seed.Players, func(p NativePlayer) bool { return p.Franchise == f.ID }):
		return fmt.Errorf("%s still has players; move or release them before removing the team", f.Name)
	case slices.ContainsFunc(seed.Picks, func(p NativePick) bool { return p.Owner == f.ID || p.Original == f.ID }):
		return fmt.Errorf("%s still holds draft picks; move them before removing the team", f.Name)
	case slices.ContainsFunc(seed.Adjustments, func(c NativeCharge) bool { return c.Franchise == f.ID }):
		return fmt.Errorf("%s still has cap charges; clear them before removing the team", f.Name)
	}
	return nil
}

// setupCommand is an entry that sets the league up without playing it or moving a player.
func setupCommand(r Record) bool {
	if r.Command.Args.People != nil {
		return true
	}
	if _, ok := noteDefinitionFor(r.Command.Command); ok {
		return true
	}
	switch r.Command.Command {
	case Create, NativeSeed, RulesSet, ContractActionSet, WindowSet, WindowMove, WindowRemove, CalendarException,
		Revert, PlayerPosition, PlayerPositionClear, "commish.franchise.edit":
		return true
	}
	return standingRule(r.Command.Command)
}

// playBegun says the league has scheduled, played or moved anything since it started.
func playBegun(records []Record) bool {
	return slices.ContainsFunc(records, func(r Record) bool { return !setupCommand(r) })
}

// checkShapeChange checks a Teams or Divisions change: a native league before play, now, within range.
func shapeField(field string) bool {
	return field == teamsField || field == divisionsField
}

func checkShapeChange(raw RawCommand, records []Record, at time.Time) error {
	if raw.Command == Revert {
		return checkTeamsRevert(raw, records)
	}
	if raw.Command != RulesSet || !shapeField(raw.Args.Field) {
		return nil
	}
	label := "Teams"
	if raw.Args.Field == divisionsField {
		label = "Divisions"
	}
	if LeagueSource(records) != SourceNative {
		return Refused(fmt.Errorf("rule change refused: %s follows the league's MFL franchises in an imported league", label))
	}
	if raw.Effective != "now" {
		return Refused(fmt.Errorf("rule change refused: a %s change takes effect now; choose now", label))
	}
	setting, err := leaguerules.DecodeSetting(leaguerules.FieldPath(raw.Args.Field), raw.Args.Value)
	if err != nil {
		return Refused(fmt.Errorf("rule change refused: %w", err))
	}
	value, ok := setting.(leaguerules.Value[int])
	if !ok || value.State != leaguerules.IsKnown {
		return Refused(fmt.Errorf("rule change refused: choose a number of %s", strings.ToLower(label)))
	}
	teams := len(LeagueFranchises(records))
	if raw.Args.Field == divisionsField {
		if value.Value < 1 || value.Value > teams {
			return Refused(fmt.Errorf("rule change refused: choose between 1 and %d divisions", teams))
		}
		return nil
	}
	if !validFranchiseCount(value.Value) {
		return Refused(fmt.Errorf("rule change refused: choose between %d and %d teams", minFranchises, maxFranchises))
	}
	return checkTeamsTrial(raw, records, at, value.Value, value.Value < teams)
}

// checkTeamsTrial projects the change: a removed team must hold nothing and seat no one, and no team
// changes once play has begun.
func checkTeamsTrial(raw RawCommand, records []Record, at time.Time, n int, removing bool) error {
	trial := append(slices.Clone(records), Record{Command: raw, Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: at}})
	if removing {
		if _, err := NativeAssetsReading(raw.League, trial); err != nil {
			return Refused(fmt.Errorf("rule change refused: %w", err))
		}
		if err := checkSeatsFree(records, n); err != nil {
			return Refused(fmt.Errorf("rule change refused: %w", err))
		}
		if _, err := GovernanceState(trial); err != nil {
			return Refused(fmt.Errorf("rule change refused: %w", err))
		}
	}
	if playBegun(records) {
		return Refused(fmt.Errorf("rule change refused: %w", errTeamsInPlay))
	}
	return nil
}

// checkSeatsFree names the first team a change to n teams would remove while someone holds its seat.
func checkSeatsFree(records []Record, n int) error {
	state, err := GovernanceState(records)
	if err != nil {
		return err
	}
	for _, id := range LeagueFranchises(records)[n:] {
		if f, ok := state.Franchise(governance.FranchiseID(id)); ok && (f.Seat != "" || f.Caretaker != "") {
			return fmt.Errorf("%s still has an owner; free the seat before removing the team", f.Name)
		}
	}
	return nil
}

// checkTeamsRevert refuses to reverse a Teams change: the count is changed again instead, through the
// same checks.
func checkTeamsRevert(raw RawCommand, records []Record) error {
	for _, r := range records {
		if r.Receipt.Entry != raw.Args.Entry {
			continue
		}
		if _, ok := teamCount(r); ok {
			return Refused(errors.New("a Teams change is not reversed; change Teams again instead"))
		}
	}
	return nil
}

// shapePreview words a Teams change by the franchises it adds or removes, and shows the alignment a
// Divisions change divides the league into.
func shapePreview(out *Preview, raw RawCommand, records []Record, now time.Time) error {
	if !shapeField(raw.Args.Field) || len(out.Fields) == 0 {
		return nil
	}
	setting, err := leaguerules.DecodeSetting(leaguerules.FieldPath(raw.Args.Field), raw.Args.Value)
	if err != nil {
		return fmt.Errorf("describe proposed rule: %w", err)
	}
	value, ok := setting.(leaguerules.Value[int])
	if !ok || value.State != leaguerules.IsKnown {
		return nil
	}
	ids := LeagueFranchises(records)
	if raw.Args.Field == teamsField {
		out.Fields[0].After = teamsWords(records, ids, value.Value, now)
		return nil
	}
	if value.Value < 1 || value.Value > len(ids) {
		return nil
	}
	rules, err := RulesNow(records, now)
	if err != nil {
		return err
	}
	before, err := alignmentWords(records, rules.League.Alignment)
	if err != nil {
		return err
	}
	after, err := alignmentWords(records, leaguerules.Known(leaguerules.EvenDivisions(ids, value.Value)))
	if err != nil {
		return err
	}
	if before != after {
		out.Fields = append(out.Fields, PreviewField{Label: "Division alignment", Before: before,
			BeforeSource: "In force now", After: after, AfterSource: "Divided evenly by this change", When: timingWords(raw.Effective)})
	}
	return nil
}

// teamsWords says "14 (adds Franchise 13, Franchise 14)" or "12 (removes Franchise 13, Franchise 14)".
func teamsWords(records []Record, ids []string, n int, now time.Time) string {
	names := []string{}
	verb := "adds"
	if n < len(ids) {
		verb = "removes"
		directory, err := NativeDirectoryReading(records[0].Command.League, records, now)
		seats := directory.Seats()
		for _, id := range ids[n:] {
			name, _ := defaultFranchiseName(id)
			if seat, ok := seats[id]; ok && err == nil {
				name = seat.Name
			}
			names = append(names, name)
		}
	}
	for _, id := range resizedFranchises(ids, n)[min(n, len(ids)):] {
		name, _ := defaultFranchiseName(id)
		names = append(names, name)
	}
	if len(names) == 0 {
		return fmt.Sprint(n)
	}
	return fmt.Sprintf("%d (%s %s)", n, verb, strings.Join(names, ", "))
}
