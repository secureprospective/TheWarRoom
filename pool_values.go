package main

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func noNFLTeam(team string) bool { return team == "" || team == "FA" || team == "—" }

func (a *App) poolAssetValues(ctx context.Context, ids []playerid.PlayerID, at time.Time) (map[string]AssetValue, string) {
	a.hydrateResearchContext(ctx)
	var model assetModel
	if held := a.assetModel.Load(); held != nil {
		model = *held
	}
	also := make([]string, 0, len(ids))
	for _, id := range ids {
		also = append(also, id.String())
	}
	board := buildAssetBoard(researchView{researchInput: a.heldResearchInput()}, model, at, also)
	out := make(map[string]AssetValue, len(board.Values))
	for _, v := range board.Values {
		out[v.ID] = v
	}
	return out, board.NowSource
}

func valuePoolRow(row *PoolRow, engine map[string]float64, assets map[string]AssetValue) {
	row.EngineText, row.PPGText, row.LabText = "Not valued", "No score", "Not valued"
	if v, ok := engine[row.ID]; ok {
		row.Engine = &v
		row.EngineText = fmt.Sprintf("%.2f points/game", v)
	}
	if v, ok := assets[row.ID]; ok {
		row.PPGText, row.LabText = v.Now.Text, v.Dynasty.Text
		if v.Now.Held {
			row.PPG = &v.NowValue
		}
		if v.Dynasty.Held {
			row.Lab = &v.DynastyValue
		} else {
			row.LabText = "Not valued"
		}
	}
}

func poolNumber(row PoolRow, key string) *float64 {
	switch key {
	case "engine":
		return row.Engine
	case "ppg":
		return row.PPG
	case "lab":
		return row.Lab
	default:
		return nil
	}
}

func poolNumericOrder(x, y PoolRow, key string) (int, bool) {
	xv, yv := poolNumber(x, key), poolNumber(y, key)
	if index := slices.Index([]string{"projected", "rank", "adp", "aav"}, key); index >= 0 {
		xv, yv = poolMarketValue(x, index), poolMarketValue(y, index)
	}
	if xv == nil && yv == nil {
		return 0, false
	}
	if xv == nil {
		return 1, true
	}
	if yv == nil {
		return -1, true
	}
	return cmp.Compare(*xv, *yv), false
}
