package commissioner

import "testing"

func TestMirrorIndependentPlayerCopies(t *testing.T) {
	t.Skip("mode1 M2: same player has different contracts and statuses on both sides")
}

func TestMirrorIndependentMarket(t *testing.T) {
	t.Skip("mode1 M4: bids and waiver clocks are independent on each side")
}

func TestMirrorChangesStayOnTheirSide(t *testing.T) {
	t.Skip("mode1 M6: corrections and drafts leave the other side byte-identical")
}

func TestMirrorSaveAndRestore(t *testing.T) {
	t.Skip("mode1 M7: two saves and a restore preserve both player copies")
}

func TestMirrorScheduleAndFinal(t *testing.T) {
	t.Skip("mode1 M2: cross-side games and one final work")
}

func TestMirrorSinglePoolUnchanged(t *testing.T) {
	t.Skip("mode1 M3: single-pool leagues retain their existing behavior")
}
