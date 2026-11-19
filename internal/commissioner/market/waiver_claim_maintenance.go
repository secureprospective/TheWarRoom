package market

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func (x *waiverExecution) maintainClaim(raw *commissioner.RawCommand) error {
	switch raw.Command {
	case commissioner.WaiverClaimOrderSet:
		return x.setClaimOrder(raw)
	case commissioner.WaiverClaimCancel:
		return x.cancelClaim(raw)
	default:
		return fmt.Errorf("unsupported claim maintenance command %q", raw.Command)
	}
}

func (x *waiverExecution) setClaimOrder(raw *commissioner.RawCommand) error {
	args := raw.Args.Waiver
	pending := x.state.PendingClaimIDsFor(args.Franchise, x.principal, x.records)
	if len(pending) == 0 {
		return fmt.Errorf("you have no pending waiver claims to reorder")
	}
	if len(args.Order) != len(pending) {
		return fmt.Errorf("list every pending waiver claim exactly once")
	}
	allowed := make(map[string]bool, len(pending))
	for _, id := range pending {
		allowed[id] = true
	}
	seen := make(map[string]bool, len(args.Order))
	for _, id := range args.Order {
		window, claim, err := x.state.Claim(id)
		if err != nil {
			return fmt.Errorf("choose one of your pending waiver claims")
		}
		if claim.Franchise != args.Franchise || !commissioner.SamePrincipal(x.records, claim.Principal, x.principal) {
			return fmt.Errorf("you may reorder only your own waiver claims")
		}
		if window.Processed {
			return fmt.Errorf("the waiver run has already processed this claim")
		}
		if !allowed[id] || seen[id] {
			return fmt.Errorf("list every pending waiver claim exactly once")
		}
		seen[id] = true
	}
	if len(seen) != len(allowed) {
		return fmt.Errorf("list every pending waiver claim exactly once")
	}
	return nil
}

func (x *waiverExecution) cancelClaim(raw *commissioner.RawCommand) error {
	args := raw.Args.Waiver
	window, claim, err := x.state.Claim(args.Claim)
	if err != nil {
		return fmt.Errorf("find waiver claim: %w", err)
	}
	if claim.Franchise != args.Franchise || !commissioner.SamePrincipal(x.records, claim.Principal, x.principal) {
		return fmt.Errorf("you may cancel only your own waiver claims")
	}
	if window.Processed {
		return fmt.Errorf("the waiver run has already processed this claim")
	}
	return nil
}
