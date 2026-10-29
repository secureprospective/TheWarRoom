package planwatch

import (
	"errors"
	"testing"
)

func TestTeamAuthorizationReadinessIdentityAndRefusals(t *testing.T) {
	for _, tc := range []struct{ mine, acting, want string }{
		{"", "0001", "Connect your MFL account to act for your team."},
		{"0001", "0002", "You can only act for your own team."},
		{"0001", "0001", ""},
	} {
		ready := false
		err := AuthorizeTeam(tc.acting, func() error { ready = true; return nil }, func(run func(string) error) error {
			if !ready {
				t.Fatal("identity before ready")
			}
			return run(tc.mine)
		})
		if tc.want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.Error() != tc.want {
			t.Fatal(err)
		}
	}
	err := AuthorizeTeam("0001", func() error { return errors.New("offline") }, func(func(string) error) error { t.Fatal("captured identity before ready"); return nil })
	if err == nil || err.Error() != "team authorization: startup: offline" {
		t.Fatal(err)
	}
}
