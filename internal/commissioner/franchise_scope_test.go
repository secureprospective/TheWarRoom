package commissioner

import "testing"

func TestPlayerExceptionNamesBothFranchiseAndPlayer(t *testing.T) {
	scope := ExceptionScope{Franchise: "franchise.afc", Player: "0099"}
	correction := RawCorrection{Rule: "ir.eligible", Scope: &scope}
	if err := correction.Validate(RuleException); err != nil {
		t.Fatalf("valid player copy: %v", err)
	}
	if !exceptionScopeMatches(scope, scope) {
		t.Fatal("own copy did not match")
	}
	if exceptionScopeMatches(scope, ExceptionScope{Franchise: "franchise.nfc", Player: "0099"}) {
		t.Fatal("other copy matched")
	}
	if exceptionScopeMatches(scope, ExceptionScope{Franchise: "franchise.afc", Player: "0098"}) {
		t.Fatal("other player matched")
	}
	if !exceptionScopeMatches(ExceptionScope{Player: "0099"}, scope) {
		t.Fatal("legacy player scope did not match")
	}
}
