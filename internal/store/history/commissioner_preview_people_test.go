package history

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func TestEveryBuiltPeopleCommandHasARealPreview(t *testing.T) {
	s, _, _ := newStore(t)
	raw := createCommand()
	proveStorePreview(t, s, raw, "creator")
	runScoreCommand(t, s, raw, "creator")
	cases := []struct {
		command, principal string
		args               commissioner.RawPeople
	}{
		{"commish.franchise.edit", "creator", commissioner.RawPeople{Franchise: "franchise.01", Name: "Falcons", Abbreviation: "FAL"}},
		{"commish.seat.replace", "creator", commissioner.RawPeople{Franchise: "franchise.01", Principal: "owner"}},
		{"commish.seat.remove", "creator", commissioner.RawPeople{Franchise: "franchise.01"}},
		{"commish.abilities.set", "creator", commissioner.RawPeople{Franchise: "franchise.01", Ability: "trades", On: false}},
		{"commish.visibility.set", "creator", commissioner.RawPeople{Report: "finances", Audience: "commish"}},
		{"commish.role.grant", "creator", commissioner.RawPeople{Principal: "judge", Role: governance.DOT}},
		{"commish.role.revoke", "creator", commissioner.RawPeople{Principal: "judge", Role: governance.DOT}},
		{"commish.seat.invite", "creator", commissioner.RawPeople{ID: "invite-one", Franchise: "franchise.01", Principal: "owner"}},
		{"commish.seat.invite.revoke", "creator", commissioner.RawPeople{ID: "invite-one"}},
		{"commish.seat.invite", "creator", commissioner.RawPeople{ID: "invite-two", Franchise: "franchise.01", Principal: "owner"}},
		{"commish.seat.invite.redeem", "owner", commissioner.RawPeople{ID: "invite-two", Principal: "owner"}},
	}
	for index, tc := range cases {
		raw := peopleCommand(tc.command, tc.args, int64(index+1))
		proveStorePreview(t, s, raw, tc.principal)
		runScoreCommand(t, s, raw, tc.principal)
	}
}

func previewSuccession(t *testing.T, f *successionFixture, principal, command string, args commissioner.RawPeople) {
	t.Helper()
	proveStorePreview(t, f.store, successionCommand(command, args, f.revision), principal)
	f.act(t, principal, command, args)
}

func TestEveryBuiltSuccessionCommandHasARealPreview(t *testing.T) {
	t.Run("inactivity", func(t *testing.T) {
		f := newSuccession(t, 2, true)
		previewSuccession(t, f, "creator", "commish.succession.inactivity.set", commissioner.RawPeople{InSeasonDays: 3, OffseasonDays: 2})
		previewSuccession(t, f, "creator", "commish.succession.activity", commissioner.RawPeople{Principal: "creator"})
		previewSuccession(t, f, "creator", "commish.succession.periods.set", commissioner.RawPeople{ElectionDays: 2, RunoffDays: 3})
		f.clock.now = f.clock.now.Add(3 * 24 * time.Hour)
		previewSuccession(t, f, "cc", "commish.succession.takeover", commissioner.RawPeople{Principal: "cc", Commissioner: "creator"})
	})
	t.Run("accept", func(t *testing.T) {
		f := newSuccession(t, 2, true)
		previewSuccession(t, f, "creator", "commish.succession.stepdown", commissioner.RawPeople{Principal: "creator"})
		previewSuccession(t, f, "cc", "commish.succession.accept", commissioner.RawPeople{Principal: "cc"})
	})
	t.Run("election", func(t *testing.T) {
		f := newSuccession(t, 2, true)
		previewSuccession(t, f, "creator", "commish.succession.stepdown", commissioner.RawPeople{Principal: "creator"})
		previewSuccession(t, f, "cc", "commish.succession.decline", commissioner.RawPeople{Principal: "cc"})
		previewSuccession(t, f, "judge", "commish.succession.election.open", commissioner.RawPeople{ID: "election", Principal: "judge"})
		castPreviewBallots(t, f)
		f.clock.now = f.clock.now.Add(7 * 24 * time.Hour)
		previewSuccession(t, f, "judge", "commish.vote.close", commissioner.RawPeople{ID: "election"})
		castPreviewBallots(t, f)
		f.clock.now = f.clock.now.Add(7 * 24 * time.Hour)
		previewSuccession(t, f, "cc", "commish.vote.tiebreak", commissioner.RawPeople{ID: "election", Principal: "cc", Choice: "owner1"})
		previewSuccession(t, f, "judge", "commish.vote.close", commissioner.RawPeople{ID: "election"})
	})
	t.Run("vote out", func(t *testing.T) {
		f := newSuccession(t, 2, true)
		previewSuccession(t, f, "judge", "commish.succession.voteout.open", commissioner.RawPeople{ID: "remove", Principal: "judge", Commissioner: "creator"})
	})
}

func castPreviewBallots(t *testing.T, f *successionFixture) {
	t.Helper()
	for _, owner := range []string{"owner0", "owner1"} {
		previewSuccession(t, f, owner, "commish.vote.cast", commissioner.RawPeople{ID: "election", Principal: owner, Choice: owner})
	}
}
