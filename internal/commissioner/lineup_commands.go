package commissioner

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	// LineupSet is an owner setting the starters of their own franchise for a week; LineupCorrect is the
	// commissioner side's CM-029 ruling on which lineup counts, lock or no lock.
	LineupSet     = "commish.lineup.set"
	LineupCorrect = "commish.lineup.correct"
)

// RawLineupChange is a lineup command's argument (Args.Lineup): the whole set of starters for one
// franchise and week. A later lineup replaces an earlier one; it is never merged with it.
type RawLineupChange struct {
	Season    int      `json:"season"`
	Franchise string   `json:"franchise"`
	Week      int      `json:"week"`
	Starters  []string `json:"starters"`
}

func isLineupCommand(command string) bool { return command == LineupSet || command == LineupCorrect }

// DecodeLineupChange reads a lineup command's argument and refuses details that belong to another command.
func DecodeLineupChange(a RawArgs, command string) (RawLineupChange, error) {
	other := a
	other.Lineup = nil
	if a.Lineup == nil || !reflect.DeepEqual(other, RawArgs{}) || !isLineupCommand(command) {
		return RawLineupChange{}, fmt.Errorf("send only the details of this lineup")
	}
	raw := *a.Lineup
	if raw.Season < 1920 || raw.Season > 9999 || raw.Week < 1 || raw.Week > 22 ||
		!strings.HasPrefix(raw.Franchise, "franchise.") || strings.TrimSpace(raw.Franchise) != raw.Franchise {
		return raw, fmt.Errorf("choose a franchise, season and week for this lineup")
	}
	return raw, raw.validateStarters()
}

func (r RawLineupChange) validateStarters() error {
	if len(r.Starters) == 0 || len(r.Starters) > maxLineupStarters {
		return fmt.Errorf("list the players who start, once each")
	}
	seen := map[string]bool{}
	for _, player := range r.Starters {
		id, err := playerid.New(player)
		if err != nil || id.String() != player || seen[player] {
			return fmt.Errorf("list each starter once, by the player's full league id")
		}
		seen[player] = true
	}
	return nil
}

// validateLineupPlacement fixes when a lineup takes effect and who may read it: an owner's lineup is
// private to the franchise and the commissioners, a correction is shown to the league with its reason.
func (r RawCommand) validateLineupPlacement() error {
	if r.Effective != "now" {
		return fmt.Errorf("lineups take effect now")
	}
	if r.Command == LineupSet && r.Visibility != "private" {
		return fmt.Errorf("a lineup stays private to its owner and the commissioners")
	}
	if r.Command == LineupCorrect && (r.Visibility != "league" || strings.TrimSpace(r.Reason) == "") {
		return fmt.Errorf("a lineup correction is shown to the league with the reason for it")
	}
	return nil
}

func lineupLabel(command string, change RawLineupChange) string {
	starters := fmt.Sprintf("%d starters", len(change.Starters))
	if len(change.Starters) == 1 {
		starters = "1 starter"
	}
	if command == LineupCorrect {
		return fmt.Sprintf("Set the week %d lineup of %s to %s by ruling", change.Week, change.Franchise, starters)
	}
	return fmt.Sprintf("Set the week %d lineup of %s to %s", change.Week, change.Franchise, starters)
}
