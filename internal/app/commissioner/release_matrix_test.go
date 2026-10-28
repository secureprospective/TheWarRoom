package commissioner

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	domain "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

type releaseDenial struct{ calls int }

func (d *releaseDenial) Admit(context.Context, domain.Access) error {
	d.calls++
	return errors.New("release matrix credential revoked")
}

func TestReleaseMatrixCatalogReadAdmission(t *testing.T) {
	t.Parallel()
	app := previewApp(t)
	executeCommissionerTest(t, app, domain.RawCommand{League: "native.other", Command: domain.Create,
		Args:    domain.RawArgs{Name: "Second matrix league", Preset: domain.PresetEmpty, Season: 2026, FranchiseCount: 2},
		Request: "create.other", Reason: "Create a separate league", Effective: "now", Visibility: "league"})
	host := serviceParityHost{Host: app.host}
	otherBefore, err := app.history.Events(t.Context(), "native.other", domain.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	before, err := app.history.Events(t.Context(), "native.preview", domain.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range NativeCommands() {
		if !slices.Contains(command.Modes, catalog.Read) {
			continue
		}
		t.Run(command.ID, func(t *testing.T) {
			for _, league := range []string{"native.preview", "native.other"} {
				for _, revoked := range []bool{false, true} {
					gate := &releaseDenial{}
					var admission domain.CredentialAdmission
					if revoked {
						admission = gate
					}
					service := leagueservice.NewHosted(host, admission)
					method := reflect.ValueOf(service).MethodByName(command.Method)
					args := releaseReadInputs(t, command.Method, method.Type(), league)
					results := method.Call(args)
					failure := results[len(results)-1]
					if failure.IsNil() {
						t.Fatalf("%s revoked=%v: read admitted", league, revoked)
					}
					want := "Your session is unavailable"
					if revoked {
						want = "Release matrix credential revoked"
					}
					var refusal *domain.Refusal
					got := failure.Interface().(error)
					if !errors.As(got, &refusal) || refusal.Reason() != want {
						t.Fatalf("%s revoked=%v: did not reach credential admission: %v", league, revoked, got)
					}
					if revoked && gate.calls != 1 {
						t.Fatalf("admission calls=%d", gate.calls)
					}
				}
			}
		})
	}
	otherAfter, err := app.history.Events(t.Context(), "native.other", domain.Stream, 0)
	if err != nil || !reflect.DeepEqual(otherBefore, otherAfter) {
		t.Fatalf("denied reads changed other history: %v", err)
	}
	after, err := app.history.Events(t.Context(), "native.preview", domain.Stream, 0)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("denied reads changed history: %v", err)
	}
}

func releaseReadInputs(t *testing.T, name string, typ reflect.Type, league string) []reflect.Value {
	t.Helper()
	args := parityInputs(t, typ, true)
	if len(args) > 0 && args[0].Kind() == reflect.String {
		args[0].SetString(league)
	}
	switch name {
	case "TargetCommissionerLedgerFiltered":
		args[1].SetString("{}")
	case "TargetCommissionerMarket":
		args[1].SetString(domain.BidPlace)
	case "TargetNativeCapCheck":
		args[1].SetString("UTC")
	case "TargetNativePlayerScore":
		args[1].SetString("0099")
		args[2].SetInt(2026)
		args[3].SetInt(1)
	case "TargetNativeResults":
		args[1].SetInt(2026)
		args[2].SetInt(1)
	case "TargetNativeStandings", "TargetNativeBracket", "TargetStatQueue":
		args[1].SetInt(2026)
	case "TargetNativeLineup":
		args[1].SetString("franchise.01")
		args[2].SetInt(1)
	case "TargetNativeBids", "TargetNativeTrades", "TargetNativeWaivers":
		args[1].SetString("franchise.01")
	}
	if len(args) > 0 && args[0].Kind() == reflect.Struct {
		value := args[0]
		if f := value.FieldByName("League"); f.IsValid() {
			f.SetString(league)
		}
		for _, field := range []string{"Franchise", "ViewerZone", "LocalZone", "Player", "Action"} {
			f := value.FieldByName(field)
			if !f.IsValid() {
				continue
			}
			f.SetString(map[string]string{"Franchise": "franchise.01", "ViewerZone": "UTC", "LocalZone": "UTC", "Player": "0099", "Action": "lineup"}[field])
		}
		if name == "TargetNativeContractQuote" {
			value.FieldByName("Action").SetString("tag")
		}
		if f := value.FieldByName("At"); f.IsValid() {
			f.Set(reflect.ValueOf(time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)))
		}
		if f := value.FieldByName("Season"); f.IsValid() {
			f.SetInt(2026)
		}
	}
	return args
}

