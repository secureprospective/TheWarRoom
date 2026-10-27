package commissioner

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
)

func cloneClosedNative(raw RawClosedBooks) (NativeAssets, MarketBook, error) {
	assets := raw.Native
	if assets.Snapshot != nil {
		snapshot := cloneRolloverAssets(*assets.Snapshot)
		snapshot.StandingsOrder = slices.Clone(assets.Snapshot.StandingsOrder)
		snapshot.Conferences = maps.Clone(assets.Snapshot.Conferences)
		assets.Snapshot = &snapshot
	}
	// The waiver book also owns nested bid and trade slices. Its persisted shape
	// is the same shape resumed here, including private, already-decided claims.
	body, err := json.Marshal(raw.Waivers)
	if err != nil {
		return NativeAssets{}, MarketBook{}, fmt.Errorf("copy closed waiver book: %w", err)
	}
	var waivers MarketBook
	if err := json.Unmarshal(body, &waivers); err != nil {
		return NativeAssets{}, MarketBook{}, fmt.Errorf("read closed waiver book: %w", err)
	}
	return assets, waivers, nil
}

func resumeClosedNative(assets *NativeAssets, records []Record, waivers *MarketBook) (int, error) {
	b := usableClosedBooks(records)
	if b == nil {
		return 0, nil
	}
	// Native projectors consume entire command bodies. Unlike the smaller shared
	// folds, their checkpoint must also reject edits to a caller's nested payloads.
	hash, err := ClosedBooksHash(records[:b.raw.Revision])
	if err != nil || hash != b.raw.SourceHash {
		return 0, nil //nolint:nilerr // Unhashable edited source bodies invalidate this derived checkpoint; full replay owns their errors.
	}
	held, book, err := cloneClosedNative(b.raw)
	if err != nil {
		return 0, err
	}
	held.Revision = int64(len(records))
	*assets, *waivers = held, book
	return b.raw.Revision, nil
}

func (r RawClosedBooks) validateNative() error {
	if r.Native.Revision != int64(r.Revision) || r.Native.League == "" {
		return fmt.Errorf("invalid closed season assets")
	}
	if r.Native.Seeded {
		if r.Native.Snapshot == nil {
			return fmt.Errorf("missing closed season assets")
		}
		if err := r.Native.Snapshot.validate(true); err != nil {
			return fmt.Errorf("invalid closed season assets: %w", err)
		}
	}
	return nil
}
