package core

import (
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func cloneAssets(rows []leaguefeed.Asset) []leaguefeed.Asset {
	out := slices.Clone(rows)
	for i := range out {
		a := &out[i]
		a.Player, a.CurrentPick, a.FuturePick, a.BlindBidCents = clonePtr(a.Player), clonePtr(a.CurrentPick), clonePtr(a.FuturePick), clonePtr(a.BlindBidCents)
	}
	return out
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	value := *p
	return &value
}
