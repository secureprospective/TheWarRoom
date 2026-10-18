package commissioner

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

type TradeVote struct {
	Who      string    `json:"who"`
	Decision string    `json:"decision"`
	At       time.Time `json:"at"`
	Reason   string    `json:"reason,omitempty"`
}

type TradeState struct {
	VetoNumber   int                `json:"vetoNumber"`
	AppealedVeto int                `json:"appealedVetoNumber"`
	Details      []TradeAssetDetail `json:"assetDetails"`
	Previous     string             `json:"previous,omitempty"`
	ID           string             `json:"id"`
	Proposer     string             `json:"proposer"`
	Parties      []string           `json:"parties"`
	Assets       []TradeAsset       `json:"assets"`
	Accepted     []string           `json:"accepted"`
	Policy       TradePolicy        `json:"policy"`
	OfferedAt    time.Time          `json:"offeredAt"`
	ExpiresAt    time.Time          `json:"expiresAt"`
	Status       string             `json:"status"`
	Words        string             `json:"words"`
	Votes        []TradeVote        `json:"votes"`
	Recusals     []TradeVote        `json:"recusals"`
	Appeals      []TradeVote        `json:"appeals"`
	Overrides    []TradeVote        `json:"overrides"`
}

func (t TradeState) clone() TradeState {
	t.Details = slices.Clone(t.Details)
	for i, d := range t.Details {
		if d.Salary != nil {
			salary := *d.Salary
			t.Details[i].Salary = &salary
		}
	}
	t.Parties, t.Assets, t.Accepted = slices.Clone(t.Parties), slices.Clone(t.Assets), slices.Clone(t.Accepted)
	t.Votes, t.Recusals, t.Appeals, t.Overrides = slices.Clone(t.Votes), slices.Clone(t.Recusals), slices.Clone(t.Appeals), slices.Clone(t.Overrides)
	return t
}

func tradeByID(trades []TradeState, id string) (TradeState, error) {
	for _, t := range trades {
		if t.ID == id {
			return t.clone(), nil
		}
	}
	return TradeState{}, fmt.Errorf("choose a recorded trade")
}

func projectTrade(seed *RawNativeSnapshot, state *waiverState, r Record) error {
	if r.Command.Args.TradeOutcome == nil {
		return fmt.Errorf("a trade action has no recorded outcome")
	}
	t := r.Command.Args.TradeOutcome.clone()
	index := slices.IndexFunc(state.Trades, func(v TradeState) bool { return v.ID == t.ID })
	before := false
	if r.Command.Command == TradeOffer {
		if index >= 0 {
			return fmt.Errorf("this trade was already offered")
		}
		t.ID = r.Receipt.Entry
	} else {
		if index < 0 {
			return fmt.Errorf("reverse the later trade actions first")
		}
		if t.Previous != tradeDigest(state.Trades[index]) {
			return fmt.Errorf("undo the later trade actions first")
		}
		before = state.Trades[index].Status == "approved"
	}
	after := t.Status == "approved"
	if before != after {
		if err := moveTradeAssets(seed, t.Assets, before); err != nil {
			return err
		}
	}
	if index < 0 {
		state.Trades = append(state.Trades, t)
	} else {
		state.Trades[index] = t
	}
	return nil
}

// Ownership checks make compensation refuse to strand a subsequent cut, award or pick transfer.
func moveTradeAssets(seed *RawNativeSnapshot, assets []TradeAsset, reverse bool) error {
	for _, a := range assets {
		from, to := a.From, a.To
		if reverse {
			from, to = to, from
		}
		if a.Player != "" {
			i := slices.IndexFunc(seed.Players, func(p NativePlayer) bool { return p.ID == a.Player && p.Franchise == from })
			if i < 0 {
				return fmt.Errorf("a traded player no longer belongs to the sending team; undo the later asset change first")
			}
			seed.Players[i].Franchise = to
		} else {
			i := slices.IndexFunc(seed.Picks, func(p NativePick) bool {
				return p.Season == a.Season && p.Round == a.Round && p.Original == a.Original && p.Owner == from
			})
			if i < 0 || seed.Picks[i].Player != "" || seed.Picks[i].Skipped || seed.Picks[i].Slot != 0 || seed.Picks[i].Compensatory {
				return fmt.Errorf("a traded pick is no longer available; undo the later pick change first")
			}
			seed.Picks[i].Owner = to
		}
	}
	return nil
}

func checkTradeReplay(raw RawCommand, records []Record, now time.Time) error {
	if raw.Command != Revert {
		return nil
	}
	trial := append(slices.Clone(records), Record{Command: raw, Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: now}})
	_, _, err := nativeAssetsState(raw.League, trial)
	return err
}

func tradeDigest(t TradeState) string {
	body, err := json.Marshal(t)
	if err != nil {
		return "invalid"
	}
	return fmt.Sprintf("%x", sha256.Sum256(body))
}
