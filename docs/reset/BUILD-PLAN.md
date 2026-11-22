# TheWarRoom build plan — from Sun 11-22 (for Tom)
Written by Claudebox (planner) 2026-11-22 for Tom (builder). Christopher approved the direction in the
11-22 planning session. Rulings source: `~/Downloads/TheWarRoom-DIRECTION-GRILL-1122.md` on the Beelink (same folder as this
file; master copy on Claudebox `/root/`). Today's state source: your own
`~/work/TheWarRoom/.worktrees/pwa-sw-shell/docs/RESUME-gameday-1122.md`.
Repo: `~/work/TheWarRoom` on the Beelink. Claudebox (CT105) is `ssh claudebox` from the Beelink.

## 0. How to use this plan
- Work the stages **in order**. Inside a stage, work in the order listed unless a stop rule fires.
- Today's game-day items are **mile markers**: fixed times you work around, not the whole day.
- **Stop and ask only for:** a hard limit (merge or push to main, production deploy, Proxmox host change,
  deleting the only copy of something), a clash with a product ruling below, or an assumption proved wrong.
  Everything else: decide, log it on the tracker, keep moving.
- **Assumption-risk flag format** (use it on the tracker): "This assumes X. If X is wrong, it costs Y. Cheapest check: Z."
- **Tracker** http://192.168.1.222:8790/ is Christopher's one steering place. He has asked for this
  specifically: it is not optional. File: Hermes `~/fleet/tracker/data/plan.json`
  (`ssh hermes` from the Beelink). Rules:
  - Read `data/answers.jsonl` first on every update; act on his answers; list each in `seen`.
  - The Build Plan already has one card per stage. Keep those titles. Change only `status`, `plain`,
    `next`, `blocker`. Status words: `queued` (grey), `running` (blue), `needs_you` (amber),
    `blocked` / `failed` (red), `done` (green). Any other word shows red, so don't invent one.
  - `next` must say what Christopher should expect next and roughly when.
  - Add a `log` entry (time in Central from the machine clock, one plain sentence) at every start,
    finish, block or change of direction. Back up plan.json before each write; keep valid JSON.
  - He must never have to ask for an update.
- **Do not edit** North Star, `docs/adr/`, GLOSSARY.md or this plan's rulings. Those are Claudebox's,
  under Christopher's rulings. Propose changes on the tracker.
- **Who builds:** you drive and verify; route code work per `~/fleet/ROUTING.md` (Gemini is the build pool while GPT is down; max 2 Neon lanes;
  Argon only for risky code). Christopher: "If you run Haiku please use Claudebox, not your own". So you don't
  launch Haiku yourself; if a job suits Haiku, use the Tom-asks-Claudebox file below or pick the next row in ROUTING.md.

- **Need a technical answer from Claudebox?** Don't guess on anything that touches a product ruling,
  data safety or a hard limit. Write the question to `~/Downloads/TOM-ASKS-CLAUDEBOX-<topic>.md`, set the
  stage card to `needs_you` with next = "Christopher: bring TOM-ASKS-CLAUDEBOX-<topic>.md to Claudebox",
  and keep working on anything that doesn't depend on it. Use this shape, under one page:
  ```
  # Tom asks Claudebox: <one-line question>
  Stage / step: <e.g. Stage 1 step 2>
  What I'm trying to do: <one sentence>
  What I found (facts, with paths/commands/output): <bullets>
  Options I see: A) ... B) ... (my pick and why)
  What breaks if we pick wrong: <one line>
  What I'm doing meanwhile: <one line>
  ```
  Small technical choices (naming, test layout, which model) are yours: decide and log it.

## 1. Product rulings every brief must respect (quote the ones a job touches)
1. Long-term aim: the juggernaut. Through 2027: one real 32-team league runs a full season on TWR; Legacy is never put at risk.
2. Two league roles: **MFL leagues** (TWR is a companion, MFL is the record) and **native leagues** (TWR is the record).
   Code serving the companion role is not "old assumption" residue.
3. No plan about which real league moves first. Until July 2027: test leagues of many sizes and shapes.
4. Build for Christopher's solo use first until about March. Exception: anything a bystander sees in phone
   live scoring gets full polish now. Phone target is **mobile mock v1 only**.
