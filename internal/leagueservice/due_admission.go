package leagueservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type dueAdmission struct {
	worker     *DueWorker
	credential func(commissioner.Access) error
	system     bool
}

func (a dueAdmission) Admit(ctx context.Context, access commissioner.Access) error {
	if a.system {
		if access.Mode != "execute" || !systemDueCommand(access.Principal, access.Operation) {
			return fmt.Errorf("the server worker may only process due bids, waivers and finished drafts")
		}
	} else if err := a.credential(access); err != nil {
		return fmt.Errorf("admit hosted credential: %w", err)
	}
	if a.system || timedCommand(access.Operation) {
		if err := a.worker.trusted(); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("stop due admission: %w", err)
	}
	return nil
}

func systemDueCommand(principal, command string) bool {
	switch principal {
	case commissioner.ReserveClockPrincipal:
		return command == commissioner.RosterReserve
	case "native.bid.clock":
		return command == commissioner.BidsProcess
	case "native.waiver.clock":
		return command == commissioner.WaiverProcess
	case commissioner.DraftClockPrincipal:
		return command == commissioner.DraftArchive
	default:
		return false
	}
}

func timedCommand(command string) bool {
	return commissioner.BidCommand(command) || commissioner.WaiverCommand(command) ||
		strings.HasPrefix(command, "draft.") || command == commissioner.LineupSet
}
