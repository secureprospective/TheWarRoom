package commissioner

import (
	"fmt"
	"slices"
	"time"
)

// NativeAssets is app-owned state, not an MFL roster or a what-if plan.
type NativeAssets struct {
	Corrections []Correction       `json:"corrections,omitempty"`
	League      string             `json:"league"`
	Revision    int64              `json:"revision"`
	Seeded      bool               `json:"seeded"`
	Entry       string             `json:"entry"`
	Source      string             `json:"source"`
	Snapshot    *RawNativeSnapshot `json:"snapshot"`
}

func NativeAssetsReading(league string, records []Record, at ...time.Time) (NativeAssets, error) {
	assets, _, err := nativeAssetsState(league, records, at...)
	return assets, err
}

func nativeAssetsState(league string, records []Record, at ...time.Time) (NativeAssets, waiverState, error) {
	waivers := waiverState{Windows: []waiverWindow{}, Order: []string{}}
	out, err := seedNativeAssets(league, records)
	if err != nil {
		return NativeAssets{}, waivers, err
	}
	if err := projectNativeOverrides(&out, records, &waivers, nil); err != nil {
		return NativeAssets{}, waivers, err
	}
	now := records[len(records)-1].Receipt.RecordedAt
	if len(at) != 0 {
		now = at[0]
	}
	changes, err := CorrectionsReading(league, records, now)
	if err != nil {
		return NativeAssets{}, waivers, err
	}
	out.Corrections = changes.Items
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

// seedNativeAssets is the league's starting snapshot, cloned so projecting over it never changes a ledger body.
func seedNativeAssets(league string, records []Record) (NativeAssets, error) {
	if len(records) == 0 {
		return NativeAssets{}, fmt.Errorf("this app league has not been created")
	}
	out := NativeAssets{League: league, Revision: int64(len(records)), Source: "App league; no starting snapshot recorded"}
	for _, r := range records {
		if r.Command.Command != NativeSeed {
			continue
		}
		if out.Seeded || r.Command.Args.Native == nil {
			return NativeAssets{}, fmt.Errorf("the league has an invalid starting snapshot history")
		}
		if err := r.Command.Args.Native.Validate(); err != nil {
			return NativeAssets{}, fmt.Errorf("read starting snapshot: %w", err)
		}
		seed := *r.Command.Args.Native
		seed.Champions = slices.Clone(seed.Champions)
		seed.StandingsOrder = slices.Clone(seed.StandingsOrder)
		seed.Franchises = slices.Clone(seed.Franchises)
		seed.Players = slices.Clone(seed.Players)
		seed.Picks = slices.Clone(seed.Picks)
		seed.Adjustments = slices.Clone(seed.Adjustments)
		seed.Pool = slices.Clone(seed.Pool)
		out.Seeded, out.Entry, out.Snapshot = true, r.Receipt.Entry, &seed
		out.Source = "App league; starting snapshot recorded from held data, never synced with MFL"
	}
	return out, nil
}
