package commissioner

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// TradeAssetDetail preserves the names and full contract offered, even if the player later leaves a roster.
type TradeAssetDetail struct {
	From     string        `json:"fromTeam"`
	To       string        `json:"toTeam"`
	Label    string        `json:"label"`
	Contract string        `json:"contract,omitempty"`
	Salary   *domain.Money `json:"yearlySalaryCents,omitempty"`
	Ends     int           `json:"contractEndsInSeason,omitempty"`
}

func (x tradeExecution) describeAssets(assets []TradeAsset) []TradeAssetDetail {
	teams := map[string]string{}
	for _, f := range x.seed.Franchises {
		teams[f.ID] = f.Name
	}
	details := make([]TradeAssetDetail, 0, len(assets))
	for _, a := range assets {
		d := TradeAssetDetail{From: teams[a.From], To: teams[a.To]}
		if a.Player == "" {
			d.Label = fmt.Sprintf("%d round %d pick, originally %s", a.Season, a.Round, teams[a.Original])
		} else {
			for _, p := range x.seed.Players {
				if p.ID != a.Player {
					continue
				}
				d.Label = p.Name
				if d.Label == "" {
					d.Label = x.facts[p.ID].Name
				}
				if d.Label == "" {
					d.Label = "Player " + p.ID
				}
				salary := p.Salary
				d.Salary, d.Ends = &salary, p.ExpirationYear
				d.Contract = nativecheck.ContractWords(p.Salary, max(0, p.ExpirationYear-x.rules.Season+1))
			}
		}
		details = append(details, d)
	}
	return details
}
