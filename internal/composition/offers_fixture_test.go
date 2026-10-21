package composition

import (
	offersapp "github.com/secureprospective/TheWarRoom/internal/app/offers"
)

type RawBidInput = offersapp.RawBidInput
type BidPlan = offersapp.BidPlan
type BidBoard = offersapp.BidBoard
type RawOfferScope = offersapp.RawOfferScope
type OfferOutcomeView = offersapp.OfferOutcomeView
type OfferStretchView = offersapp.OfferStretchView
type OffersView = offersapp.OffersView
type PoolRow = offersapp.PoolRow
type Pool = offersapp.Pool
type RawPoolScope = offersapp.RawPoolScope
type offersFixture struct {
	*App
	*offersapp.API
}

func offersTestAPI(a *App) *offersFixture {
	return &offersFixture{App: a, API: offersapp.New(offersHost{State: a.core, app: a})}
}
func (a *offersFixture) TargetOffers(raw RawOfferScope) Reading[OffersView] {
	return wireReading(a.API.TargetOffers(raw), func(v OffersView) OffersView { return v })
}
func (a *offersFixture) TargetPool(raw RawPoolScope) Reading[Pool] {
	return wireReading(a.API.TargetPool(raw), func(v Pool) Pool { return v })
}
func (a *offersFixture) TargetBidPlans(mine string) Reading[BidBoard] {
	return wireReading(a.API.TargetBidPlans(mine), func(v BidBoard) BidBoard { return v })
}
func wireOffers(v offersapp.View) OffersView { return offersapp.WireOffers(v) }
