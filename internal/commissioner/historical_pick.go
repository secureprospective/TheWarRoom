package commissioner

import (
	"fmt"
	"slices"
)

// PickEvidence links historical ownership to a held selection, or explicitly leaves it unverified.
type PickEvidence struct {
	Slot       int  `json:"slot,omitempty"`
	Unverified bool `json:"unverified,omitempty"`
}

func (p PickEvidence) Validate() error {
	if p.Unverified && p.Slot == 0 {
		return nil
	}
	if !p.Unverified && p.Slot > 0 && p.Slot <= 1000 {
		return nil
	}
	return fmt.Errorf("link the historical pick to a selection or mark it from import, unverified")
}

// HistoricalPick records ownership history, never an additional selectable or tradable asset.
type HistoricalPick struct {
	Pick     CorrectionPick `json:"pick"`
	Owner    string         `json:"owner"`
	Evidence PickEvidence   `json:"evidence"`
}

func correctHistoricalPick(seed *RawNativeSnapshot, c *RawCorrection) error {
	if c.Pick.Original != "" && !slices.ContainsFunc(seed.Franchises, func(f NativeFranchise) bool { return f.ID == c.Pick.Original }) {
		return fmt.Errorf("choose an original franchise in this league")
	}
	if !c.History.Unverified && !slices.ContainsFunc(seed.Picks, func(p NativePick) bool {
		return p.Season == c.Pick.Season && p.Round == c.Pick.Round && p.Slot == c.History.Slot && (p.Player != "" || p.Skipped)
	}) {
		return fmt.Errorf("link historical ownership to a recorded selection")
	}
	for i := range seed.PickHistory {
		p := &seed.PickHistory[i]
		if p.Pick == *c.Pick {
			p.Owner, p.Evidence = c.Owner, *c.History
			return nil
		}
	}
	seed.PickHistory = append(seed.PickHistory, HistoricalPick{Pick: *c.Pick, Owner: c.Owner, Evidence: *c.History})
	return nil
}

func historicalPickJobs(assets NativeAssets) []Job {
	if !assets.Seeded {
		return nil
	}
	var jobs []Job
	for _, p := range assets.Snapshot.PickHistory {
		if !p.Evidence.Unverified {
			continue
		}
		jobs = append(jobs, Job{ID: fmt.Sprintf("pick.%d.%d.%s.%d", p.Pick.Season, p.Pick.Round, p.Pick.Original, p.Pick.Slot),
			Name: "Pick from import — verify it", State: JobLive, Controls: []string{"CM-038"},
			Words: fmt.Sprintf("From import, unverified: %d round %d, original %s, slot %d. Check the recorded ownership and link its held draft selection with a pick correction.", p.Pick.Season, p.Pick.Round, p.Pick.Original, p.Pick.Slot)})
	}
	return jobs
}

func (r RawNativeSnapshot) validateHistoricalPicks(franchises map[string]bool) error {
	seen := map[CorrectionPick]bool{}
	for _, p := range r.PickHistory {
		c := RawCorrection{Pick: &p.Pick, Owner: p.Owner, History: &p.Evidence}
		if err := c.Validate(CorrectPick); err != nil {
			return err
		}
		if seen[p.Pick] || !franchises[p.Owner] {
			return fmt.Errorf("historical picks need unique identities and recorded owners")
		}
		seen[p.Pick] = true
		trial := r
		trial.PickHistory = nil
		if err := correctHistoricalPick(&trial, &c); err != nil {
			return err
		}
	}
	return nil
}
