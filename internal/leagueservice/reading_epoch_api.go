package leagueservice

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// AdmitRead runs the admission a league reading of operation makes, without reading. A server that
// answers a reading from its reply cache calls it first, so a cached answer is never an extra grant.
func (a *Service) AdmitRead(league, operation, franchise string) error {
	if err := a.host.Ready(); err != nil {
		return fmt.Errorf("league reading startup: %w", err)
	}
	return a.admitRead(league, operation, franchise)
}

// ReadingEpoch is the league's reading epoch at this host's clock (commissioner.ReadingEpoch) and when
// it ends; evidence is the football kickoffs the host reads against. ok is false when the league's
// readings cannot be shared.
func (a *Service) ReadingEpoch(league string, evidence []time.Time) (epoch int, next time.Time, ok bool, err error) {
	records, err := a.host.CommissionerRecords(a.requestContext(a.host.CommissionerContext()), league)
	if err != nil {
		return 0, time.Time{}, false, fmt.Errorf("reading epoch: %w", err)
	}
	epoch, next, ok = commissioner.ReadingEpoch(records, a.host.CommissionerNow(), evidence)
	return epoch, next, ok, nil
}
