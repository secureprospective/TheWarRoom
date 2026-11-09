package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// NativeAssets is app-owned state, not an MFL roster or a what-if plan.
type NativeAssets struct {
	PoolPointsNote string                `json:"poolPointsNote,omitempty"`
	Pool           []NativePoolPlayer    `json:"pool,omitempty"`
	Identities     []NativeAssetIdentity `json:"identities,omitempty"`
	Corrections    []Correction          `json:"corrections,omitempty"`
	League         string                `json:"league"`
	Revision       int64                 `json:"revision"`
	Seeded         bool                  `json:"seeded"`
	Entry          string                `json:"entry"`
	Source         string                `json:"source"`
	Snapshot       *RawNativeSnapshot    `json:"snapshot"`
}

type NativePoolPlayer struct {
	Age      string                   `json:"age,omitempty"`
	Value    string                   `json:"value,omitempty"`
	Minimum  *snapshot.ContractAmount `json:"minimum,omitempty"`
	Player   string                   `json:"player"`
	Name     string                   `json:"name"`
	Position string                   `json:"position"`
	String   string                   `json:"string"`
	Points   string                   `json:"points"`
}

// NativeAssetIdentity holds public football facts separately from recorded roster state.
type NativeAssetIdentity struct {
	Player    string `json:"player"`
	Franchise string `json:"franchise"`
	Side      string `json:"side,omitempty"`
	Name      string `json:"name"`
	Position  string `json:"position"`
}

// NativeAssetsAsOf projects effective-dated history from its earliest recorded roster state.
// Entries learned later may repair earlier history; receipt times remain unchanged.
func NativeAssetsAsOf(league string, records []Record, at time.Time) (NativeAssets, error) {
	if at.IsZero() {
		return NativeAssets{}, fmt.Errorf("choose the roster observation time")
	}
	seed, err := seedNativeAssets(league, records)
	if err != nil {
		return NativeAssets{}, err
	}
	visible, err := nativeRecordsAt(records, at)
	if err != nil {
		return NativeAssets{}, err
	}
	if seed.Seeded && at.Before(rosterHistoryStart(seed.Snapshot.RosterAt, visible)) {
		return NativeAssets{}, fmt.Errorf("roster history before the starting observation is not recorded")
	}
	return NativeAssetsReading(league, records, at)
}

func NativeAssetsReading(league string, records []Record, at ...time.Time) (NativeAssets, error) {
	assets, _, err := nativeAssetsState(league, records, at...)
	return assets, err
}

// foldNativeAssets projects the ledger; nativeAssetsState serves it from the held revision's cache.
func foldNativeAssets(league string, records []Record, at ...time.Time) (NativeAssets, MarketBook, error) {
	waivers := MarketBook{Windows: []WaiverWindow{}, Order: []string{}}
	out, err := seedNativeAssets(league, records)
	if err != nil {
		return NativeAssets{}, waivers, err
	}
	now := records[len(records)-1].Receipt.RecordedAt
	if len(at) != 0 {
		now = at[0]
	}
	visible := records
	if len(at) != 0 {
		visible, err = nativeRecordsAt(records, now)
		if err != nil {
			return NativeAssets{}, waivers, err
		}
	}
	if err := projectNativeOverrides(&out, visible, &waivers, nil); err != nil {
		return NativeAssets{}, waivers, err
	}
	changes, err := CorrectionsReading(league, visible, now)
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
		sides, err := snapshotSides(records)
		if err != nil {
			return NativeAssets{}, err
		}
		seed := *r.Command.Args.Native
		seed.Sides = sides
		if err := seed.Validate(); err != nil {
			return NativeAssets{}, fmt.Errorf("read starting snapshot: %w", err)
		}
		seed.Champions = slices.Clone(seed.Champions)
		seed.StandingsOrder = slices.Clone(seed.StandingsOrder)
		seed.Franchises = slices.Clone(seed.Franchises)
		seed.Players = slices.Clone(seed.Players)
		seed.Picks = slices.Clone(seed.Picks)
		seed.PickHistory = slices.Clone(seed.PickHistory)
		seed.Adjustments = slices.Clone(seed.Adjustments)
		seed.Pool = slices.Clone(scopedOLineLots(seed))
		out.Seeded, out.Entry, out.Snapshot = true, r.Receipt.Entry, &seed
		out.Source = "App league; starting snapshot recorded from held data, never synced with MFL"
	}
	return out, nil
}

func rosterHistoryStart(observed time.Time, records []Record) time.Time {
	active := Replay(records).Active
	for _, r := range records {
		if !active[r.Receipt.Entry] || !RosterTimed(r.Command.Command) || r.Command.Command == CorrectPick {
			continue
		}
		effective, err := nativeEntryTime(r)
		if err == nil && effective.Before(observed) {
			observed = effective
		}
	}
	return observed
}
