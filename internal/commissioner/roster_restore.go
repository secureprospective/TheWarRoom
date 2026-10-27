package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// RawRosterRestorePoint preserves roster and contract state independently of event history.
// It is private league data, not a grant of authority to operate the league.
type RawRosterRestorePoint struct {
	Identities []playerid.RawIdentity `json:"identities,omitempty"`
	Version    int                    `json:"version"`
	League     string                 `json:"league"`
	Revision   int64                  `json:"revision"`
	CapturedAt time.Time              `json:"capturedAt"`
	Snapshot   RawNativeSnapshot      `json:"snapshot"`
}

func (r RawRosterRestorePoint) Validate() error {
	if r.Version != 1 || !strings.HasPrefix(r.League, "native.") || r.League == "native." ||
		strings.TrimSpace(r.League) != r.League || r.Revision < 2 || r.CapturedAt.IsZero() {
		return fmt.Errorf("invalid league roster restore point")
	}
	if err := r.validateIdentities(); err != nil {
		return err
	}
	return r.validateRoster()
}

func (r RawRosterRestorePoint) validateRoster() error {
	s := r.Snapshot
	if !validFranchiseCount(len(s.Franchises)) {
		return fmt.Errorf("a league needs between 2 and 32 franchises")
	}
	if s.Season < 1 || s.Season > 9999 || s.RosterAt.IsZero() || s.DraftAt.IsZero() ||
		s.FutureAt.IsZero() || len(s.Picks) != 0 ||
		len(s.Pool) != 0 || len(s.Champions) != 0 || s.Champion != "" || len(s.StandingsOrder) != 0 {
		return fmt.Errorf("invalid roster restore point directory or season")
	}
	franchises, err := s.franchiseIndex()
	if err != nil {
		return fmt.Errorf("invalid restore point franchises: %w", err)
	}
	if err := s.validatePlayers(franchises); err != nil {
		return fmt.Errorf("invalid restore point contracts: %w", err)
	}
	if err := s.validateArchive(franchises); err != nil {
		return fmt.Errorf("invalid restore point divisions: %w", err)
	}
	return s.validateCharges(franchises)
}

// RosterRestorePoint omits draft and historical results; player contract fields are exact.
func RosterRestorePoint(league string, records []Record) (RawRosterRestorePoint, bool, error) {
	if len(records) == 0 {
		return RawRosterRestorePoint{}, false, nil
	}
	last := records[len(records)-1]
	assets, err := NativeAssetsReading(league, records, last.Receipt.RecordedAt)
	if err != nil {
		return RawRosterRestorePoint{}, false, fmt.Errorf("capture league rosters: %w", err)
	}
	if !assets.Seeded || assets.Snapshot == nil {
		return RawRosterRestorePoint{}, false, nil
	}
	snapshot := *assets.Snapshot
	snapshot.Picks, snapshot.Pool, snapshot.Champions = nil, nil, nil
	snapshot.Champion, snapshot.StandingsOrder = "", nil
	point := RawRosterRestorePoint{Version: 1, League: league, Revision: last.Receipt.Revision,
		CapturedAt: last.Receipt.RecordedAt, Snapshot: snapshot}
	if err := point.Validate(); err != nil {
		return RawRosterRestorePoint{}, false, err
	}
	return point, true, nil
}

func (r RawRosterRestorePoint) validateIdentities() error {
	if len(r.Identities) == 0 {
		return nil
	}
	players := make(map[string]bool, len(r.Snapshot.Players))
	for _, p := range r.Snapshot.Players {
		players[p.ID] = true
	}
	seen := make(map[string]bool)
	links := make(map[playerid.RawOutsideID]bool)
	for _, identity := range r.Identities {
		if err := identity.Validate(); err != nil {
			return fmt.Errorf("restore player identity: %w", err)
		}
		if seen[identity.Canonical] {
			return fmt.Errorf("repeated restore player identity")
		}
		seen[identity.Canonical] = true
		mfl := ""
		for _, link := range identity.Links {
			if links[link] {
				return fmt.Errorf("repeated restore outside link")
			}
			links[link] = true
			if link.Provider == "mfl" {
				mfl = link.Value
			}
		}
		if !players[mfl] {
			return fmt.Errorf("restore identity does not name an owned player")
		}
		delete(players, mfl)
	}
	if len(players) != 0 {
		return fmt.Errorf("restore identities do not cover every owned player")
	}
	return nil
}
