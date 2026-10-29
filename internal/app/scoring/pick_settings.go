package scoring

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// PickSettingsSource updates held value readings without fetching or rerunning player values.
type PickSettingsSource struct {
	Lock, Unlock func()
	Timeout      time.Duration
	Model        func(context.Context)
	Held         func() season.Held
	Board        Source
	Publish      func(season.Held)
}

func PickSettingChanged(parent context.Context, key string, ok bool, src PickSettingsSource) {
	if !ok || (key != params.KeyPickValueDiscount && key != params.KeyPowerPickWeight) {
		return
	}
	src.Lock()
	defer src.Unlock()
	ctx, cancel := context.WithTimeout(parent, src.Timeout)
	defer cancel()
	if key == params.KeyPickValueDiscount {
		src.Model(ctx)
	}
	held := src.Held()
	if len(held.Standings.Value) == 0 {
		return
	}
	if held.Power == nil {
		held.Power = map[string]season.PowerRankingsResult{}
	}
	for _, mode := range []string{m2service.AggLineup, m2service.AggSum, m2service.AggTopN} {
		key := season.PowerKey(season.ViewFranchise, mode)
		prior := held.Power[key]
		rebuilt := src.Board.Build(ctx, held.Standings.Value, prior.Freshness, func(context.Context) []leagueschedule.RawScheduleWeek { return nil }, 0, false, mode, season.ViewFranchise)
		if !rebuilt.OK {
			return
		}
		held.Power[key] = rebuilt
	}
	src.Publish(held)
}
