package commissioner

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func nativePreviewValues(raw RawCommand, records []Record, now time.Time, out map[string]string) error {
	assets, book, err := nativeAssetsState(raw.League, records, now)
	if err != nil {
		return err
	}
	names := map[string]string{}
	var seats []PeopleSeat
	if assets.Seeded {
		directory, err := NativeDirectoryReading(raw.League, records, now)
		if err != nil {
			return err
		}
		for _, f := range directory.Franchises {
			names[f.Franchise] = f.Name
			seats = append(seats, f.PeopleSeat)
		}
		if raw.Command == NativeSeed {
			seedPreviewValues(assets.Snapshot, names, out)
		} else {
			snapshotPreviewValues(assets.Snapshot, names, out)
		}
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
	marketPreviewValues(raw, book, names, out)
	if raw.Args.Trade != nil {
		labels := map[string]int{}
		for _, t := range book.Trades {
			label := tradePreviewLabel(raw, t, names, labels)
			out[label] = tradePreviewWords(t)
			tradePreviewVotes(t, label, seats, records, out)
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

// rosterStatusWords names a squad the way the owner screens do, never by its stored code.
func rosterStatusWords(status domain.RosterStatus) string {
	switch status {
	case domain.RosterIR:
		return "Injured reserve"
	case domain.RosterTaxi:
		return "Taxi squad"
	case domain.RosterActive:
	}
	return "Active roster"
}

func acquisitionWords(via string) string {
	words := map[string]string{"held": "Starting roster", "draft": "Rookie draft", "waiver": "Waivers",
		"trade": "Trade", "free_agent": "Free agency", "commissioner": "Commissioner ruling"}
	if words[via] != "" {
		return words[via]
	}
	return "No acquisition source recorded"
}

func pickPreviewWords(p NativePick, players map[string]string) string {
	if p.Player != "" {
		return "; used on " + cmp.Or(players[p.Player], "a player no longer on a roster")
	}
	if p.Skipped {
		return "; skipped"
	}
	return "; not used"
}

// seedPreviewValues sums up starting assets per franchise. Recording them adds every rostered player at
// once, and a row per player would bury the reason under thousands of lines.
func seedPreviewValues(seed *RawNativeSnapshot, names map[string]string, out map[string]string) {
	statusWords := []struct {
		status domain.RosterStatus
		words  string
	}{{domain.RosterActive, "on the active roster"}, {domain.RosterIR, "on IR"}, {domain.RosterTaxi, "on the taxi squad"}}
	for _, f := range seed.Franchises {
		counts := map[domain.RosterStatus]int{}
		players := 0
		var salary domain.Money
		for _, p := range seed.Players {
			if p.Franchise == f.ID {
				players++
				counts[p.Status]++
				salary += p.Salary
			}
		}
		parts := []string{}
		for _, s := range statusWords {
			if counts[s.status] > 0 {
				parts = append(parts, fmt.Sprintf("%d %s", counts[s.status], s.words))
			}
		}
		roster := "No rostered players"
		if players > 0 {
			roster = fmt.Sprintf("%d players: %s; %s salary", players, strings.Join(parts, ", "), salary)
		}
		out[names[f.ID]+" starting roster"] = roster
		out[names[f.ID]+" draft picks"] = seedPickWords(seed.Picks, f.ID)
		out[names[f.ID]+" cap charges"] = seedChargeWords(seed.Adjustments, f.ID)
	}
}

// seedChargeWords totals a franchise's starting cap charges; each charge's own words stay in the ledger.
func seedChargeWords(charges []NativeCharge, franchise string) string {
	count := 0
	var total domain.Money
	for _, c := range charges {
		if c.Franchise == franchise {
			count++
			total += c.Amount
		}
	}
	switch count {
	case 0:
		return "No cap charges"
	case 1:
		return fmt.Sprintf("1 cap charge totalling %s", total)
	}
	return fmt.Sprintf("%d cap charges totalling %s", count, total)
}

// seedPickWords lists a franchise's starting draft picks by season, such as "2027 rounds 1, 2, 2, 3".
func seedPickWords(picks []NativePick, franchise string) string {
	rounds := map[int][]string{}
	seasons := []int{}
	for _, p := range picks {
		if p.Owner != franchise {
			continue
		}
		if rounds[p.Season] == nil {
			seasons = append(seasons, p.Season)
		}
		rounds[p.Season] = append(rounds[p.Season], fmt.Sprint(p.Round))
	}
	if len(seasons) == 0 {
		return "No draft picks"
	}
	slices.Sort(seasons)
	words := []string{}
	for _, season := range seasons {
		words = append(words, fmt.Sprintf("%d rounds %s", season, strings.Join(rounds[season], ", ")))
	}
	return strings.Join(words, "; ")
}

func snapshotPreviewValues(seed *RawNativeSnapshot, names map[string]string, out map[string]string) {
	players := map[string]string{}
	for _, p := range seed.Players {
		name := p.Name
		if name == "" {
			name = "Player " + p.ID
		}
		players[p.ID] = name
		out[name+" roster"] = names[p.Franchise] + ", " + rosterStatusWords(p.Status)
		contract := fmt.Sprintf("%s salary; contract ends in %d", p.Salary, p.ExpirationYear)
		if info := strings.TrimSpace(p.ContractInfo); info != "" {
			contract += "; " + info
		}
		out[name+" contract"] = contract
		out[name+" acquisition"] = acquisitionWords(p.AcquiredVia)
	}
	for _, p := range seed.Picks {
		label := fmt.Sprintf("Season %d, round %d, %s pick", p.Season, p.Round, cmp.Or(names[p.Original], p.Original))
		if p.Slot > 0 {
			label += fmt.Sprintf(" (slot %d)", p.Slot)
		}
		out[label] = names[p.Owner] + pickPreviewWords(p, players)
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

func marketPreviewValues(raw RawCommand, book waiverState, names map[string]string, out map[string]string) {
	if raw.Args.Waiver != nil {
		out["Waiver priority"] = strings.Join(book.Order, ", ") + " — " + book.Source
		for _, w := range book.Windows {
			words := "Waiting for processing"
			if w.Processed {
				words = "Available as a free agent"
				if w.Winner != "" {
					words = "Awarded to " + cmp.Or(names[w.Winner], w.Winner)
				}
			}
			out["Waivers for "+cmp.Or(w.Player.Name, "player "+w.Player.ID)] = words
		}
	}
	if raw.Args.Bid != nil {
		for _, a := range book.Bids.Auctions {
			words := countWords(len(a.liveBids()), "bid", "bids") + " awaiting a decision"
			if a.Decided {
				words = "No award"
				if a.Winner != "" {
					words = "Awarded to " + cmp.Or(names[a.Winner], a.Winner)
				}
			}
			out["Bids for player "+a.Player] = words
		}
	}
}

// tradePreviewLabel names a trade by its teams, never by its ledger id. The trade being changed reads
// "This trade"; repeat offers between the same teams are numbered in the order they were made.
func tradePreviewLabel(raw RawCommand, t TradeState, names map[string]string, seen map[string]int) string {
	teams := make([]string, len(t.Parties))
	for i, f := range t.Parties {
		teams[i] = cmp.Or(names[f], f)
	}
	label := "Trade between " + joinWords(teams)
	if t.ID == raw.Args.Trade.ID || t.ID == "proposed."+raw.Request {
		label = "This trade, between " + joinWords(teams)
	}
	seen[label]++
	if seen[label] > 1 {
		label += fmt.Sprintf(" (offer %d)", seen[label])
	}
	return label
}

// tradePreviewWords says where a trade stands and counts only the review steps that have happened.
func tradePreviewWords(t TradeState) string {
	parts := []string{t.Words}
	for _, c := range []struct {
		n         int
		one, many string
	}{{len(t.Votes), "review vote", "review votes"}, {len(t.Recusals), "recusal", "recusals"},
		{len(t.Appeals), "appeal", "appeals"}, {len(t.Overrides), "override", "overrides"}} {
		if c.n > 0 {
			parts = append(parts, countWords(c.n, c.one, c.many))
		}
	}
	return strings.Join(parts, "; ")
}

func joinWords(words []string) string {
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

// seatPersonWords names a voter by the team they own, falling back to the shared person wording.
func seatPersonWords(who string, seats []PeopleSeat, records []Record) string {
	for _, s := range seats {
		if s.Owner != "" && SamePrincipal(records, s.Owner, who) {
			return s.Name + "'s owner"
		}
	}
	if words := personWords(who); words != who {
		return words
	}
	return "a league official"
}

func tradePreviewVotes(trade TradeState, label string, seats []PeopleSeat, records []Record, out map[string]string) {
	trails := []struct {
		label string
		votes []TradeVote
	}{
		{"Review", trade.Votes}, {"Recusal", trade.Recusals}, {"Appeal", trade.Appeals}, {"Override", trade.Overrides},
	}
	for _, trail := range trails {
		for _, vote := range trail.votes {
			key := label + " — " + trail.label + " by " + seatPersonWords(vote.Who, seats, records)
			for n := 2; out[key] != ""; n++ {
				key = fmt.Sprintf("%s — %s %d by %s", label, trail.label, n, seatPersonWords(vote.Who, seats, records))
			}
			out[key] = tradeVoteWords(vote.Decision) + "; " + vote.Reason
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
