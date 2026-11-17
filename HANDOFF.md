# Handoff

- Baton: Christopher / ClaudeBox, 2026-11-17, session/pulse-hqmoves.
- State: source mismatch documented in `docs/build-handoffs/HQ-MOVES-SOURCE-2026-11-17.md`; draft PR pending push. No functional changes. Desktop My moves uses local append-only envelope audits; HQ season ledger is empty. Server has a distinct native ledger, not the desktop audit source.
- Next: review source trace and confirm native committed franchise history versus desktop audit parity, then resume the appropriate slice. Do not merge this draft as a completed feature.
- Blocked on: source contract mismatch; loader takes history.db, not thewarroom.db envelope audits. Native ledger data DOES exist; no claim that all server history is absent. LEDGER is every member, filtered by Audience; do not impose owner-only privacy or broaden moves.read.
- Evidence: initial frozen install/build passed; frontend 114 files / 881 tests passed; make lint passed, 0 issues. No touched Go packages; feature access/paging/render tests not implemented. No live host/DB inspection.
- Tried/rejected: relabel lifecycle receipts as full history (incomplete/private); assume desktop season-ledger implementation exists (empty renderer); import desktop audit data (outside read-only/no-migration task).
- Hygiene: tidy-check names six pre-existing Chromium symlinks and an unrelated pulse-lane launcher; outside scope, left untouched. No session services started. Build outputs are ignored. An accidentally written run-directory test log was immediately removed; retained changes are inside this worktree.
