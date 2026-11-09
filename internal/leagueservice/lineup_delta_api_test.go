package leagueservice

import "testing"

func TestLineupDeltaScopeValidate(t *testing.T) {
	ok := RawLineupDeltaScope{League: "native.x", Franchise: "0001", Partner: "0002", Give: []string{"13604"}, Season: 2026, Week: 3}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid scope refused: %v", err)
	}
	for name, mut := range map[string]func(*RawLineupDeltaScope){
		"same franchise": func(r *RawLineupDeltaScope) { r.Partner = r.Franchise },
		"no players":     func(r *RawLineupDeltaScope) { r.Give = nil },
		"named twice":    func(r *RawLineupDeltaScope) { r.Get = []string{"13604"} },
		"bad id":         func(r *RawLineupDeltaScope) { r.Give = []string{"x y"} },
	} {
		r := ok
		mut(&r)
		if err := r.Validate(); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
}
