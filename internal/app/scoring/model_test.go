package scoring

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestFreeAgentIDsPreserveScopeAndNil(t *testing.T) {
	t.Parallel()
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	ids := []playerid.PlayerID{id, id}
	got := FreeAgentIDs(ids)
	if !reflect.DeepEqual(got, []string{"0042", "0042"}) || FreeAgentIDs(nil) != nil || FreeAgentIDs([]playerid.PlayerID{}) != nil {
		t.Fatalf("scope: %v", got)
	}
	got[0] = "changed"
	if ids[0].String() != "0042" {
		t.Fatal("model input changed the held IDs")
	}
}
