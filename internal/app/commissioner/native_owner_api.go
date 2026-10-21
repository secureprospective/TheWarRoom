package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func (a *API) TargetNativeTradeAction(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if commissioner.OwnerTradeAction(raw.Command) == "" || raw.Args.TradeOutcome != nil {
		return commissioner.Receipt{}, fmt.Errorf("choose to offer, accept, decline or withdraw a trade")
	}
	//nolint:wrapcheck // Preserve the existing refusal text and error identity.
	return a.host.SubmitNativeTrade(raw)
}

// TargetNativeTrades reads the league's native trade desk; viewer is the seated owner's franchise.
func (a *API) TargetNativeTrades(league, viewer string) (commissioner.NativeTrades, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeTrades{}, fmt.Errorf("trade startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.NativeTrades{}, err
	}
	records, err := a.host.CommissionerRecords(a.host.CommissionerContext(), league)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.NativeTrades{}, err
	}
	reading, err := commissioner.NativeTradesReading(league, viewer, principal, records,
		a.host.CommissionerNow(), a.host.CommissionerZone())
	if err != nil {
		return commissioner.NativeTrades{}, fmt.Errorf("read trades: %w", err)
	}
	return reading, nil
}

func (a *API) TargetNativeWaiverAction(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if commissioner.OwnerWaiverAction(raw.Command) == "" || raw.Args.WaiverOutcome != nil {
		return commissioner.Receipt{}, fmt.Errorf("choose a native cut or waiver claim")
	}
	//nolint:wrapcheck // Preserve the existing refusal text and error identity.
	return a.host.SubmitNativeWaiver(raw)
}

func (a *API) TargetNativeWaivers(league, franchise string, at time.Time) (commissioner.NativeWaivers,
	error) {
	if at.IsZero() {
		return commissioner.NativeWaivers{}, fmt.Errorf("choose when to read waivers")
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeWaivers{}, fmt.Errorf("waivers startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.NativeWaivers{}, err
	}
	reading, err := a.host.NativeWaivers(a.host.CommissionerContext(), league, franchise, principal, at,
		a.host.CommissionerZone(), a.host.NativeFacts())
	if err != nil {
		return commissioner.NativeWaivers{}, fmt.Errorf("native waivers: %w", err)
	}
	return reading, nil
}

func (a *API) TargetNativeBidAction(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if commissioner.OwnerBidAction(raw.Command) == "" || raw.Args.BidOutcome != nil {
		return commissioner.Receipt{}, fmt.Errorf("choose to place, raise or withdraw a bid")
	}
	//nolint:wrapcheck // Preserve the existing refusal text and error identity.
	return a.host.SubmitNativeBid(raw)
}

func (a *API) TargetNativeBids(league, franchise string) (commissioner.NativeBids, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeBids{}, fmt.Errorf("bids startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.NativeBids{}, err
	}
	reading, err := a.host.NativeBids(a.host.CommissionerContext(), league, franchise, principal,
		a.host.CommissionerNow(), a.host.CommissionerZone(), a.host.NativeFacts())
	if err != nil {
		return commissioner.NativeBids{}, fmt.Errorf("native bids: %w", err)
	}
	return reading, nil
}
