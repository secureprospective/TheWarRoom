package commissioner

import (
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func (a *API) TargetCommissionerExecute(body string) (commissioner.Receipt, error) {
	raw, err := commissioner.Decode(body)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league change: %w", err)
	}
	//nolint:wrapcheck // The shared service supplies the user-facing context.
	return a.reads.Execute(a.host.CommissionerContext(), raw)
}

func (a *API) TargetCommissionerLedger(league string) (commissioner.Ledger, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerLedger(league)
}

func (a *API) TargetCommissionerLedgerFiltered(league, body string) (commissioner.Ledger, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerLedgerFiltered(league, body)
}

func (a *API) TargetCommissionerPeople(league string) (commissioner.People, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerPeople(league)
}

func (a *API) TargetCommissionerLeagues() ([]commissioner.NativeLeague, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerLeagues()
}

func (a *API) TargetCommissionerRules(league string) (commissioner.Rules, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerRules(league)
}

func (a *API) TargetCommissionerJobs(league string) (commissioner.Jobs, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerJobs(league)
}

// TargetCommissionerControls lists every commissioner control, grouped, and whether it is built yet.
func (a *API) TargetCommissionerControls() commissioner.Controls {
	// Desktop metadata is available offline; hosted callers use the admitted service read.
	return commissioner.ControlsReading()
}

// TargetCommissionerNotices reads the posted notices, league notes, release notes and the mail log. Every
// owner may read it.
func (a *API) TargetCommissionerNotices(league string) (commissioner.LeagueNotes, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerNotices(league)
}

// TargetCommissionerDigests writes a plain sentence for each change that took a hold or a typed reason.
func (a *API) TargetCommissionerDigests(league string) (commissioner.Digests, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerDigests(league)
}

// TargetCorrections reads only this app league's public, active corrections and exceptions.
func (a *API) TargetCorrections(league string) (commissioner.Corrections, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCorrections(league)
}

// TargetCommissionerMarket reads held targets under the permission of the requested act.
func (a *API) TargetCommissionerMarket(league, command string) (market.Market, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerMarket(league, command)
}

// TargetCommissionerActor supplies the session identity for acts that must be performed as oneself.
func (a *API) TargetCommissionerActor() (string, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerActor()
}

func (a *API) TargetCommissionerSchedule(league string) (commissioner.ManagedSchedule, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerSchedule(league)
}

// SeatHost links this computer's identity to the desktop's local sign-in, and backs the league up
// with that sign-in inside.
type SeatHost interface {
	CommissionerSeat() (accounts.SeatStatus, error)
	CommissionerSeatSignIn(accounts.RawCredentials) (accounts.SeatStatus, error)
	CommissionerBackupNow() (BackupResult, error)
}

// ErrBackupNeedsSignIn refuses a backup without a local sign-in: restored on another computer,
// such a backup would keep no one's league roles.
var ErrBackupNeedsSignIn = errors.New("create a sign-in first, so the backup keeps your league roles on another computer")

// BackupResult is a saved on-demand backup: the set's name, which a restore asks for, and a
// plain sentence to show.
type BackupResult struct {
	Set     string `json:"set"`
	Message string `json:"message"`
}

// TargetCommissionerSeat says whether a local sign-in exists and whether this computer uses it.
func (a *API) TargetCommissionerSeat() (accounts.SeatStatus, error) {
	return a.host.CommissionerSeat() //nolint:wrapcheck // the seat's errors are already the words shown
}

// TargetCommissionerSeatSignIn creates the local sign-in, or proves it on a restored copy, and
// links this computer so the league keeps every role its owner held. A refusal is shown as is.
func (a *API) TargetCommissionerSeatSignIn(username, password string) (accounts.SeatStatus, error) {
	//nolint:wrapcheck // the seat's errors are already the words shown
	return a.host.CommissionerSeatSignIn(accounts.RawCredentials{Username: username, Password: password})
}

// TargetBackupNow saves a matched backup of all the league data now. It is refused with
// ErrBackupNeedsSignIn until a local sign-in exists.
func (a *API) TargetBackupNow() (BackupResult, error) {
	return a.host.CommissionerBackupNow() //nolint:wrapcheck // the host's errors are already the words shown
}
