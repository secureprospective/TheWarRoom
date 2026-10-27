package offers

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestPersonalSnapshot(t *testing.T) {
	t.Parallel()
	snap := snapshot.Snapshot{Franchises: snapshot.Sourced[[]snapshot.Franchise]{Value: []snapshot.Franchise{{ID: "0001"}}}}
	for _, tc := range []struct {
		loaded     bool
		mine, want string
	}{{false, "0001", "personal: snapshot not loaded"}, {true, "0002", "personal: unknown franchise"}, {true, "0001", ""}} {
		got, err := PersonalSnapshot(snap, tc.loaded, tc.mine)
		if !reflect.DeepEqual(got, snap) {
			t.Fatal("snapshot changed")
		}
		if tc.want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.Error() != tc.want {
			t.Fatalf("%+v: %v", tc, err)
		}
	}
}
