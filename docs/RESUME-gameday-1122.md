# RESUME — game day Sun 11-22 (Tom session on the Beelink, ~9:45–10:40 AM CT)

For Claudebox to plan from. Written by Tom (Claude Opus 5.5 on the Beelink).
Christopher's own words are quoted; everything else is what was observed.

## COMPACT POINT 2 — 11-22 12:20 PM CT (working the build plan)
On resume Christopher said: "after compaction make sure everything is healthy". Do that first (list below).
- **Plan:** `~/Downloads/TWR-BUILD-PLAN-1122.md` (Claudebox; re-read Stages 4-6 before them, updated
  12:00), rulings `~/Downloads/TheWarRoom-DIRECTION-GRILL-1122.md`. Tom = head-brain: briefs, verify;
  builds via AGY (`~/fleet/bin/agy-run.sh`). Haiku: only via `~/Downloads/TOM-ASKS-CLAUDEBOX-<topic>.md`.
  Questions to Claudebox: write that file, card `needs_you`; answers land as `~/Downloads/CLAUDEBOX-ANSWERS-TOM-*.md`.
- **Tracker** Hermes `~/fleet/tracker/data/plan.json`: Claudebox writes it too. Find cards BY TITLE
  (`next(l for l in d["lanes"] if l["title"].startswith("Stage 1"))`), never by index; back up first
  (`plan.json.bak-1122<letter>`, last used `o`); no apostrophes inside the ssh-quoted python (breaks quoting).
  Read `data/answers.jsonl` first (only q1 from 11-21, already in `seen`).
- **AGY quota 12:15 PM (REMAINING):** Gemini week 44.33% (refresh Sat 11/28 ~4 AM), 5-hour 57.26%
  (refresh ~4:14 PM), Argon week 84.31%, Claude/GPT via AGY 0% until Fri 11/27. Saved as USED in
  `~/fleet/agy-quota.json`.

### In flight at compaction
| Job | Check alive | Output | On result |
|---|---|---|---|
| Neon Medium fix pass T677+T679 (pid 1462641, started 12:04 PM) | `cat ~/fleet/runs/twr-t677-t679-fix-1122/exit`; `ps -p 1462641` | `~/fleet/runs/twr-t677-t679-fix-1122/REPORT.md`, commit on `session/t677-t679` | Verify: each of 8 Argon findings fixed with fail-without test; run gates (cmd/twr-server is slow under race: `-run` focused + `-timeout 900s`); then a short Argon re-review of finding #1 (startup order) and #4 (cache). Then push branch. |
| PR #460 auto-merge loop (bg shell, `gh pr merge 460 --squash` when green; Christopher said "yes merge #460") | `gh pr view 460 --json state -q .state` | merge commit on main | If loop died and checks green: run `gh pr merge 460 --squash` (approved). Then card: Claudebox runs the Q27 cold-start test. Then `git -C ~/work/TheWarRoom pull --ff-only` on main. |
| Vite pid 557753 (127.0.0.1:5199, wall/dev page) | `ps -p 557753` | `~/logs/vite-lab-dev.log` | Keep up all day (Christopher: "the wall is working well"). |

### Done today since 10:40 (all verified)
- Stage 1 main checkout: 592 files existed only in the git INDEX (status `AD`), so the plan's
  `git diff HEAD` would save nothing; saved index patch + worktree patch + untracked tar + raw .git/index
  to CT105 `/mnt/storage/claudebox/backups/twr-clean-1122/` (sha256 checked). Archive branch
  `archive/league-history-pwa-1122` pushed (476c330ca index state, ae856d1d0 worktree state). Main checkout
  on main, clean. `tools/league-history/` (849 MB ignored data incl. mfl.db) backed up (tgz on CT105) and
  hidden via `.git/info/exclude`.
- Workspaces 165 -> 58; branches local 370 -> 135, remote 376 -> 51; disk 112 -> 130 GB free. All tips
  bundled+verified on CT105 (`worktrees/`, `branches-unmerged-gt30d-1122.bundle`, `branches-merged-recent-1122.bundle`
  541 tips 70,594,803 B). Claudebox ruled merged work under 7/30 days may go (Option A).
  Left: `.worktrees/pulse-a010u2`, `pulse-a010u3` half-removed (read-only Go cache; git refused; ~2 GB,
  unregistered) — needs Christopher's ok to `chmod -R u+w` + delete. Kept `session/split-scoring` (T473 parked).
  Inventory: `~/fleet/runs/twr-clean-1122/INVENTORY.md` (+ `inventory.py`).
- Docs PR #460 (Hydrogen built, Tom verified nothing lost from README): README one page, `docs/product-tour.md`,
  `docs/resume/` (7), HANDOFF.md retired, 2 docs to `docs/archive/` + INDEX. Links North_Star + docs/adr/product/README.md.
