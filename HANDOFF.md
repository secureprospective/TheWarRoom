# Handoff

## Baton
ClaudeBox / review — 2026-11-01, build/SCHED-1, APP-SCHED-9.
Local commits only; no upstream configured, no push.

## Where it stands
- Through db6b642bc: reconciled full-snap/no-event zeros + dated depth membership; schedule
  bye zeros; partialLineupAllowed YES maps to ordinary penalty none. No scoring math/forks.
- Same 148 rows: 61 exact / 1 within .01 / 15 differ / 71 unavailable (was 45/1/13/89).
  55 no-snap + 32 bye zeros newly admitted. 198 player-weeks remain unavailable (was 285).
- Childress 0835 pending: no source-ID bridge; same-name IDs belong to canonical MFL 17372.
- Bills MFL 6–1 / PF 1912.35; native still 0/7 complete, W-L/PF unavailable.
- Evidence + full ledgers: docs/build-handoffs/APP-SCHED-9-evidence.md and matching TSVs.
- Race: 319 top-level PASS / 9 opt-in SKIP across source/history/league/settings packages.
  Final copied replay PASS 169.18 s, byte-identical results/standings, ledger idempotent 10478.
  Plain make build PASS. Commit hooks PASS. No independent review or GUI evidence.
- make lint blocked by inherited tiny_files 26 > 25. Separate source-branch and boundary
  gates also fail on unchanged starting-ref files; details in evidence. No gate weakened.

## Next move / blocked on
- Review the new zero admission and partial-lineup translation; live-check imported partial
  lineups and queue evidence on a disposable profile. Existing ledgers need an explicit ordinary
  setting change; importer does not silently rewrite them.
- Need historical membership / stronger snap completeness evidence for remaining gaps; no
  identity guess or current-team zero. Bills Gonzalez W1–6 still lacks dated membership.
- 15 complete residual differences, secondary adapters, pinned absence evidence, rare stat
  fallback completeness, whole-season cost, 76 backfill refusals and inherited gates remain.
- Pre-existing untracked sched8 measurement test/cache untouched; own one-use tests removed.

## Tried and rejected
- Name-only merge of 0835 into 17372: external IDs already owned by another canonical player.
- Rounded snap shares alone: partial files can fit adjacent totals; now require unique totals.
- Zero for missing roster rows, current affiliations, absent partial-schedule entries or
  conflicting team games. Those stay pending.
