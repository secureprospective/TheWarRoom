package commissioner

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	StatApprove = "commish.stat.queue.approve"
	StatReject  = "commish.stat.queue.reject"
	StatCorrect = "commish.stat.correct"

	// statDigestLen is how much of a version's digest an item name carries.
	statDigestLen = 16
	// maxStatQuantity bounds a hand-entered stat; no player records a million of anything in a week.
	maxStatQuantity = leaguerules.Decimal(1_000_000 * 10000)
	maxSourceLen    = 200
)

// RawStatChange is a stat command's argument (Args.Stat). Approve and reject name a waiting item; a
// correction names a player's week, one stat, its value and where the value came from.
type RawStatChange struct {
	Item   string               `json:"item,omitempty"`
	Player string               `json:"player,omitempty"`
	Season int                  `json:"season,omitempty"`
	Week   int                  `json:"week,omitempty"`
	Stat   leaguerules.StatID   `json:"stat,omitempty"`
	Value  *leaguerules.Decimal `json:"value,omitempty"`
	Source string               `json:"source,omitempty"`
}

// StatItemID names one waiting change: a player's week and the version of it the feed now reports.
type StatItemID struct {
	Season, Week int
	Player       string
	// Seq numbers the versions the feed has given for the week; Digest is the start of that version's digest.
	Seq    int
	Digest string
}

func (id StatItemID) String() string {
	return fmt.Sprintf("stat.%d.%d.%s.%d.%s", id.Season, id.Week, id.Player, id.Seq, id.Digest)
}

// ParseStatItemID is the only way in from the outside: it refuses anything String would not write.
func ParseStatItemID(text string) (StatItemID, error) {
	parts := strings.Split(text, ".")
	if len(parts) != 6 || parts[0] != "stat" {
		return StatItemID{}, errNotAnItem
	}
	numbers, err := atois(parts[1], parts[2], parts[4])
	if err != nil {
		return StatItemID{}, errNotAnItem
	}
	id := StatItemID{Season: numbers[0], Week: numbers[1], Player: parts[3], Seq: numbers[2], Digest: parts[5]}
	player, err := playerid.New(id.Player)
	if err != nil || player.String() != id.Player || !id.inRange() || id.String() != text {
		return StatItemID{}, errNotAnItem
	}
	return id, nil
}

var errNotAnItem = errors.New("choose a change from the statistics queue")

func (id StatItemID) inRange() bool {
	return id.Season >= 1920 && id.Season <= 9999 && id.Week >= 1 && id.Week <= 22 && id.Seq >= 1 &&
		len(id.Digest) == statDigestLen && strings.Trim(id.Digest, "0123456789abcdef") == ""
}

func atois(texts ...string) ([]int, error) {
	out := make([]int, len(texts))
	for i, text := range texts {
		n, err := strconv.Atoi(text)
		if err != nil {
			return nil, fmt.Errorf("read number: %w", err)
		}
		out[i] = n
	}
	return out, nil
}

func isStatCommand(command string) bool {
	return command == StatApprove || command == StatReject || command == StatCorrect
}

// IsStatCommand says whether the command decides or enters a statistic.
func IsStatCommand(command string) bool { return isStatCommand(command) }

// DecodeStatChange reads a stat command's argument and refuses details that belong to another command.
func DecodeStatChange(a RawArgs, command string) (RawStatChange, error) {
	other := a
	other.Stat = nil
	if a.Stat == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return RawStatChange{}, fmt.Errorf("send only the details of this statistic change")
	}
	raw := *a.Stat
	if command == StatCorrect {
		return raw, raw.validateCorrection()
	}
	if raw.Player != "" || raw.Season != 0 || raw.Week != 0 || raw.Stat != "" || raw.Value != nil || raw.Source != "" {
		return raw, fmt.Errorf("send only the change you are deciding")
	}
	_, err := ParseStatItemID(raw.Item)
	return raw, err
}

func (r RawStatChange) validateCorrection() error {
	if r.Item != "" {
		return fmt.Errorf("send the player, week, statistic, value and source")
	}
	id, err := playerid.New(r.Player)
	if err != nil || id.String() != r.Player {
		return fmt.Errorf("choose a player by the league's full id")
	}
	if r.Season < 1920 || r.Season > 9999 || r.Week < 1 || r.Week > 22 {
		return fmt.Errorf("choose a season and week")
	}
	if !validStatID(r.Stat) {
		return fmt.Errorf("choose a recorded statistic")
	}
	if r.Value == nil || *r.Value > maxStatQuantity || (*r.Value < 0 && !strings.Contains(string(r.Stat), "yards")) {
		return fmt.Errorf("enter the value the statistic should have")
	}
	return validStatSource(r.Source)
}

// validStatSource requires the person to say where a hand-entered value came from.
func validStatSource(text string) error {
	if strings.TrimSpace(text) != text || text == "" || len(text) > maxSourceLen {
		return fmt.Errorf("say where this value comes from, in up to %d characters", maxSourceLen)
	}
	for _, r := range text {
		if unicode.IsControl(r) {
			return fmt.Errorf("say where this value comes from in plain text")
		}
	}
	return nil
}

func validStatID(id leaguerules.StatID) bool {
	if id == "" || len(id) > 64 || id[0] < 'a' || id[0] > 'z' {
		return false
	}
	return strings.Trim(string(id), "abcdefghijklmnopqrstuvwxyz0123456789_") == ""
}

// statKey is one player's week in one season.
type statKey struct {
	player       string
	season, week int
}

// key is the player's week the change is about; for an item it comes from the item's name.
func (r RawStatChange) key() statKey {
	if id, err := ParseStatItemID(r.Item); err == nil {
		return statKey{id.Player, id.Season, id.Week}
	}
	return statKey{r.Player, r.Season, r.Week}
}

// Scope is the player's week the change is about; for an approval or rejection it comes from the item.
func (r RawStatChange) Scope() (player string, season, week int) {
	key := r.key()
	return key.player, key.season, key.week
}

func statChangeLabel(command string, change RawStatChange) string {
	key := change.key()
	where := fmt.Sprintf("player %s in season %d, week %d", key.player, key.season, key.week)
	switch command {
	case StatApprove:
		return "Counted the NFL's updated statistics for " + where
	case StatReject:
		return "Kept the league's count and turned down the NFL's update for " + where
	default:
		return fmt.Sprintf("Corrected %s to %s for %s; source: %s", change.Stat, scorePointsWords(*change.Value),
			where, change.Source)
	}
}
