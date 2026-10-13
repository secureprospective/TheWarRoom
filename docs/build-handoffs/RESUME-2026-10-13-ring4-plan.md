# RESUME: TheWarRoom ring 4 (commissioner core), 2026-10-13 ~14:05

## 1. What we are doing
Ring 4 planning is done. This session is on standby until the other session merges ring 3, then it
starts the ring 4 build. The first step is moving K4 (the commissioner section of the spec) into the repo.
- Repo: Beelink `~/work/TheWarRoom` (Beelink = 192.168.1.191, user chris; this box). The main checkout is on
  `session/league-history-pwa` (Christopher's Lab, never mix). Do not commit there.
- Ring 4 work goes on a NEW worktree `session/ring-4-<desc>` branched from main AFTER the ring 3 close PR merges.

## 2. Agents and harnesses
- Another Claude session (ring 3 close) owns: `session/ring-3-land-close`, the Sol audits of findings 1-52
  and 53-95, spec §15b, the final ring 3 PR. Do not touch its worktrees or files. Its resume:
  CT105 `/root/TheWarRoom-ring3-RESUME.md`, task T428.
- Its ETA (Christopher, 14:00): audits finish 14:45-15:00 · §15b findings written ~15:20 · final PR,
  full gate, merge ~15:45.
- Bee (Luna) is the default for bulk dispatch; Sol codes chunks in the ring loop; Claude reviews.

## 3. Status
| Item | State |
|---|---|
| League facts (14 + follow-ups) | RULED, `league-facts-rulings-2026-10-13.md` |
| K4 outline §16.1-§16.12 | ALL AGREED, `K4-outline-draft.md` |
| K4 census (§16.7) | ALL 7 GROUPS SIGNED OFF, `K4-census-draft.md`: 56 controls CM-001..CM-056, 144 typed verbs |
| CT105 backbone pointer | WRITTEN 13:59 in `/root/.claude/backbone/context.md` |
| K4 in repo (spec rev 4 §16 + registry rows) | NOT STARTED, waiting on ring 3 merge |
| Rulebook corrections | OWED, not done (list in §7) |

## 4. Artifacts (Beelink run dir `~/fleet/runs/warroom-commish-k3-2026-10-13/`)
`K4-outline-draft.md`, `K4-census-draft.md`, `league-facts-rulings-2026-10-13.md`, `k3-triage.md`
(pointer section appended), `K3-HANDOFF-to-Tom-2026-10-13.md`, `bee-findings.md` (= Sol), `glm-findings.md`.
Process doc: `~/fleet/runs/bee-mfl-commish-map-2026-10-13/reports/Commissioner_Surface_Process_2026-10-13.md`.
Brief: `~/fleet/briefs/warroom-commish-k3-glm-2026-10-13.md`. Rulebook: repo `docs/league-rules/`.
Memories: `warroom-rules-are-commissioner-settings-not-mfl-or-legacy`, `warroom-league-rulebook-lives-in-the-repo`.

## 5. Current bug
None. Planning only.

## 6. Refuted / do not redo
- Do NOT re-ask the league facts. The rulebook (`docs/league-rules/Official_Rulebook.md` v2026,
  `Math_Rules_Reference.md`) already answers most; Christopher ruled the rest. K3's "open facts" list was stale.
- Do NOT design commissioner Acts with MFL read-back or "Handed off". Ruled independent (§16.2).
- The pre-rollover owner window (first 9b answer) is SUPERSEDED: rollover on Mar 1, spring windows decide.

## 7. Decisions (binding; full text in the three run-dir files)
- App is coded to no league's standard; every rule is a commissioner setting; Legacy rulebook values are the preset only.
- Commissioner side is independent of MFL/ProBoards; the owner-side hand-offs stay until a later ring.
- Ring placement: ring 4 = census groups A, B, C, D, E, and G (minus CM-055) + commissioner-facing §15 carry
  (M-093, M-095, M-024/M-156/M-157, M-137). Ring 5 = owner-only carry (M-125..127, M-050, M-108, S-14,
  Lab runtime compile, Inspector dead cap/form). Later = group F playoffs/history, CM-055 engine params,
  CM-050 awards (deferred), new contract action kinds.
- Rulebook edits owed (later, ask before doing): §2 IR default wider, §2 roster 18-80, §5 3 wild cards,
  §3 lineup = min/max model.

## 8. Ledger state
Nothing committed to any repo by this session. Run-dir files are on disk only (not a git repo).
The repo copy of this RESUME is deferred until the ring-4 branch exists, because committing now would land on
the Lab branch or in the other session's worktree. Copies: this file, CT105 `/root/TheWarRoom-ring4-RESUME.md`.

## 9. Next actions, in order
1. Check ring 3 merged: `cd ~/work/TheWarRoom && git fetch -q && git log origin/main --oneline -3`, plus CT105
   backbone tail and T428 on Hermes. Wait if it isn't merged. Don't ping the other session.
2. Once merged, run `git worktree add .worktrees/ring-4-k4 -b session/ring-4-k4 origin/main`.
3. Commit this RESUME into the worktree at `docs/build-handoffs/RESUME-2026-10-13-ring4-plan.md`.
4. Fold `K4-outline-draft.md` + `K4-census-draft.md` into `docs/ui/Target_UI_Spec_2026-10.md` as §16 (rev 4,
   following the doc's own revision rules). Read the post-ring-3 spec (§15b) first.
5. Add commissioner rows (prefix CM-) to `docs/ui/endpoint-registry.csv` with the same columns plus `governs`.
   Check every governs ID exists in the registry.
6. Show Christopher the spec diff for K4 sign-off. Plan the ring 4 chunks (Sol briefs) only after his go.

## 10. Environment
CT105 reachable from here: `ssh root@192.168.1.105`. Hermes: `ssh hermes`, task list via `cc`.
CT105 backbone log is actively written by CT105/the other session: append only, small single writes.

## 11. Honest status
Plan complete and agreed by Christopher. Nothing is built. The ring 3 merge ETA (~15:45) comes from the other session and hasn't been checked.
