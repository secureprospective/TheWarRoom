package market

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func (x *waiverExecution) reserveAction(raw commissioner.RawCommand) leaguerules.Action {
	if raw.Command != commissioner.RosterReserve {
		return commissioner.OwnerWaiverAction(raw.Command)
	}
	if raw.Args.Waiver.Bucket == "taxi" {
		return "taxi"
	}
	if raw.Args.Waiver.Bucket == "active" {
		for _, p := range x.assets.Snapshot.Players {
			if p.ID == raw.Args.Waiver.Player && p.Franchise == raw.Args.Waiver.Franchise && p.Status == domain.RosterTaxi {
				return "taxi"
			}
		}
	}
	return "ir"
}

func (x *waiverExecution) reserve(raw *commissioner.RawCommand) error {
	args := raw.Args.Waiver
	for _, p := range x.assets.Snapshot.Players {
		if p.ID != args.Player || p.Franchise != args.Franchise {
			continue
		}
		if p.Pending {
			return fmt.Errorf("settle this player's pending contract first")
		}
		target := reserveStatus(args.Bucket)
		if target == p.Status {
			return fmt.Errorf("the player is already on that squad")
		}

		used := 0
		for _, other := range x.assets.Snapshot.Players {
			if other.Franchise == p.Franchise && other.Status == target {
				used++
			}
		}
		exceptions := commissioner.ScopedDecisionExceptions(x.records, commissioner.ExceptionScope{Franchise: p.Franchise, Player: p.ID}, x.now)
		entry := nativecheck.Entry{Allowed: true}
		switch target {
		case domain.RosterIR:
			entry = nativecheck.CanEnterIR(x.rules.IR, x.facts[p.ID], used, exceptions...)
		case domain.RosterTaxi:
			entry = nativecheck.CanEnterTaxi(x.rules.Taxi, x.facts[p.ID], x.rules.Season, used, exceptions...)
		case domain.RosterActive:
		}
		if !entry.Allowed {
			return fmt.Errorf("%s", entry.Words)
		}
		raw.Args.WaiverOutcome = &commissioner.WaiverOutcome{Player: &p, Season: x.rules.Season}
		return nil
	}
	return fmt.Errorf("choose a player on your recorded roster")
}

func reserveStatus(bucket string) domain.RosterStatus { return commissioner.ReserveStatus(bucket) }
