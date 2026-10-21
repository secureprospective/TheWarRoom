package pool

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

func TestAgeTextCountsBirthdays(t *testing.T) {
	t.Parallel()
	born := time.Date(1999, 3, 15, 0, 0, 0, 0, time.UTC).Unix()
	facts := normalize.PlayerFacts{Birthdate: born, HasBirthdate: true}
	for now, want := range map[time.Time]string{
		time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC):  "26",
		time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC):  "27",
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC): "27",
	} {
		if got := AgeText(facts, now); got != want {
			t.Errorf("age on %v = %s, want %s", now, got, want)
		}
	}
	if got := AgeText(normalize.PlayerFacts{}, time.Now()); got != "—" {
		t.Errorf("no birthdate = %s", got)
	}
}
