package now

import "testing"

func TestKnownFranchiseMembershipNotDisplayName(t *testing.T) {
	t.Parallel()
	if KnownFranchise(nil, "0001") || KnownFranchise(map[string]string{"0001": "Name"}, "0002") || !KnownFranchise(map[string]string{"0001": ""}, "0001") {
		t.Fatal("membership must use the key, not name availability")
	}
}
