package control

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

type ParameterStore interface {
	Snapshot() params.Set
	Definitions() []params.ParamDef
	Overridden(string, string) bool
	SetOverride(context.Context, string, string, float64, string) error
	ClearOverride(context.Context, string, string) error
}

func ReadParams(ready func() error, storeFor func() ParameterStore) ParamsResult {
	if err := ready(); err != nil {
		return ParamsResult{OK: false, Error: err.Error()}
	}
	store := storeFor()
	values := store.Snapshot()
	defs := store.Definitions()
	out := make([]ParamView, len(defs))
	for i, d := range defs {
		v, err := values.GetPosition(d.Key, d.Position)
		if err != nil {
			return ParamsResult{OK: false, Error: err.Error()}
		}
		label, group := ParameterWords(d.Key, d.Description)
		out[i] = ParamView{Key: d.Key, Label: label, Group: group, Position: d.Position, Description: d.Description,
			Default: d.Default, Min: d.Min, Max: d.Max, Value: v, Calibrated: d.IsCalibrated,
			Overridden: store.Overridden(d.Key, d.Position)}
	}
	slices.SortFunc(out, func(a, b ParamView) int {
		if n := cmp.Compare(a.Group, b.Group); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Label, b.Label); n != 0 {
			return n
		}
		return cmp.Compare(a.Position, b.Position)
	})
	return ParamsResult{OK: true, Params: out}
}

func SetParameter(contextFor func() context.Context, ready func() error, storeFor func() ParameterStore, key, position string, value float64) SetParamResult {
	if err := ready(); err != nil {
		return SetParamResult{OK: false, Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(contextFor(), 3*time.Second)
	defer cancel()
	if err := storeFor().SetOverride(ctx, key, position, value, "admin console"); err != nil {
		return SetParamResult{OK: false, Error: err.Error()}
	}
	return SetParamResult{OK: true}
}

func ResetParameter(contextFor func() context.Context, ready func() error, storeFor func() ParameterStore, key, position string) SetParamResult {
	if err := ready(); err != nil {
		return SetParamResult{OK: false, Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(contextFor(), 3*time.Second)
	defer cancel()
	if err := storeFor().ClearOverride(ctx, key, position); err != nil {
		return SetParamResult{OK: false, Error: err.Error()}
	}
	return SetParamResult{OK: true}
}
