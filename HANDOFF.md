# Handoff

## Baton
Review / ClaudeBox — 2026-11-01, build/SCHED-1. APP-SCHED-4 **partial**, local commits only;
no remote contact. Do not claim full scoring or parity.

## Where it stands
- 28ee30a3: traced existing acquisition and copied-profile gap.
- b311aae1: ordinary launch/manual/watched sync acquires cached, validated 2026 nflverse plays
  through the existing archived transport, for any already-published NFL league season.
  History offers a completed newest revision, preserving all original weekly revisions;
  native results and stat queue see it. No math/settings/player exceptions changed.
  Scorer retains the counted revision's missing-evidence labels (regression prevents false completion).
- ab6b7391: native-league cold launch acquires weekly + plays, repeat fetches neither; held evidence verified.
- Copy proof: scoring acquisition 1.472 s cold / .002–.003 s cached; history 461,660,160 →
  469,966,848 B (+7.92 MiB including reference sync), one distinct 2026 plays body after repeats.
  Whole proof 126.83 s first / 39.32 s repeat; page latency not certified.
- Recorded-lineup-only comparison: 148 rows; before AND after exact=0 / within .01=0 /
  complete differ=0 / unavailable=148. New complete evidence is not automatically approved.
  Current recorded-starter occurrences: 2,304 counted revisions lack plays, 472 absent weeks,
  300 unresolved positions, 7 unlinked identities. No fabricated zeroes or partial-score totals.
- Targeted race: 623 top-level / 1,004 named pass, 9 skip, 1 inherited failure:
  TestSinglePoolStartupDraftIsUnchanged digest 8a6936145d37309e != 90ca21fd230f0aa6;
  same failure proven on clean local archive of starting f5366b23. Final focused tests pass.
  Plain make build and commit hooks pass. No GUI/full verify/independent review.
- Durable evidence: `docs/build-handoffs/APP-SCHED-4-evidence.md` and its comparison TSV.
  Raw logs/copied profile under `.tmp/sched4`; one-use harness removed. Pre-existing
  untracked `.golangci-cache/` left alone. Prior part-3 evidence remains.

## Next move / blocked on
- Head brain decision: default `countedVersion` is earliest revision; newer complete evidence does not
  silently approve a correction. Resolve completeness/approval policy without changing scoring math.
- Finish ordinary Sleeper edge/advanced-rushing acquisition and live ESPN scoring integration.
  Existing ESPN live display path is unchanged; full live-to-final suite NOT delivered.
- Resolve true historical positions and absent weekly evidence through approved data.
  Zion Childress custom MFL 0835 has no verified NFL link: fresh DP candidate is MFL 17372,
  GSIS 00-0040044. Same name alone is not a verified identity. No manual exception installed.
- First schedule publication is seen by scoring acquisition on next ordinary sync; immediate first-sync
  acquisition and source-health UI failure integration remain debt. Re-run parity after blockers.
- Prior 76 historical-roster refusals and empty-lineup representation remain separate issues.

## Tried and rejected
- Refreshing weekly stats alone: already loaded 2026, does not supply play-by-play.
- Relabelling the old counted weekly revision as complete because a newer revision has plays:
  regression showed false Complete; missing metadata now follows the selected revision too.
- Replacing older revisions, auto-approving corrections, inventing inactive zeroes, mapping a custom ID
  by name alone, using MFL totals or changing scoring settings: not done.
