# Handoff

## Baton
Review / ClaudeBox — 2026-11-01, build/SCHED-1. APP-SCHED-5 implemented; parity partial.
Local commits only; no git remote contact or push authorized.

## Where it stands
- Automatic counted revision: newest live, then nflverse authority; later live cannot displace it.
  Commissioner revision pins and per-stat corrections win. Ordinary scheduled scoring sync records
  current selections via existing StatApprove ledger/queue/provenance. Worker uses a locked batch,
  standard admission/checks/event/audit writes; stat-only changes leave roster recovery unchanged.
- Player-record fallback resolves all 300 position-blocked starter occurrences (93 players), no guesses
  or exceptions. Specific NFL positions remain authoritative over differing player-record positions.
- Copied scratch profile: 148 recorded franchise-weeks → exact 4 / within .01 0 / differ 33 /
  unavailable 111. Largest differences are existing invalid-lineup penalties caused by specific
  NFL-vs-league position differences, plus positional rates. No settings/math fixes.
- Bills MFL 6–1 / PF 1912.35; app unsupported (no complete Bills week). Missing weekly evidence:
  472 starter occurrences; 0835 Zion Childress unlinked in seven. No missing-position/play blockers.
- Full affected race run: 961 named PASS (588 top-level), 9 skip, 2 inherited/environment failures:
  startup draft digest (already clean-baseline proven) and preset-name guard scanning the pre-existing
  `.tmp/sched4/baseline` archive. Final focused race 51 PASS; after helper extraction 11/11 PASS.
  Final plain make build + commit hooks PASS. No GUI/full verify/independent reviewer.
- Initial remaining 6,988 feed selections committed in 91.710 s; repeat 2.193 s. Final copy proof
  121.96 s (seven results + standings + reconciliation). Not fast page/first-sync certification.
- Evidence + 148-row detail: `docs/build-handoffs/APP-SCHED-5-evidence.md` and comparison TSV.
  Raw logs/copy under `.tmp/sched5`; one-use harnesses removed. Pre-existing cache/archive untouched.

## Next move / blocked on
- Review specific NFL-vs-league position precedence: e.g. Collins/Chaisson LB vs listed DE triggers
  existing minimum-DE failure and top-starter penalty. No origin-specific branch or manual fix added.
- Establish missing NFL weekly/inactivity evidence and verified custom-ID alias (0835); no zero fiction.
- Trace Schultz W4 one-point residual. Complete ordinary live ESPN/Sleeper/advanced-rushing inputs.
- Improve slow ledger bootstrap and result reads; first-publication timing/source-health UI remain debt.
- Prior 76 historical-roster refusals and empty-lineup limitations remain separate.

## Tried and rejected
- Replaying all archived revisions as if each had been counted: wrong bootstrap history and too slow.
  Adoption records only actual current selection; future moves each use the same ledger command.
- Per-move archive rereads/roster reconstruction: trial only recorded 142 approvals in 45 s.
  Batch and unchanged-stat-only recovery optimization preserve append/audit flow; cancelled batch
  observed zero writes. No settings tweaks, name-only links or absent-stat zeroes installed.
