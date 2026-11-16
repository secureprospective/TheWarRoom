# Handoff

- Baton: ClaudeBox review, 2026-11-16; branch session/pulse-storm47.
- State: Hosted Pulse odds read separately, once per season/open and on league topics only. Scores, unrelated topics, manual Refresh and other-read retries cannot re-request refused odds; unavailable wording stays visible. StoredHost has no NativeOdds implementation; server admission unchanged.
- Next: Review PR and test hosted Now/Race with odds refused: score updates continue, odds stay unavailable; a league invalidation makes one fresh attempt.
- Blocked on: Live click-through, not attempted; prohibited lab hosts were not contacted.
- Tried/rejected: Server odds implementation is beyond the request-storm fix. Baseline already skipped score-only bursts, but grouped odds with all other league refreshes.
- Evidence: hostedPulse/nativePulse/pulseRace targeted tests pass (27); score-refresh and refusal-resubscription mutants each rejected, then restored. Push hook supplies full frontend gates.
