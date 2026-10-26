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

type EngineRefreshSource struct {
	Available  func() bool
	TryLock    func() bool
	Unlock     func()
	Changed    func(context.Context)
	Timeout    time.Duration
	Signals    func(context.Context)
	Points     func(context.Context) (string, error)
	FreeAgents func(context.Context)
	Check      func(context.Context)
	Log        func(string)
}

func RefreshEngine(parent context.Context, source EngineRefreshSource) {
	if !source.Available() || !source.TryLock() {
		return
	}
	defer source.Unlock()
	defer source.Changed(parent)
	ctx, cancel := context.WithTimeout(parent, source.Timeout)
	defer cancel()
	source.Signals(ctx)
	if warning, err := source.Points(ctx); err != nil {
		source.Log(fmt.Sprintf("engine inputs: %v", err))
		return
	} else if warning != "" {
		source.Log("engine inputs: " + warning)
	}
	source.FreeAgents(ctx)
	source.Check(ctx)
}

type EngineCheckSource struct {
	Latest        func(context.Context) (history.Run, bool, error)
	InputsChanged func(context.Context, time.Time) (bool, error)
	Now           func() time.Time
	Started       func() time.Time
	DirtyLoad     func() bool
	DirtySwap     func(bool) bool
	DirtyStore    func(bool)
	Directory     func(context.Context) (normalize.Lookup, error)
	Run           func(context.Context, normalize.Lookup) (modelrun.Report, error)
	Finished      func(context.Context, time.Time, modelrun.Report)
	Log           func(string)
}

// CheckEngine consumes the dirty flag immediately before running, restoring it on failure.
func CheckEngine(ctx context.Context, source EngineCheckSource) {
	last, exists, err := source.Latest(ctx)
	if err != nil {
		source.Log(fmt.Sprintf("engine latest run: %v", err))
		return
	}
	changed, err := source.InputsChanged(ctx, last.AsOf)
	if err != nil {
		source.Log(fmt.Sprintf("engine input changes: %v", err))
		return
	}
	if !EngineDue(source.Now(), last, exists, changed || source.DirtyLoad()) {
		return
	}
	lk, err := source.Directory(ctx)
	if err != nil {
		source.Log(fmt.Sprintf("engine directory: %v", err))
		return
	}
	// Changes arriving during the pass must remain pending for the next refresh.
	dirty := source.DirtySwap(false)
	started := source.Started()
	rep, err := source.Run(ctx, lk)
	if err != nil {
		if dirty {
			source.DirtyStore(true)
		}
		source.Log(fmt.Sprintf("engine run failed: %v", err))
		return
	}
	source.Finished(ctx, started, rep)
}

func EngineFinished(ctx context.Context, started time.Time, rep modelrun.Report, since func(time.Time) time.Duration, changed func(context.Context), logLine func(string)) {
	logLine(fmt.Sprintf("Engine run %s: %d players valued (%d rostered, %d free agents), %s",
		started.Local().Format(time.RFC3339), rep.Scored, rep.RosteredScored, rep.FreeAgentScored, since(started).Round(time.Millisecond)))
	SourceHealthChanged(ctx, changed)
}

func SourceHealthChanged(ctx context.Context, changed func(context.Context)) {
	if changed != nil {
		changed(ctx)
	}
}
