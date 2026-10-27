package commissioner

import (
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func (a *API) TargetNativeCapCheck(leagueID, viewerZone string) (commissioner.NativeCapCheck, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeCapCheck(leagueID, viewerZone)
}

func (a *API) TargetNativeRosterCheck(leagueID string) (commissioner.NativeRosterCheck, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeRosterCheck(leagueID)
}

func (a *API) TargetNativeReserveCheck(leagueID string) (commissioner.NativeReserveCheck, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeReserveCheck(leagueID)
}

// TargetNativeLineupCheck judges a lineup of one native franchise's active players; it records nothing.
func (a *API) TargetNativeLineupCheck(
	input commissioner.RawNativeLineup,
) (commissioner.NativeLineupCheck, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeLineupCheck(input)
}

func (a *API) TargetNativeHouseRules(leagueID string) (commissioner.NativeHouseRules, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeHouseRules(leagueID)
}

// TargetNativePlayerScore reads only held NFL evidence and the app league's recorded decisions.
func (a *API) TargetNativePlayerScore(league, player string, season,
	week int) (commissioner.NativePlayerScore, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativePlayerScore(league, player, season, week)
}

// TargetNativeContractQuote prices one contract action on one recorded native contract: allowed or
// refused with the plain reason, the new salary and years, the dead money by season and the cap effect.
// It reads the app league's ledger only and records nothing; the owner action that follows runs the same
// check.
func (a *API) TargetNativeContractQuote(raw market.RawContractQuote) (market.ContractQuote,
	error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeContractQuote(raw)
}

type RawNativeResultsScope leagueservice.RawNativeResultsScope

func (r RawNativeResultsScope) Validate() error {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return leagueservice.RawNativeResultsScope(r).Validate()
}

// TargetNativeResults scores one week of the app league's published schedule from its lineups and held
// NFL evidence. A game missing a score has no result yet.
func (a *API) TargetNativeResults(league string, season, week int) (commissioner.NativeResults, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeResults(league, season, week)
}

// TargetNativeStandings ranks a season's regular-season results under the league's tie-breakers, with any
// commissioner override beside the computed standings.
func (a *API) TargetNativeStandings(league string, season int) (commissioner.NativeStandings, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeStandings(league, season)
}

// TargetNativeScorePreview reads a score adjustment, in the form it would be sent, and warns when it
// would turn over a game's result. It records nothing.
func (a *API) TargetNativeScorePreview(body string) (commissioner.ScorePreview, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeScorePreview(body)
}

// TargetNativeBracket reads a season's playoffs: the seeds, each round's games, anything the commissioner set
// by hand, and the champion with the season's title once the final is decided.
func (a *API) TargetNativeBracket(league string, season int) (commissioner.NativeBracket, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeBracket(league, season)
}

// TargetStatQueue lists the statistics the NFL changed after the league counted them, newest first, with the
// decisions already made. Nothing in the queue moves a score until it is approved.
func (a *API) TargetStatQueue(league string, season int) (commissioner.StatQueue, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetStatQueue(league, season)
}
