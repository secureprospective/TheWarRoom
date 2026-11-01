# Handoff

## Baton
Review — 2026-11-01, build/WR-wire. PICKER-7; local commits only, no remote contact.

## Where it stands
- 95de331b: embedded HQ spans the shell workspace + unused outer inspector track; only its own inspector remains. Shell inspector CSS no longer leaks into native inspectors; HQ uses shell tabs only.
- b192ba2e: import records assets/settings, not a published schedule. A successful week.read with no current/next week previously skipped lineup.read entirely. Read week 1 explicitly labelled as a viewing week; failed/wrong-scope week reads still fail closed. No parser checks relaxed. Partial HQ errors are logged and attached to unavailable board regions.
- dda3b16d: War Room uses the same space, portals its existing tabs into the shell strip, keeps the franchise chooser, and contains narrow panels/rows. Read failures log diagnostics. Imported-shaped 35-player shell regression covers both screens (blank contractStatus, heldContractStatus, contractInfo).
- Targeted Vitest: 60 PASS / 7 files; tsc PASS. Old no-week behavior deliberately restored temporarily: regression failed as expected, then source restored.
- Focused Go race run: 6 tests / 4 packages PASS; one helper package has no tests. Broader four-package race attempt exceeded the 120-second tool limit: migration, mflsource and leagueservice completed, commissioner did not (376 individual passes before termination, no failures). No full repository suite run.
- make build PASS (stamped Makefile build); existing Wails unresolved-type generator warnings remain. Logs: frontend/.tmp/picker7/ (ignored). Mechanical layout detector: no findings. No desktop walk or real database access.
- Handwritten production footprint, fc7ad291 → dda3b16d: 3319 → 3360 physical lines, 216578 → 219523 Unicode code points including LF; tests/docs/generated excluded.

## Next move / blocked on
- Reviewer confirms on the real imported league: one HQ tab strip, one inspector, no overlapping starting/bench rows; 35 players available; absent schedule says “Viewing week 1”; shell War Room tabs switch functional regions. Real-league diagnosis and rendered geometry remain unverified here.
- Native week 1 is an explicit initial viewing week, not inferred current football time; no schedule/kickoff/starting lineup is fabricated.
- Existing debt: unavailable model/projection/private-plan regions remain honest placeholders; prior transfer replay parity failure belongs to another branch. A max-only offensive minimum still has no numeric minimum to show.

## Tried and rejected
- No import-source/league-ID branches, no fabricated starters or dates, no fallback after a failed or wrong-scope calendar read.
- Existing untracked .golangci-cache/, .picker-state.md and .picker-tmp/ remain untouched.
