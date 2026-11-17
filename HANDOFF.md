# Handoff

- Baton: Bee, 2026-11-17.
- Previous: current main includes PRs #330 and #331 hosted player-usage facts.
- State: PR #325 supplies `private_entries`; hosted `draft.queue.read` / `draft.queue.set` now store the authenticated seated franchise's ordered queue. Session and delegated-token paths share per-league `personal.NewLeague`; no franchise is accepted from the request, and queue reads bypass reply cache. Embedded DraftRoom is unchanged; ADP/tier stay unavailable.
- Tests: `TestHostedDraftQueueIsSeatedPrivateAndReplacedAsAWhole`, `TestHostedDraftQueueReplaceIsOrderedAtomicAndPrivate`, server-pin tests, leagueservice/account/commandcatalog race tests, hosted-reorder and embedded-path War Room Draft tests, frontend typecheck, and `make lint` passed.
- Next: run push-hook verification, push/open PR, wait for green CI, and close PR #323 with the new link.
- Blocked: none.
- Tried/rejected: no schema change, shared league cache, commissioner-wide view, or local queue route.
