package commissioner

import (
	"fmt"
	"strings"
	"time"
)

func nativePreviewValues(raw RawCommand, records []Record, now time.Time, out map[string]string) error {
	assets, book, err := nativeAssetsState(raw.League, records, now)
	if err != nil {
		return err
	}
	if assets.Seeded {
		directory, err := NativeDirectoryReading(raw.League, records, now)
		if err != nil {
			return err
		}
		names := map[string]string{}
		for _, f := range directory.Franchises {
			names[f.Franchise] = f.Name
		}
		snapshotPreviewValues(assets.Snapshot, names, out)
	}
	if assets.Seeded {
		capView, capErr := NativeCapCheckReading(RawNativeCapScope{League: raw.League, ViewerZone: "UTC"}, records, now)
		if capErr != nil {
			return capErr
		}
		for _, f := range capView.Franchises {
			out[f.Name+" cap room"] = f.Words
			if f.CommittedCents != nil {
				out[f.Name+" counted salary"] = fmt.Sprintf("%s counted salary", *f.CommittedCents)
			}
		}
	}
	marketPreviewValues(raw, book, out)
	if raw.Args.Trade != nil {
		for _, t := range book.Trades {
			out["Trade "+t.ID] = fmt.Sprintf("%s; %s; %d review votes, %d recusals, %d appeals, %d overrides",
				t.Status, t.Words, len(t.Votes), len(t.Recusals), len(t.Appeals), len(t.Overrides))
			tradePreviewVotes(t, out)
		}
	}
	if raw.Args.Correction != nil {
		changes, err := CorrectionsReading(raw.League, records, now)
		if err != nil {
			return err
		}
		for _, c := range changes.Items {
			out["Correction "+c.Entry] = c.What
		}
	}
	return nil
}

func acquisitionWords(via string) string {
	words := map[string]string{"held": "Starting roster", "draft": "Rookie draft", "waiver": "Waivers",
		"trade": "Trade", "free_agent": "Free agency", "commissioner": "Commissioner ruling"}
	if words[via] != "" {
		return words[via]
	}
	return "No acquisition source recorded"
}

func pickPreviewWords(p NativePick) string {
	if p.Player != "" {
		return "; player " + p.Player
	}
	if p.Skipped {
		return "; skipped"
	}
	return "; not used"
}

func snapshotPreviewValues(seed *RawNativeSnapshot, names map[string]string, out map[string]string) {
	for _, p := range seed.Players {
		name := p.Name
		if name == "" {
			name = "Player " + p.ID
		}
		out[name+" roster"] = names[p.Franchise] + ", " + string(p.Status)
		out[name+" contract"] = fmt.Sprintf("%s salary; contract ends in %d; %s", p.Salary, p.ExpirationYear, p.ContractInfo)
		out[name+" acquisition"] = acquisitionWords(p.AcquiredVia)
	}
	for _, p := range seed.Picks {
		out[fmt.Sprintf("Season %d, round %d, %s pick (slot %d)", p.Season, p.Round, p.Original, p.Slot)] =
			names[p.Owner] + pickPreviewWords(p)
	}
	for _, f := range seed.Franchises {
		charges := []string{}
		for _, c := range seed.Adjustments {
			if c.Franchise == f.ID {
				charges = append(charges, fmt.Sprintf("%s: %s", c.Description, c.Amount))
			}
		}
		out[names[f.ID]+" cap charges"] = strings.Join(charges, "; ")
	}

}

func marketPreviewValues(raw RawCommand, book waiverState, out map[string]string) {
	if raw.Args.Waiver != nil {
		out["Waiver priority"] = strings.Join(book.Order, ", ") + " — " + book.Source
		for _, w := range book.Windows {
			words := "Waiting for processing"
			if w.Processed {
				words = "Available as a free agent"
				if w.Winner != "" {
					words = "Awarded to " + w.Winner
				}
			}
			out["Waivers for player "+w.Player.ID] = words
		}
	}
	if raw.Args.Bid != nil {
		for _, a := range book.Bids.Auctions {
			words := fmt.Sprintf("%d bids awaiting a decision", len(a.liveBids()))
			if a.Decided {
				words = "No award"
				if a.Winner != "" {
					words = "Awarded to " + a.Winner
				}
			}
			out["Bids for player "+a.Player] = words
		}
	}
}

func tradePreviewVotes(trade TradeState, out map[string]string) {
	trails := []struct {
		label string
		votes []TradeVote
	}{
		{"Review", trade.Votes}, {"Recusal", trade.Recusals}, {"Appeal", trade.Appeals}, {"Override", trade.Overrides},
	}
	for _, trail := range trails {
		for _, vote := range trail.votes {
			out["Trade "+trade.ID+" — "+trail.label+" by "+personWords(vote.Who)] = tradeVoteWords(vote.Decision) + "; " + vote.Reason
		}
	}
}

func tradeVoteWords(decision string) string {
	words := map[string]string{"approve": "Approve", "veto": "Veto", "appeal": "Appeal filed", TradeRecuse: "Sitting out"}
	if words[decision] != "" {
		return words[decision]
	}
	return decision
}

func PreviewFieldSources(raw RawCommand, fields []PreviewField) {
	for i := range fields {
		f := &fields[i]
		if f.Label == "League ledger" {
			continue
		}
		if seed := raw.Args.Native; seed != nil {
			f.AfterSource = "Held rosters captured " + seed.RosterAt.Format(time.RFC3339)
			if strings.Contains(f.Label, "pick") {
				f.AfterSource = "Held draft reports captured " + seed.DraftAt.Format(time.RFC3339) +
					"; future picks captured " + seed.FutureAt.Format(time.RFC3339)
			}
		}
		if raw.Command == ReserveException && raw.Args.Override.Until != "" {
			f.When = "Now, until " + raw.Args.Override.Until
		}
	}
}
