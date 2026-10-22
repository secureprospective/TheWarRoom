package core

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func (s *State) ExecuteCommissioner(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer) (commissioner.Receipt, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.ExecuteCommissioner(ctx, raw, principal, auth)
}
func (s *State) ExecuteCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, facts nativecheck.Facts,
	zone *time.Location) (commissioner.Receipt, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.ExecuteCommissionerWithFacts(ctx, raw, principal, auth, facts, zone)
}
func (s *State) PreviewCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, catalogue commissioner.PreviewCatalogue,
	playerFacts nativecheck.Facts, zone *time.Location) (commissioner.Preview, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.PreviewCommissionerWithFacts(ctx, raw, principal, auth, catalogue, playerFacts, zone)
}
func (s *State) CommissionerLedgerFiltered(ctx context.Context, league, principal string,
	auth commissioner.Authorizer, filter commissioner.RawLedgerFilter) (commissioner.Ledger, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.CommissionerLedgerFiltered(ctx, league, principal, auth, filter)
}
func (s *State) CommissionerPeople(ctx context.Context, league, principal string) (commissioner.People,
	error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.CommissionerPeople(ctx, league, principal)
}
func (s *State) NativeLeagues(ctx context.Context) ([]commissioner.NativeLeague, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeLeagues(ctx)
}
func (s *State) CommissionerRules(ctx context.Context, league string) (commissioner.Rules, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.CommissionerRules(ctx, league)
}
func (s *State) CommissionerJobs(ctx context.Context, league string) (commissioner.Jobs, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.CommissionerJobs(ctx, league)
}
func (s *State) CommissionerNotes(ctx context.Context, league string) (commissioner.LeagueNotes, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.CommissionerNotes(ctx, league)
}
func (s *State) CommissionerDigests(ctx context.Context, league, principal string,
	auth commissioner.Authorizer) (commissioner.Digests, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.CommissionerDigests(ctx, league, principal, auth)
}
func (s *State) Corrections(ctx context.Context, league string) (commissioner.Corrections, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.Corrections(ctx, league)
}
func (s *State) Events(ctx context.Context, league, stream string, after int64) ([]history.LeagueEvent,
	error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.Events(ctx, league, stream, after)
}
func (s *State) NativeDraft(ctx context.Context, scope commissioner.RawNativeDraftScope,
	local *time.Location, facts nativecheck.Facts) (commissioner.NativeDraft, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeDraft(ctx, scope, local, facts)
}
func (s *State) NativeLineup(ctx context.Context, league, franchise, principal string, week int,
	at time.Time) (commissioner.NativeLineup, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeLineup(ctx, league, franchise, principal, week, at)
}
func (s *State) NativePlayerScore(ctx context.Context, league string,
	scope source.RawNativeScope) (commissioner.NativePlayerScore, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativePlayerScore(ctx, league, scope)
}
func (s *State) NativeResults(ctx context.Context, league string, season,
	week int) (commissioner.NativeResults, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeResults(ctx, league, season, week)
}
func (s *State) NativeStandings(ctx context.Context, league string,
	season int) (commissioner.NativeStandings, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeStandings(ctx, league, season)
}
func (s *State) NativeScorePreview(ctx context.Context,
	raw commissioner.RawCommand) (commissioner.ScorePreview, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeScorePreview(ctx, raw)
}
func (s *State) NativeBracket(ctx context.Context, league string,
	season int) (commissioner.NativeBracket, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeBracket(ctx, league, season)
}
func (s *State) StatQueue(ctx context.Context, league string, season int) (commissioner.StatQueue, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.StatQueue(ctx, league, season)
}
func (s *State) NativeCapCheck(ctx context.Context,
	scope commissioner.RawNativeCapScope) (commissioner.NativeCapCheck, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeCapCheck(ctx, scope)
}
func (s *State) NativeRosterCheck(ctx context.Context, league string,
	facts nativecheck.Facts) (commissioner.NativeRosterCheck, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeRosterCheck(ctx, league, facts)
}
func (s *State) NativeReserveCheck(ctx context.Context, league string,
	facts nativecheck.Facts) (commissioner.NativeReserveCheck, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeReserveCheck(ctx, league, facts)
}
func (s *State) NativeLineupCheck(ctx context.Context, in commissioner.RawNativeLineup,
	facts nativecheck.Facts) (commissioner.NativeLineupCheck, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeLineupCheck(ctx, in, facts)
}
func (s *State) NativeHouseRules(ctx context.Context, league string) (commissioner.NativeHouseRules, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeHouseRules(ctx, league)
}
func (s *State) NativeDirectory(ctx context.Context, league string) (commissioner.NativeDirectory, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeDirectory(ctx, league)
}
func (s *State) NativeAssets(ctx context.Context, league string) (commissioner.NativeAssets, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeAssets(ctx, league)
}
func (s *State) NativeWaivers(ctx context.Context, league, franchise, principal string, at time.Time,
	zone *time.Location, facts nativecheck.Facts) (commissioner.NativeWaivers, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeWaivers(ctx, league, franchise, principal, at, zone, facts)
}
func (s *State) NativeBids(ctx context.Context, league, franchise, principal string, at time.Time,
	zone *time.Location, facts nativecheck.Facts) (commissioner.NativeBids, error) {
	//nolint:wrapcheck // Preserve the original store error for the API wording.
	return s.History.NativeBids(ctx, league, franchise, principal, at, zone, facts)
}
