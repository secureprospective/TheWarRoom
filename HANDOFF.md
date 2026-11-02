# Handoff

## Baton
Bee → ClaudeBox / APP-SCORE-2 — 2026-11-02, `build/SCORE-2`. Local only.

## Where it stands
- `docs/build-handoffs/APP-SCORE-2.md` records changes, flow, verification and remaining acceptance debt.
- Runtime chain: ESPN → Sleeper gaps/outage; MFL never counted; nflverse field authority plus live gaps;
  commissioner pins/overrides last. New merged revisions preserve existing ones in process.
- Worker and readings share revisions; player/field sources reach results labels and starter inspector.
- Targeted race: 225 PASS, 1 opt-in SKIP; frontend 13 PASS; changed-package lint 0 issues; make build PASS.
- Full make lint blocked by inherited tiny-file count 28 vs baseline 24 (base also 28); no baseline change.

## Next move / blocked on
- Durable live/merged revision capture and restart tests: current chronology is in-process only.
- Establish exhaustive Sleeper IDP/kicker and ESPN recovery coverage from checked responses; one public
  Sleeper contract request returned 403. Do not certify full APP-SCORE-2 acceptance yet.
- Integrate with Tom's held-season cache without hiding new revision/digest changes.

## Tried and rejected
- No MFL totals inversion, guessed distances, fuzzy attribution or promotion of live fills to final authority.
- No archived digest rewriting, real-profile access, full suite, desktop launch, git remote or hook bypass.