5. Feature guideline (not a gate): saves time / makes someone ask "what is that?" / protects the league. Stay inventive.
6. Off-season direction: every commissioner job redesigned until it is faster and clearer than the market,
   mock first. The engines already work; the UI/UX is the hard part.
7. Market study per commissioner job before any mock (Sleeper, ESPN, Yahoo accounts exist; read only,
   never a real move in any app).
8. **Stand-in** = an AI holding an empty seat and playing the team like an owner. Test leagues only for now.
   Caretaker = a commissioner keeping an empty team alive.

## 2. Mile markers today (Sun 11-22, Central)
| Time | Marker | Owner |
|---|---|---|
| 12:00 PM | T673 live field proof on Christopher's S25; Hermes recorder goes live | Christopher walks; Tom watches `server live` log |
| 12:00 PM | T670 wall at `127.0.0.1:5199/app/wall?n=7` (signed in) | Tom, if Christopher wants it |
| 12:00–~10 PM | Lab rules: CLAUDE.md "Environments" (noon–10 PM rule retired) | — |
| after the SNF game ends (~10:30 PM) | T668 release v50 (Stage 2) | Tom; Christopher already said yes |

During the games, Tom works Stage 1 (it never touches the lab) and builds Stage 2 code on branches.

## Stage 1 — Clean the workspace (starts when Christopher hands you this file)
Rulings Q22–Q27 (11-22). **Never touch:** `.worktrees/pwa-sw-shell`, `.worktrees/lab-dev`,
`.worktrees/glossary-adr`, Vite pid 557753, anything modified in the last 7 days, anything another agent is running in.
(The main checkout in step 2 is the one exception: it is handled with the backup steps below.)

1. **Inventory first** (read only): write `~/fleet/runs/twr-clean-1122/INVENTORY.md`: every worktree
   (path, branch, last change date, merged into origin/main y/n, clean y/n, size), every local and remote
   branch (merged y/n, last commit date). Note: squash merges make "commits ahead" lie; judge "merged" by
   `git diff origin/main...<branch>` being empty, or by the PR state on GitHub.
2. **Main checkout** (`~/work/TheWarRoom`, on `session/league-history-pwa`, ~596 changed files; may be the
   only copy of some work):
   a. Back up first: `git diff --binary HEAD > league-history-pwa-1122.patch` plus a tar of the untracked
      files (`git ls-files -o --exclude-standard`), copy both to Claudebox
      `/mnt/storage/claudebox/backups/twr-clean-1122/`, and check the sizes match on both ends.
   b. Then commit the state as-is to a new branch `archive/league-history-pwa-1122` and push it.
      If the pre-commit hooks refuse (likely, with broken files), **do not use `--no-verify`**: the
      verified patch + tar from (a) is the saved copy; log that on the tracker and move on.
   c. Only after (a) is verified: `git switch main`, then `git pull --ff-only` (if `git switch` refuses
      because of the changes, use `git stash -u` and leave the stash in place).
   Check: on main, `git status --porcelain | wc -l` = 0.
3. **Worktrees:** remove the merged and clean ones (`git worktree remove`, never `rm -rf` on a dirty one).
   Anything idle for more than 7 days and unmerged: back it up to Claudebox first, into
   `/mnt/storage/claudebox/backups/twr-clean-1122/` (a git bundle of its branch plus a tar of any uncommitted files), check the backup's size and that `git bundle verify` passes, then remove.
4. **Branches:** delete the local and remote branches already merged into main. Bundle the unmerged ones
   older than 30 days into one file in the same Claudebox folder (`git bundle create`), verify it, then delete them. Never delete `main`,
   `archive/*`, `bundle/*` with open PRs, any branch with an open PR, or anything in use or under 30 days.
   Deleting remote branches (`git push origin --delete`) is allowed; if a hook refuses, list the
   commands under "Needs Christopher" in the stage card instead.
5. **One front door:** README is the only "start here" (one page: what TWR is, where direction lives,
   where the current handoff lives, how to run it). North_Star.md = direction (Claudebox rewrites it as v3).
   One handoff per workstream in `docs/resume/RESUME-<workstream>.md`. HANDOFF.md is retired (its live
   content moves to the matching resume). Docs-only PR. Checked 11-22: README PR #339 is already merged
   (11-17) and #342 was closed unmerged (11-19), so start from main's README. Keep its mock images and the touchdown GIF.
