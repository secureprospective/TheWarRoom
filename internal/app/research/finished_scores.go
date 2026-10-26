package research

import (
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type Finished struct {
	Rows   map[string]*Score
	Held   int
	Weeks  int
	Latest time.Time
}

// FinishedScores uses only held weeks before the current week, never YTD or live scores.
func FinishedScores(in Snapshot) Finished {
	result := Finished{Rows: map[string]*Score{}, Weeks: max(in.Week-1, 0)}
	for week := 1; week <= result.Weeks; week++ {
		feed := in.Held.Feeds[Key("playerScores", strconv.Itoa(week))]
		if len(feed.Value) == 0 {
			continue
		}
		result.Held++
		if feed.FetchedAt.After(result.Latest) {
			result.Latest = feed.FetchedAt
		}
		for _, entry := range Rows(feed) {
			id := entry.ID.String()
			score := result.Rows[id]
			if score == nil {
				score = &Score{Position: Player(in.Lookup, in.Mirror, in.Config, id).Position}
				result.Rows[id] = score
			}
			score.Sum += entry.Value
			score.Games++
		}
	}
	return result
}

// PrepareSnapshot enriches a detached capture without mutating its published inputs.
func PrepareSnapshot(in Snapshot, now func() time.Time, cutCharge func(state.MirrorPlayer, int) (domain.Money, bool)) Snapshot {
	if in.Now.IsZero() {
		in.Now = now()
	}
	finished := FinishedScores(in)
	in.Finished, in.Latest = finished.Rows, finished.Latest
	in.CutCharges = map[string]domain.Money{}
	for _, p := range in.Mirror.Players {
		if charge, ok := cutCharge(p, in.Mirror.Season); ok {
			in.CutCharges[p.MFLID] = charge
		}
	}
	return in
}
