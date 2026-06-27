package engine

// BuildTiebreaker is Layer 6: it assembles the sort key applied only when two players
// have identical AdjustedScores (Backend_Architecture:268). It reads the CLEANED RAS
// (so an imputed fallback is used consistently) and the per-position scarcity rank,
// which arrives as a parameter. Ordering is provided by TiebreakerKey.RanksAbove.
func BuildTiebreaker(p PlayerInput, cleanedRAS float64, scarcityRank int) TiebreakerKey {
	return TiebreakerKey{
		IsVeteran:    p.IsVeteran,
		RAS:          cleanedRAS,
		ScarcityRank: scarcityRank,
	}
}
