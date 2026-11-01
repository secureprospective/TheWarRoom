# Handoff

## Baton
Review — 2026-11-01, build/MOVE-mfl. Local only; no remote contact.

## Where it stands
- Control Room → League → Move my MFL league now uses the existing migration service.
- Per-league keyring credentials, preset Base threading, catalogue inputs, atomic import answers,
  and selection of the imported league are wired. Flow: `docs/ui/move-mfl.md`.
- Final targeted Go run: 11 top-level / 26 including subtests PASS with race detection;
  frontend flow: 5 PASS. Build, typecheck, bundle/CSP, binding freshness and scoped lint pass.

## Next move / blocked on
- Full suite not run: heavy-ok refused because another race job was running.
- Bloat gate inherited: HEAD and candidate both have 25 tiny files; baseline is 24.
- Native Wails click-through and expanded desktop capture still need verification.
  Synthetic mobile capture inspected in `.impeccable/review/mobile.png` (untracked).
- Generated door refresh also corrected pre-existing lineup/projection/personal type drift.

## Tried and rejected
- Ordinary RulesSet was not batchable; permitted it only in creation batches, preserving checks.
- Rollback fixture proves a failed answer leaves no creation, snapshot or partial answers.
