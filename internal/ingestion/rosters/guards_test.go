package rosters

import (
	"context"
	"testing"
)

func TestFlatten_MalformedDuplicateFailsLoud(t *testing.T) {
	env := rostersEnvelope{}
	env.Rosters.Franchise = []franchiseBlock{{ID: "0001", Player: []playerBlock{
		{ID: "0835", Salary: "7", Status: "TAXI_SQUAD"},
		{ID: "0835", Salary: "invalid", Status: "ROSTER"},
	}}}
	got, err := flatten(context.Background(), env)
	if err == nil || got != nil {
		t.Fatalf("malformed duplicate must fail without partial rows: got %+v, err %v", got, err)
	}
}
