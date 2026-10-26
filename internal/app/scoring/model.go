package scoring

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// ModelSource supplies the runner and the shared-state snapshot at its original read point.
type ModelSource struct {
	EndWeek    func() (string, bool)
	State      func() state.Reader
	History    modelrun.History
	BuildLabel string
	Season     int
	FreeAgents func() []string
}

// EngineDue reports whether a synchronous model pass is needed.
func EngineDue(now time.Time, last history.Run, exists, changed bool) bool {
	return changed || !exists || now.Sub(last.AsOf) >= 24*time.Hour
}

// RunModel values the league with the measurables model (plan Stage 7) and writes a model
// run. The caller holds its refresh lock.
func (a ModelSource) RunModel(ctx context.Context, lk normalize.Lookup, set params.Set) (modelrun.Report, error) {
	end, _ := a.EndWeek()
	last, err := strconv.Atoi(end)
	if err != nil || last < 1 {
		return modelrun.Report{}, fmt.Errorf("app: the league's last scoring week %q is unreadable", end)
	}
	runner, err := modelrun.New(a.State(), lk, a.History, a.BuildLabel)
	if err != nil {
		return modelrun.Report{}, fmt.Errorf("app: %w", err)
	}
	free := a.FreeAgents()
	rep, err := runner.Run(ctx, modelrun.Spec{Season: a.Season, AsOf: time.Now(), Params: set, LastWeek: last, FreeAgents: free, Refresh: true})
	if err != nil {
		return modelrun.Report{}, fmt.Errorf("app: %w", err)
	}
	return rep, nil
}

func FreeAgentIDs(ids []playerid.PlayerID) []string {
	var free []string
	for _, id := range ids {
		free = append(free, id.String())
	}
	return free
}