6. **Old docs:** superseded docs move to `docs/archive/` with a one-line-per-file `docs/archive/INDEX.md`.
   Docs-only PR. Do not move docs/adr, GLOSSARY.md or SYSTEM_MAP.md.
7. **Disk + log:** `df -h /home` before and after; log the GB freed on the tracker.
**Done lines:** main checkout clean on main; worktrees cut to those in use (count logged); branches cut to
live ones (count logged); every backup verified on CT105; docs PRs open and green (Christopher signs the merge);
tracker shows before/after numbers. Never run `rm -rf` on a path built from a variable without
checking it is non-empty and inside `~/work/TheWarRoom`. **Proof (Q27):** Claudebox (not Tom) runs a cold-start agent ("start work on
TheWarRoom") after the merge; it must find current work, direction and rules within 5 minutes.

## Stage 2 — Finish the alpha loose ends (branches during games; see CLAUDE.md "Environments")
1. **T668 phones update themselves.** Fix is on `session/pwa-sw-shell` (3be79c9d7). YES GIVEN 11-22;
   after 10 PM: lab release v50 from that branch, phone check on S8 + iPhone (registration exists,
   `caches.keys()` holds `twr-static-shell-<new BUILD>`). Steps: RESUME-gameday-1122.md section 4.
2. **T677 startup reads** (results/standings/odds/bracket warm at startup). Build during games on a branch
   **cut from `session/pwa-sw-shell`**, so v51 still carries the T668 fix. (A v51 built from main would
   bring the broken update worker back.)
3. **T679 pre-game follow-ups** (pre-game updates trigger a phone refresh; playoff season type 3; debug log; dedupe weather).
   Same base as T677; T677 and T679 can share one branch. New data goes in a new read, never new fields on an
   existing read (11-21 lesson: that broke installed apps).
4. **Second release v51** = T677 + T679. It is also the T668 proof: both phones must switch to the new BUILD on their own.
5. Then one merge ask to Christopher: open the PR, get CI green, put the exact merge command in the
   stage card as `needs_you`. Never merge to main yourself. The ask covers `pwa-sw-shell` + T677 + T679 (hand him the exact command). Check T668 on the task list after he verifies.
6. Check whether `session/live-wall`, `session/error-codes` and `session/glossary-adr` still hold work main lacks
   (diff, not commit count). Real leftovers go in that same merge ask. When a branch is confirmed fully on main, its worktree and branch
   are removed with the Stage 1 rules. `glossary-adr` stays: it is T666 work for Stage 5.
**Done lines:** both test phones auto-updated v50 → v51 with no reinstall (screenshots); first app open after
a restart has no 5 s timeouts (lab log); one merge ask on the tracker.

## Stage 3 — Stability and safety carry-overs (before any reset building)
1. **T640 / T631 load** (history: Claudebox `/root/TheWarRoom-RESUME.md`, read with `ssh claudebox cat ...`): 0.10 stress was not a pass (502s at 80 devices, peak 1,018 MiB). A 32-team league is
   the target, so re-run the stress on the current main; fix the CPU and memory cost; GOMEMLIMIT per F51.
2. **T662** owner lineup.read refused for stress owner #1: product bug or test-account artefact. Settle it.
3. **T663** lab release runs automatically on merge to main (from CI or Hermes); production stays his sign-off.
   It must hold releases during NFL game windows (the no-release rule above) and queue them for after.
4. **Needs Christopher (list on the tracker, do not do):** T621 rotate the CT 257 keys; T632 Cloudflare TLS rule;
   T649 picomatch audit ignore (weekly check only).
**Done lines:** stress at 32-team load passes with zero 502s inside scripted stops (report); T662 closed with evidence.

## Stage 4 — Find the old assumptions (read only; Christopher rules)
Argon (if open) or Neon High maps the code, read only. Output `docs/reset/ASSUMPTIONS.md`: each old idea, where it lives,
how big it is, and a keep / retire / rebuild recommendation judged against Section 1. Known candidates:
legacy desktop (Wails) mode, the old Control Room, MFL-as-record for native leagues, offline desktop
acceptance (T489), the 3-OS freeze (T535), Flatpak/CasaOS packaging (T457), the 52-slice client/server plan (T320, T565).
**Also triage the task list:** every open TWR task gets a recommendation (still live / done / superseded /
needs ruling) in the same file. Never delete tasks; Christopher rules, then they are checked or reworded.
**Cut rule (Christopher 11-22):** clashes with a Section 1 ruling = recommend cut. Merely unproven = keep, with an
assumption flag ("This assumes X. If X is wrong, it costs Y. Cheapest check: Z."). Nothing is cut just for being old.
**Done line:** ASSUMPTIONS.md on a docs PR; tracker card "needs you: rule on N assumptions".

## Stage 5 — De-slop by area (only after Stage 4 rulings)
Rank areas by where agents get lost or break things (use the restart history and the review findings). Rebuild the
worst few properly; leave working code alone. Every change must make agents faster or the product clearer, and
owners must never notice. Folds in T591 (debloat), T570 (Market slim-down) and T666 (96 old-word renames).
Each area is one branch, an Argon review for risky code, and one merge ask.
**Slop means (Christopher 11-22):** (a) code or docs nothing uses; (b) two copies of one thing; (c) wording an owner
would find confusing; (d) tests that pass without really checking. Style preference is NOT slop.
**Order:** 1 what owners see on the phone in live games, 2 anything that could risk a trusted league, 3 what slows
agents down, 4 pure tidying.
**Tom's freedom:** delete unused code freely when tests pass. Any change to what an owner sees: stop, put a
needs_you card on the tracker first.
**Done line per area:** fewer lines, the same decisions (correctness first, then speed; table tests prove identical output).

## Stage 6 — Lock it in (runs alongside Stage 5)
1. **T664** fast checks (TS line length, controls test, tsc) at commit time, not pre-push.
2. Build guards like the T668 sw.js check: a build that would ship broken must fail. Add one per class of bug found.
3. The phone test that failed once on 11-20 during a push and passed on retry (noted in Claudebox
   `/root/TheWarRoom-RESUME-desktop-v3.md`): find it, fix it or quarantine it with a task.
4. **T665** daily cleanup PR job (Hydrogen from Hermes, spot-check 1 in 5), after the reset settles.
5. **Weekly habit check**, posted on the tracker: no running card more than a day stale; every new task has done
   lines; the cold-start test (fresh agent finds current work, direction, rules in 5 min) still passes.
**Done line:** each lesson from Stages 1–5 has an automatic check, listed in `docs/reset/LOCKS.md`.

**When Stages 4–6 start (Christopher 11-22):** after v51 ships and the 32-team load test passes. Game days are
NOT a hard stop for this work: we are the only users. (The lab-release hold during games, T663, is unchanged
unless Christopher says otherwise.)

## Stage 7 — Product backlog after the reset (order follows Section 1; Christopher threads timing)
- **Phone live "wow"** (mobile mock v1 gaps, `design-mocks/FIXES-mobile-batch-1121.md`, T650 big-play
  animations, T670 wall polish). Christopher picks what ships when.
- **Test leagues + Stand-ins (T681):** copies of real MFL league shapes (10, 12, 16, 32, twin-pool), agent-played owners;
  doubles as the 32-team load test.
- **Market study soak (T682):** Sleeper, ESPN and Yahoo, one page per commissioner job. 
- **Commissioner UX, mock first:** T644 Control Room real build (4 open decisions), T641 league presets,
  T592 league name, T617 one account holds the commissioner seat and a team, T638 private notes (panel first), T655 Home layout.
- **Owner tools:** T610 trade desk, T559 bid contract years, T560 parked stat reports.
- **Import door:** T456 MFL → native league import (the pre-public gate; prepares any real migration).
- Parked until ruled: T495 default league mode 2, T503 Legacy study round 2, T659 server on the S8, T473 split-scoring docs commit.

## Needs Christopher (current)
1. ~~T668 v50~~ APPROVED 11-22: Christopher said yes to lab release v50 after 10 PM tonight.
2. Merge sign-offs as asked (Stage 1 docs PRs, Stage 2 bundle).
3. T621 and T632 security items.
4. Rule on Stage 4's assumptions list when it lands.
