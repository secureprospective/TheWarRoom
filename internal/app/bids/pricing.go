package bids

import (
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

func HeldCutCharge(p state.MirrorPlayer, season int) (domain.Money, bool) {
	if p.ContractYear < season {
		return 0, false
	}
	cells := []state.LedgerCell{}
	for year := season; year <= p.ContractYear; year++ {
		cells = append(cells, state.LedgerCell{Year: year, Salary: p.Salary})
	}
	return transactions.CutCharge(cells, season, false), true
}
func PointsText(salary domain.Money, years int) string {
	points, err := bid.Score(salary, years)
	if err != nil {
		return "Not scored"
	}
	return points.String()
}

func PostText(line string, f envelope.BidFacts) string {
	return bid.Post{Conditional: f.ConditionalName, Player: line, Salary: f.Salary, Years: f.Years, Points: f.Points, Snipe: f.Snipe}.Text()
}

func Active(s envelope.State) bool {
	return slices.Contains([]envelope.State{envelope.Ready, envelope.HandedOff, envelope.NotYetDone,
		envelope.NotVerified, envelope.BidPending}, s)
}
