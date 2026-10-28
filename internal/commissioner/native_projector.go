package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// projectNativeOverrides starts from cloned seed slices; neither ledger bodies nor held inputs mutate. visit, when
// given, sees the assets after each entry is applied.
func projectNativeOverrides(assets *NativeAssets, records []Record, waivers *MarketBook, visit func(*NativeAssets)) error {
	active := Replay(records).Active
	since := 0
	if visit == nil {
		var err error
		since, err = resumeClosedNative(assets, records, waivers)
		if err != nil {
			return err
		}
	}
	for i := since; i < len(records); i++ {
		record := records[i]
		if !active[record.Receipt.Entry] {
			continue
		}
		if err := projectNativeEntry(assets, records, i, waivers); err != nil {
			return err
		}
		if visit != nil {
			visit(assets)
		}
	}
	return nil
}

// projectNativeEntry applies one active ledger entry to the native assets.
func projectNativeEntry(assets *NativeAssets, records []Record, i int, waivers *MarketBook) error {
	record := records[i]
	if TradeCommand(record.Command.Command) {
		if !assets.Seeded {
			return fmt.Errorf("record starting assets before trading")
		}
		return projectTrade(assets.Snapshot, waivers, record)
	}
	if WaiverCommand(record.Command.Command) {
		if !assets.Seeded {
			return fmt.Errorf("record starting assets before waiver actions")
		}
		return projectWaiver(assets.Snapshot, waivers, record)
	}
	if BidCommand(record.Command.Command) {
		if !assets.Seeded {
			return fmt.Errorf("record starting assets before bid actions")
		}
		return projectBid(assets.Snapshot, waivers, record)
	}
	switch record.Command.Command {
	case RulesSet:
		return projectTeamsChange(assets, record)
	case CorrectRoster, CorrectTransaction, CorrectProvenance, CorrectPick, RuleException:
		return applyAssetCorrection(assets.Snapshot, records[:i], record)
	case SeasonRollover:
		return applyNativeRollover(assets, record)
	case DraftPick, DraftAssign, DraftInsert, DraftChange, DraftArchive:
		projectDraftEntry(assets.Snapshot, record)
	case ContractEdit, CapAdjust, CapRelief, ReserveException:
		if !assets.Seeded {
			return fmt.Errorf("a native correction has no starting assets")
		}
		value, err := decodeOverride(record.Command.Command, record.Command.Args)
		if err != nil {
			return err
		}
		rules, err := RulesNow(records[:i+1], record.Receipt.RecordedAt)
		if err != nil {
			return err
		}
		return applyNativeOverride(assets.Snapshot, record, value, rules.Season)
	}
	return nil
}

// projectDraftEntry applies a live draft entry: a pick, a commissioner's tray move or the draft's close.
func projectDraftEntry(seed *RawNativeSnapshot, record Record) {
	switch record.Command.Command {
	case DraftPick:
		applyDraftPick(seed, record)
	case DraftAssign, DraftInsert:
		applyDraftTray(seed, record)
	case DraftChange:
		applyDraftChange(seed, record)
	case DraftArchive:
		applyDraftClose(seed, record)
	}
}

func applyNativeOverride(seed *RawNativeSnapshot, record Record, value RawNativeOverride, season int) error {
	if record.Command.Command == CapAdjust {
		for _, franchise := range seed.Franchises {
			if franchise.ID == value.Franchise {
				seed.Adjustments = append(seed.Adjustments, NativeCharge{ID: record.Receipt.Entry,
					Franchise: value.Franchise, Amount: *value.Amount, Season: value.Season,
					Description: "Commissioner salary adjustment: " + record.Command.Reason})
				return nil
			}
		}
		return fmt.Errorf("choose a franchise in this app league")
	}
	side, err := seed.TargetSide(value.Franchise)
	if err != nil {
		return err
	}
	if i := seed.PlayerOn(side, value.Player); i >= 0 {
		player := &seed.Players[i]
		if value.Franchise != "" && player.Franchise != value.Franchise {
			return fmt.Errorf("choose a player on that franchise's recorded roster")
		}
		switch record.Command.Command {
		case ContractEdit:
			expiration := season + *value.Years - 1
			if expiration > 9999 {
				return fmt.Errorf("the contract ends beyond the supported calendar")
			}
			player.Salary, player.ExpirationYear = *value.Salary, expiration
		case CapRelief:
			seed.Adjustments = append(seed.Adjustments, NativeCharge{ID: record.Receipt.Entry,
				Franchise: player.Franchise, Amount: -*value.Amount, Season: season,
				Description: "Salary relief for player " + player.ID + ": " + record.Command.Reason})
		case ReserveException:
			// The target is checked here; the reserve reader applies time and bucket scope.
		}
		return nil
	}
	return fmt.Errorf("choose a player on this app league's recorded roster")
}

func applyReserveExceptions(verdicts []nativecheck.ReserveVerdict, records []Record, now time.Time,
	franchise string, perSide bool) error {
	active := Replay(records).Active
	for _, record := range records {
		if record.Command.Command != ReserveException || !active[record.Receipt.Entry] {
			continue
		}
		value, err := decodeOverride(record.Command.Command, record.Command.Args)
		if err != nil {
			return err
		}
		if (perSide && value.Franchise == "") || (value.Franchise != "" && value.Franchise != franchise) {
			continue
		}
		if value.Until != "" {
			until, parseErr := time.Parse(time.RFC3339Nano, value.Until)
			if parseErr != nil {
				return fmt.Errorf("read exception expiry: %w", parseErr)
			}
			if !now.Before(until) {
				continue
			}
		}
		for i := range verdicts {
			verdict := &verdicts[i]
			if verdict.Player != value.Player || strings.ToLower(verdict.Squad) != value.Bucket {
				continue
			}
			verdict.Standing = nativecheck.Stays
			verdict.Words = "May stay on " + verdict.Squad + ": commissioner exception — " + record.Command.Reason
		}
	}
	return nil
}
