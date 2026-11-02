package mflsource

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
)

func annotateHistoricalPick(c *commissioner.RawCorrection, seed commissioner.RawNativeSnapshot, drafts [][]draftreports.Pick) {
	if c == nil || c.Pick == nil {
		return
	}
	p := c.Pick
	live := slices.IndexFunc(seed.Picks, func(held commissioner.NativePick) bool {
		return held.Season == p.Season && held.Round == p.Round && held.Slot == p.Slot && held.Original == p.Original
	})
	if live >= 0 && seed.Picks[live].Player == "" && !seed.Picks[live].Skipped {
		return
	}
	c.History = &commissioner.PickEvidence{Unverified: true}
	if p.Season != seed.Season {
		return
	}
	if p.Original == "" && live >= 0 {
		c.History = &commissioner.PickEvidence{Slot: p.Slot}
		return
	}
	c.History = originalPickEvidence(*p, seed, drafts)
}

func originalPickEvidence(p commissioner.CorrectionPick, seed commissioner.RawNativeSnapshot, drafts [][]draftreports.Pick) *commissioner.PickEvidence {
	unverified := &commissioner.PickEvidence{Unverified: true}
	name := ""
	for _, f := range seed.Franchises {
		if f.ID == p.Original {
			name = f.Name
		}
	}
	if name == "" {
		return unverified
	}
	slot := 0
	for _, draft := range drafts {
		for _, selection := range draft {
			if selection.Round != p.Round || !strings.HasPrefix(selection.Comments, "[Pick traded from "+name+".") {
				continue
			}
			if slot != 0 && slot != selection.Slot {
				return unverified
			}
			slot = selection.Slot
		}
	}
	if slot != 0 {
		return &commissioner.PickEvidence{Slot: slot}
	}
	return unverified
}

func annotateHistoricalGroups(groups [][]commissioner.RawCommand, seed commissioner.RawNativeSnapshot, drafts [][]draftreports.Pick) error {
	for _, group := range groups {
		for _, command := range group {
			at, err := command.EffectiveTime(seed.RosterAt)
			if err != nil {
				return fmt.Errorf("date historical pick: %w", err)
			}
			if at.Before(seed.RosterAt) {
				annotateHistoricalPick(command.Args.Correction, seed, drafts)
			}
		}
	}
	return nil
}
