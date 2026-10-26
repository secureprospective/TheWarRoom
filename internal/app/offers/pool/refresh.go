package pool

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/freeagents"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func FreeAgentIDs(rows []freeagents.RawFreeAgent, err error) ([]playerid.PlayerID, error) {
	var ids []playerid.PlayerID
	if err == nil {
		ids, err = freeagents.ToPlayerIDs(rows)
		SortIDs(ids)
	}
	return ids, err //nolint:wrapcheck // Preserve the existing free-agent conversion refusal.
}

func HoldFreeAgents(held *report.HeldFeed[[]playerid.PlayerID], ids []playerid.PlayerID, fresh domain.Freshness, err error, changed func()) {
	if err == nil && !slices.Equal(held.Value, ids) {
		changed()
	}
	report.HoldFeed(held, ids, fresh, err)
}

func RestoreFreeAgents(ctx context.Context, held bool, season int,
	bodies func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error),
	publish func([]playerid.PlayerID, time.Time), logFailure func(string)) {
	if held {
		return
	}
	_, err := bodies(ctx, "TYPE="+freeagents.Export, func(src string, body []byte, at time.Time) bool {
		if !ingestion.SameExport(src, freeagents.Export, season) {
			return false
		}
		rows, err := freeagents.Parse(body)
		if err != nil {
			return false
		}
		ids, err := freeagents.ToPlayerIDs(rows)
		if err != nil {
			return false
		}
		SortIDs(ids)
		publish(ids, at)
		return true
	})
	if err != nil {
		logFailure("engine: previous free-agent list unavailable: " + err.Error())
	}
}

func RefreshPool(force bool, ready func() error, held func() (time.Time, string), since func(time.Time) time.Duration,
	market func() error, refresh func()) error {
	if err := ready(); err != nil {
		return fmt.Errorf("target pool refresh: startup: %w", err)
	}
	at, failed := held()
	if !force && failed == "" && since(at) < 5*time.Minute {
		return market()
	}
	refresh()
	return nil
}
