package commissioner

import (
	"fmt"
	"slices"
	"testing"
)

func TestChangedPreviewFieldsReadWeeksInNumberOrder(t *testing.T) {
	after := map[string]string{}
	for _, week := range []int{1, 10, 2, 13, 3} {
		after[fmt.Sprintf("Proposed schedule, week %d", week)] = "games"
	}
	var got []string
	for _, field := range ChangedPreviewFields(map[string]string{}, after) {
		got = append(got, field.Label)
	}
	want := []string{"Proposed schedule, week 1", "Proposed schedule, week 2", "Proposed schedule, week 3",
		"Proposed schedule, week 10", "Proposed schedule, week 13"}
	if !slices.Equal(got, want) {
		t.Fatalf("weeks out of order: %v", got)
	}
}
