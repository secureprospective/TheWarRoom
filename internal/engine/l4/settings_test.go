package l4

import (
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// A knob is a setting only where the shipped value is on, and every shipped value is in range.
func TestKnobsCoverTheOnMechanics(t *testing.T) {
	offAtQB := []string{"l4.ras.steepness", "l4.ras.cap", "l4.ras.weight", "l4.breakout.athletic_lift"}
	onAtK := []string{"l4.film.steepness", "l4.film.cap"}
	qb, k := Defaults()[domain.PosQB], Defaults()[domain.PosK]
	for _, knob := range Knobs() {
		if knob.AppliesTo(qb) == slices.Contains(offAtQB, knob.Key) {
			t.Errorf("QB: %s applies = %v", knob.Key, knob.AppliesTo(qb))
		}
		if knob.AppliesTo(k) != slices.Contains(onAtK, knob.Key) {
			t.Errorf("K: %s applies = %v", knob.Key, knob.AppliesTo(k))
		}
		for pos, s := range Defaults() {
			if v := *knob.Field(&s); knob.AppliesTo(s) && (v < knob.Min || v > knob.Max) {
				t.Errorf("%s at %s: %v outside [%v, %v]", knob.Key, pos, v, knob.Min, knob.Max)
			}
		}
	}
}
