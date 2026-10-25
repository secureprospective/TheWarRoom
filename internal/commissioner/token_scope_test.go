package commissioner

import "testing"

func TestTokenArgumentScopeIncludesIndirectReferences(t *testing.T) {
	b := seededLedger(t)
	player := b.records[1].Command.Args.Native.Players[0]
	own := player.Franchise
	other := "franchise.32"
	if other == own {
		other = "franchise.31"
	}
	bid := b.add(RawCommand{Command: BidPlace, Args: RawArgs{Bid: &RawBidArgs{Franchise: other}}}, seasonMoment(3))
	claim := b.add(RawCommand{Command: WaiverClaim, Args: RawArgs{Waiver: &RawWaiverArgs{Franchise: other}}}, seasonMoment(3))
	pick := b.records[1].Command.Args.Native.Picks[0]
	pickOwner := pick.Owner
	for name, raw := range map[string]RawCommand{
		"people":       {Args: RawArgs{People: &RawPeople{Franchise: other}}},
		"lineup":       {Args: RawArgs{Lineup: &RawLineupChange{Franchise: other}}},
		"score":        {Args: RawArgs{Score: &RawScoreChange{Franchise: other}}},
		"cap":          {Args: RawArgs{Override: &RawNativeOverride{Franchise: other}}},
		"player":       {Args: RawArgs{Override: &RawNativeOverride{Player: player.ID}}},
		"bid-id":       {Args: RawArgs{Bid: &RawBidArgs{Bid: bid}}},
		"claim-id":     {Args: RawArgs{Waiver: &RawWaiverArgs{Claim: claim}}},
		"trade":        {Args: RawArgs{Trade: &RawTradeArgs{Franchise: other}}},
		"bracket":      {Args: RawArgs{Bracket: &RawBracketChange{Team: other}}},
		"history":      {Args: RawArgs{Link: &RawHistoryLink{Franchise: other}}},
		"calendar":     {Args: RawArgs{Exception: &RawWindowException{Franchise: other}}},
		"correction":   {Args: RawArgs{Correction: &RawCorrection{Franchise: other}}},
		"compensation": {Args: RawArgs{Correction: &RawCorrection{Entry: bid}}},
		"pick":         {League: "native.test", Args: RawArgs{Correction: &RawCorrection{Owner: other, Pick: &CorrectionPick{Season: pick.Season, Round: pick.Round, Original: pick.Original}}}},
	} {
		t.Run(name, func(t *testing.T) {
			franchise := own
			if name == "player" {
				franchise = other
			}
			if name == "pick" && franchise != pickOwner {
				franchise = pickOwner
			}
			records := b.records
			if name == "player" || name == "pick" {
				records = records[:2]
			}
			if err := CheckTokenFranchise(Access{Raw: raw, Records: records}, franchise); err == nil {
				t.Fatal("argument escaped franchise scope")
			}
		})
	}
	if err := CheckTokenFranchise(Access{Raw: RawCommand{Args: RawArgs{Override: &RawNativeOverride{Player: player.ID}}}, Records: b.records[:2]}, own); err != nil {
		t.Fatalf("own player refused: %v", err)
	}
}
