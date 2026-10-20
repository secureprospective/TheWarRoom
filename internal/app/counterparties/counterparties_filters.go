package counterparties

import (
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func (svc Service) filteredPostures(cells []PostureCell) {
	for i := range cells {
		cells[i].Tone = ""
		cells[i].Title = cells[i].Label + " · share of the filtered trades"
	}
}
func (svc Service) counterFilterWords(s lab.TeamScope, words *CounterWords) string {
	if len(s.Postures) == 0 {
		return ""
	}
	labels := []string{}
	for _, id := range s.Postures {
		label := strings.ToLower(svc.postureLabel(id))
		if id == "swap" {
			label = "were like for like"
		}
		labels = append(labels, label)
	}
	words.Legend = "Posture shares describe the selected trades, not a finding against chance. Red ranges are losses beyond chance; green ranges are gains beyond chance."
	return "Showing only trades that " + strings.Join(labels, " or ") + "."
}
func (svc Service) tierExcluded(s lab.TeamScope, tier string, sides, unfiltered []lab.TeamSide) bool {
	if len(s.Tiers) > 0 && !slices.Contains(s.Tiers, tier) {
		return true
	}
	return len(svc.sidesOf(sides, tier)) == 0 && len(svc.sidesOf(unfiltered, tier)) > 0 &&
		(len(s.Windows) > 0 || len(s.Postures) > 0 || s.Team != "")
}
func (svc Service) applyCounterFilters(v *Counterparties, s lab.TeamScope, sides, league []lab.TeamSide) {
	if line := svc.counterFilterWords(s, &v.Words); line != "" {
		v.Summary += " " + line
		v.TrendSay = []string{}
		for i := range v.Cards {
			svc.filteredPostures(v.Cards[i].Posture)
		}
		for i := range v.Rows {
			v.Rows[i].ForNow.Tone, v.Rows[i].ForLater.Tone = "", ""
		}
		for i := range v.Trends.Tiers {
			v.Trends.Tiers[i].Shift = CounterShifts{}
			for j := range v.Trends.Tiers[i].Rows {
				row := &v.Trends.Tiers[i].Rows[j]
				row.Now.Tone, row.Later.Tone = "", ""
			}
		}
	}
	for i := range v.Cards {
		card := &v.Cards[i]
		card.Excluded = svc.tierExcluded(s, card.ID, sides, league)
		if card.Excluded {
			card.Thin = false
		}
	}
	for i := range v.Pairs {
		pair := &v.Pairs[i]
		if pair.Excluded = svc.tierExcluded(s, pair.tier, sides, league); pair.Excluded {
			pair.Trades = "Not in the current filter"
			pair.Title = svc.tierLabel(pair.tier) + " · not in the current filter"
		}
	}
	v.Trends.Tiers = slices.DeleteFunc(v.Trends.Tiers, func(t CounterTrend) bool {
		return svc.tierExcluded(s, t.Tier, sides, league)
	})
}
func (svc Service) teamExcluded(s lab.TeamScope, sides, unfiltered []lab.TeamSide) bool {
	return len(sides) == 0 && len(unfiltered) > 0 &&
		(len(s.Tiers) > 0 || len(s.Windows) > 0 || len(s.Postures) > 0)
}
func (svc Service) applyTeamFilters(v *TeamCounterpart, s lab.TeamScope, sides, league []lab.TeamSide) {
	if line := svc.counterFilterWords(s, &v.Words); line != "" {
		v.AsOf += " " + line
		svc.filteredPostures(v.Posture)
		for id, share := range v.Behaviour.Posture {
			share.Tone = ""
			v.Behaviour.Posture[id] = share
		}
	}
	if len(s.Windows) > 0 {
		for id, share := range v.Behaviour.Windows {
			share.Tone = ""
			v.Behaviour.Windows[id] = share
		}
	}
	for i := range v.AsTier {
		tier := &v.AsTier[i]
		tier.Excluded = svc.tierExcluded(s, tier.Tier, sides, league)
		if tier.Excluded {
			tier.Text = "Not in the current filter"
		}
	}
}

func (svc Service) emptyTeamReading(r report.Reading[TeamCounterpart], v TeamCounterpart, s lab.TeamScope, league []lab.TeamSide) report.Reading[TeamCounterpart] {
	reason := "No trade sides for this team in the seasons and filters in view"
	v.Empty = true
	v.Note = "This team shows no valued trades in view. Widen the seasons, time of year or what the trades did."
	v.PartnersSay = "No partners in view."
	r.Status, r.AbsenceReason = report.ReadingThin, &reason
	unfiltered := lab.FilterSides(league, func(side lab.TeamSide) bool { return side.Fid == v.Fid })
	if svc.teamExcluded(s, nil, unfiltered) {
		r.Status, r.AbsenceReason = report.ReadingOK, nil
		v.Note = "Not in the current filter"
	}
	r.Value = v
	return r
}
