package pool

import "testing"

func TestCompareKeepsMissingLastInBothDirections(t *testing.T) {
	t.Parallel()
	value := 1.0
	held := Order{ID: "0042", Name: "Z", Engine: &value, Market: [4]*float64{&value}}
	missing := Order{ID: "0043", Name: "A"}
	for _, key := range []string{"engine", "projected"} {
		for _, desc := range []bool{false, true} {
			if Compare(held, missing, key, desc) >= 0 || Compare(missing, held, key, desc) <= 0 {
				t.Fatalf("missing first for %s desc=%v", key, desc)
			}
		}
	}
	if Compare(Order{ID: "0042", Name: "Same"}, Order{ID: "0043", Name: "Same"}, "name", false) >= 0 {
		t.Fatal("lost ID tie break")
	}
}