func TestReleaseMatrixCatalogDelegationCeiling(t *testing.T) {
	t.Parallel()
	forbidden := []string{"commish.league.create", "commish.correction.revert", "native.cap.check", "native.reserve.check", "native.roster.check", "lineup.check", "commish.seat.caretaker", "commish.seat.invite", "commish.seat.invite.revoke", "commish.seat.invite.redeem", "commish.seat.remove", "commish.seat.replace", "commish.abilities.set", "commish.visibility.set", "commish.role.grant", "commish.role.revoke", "commish.deputy.order", "commish.succession.stepdown", "commish.succession.accept", "commish.succession.decline", "commish.succession.inactivity.set", "commish.succession.activity", "commish.succession.takeover", "commish.succession.voteout.open", "commish.succession.election.open", "commish.succession.periods.set", "commish.vote.cast", "commish.vote.close", "commish.vote.tiebreak"}
	reviewed := strings.Fields(`
bid.place
bid.raise
bid.withdraw
bids.award.override
bids.process
bids.read
bracket.advance
bracket.build
bracket.edit
bracket.preset.save
bracket.read
bracket.seed
calendar.read
calendar.window.move
calendar.window.remove
calendar.window.set
commish.abilities.set
commish.actor.read
commish.calendar.exception
commish.cap.adjust
commish.cap.relief
commish.contract.edit
commish.controls.read
commish.correction.revert
commish.corrections.read
commish.deputy.order
commish.digests.read
commish.franchise.edit
commish.jobs.read
commish.league.create
commish.league.seed_held
commish.leagues.read
commish.ledger.filtered.read
commish.ledger.read
commish.lineup.correct
commish.lineup.set
commish.market.read
commish.notice.publish
commish.notices.read
commish.people.read
commish.player.position
commish.player.position.clear
commish.reserve.exception
commish.role.grant
commish.role.revoke
commish.rules.read
commish.rules.set
commish.schedule.read
commish.score.franchise
commish.score.player
commish.seat.caretaker
commish.seat.invite
commish.seat.invite.redeem
commish.seat.invite.revoke
commish.seat.remove
commish.seat.replace
commish.standings.override
commish.standings.override.clear
commish.standings.record.adjust
commish.stat.correct
commish.stat.queue.approve
commish.stat.queue.reject
commish.succession.accept
commish.succession.activity
commish.succession.decline
commish.succession.election.open
commish.succession.inactivity.set
commish.succession.periods.set
commish.succession.stepdown
commish.succession.takeover
commish.succession.voteout.open
commish.visibility.set
commish.vote.cast
commish.vote.close
commish.vote.tiebreak
commish.whatsnew.post
contract.action.set
contract.quote
correct.pick
correct.provenance
correct.roster
correct.transaction
draft.archive
draft.assign
draft.change
draft.insert
draft.order.auto
draft.order.pattern
draft.order.set
draft.order.tiebreak
draft.pick
draft.read
draft.revert
draft.skip
draft.start
draft.startup.open
draft.timer
history.link
lineup.check
lineup.read
native.assets.read
native.cap.check
native.directory.read
native.reserve.check
native.roster.check
native.rules.read
results.read
roster.cut
roster.reserve
rule.exception
schedule.edit
schedule.franchise.read
schedule.generate
schedule.publish
schedule.read
score.player.read
oline.auction.close
season.rollover
standings.read
stats.queue.read
title.set
trade.accept
trade.appeal
trade.appeal.rule
trade.decline
trade.offer
trade.override
trade.recuse
trade.review
trade.withdraw
trades.read
waivers.award.override
waivers.claim
waivers.order.auto
waivers.order.set
waivers.process
waivers.read
week.read
`)
	commands := NativeCommands()
	if len(commands) != len(reviewed) {
		t.Fatal("catalog changed; review delegation ceilings and extend the release matrix inventory")
	}
	for _, command := range NativeCommands() {
		t.Run(command.ID, func(t *testing.T) {
			if !slices.Contains(reviewed, command.ID) {
				t.Fatal("command lacks a reviewed delegation ceiling")
			}
			scope := accounts.RawTokenScope{League: "native.preview", Franchise: "franchise.01", Commands: []string{command.ID}, Modes: []string{"preview"}}
			if got := scope.Validate() == nil; got == slices.Contains(forbidden, command.ID) {
				t.Fatalf("delegation ceiling changed: accepted=%v; review release matrix", got)
			}
		})
	}
}
