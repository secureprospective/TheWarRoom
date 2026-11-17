package leagueservice

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestPlayerReadingsValidateAdmitAndReportUnavailable(t *testing.T) {
	for _, read := range []struct {
		operation string
		call      func(*Service, RawPlayerReadScope) (json.RawMessage, error)
	}{
		{"player.usage.read", (*Service).TargetNativePlayerUsage},
		{"player.agearc.read", (*Service).TargetNativePlayerAgeArc},
	} {
		t.Run(read.operation, func(t *testing.T) {
			calls := 0
			service := NewHosted(admissionHost{}, admissionFunc(func(_ context.Context, access commissioner.Access) error {
				calls++
				if access.Operation != read.operation || access.League != "native.test" || access.Player != "0099" {
					return errors.New("wrong player reading scope")
				}
				return nil
			}))
			for _, id := range []string{"", "99", "x y", " 0099"} {
				if _, err := read.call(service, RawPlayerReadScope{League: "native.test", Player: id}); err == nil || calls != 0 {
					t.Fatalf("invalid player reached admission: %q %v", id, err)
				}
			}
			scope := RawPlayerReadScope{League: "native.test", Player: "0099"}
			if data, err := read.call(service, scope); err == nil || !strings.Contains(err.Error(), "hosted player reading is unavailable") || data != nil || calls != 1 {
				t.Fatalf("missing inputs must not become data: %s %v calls=%d", data, err, calls)
			}
			denied := errors.New("outsider refused")
			outsider := NewHosted(admissionHost{}, admissionFunc(func(context.Context, commissioner.Access) error { return denied }))
			if data, err := read.call(outsider, scope); !errors.Is(err, denied) || data != nil {
				t.Fatalf("outsider reached reading: %s %v", data, err)
			}
		})
	}
}
