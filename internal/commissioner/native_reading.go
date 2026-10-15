package commissioner

import (
	"fmt"
	"slices"
)

// NativeAssets is app-owned state, not an MFL roster or a what-if plan.
type NativeAssets struct {
	League   string             `json:"league"`
	Revision int64              `json:"revision"`
	Seeded   bool               `json:"seeded"`
	Entry    string             `json:"entry"`
	Source   string             `json:"source"`
	Snapshot *RawNativeSnapshot `json:"snapshot"`
}

func NativeAssetsReading(league string, records []Record) (NativeAssets, error) {
	assets, _, err := nativeAssetsState(league, records)
	return assets, err
}

func nativeAssetsState(league string, records []Record) (NativeAssets, waiverState, error) {
	waivers := waiverState{Windows: []waiverWindow{}, Order: []string{}}
	if len(records) == 0 {
		return NativeAssets{}, waivers, fmt.Errorf("this app league has not been created")
	}
	out := NativeAssets{League: league, Revision: int64(len(records)), Source: "App league; no starting snapshot recorded"}
	for _, r := range records {
		if r.Command.Command != NativeSeed {
			continue
		}
		if out.Seeded || r.Command.Args.Native == nil {
			return NativeAssets{}, waivers, fmt.Errorf("the league has an invalid starting snapshot history")
		}
		if err := r.Command.Args.Native.Validate(); err != nil {
			return NativeAssets{}, waivers, fmt.Errorf("read starting snapshot: %w", err)
		}
		seed := *r.Command.Args.Native
		seed.StandingsOrder = slices.Clone(seed.StandingsOrder)
		seed.Franchises = slices.Clone(seed.Franchises)
		seed.Players = slices.Clone(seed.Players)
		seed.Picks = slices.Clone(seed.Picks)
		seed.Adjustments = slices.Clone(seed.Adjustments)
		out.Seeded, out.Entry, out.Snapshot = true, r.Receipt.Entry, &seed
		out.Source = "App league; starting snapshot recorded from held data, never synced with MFL"
	}
	if err := projectNativeOverrides(&out, records, &waivers); err != nil {
		return NativeAssets{}, waivers, err
	}
	return out, waivers, nil
}

// NativeSeedRetry reuses the recorded body even if the held mirror changed after the first attempt.
func NativeSeedRetry(records []Record, request string) (RawCommand, bool) {
	for _, r := range records {
		if r.Command.Request == request {
			return r.Command, true
		}
	}
	return RawCommand{}, false
}
