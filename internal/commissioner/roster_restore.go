package commissioner

import (
	"fmt"
	"strings"
	"time"
)

// RawRosterRestorePoint preserves roster and contract state independently of event history.
// It is private league data, not a grant of authority to operate the league.
type RawRosterRestorePoint struct {
	Version    int               `json:"version"`
	League     string            `json:"league"`
	Revision   int64             `json:"revision"`
	CapturedAt time.Time         `json:"capturedAt"`
	Snapshot   RawNativeSnapshot `json:"snapshot"`
}

func (r RawRosterRestorePoint) Validate() error {
	if r.Version != 1 || !strings.HasPrefix(r.League, "native.") || r.League == "native." ||
		strings.TrimSpace(r.League) != r.League || r.Revision < 2 || r.CapturedAt.IsZero() {
		return fmt.Errorf("invalid league roster restore point")
	}
	return r.validateRoster()
}

func (r RawRosterRestorePoint) validateRoster() error {
	s := r.Snapshot
	if s.Season < 1 || s.Season > 9999 || s.RosterAt.IsZero() || s.DraftAt.IsZero() ||
		s.FutureAt.IsZero() || len(s.Franchises) != legacyTeams || len(s.Picks) != 0 ||
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
