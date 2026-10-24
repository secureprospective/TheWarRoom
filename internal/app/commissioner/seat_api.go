package commissioner

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

// SeatHost links this computer's identity to the desktop's local sign-in.
type SeatHost interface {
	CommissionerSeat() (accounts.SeatStatus, error)
	CommissionerSeatSignIn(accounts.RawCredentials) (accounts.SeatStatus, error)
}

// TargetCommissionerSeat says whether a local sign-in exists and whether this computer uses it.
func (a *API) TargetCommissionerSeat() (accounts.SeatStatus, error) {
	status, err := a.host.CommissionerSeat()
	if err != nil {
		return accounts.SeatStatus{}, fmt.Errorf("league sign-in: %w", err)
	}
	return status, nil
}

// TargetCommissionerSeatSignIn creates the local sign-in, or proves it on a restored copy, and
// links this computer so the league keeps every role its owner held.
func (a *API) TargetCommissionerSeatSignIn(username, password string) (accounts.SeatStatus, error) {
	status, err := a.host.CommissionerSeatSignIn(accounts.RawCredentials{Username: username, Password: password})
	if err != nil {
		return accounts.SeatStatus{}, fmt.Errorf("league sign-in: %w", err)
	}
	return status, nil
}
