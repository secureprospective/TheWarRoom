# Handoff

- Baton: Bee, 2026-11-17.
- Previous: merged origin/main (#335 audit fix) into PR #333 branch.
- State: PR #333 https://github.com/secureprospective/TheWarRoom/pull/333 is open; PR #323 is closed with a link to #333. Hosted queue uses existing per-league `private_entries`; franchise comes from authenticated seated identity. Embedded DraftRoom is unchanged; ADP/tier stay unavailable.
- Next: remove agent-token draft.queue.set delegation per task; run gates, update PR and confirm CI.
- Blocked: none known.
- Tried/rejected: no schema change, shared league cache, commissioner-wide view, local queue route, or dependency-audit bypass.