- PR #459 (Claudebox, North Star v3 + product ADRs) merged 12:00 PM, main 6c3f13e3f.
- T677+T679 built by Neon (7acfa36b9, aa6fc20e4 on `session/t677-t679`, cut from pwa-sw-shell). Argon High
  review: FIX FIRST, 4 high (warm-up runs before live restore → wrong cache keys; warm test proves nothing;
  playoff scoreboards dropped by readBoard; recordedStartersCache writes empty/partial entries), 3 medium, 1 low.
  `~/fleet/runs/twr-t677-t679-review-1122/REVIEW.md`.
- Noon: lab live ticks every 10 s, wall working (Christopher), Hermes recorder LIVE recording full plays
  (ESPN plays/summary/scoreboard + Sleeper). Yahoo scoreboard URL 404 since kickoff (flagged; after Sunday).

### Refuted — do not retest
- `cmd/twr-server` race-test timeout at 10 m is NOT a hang from T677: the running test
  `TestPostCommitPinAndReportErrorsNoteAlreadyCommitted` passes on branch (66 s) and base (102 s).
- 503 `nfl.game.read` at 08:02 UTC were 4 isolated errors; none since. value.read 409s are a known host limit.

### Health check on resume (Christopher asked)
`ps -p 557753`; lab `grep "server live" /var/lib/twr/server.log | tail -2` (heredoc form); Hermes
`journalctl --user -u live-recorder -n 2`; Neon exit file; `gh pr view 460`; tracker updated stamp.

### Next actions, in order
1. Run the health check above; post one tracker log line.
2. Verify Neon's fix pass (table above), Argon re-check of #1/#4, push `session/t677-t679`.
3. Stage 1 done lines: #460 merged, Claudebox cold-start test (theirs). Then Stage 1 card `done`.
4. After SNF ends (~10:30 PM): T668 lab release v50 from `origin/session/pwa-sw-shell` (section 4), phone check.
5. Then v51 from `session/t677-t679`; both phones must self-update; one merge ask (Stage 2 step 5).
6. Stage 2 step 6: diff `session/live-wall`, `session/error-codes`, `session/glossary-adr` against main.

### Needs Christopher
- OK to delete the two half-removed workspaces (pulse-a010u2/u3, ~2 GB).
- Later: one merge ask for pwa-sw-shell + T677 + T679 (Stage 2).
- Filing gate fails on ~27 `~/scratch-*.png/.log` at home top: another agent's live OBS/overlay job (bridge pid
  1221803), not Tom's; left alone.

## 1. Live games readiness (done, verified 9:45–9:55 AM)

| Thing | State | Evidence |
|---|---|---|
| Lab CT260 | v49 = main bc7a8bdff, all 8 status checks OK, 5.3 GB free | `~/fleet/runs/labrel-v49-1121.log` |
| Lab live collector | Holds week 11 board: 7 games 12:00 PM CT, NYJ@LAC 3:05, LV@DEN + JAX@NYG 3:25, MIN–SF 7:20 PM, CIN@WSH Mon | checkpoint `/var/lib/twr/live-scores/checkpoint.json` |
| Beelink dev page 127.0.0.1:5199 | Vite pid 557753, worktree `.worktrees/lab-dev`. Was 2 commits behind main; rebased `session/lab-dev` onto origin/main (5c10eb16e = main + resume doc); HMR reloaded | `~/logs/vite-lab-dev.log` |
| Wall `/app/wall` | Draws 6 and 7 phones correctly with `?demo`. Real mode needs the page signed in (headless got "Sign in to the league first", expected). Use `?n=7` at noon (7 games, default is 6) | `~/scratch/live-ready-1122/demo.png`, `demo7.png` |
| Hermes live-recorder | IDLE, sleeps until earliest kickoff (capped 30 min), so goes LIVE at noon | `journalctl --user -u live-recorder` on Hermes |

- A signed-in account opened the app at 9:46 AM CT (lab log, account 564f67ef…), probably Christopher's
  Brave on the Beelink. T677 struck again then: results/standings/bracket/odds 503 at the 5 s deadline.
- 127.0.0.1:5199 works only on the Beelink. The phone uses https://twr-lab-260.secureprospective.com/app/.
- No lab release noon–~10 PM CT (game rule; restart triggers T677 timeouts).

