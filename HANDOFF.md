# Handoff

## Baton
Review — 2026-11-01, build/WR-wire. PICKER-8; local commits only, no remote contact.

## Where it stands
- 5fe156c0: renamed War Room's `table fixed` header to `franchise-context` (avoids Tailwind fixed positioning). Embedded header stays in flow; shelf context cannot flex-shrink over its own content; draft controls wrap. Workspace row count distinguishes local versus portaled tabs.
- 7394b2b8: successful unscheduled week reads use the same provider clock as the shell, checked for season and failed freshness. No week-1 fallback. Unknown NFL week is labelled; failed/wrong-scope schedule reads remain rejected.
- 1051ba2d: removed the app-league HQ's meaningless MFL handoff button. Existing mirror action remains unchanged.
- cbedf643: HQ contract column and War Room share roster salary/remaining-years formatting, including zero salary and unknown years. No import-source branches.
- make build PASS (dev/picker-8 stamp); tsc PASS; targeted Vitest 65 tests / 7 files PASS; Go -race: 116 tests / 3 packages PASS (app/control, leagueclock, leagueweek). Layout detector: zero findings.
- Logs: .picker-8-build.log, .picker-8-vitest.log, .picker-8-go.json, .picker-8-design.json (local, untracked). Existing untracked cache/state/scratch paths untouched.

## Next move / blocked on
- Reviewer checks the imported league at 1280px: context above trade desk; header, partner context and draft controls do not overlap; HQ shows week 8 when the shared clock holds 8; no MFL button; contract labels agree between HQ and War Room.
- Rendered geometry and real-database behavior remain unverified: no desktop walk, as requested. Static/DOM regressions are not geometry proof.
- Existing Wails generic-type generator warnings remain; build exits successfully. Model/projection/private-plan placeholders and max-only offensive minimum remain existing debt.

## Tried and rejected
- No fabricated week, matchup, kickoff or starters; no imported flag, league-ID branch, full suite, desktop launch or remote operation.