## 2. Tracker http://192.168.1.222:8790 (done)
- Source `~/fleet/tracker` (Beelink), deployed copy Hermes `~/fleet/tracker`; code identical (md5).
- Fixed: favicon 404 (commit 31b7d6c, deployed). Hand-kept `data/plan.json` on Hermes refreshed:
  T672 done (#457/#458, lab v49), new card "Live wall for streaming (T670)", T673 text with kickoff
  times, T668 card rewritten (section 4), order = T673+T670 today → T677 → T679, new flag
  "127.0.0.1 only works on the Beelink", side workspaces now 164 (23 GB), log entries 9:55 + 10:15 AM.

## 3. Routing (done)
- `~/fleet/ROUTING.md` (not under git): Sonnet dropped (Christopher: "update the routing file to drop
  Sonnet"). T3 row is now Neon Medium → Luna high → Opus Low (AGY). AGY's `agy models` no longer lists Sonnet.
- AGY quota screen 9:50 AM (shows REMAINING): Gemini group week 56.64% left (refresh ~Sat 11/28 4 AM CT),
  5-hour 100%; Claude/GPT via AGY 0% until ~Fri 11/27 1:10 PM CT; Argon 100%.
- Argon probe (one-line prompt, Argon Low) answered at 9:52 AM: Argon is OPEN. agy-usage.py still says
  "LOCKED until 11/27": stale. Christopher: "Argon runs out fast so use it wisely."
- `~/fleet/agy-quota.json` rewritten as percent USED (the tracker reads it that way).
- GPT side is down today. Christopher: "If you run Haiku please use Claudebox, not your own."
  Saved as memory `haiku-runs-on-claudebox`.

## 4. T668 PWA auto-update: ROOT CAUSE FOUND, fix pushed, NOT released
Christopher asked to complete and verify T668 with the S8 and iPhone.

**Finding:** no phone has ever installed the update worker. On both test phones (Chrome) the app
calls `register('/app/sw.js')`, it resolves, then the registration vanishes. Cause: the v49 `sw.js`
(BUILD 157e0eade99a, embedded in the server binary) precaches 236 files; 7 were never written by the
build (`commissioner.M5lPOGmH.js`, `control.ChxWdhoY.js`, `lineup.D044rNl8.js`, `look.CEw7PUGx.js`,
`nativeAccount.B3SDfDs5.js`, `problemReport.Cm4uWFgU.js`, `registry.D1g2x8OO.js`). They are the empty
script stubs of CSS-only chunks that Vite deletes after our `generateBundle` ran. `cache.addAll` fails on
one 404, so install fails. Same file names the S25 kept requesting on 11-21. Side effect: each 404 on
`/app/assets/*` answers `Clear-Site-Data: "cache", "storage"`, so every app open wiped the phone's
storage; may explain the S8's "league server could not be reached" (TWR-V7FL-WX3B). Not confirmed.

**Fix:** branch `session/pwa-sw-shell`, commit 3be79c9d7, pushed (pre-push passed; PR not opened).
`frontend/vite.config.ts`: `pwaShell()` gets `enforce: 'post'`, and a `writeBundle` guard fails the
build if sw.js lists a file not on disk. Proof: build without the fix fails naming the 7 files;
with it 229 listed, 0 missing, BUILD e45a22cef5c1. Not merged, not on the lab.

**Ruled out:** Cloudflare. Public sw.js shows `max-age=14400` but origin sends `no-cache` and the app
registers with `updateViaCache: 'none'`; the bad list is in the binary itself.

**To finish T668 (needs Christopher's yes on timing; asked, not answered):**
1. After 10 PM CT: lab release of `origin/session/pwa-sw-shell`
   (`REF=origin/session/pwa-sw-shell ~/fleet/bin/twr-lab-release-ref.sh 50 ~/fleet/runs/labrel-v49-1121/release-v49/metadata/targets.json <run dir>`).
2. On S8 + iPhone (`~/fleet/bin/phone-chrome.py <s8|iphone> goto|eval`), open /app/ and confirm a
   registration exists and `caches.keys()` holds `twr-static-shell-<new BUILD>`.
   Note: phone-chrome `eval` does not await promises; store results on `window` and read them back.
3. A second release that changes frontend files (T677/T679 batch) → confirm both phones switch to
   the new BUILD on their own. Then merge (Christopher's sign-off) and `cc task check T668`.
- Task list text for T668 was NOT edited (`cc task` has no edit command); the tracker card carries it.

## 5. T679 (Christopher asked "anything to do now?"): answered no
Only owner-visible item: pre-game updates do not trigger a phone reload (stale preview before noon and
between the late games and SNF). Workaround: tap another game and back. Plan: one small Hydrogen/Neon
Medium job after Sunday, bundled into the same lab release as T677; no Argon needed (low risk).

## 6. Today / open
- T673 live field proof on S25 from 12:00 PM; T670 wall at `127.0.0.1:5199/app/wall?n=7`.
- Watch `grep "server live" /var/lib/twr/server.log` on CT260 (heredoc form only:
  `ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab pct exec 260 -- sh -s <<'EOF'`;
  a quoted `sh -c '…'` ran on the Proxmox host instead).
- After Sunday: T677 (startup reads), T679, T668 steps above. Gemini is the only build pool until
  GPT returns; Neon max 2 lanes; small jobs Haiku on Claudebox; Argon only for risky code.

## Leftovers
- `~/scratch/live-ready-1122/` (screenshots, harness scripts, build outputs). Delete after T668 is verified.
- Worktree `.worktrees/pwa-sw-shell` (has an untracked copied `frontend/dist`, required by pre-push).
- Vite dev server pid 557753 kept running on purpose (Christopher's wall and dev page).
- S8 adb connected at 192.168.1.122:5555; both phones left on the lab page.
