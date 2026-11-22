# RESUME — game day Sun 11-22 (Tom session on the Beelink, ~9:45–10:40 AM CT)

For Claudebox to plan from. Written by Tom (Claude Opus 5.5 on the Beelink).
Christopher's own words are quoted; everything else is what was observed.

## COMPACT POINT 11-22 ~10:40 AM CT — WAIT FOR CHRISTOPHER'S FIRST JOBS
- New role (Christopher, 10:40 AM): "I have Claudebox setting up your plan now, after compaction I will
  give you your 1st jobs - you are the headbrain, use agy and Haiku on claudebox when appropriate."
  → Tom is head-brain for this workstream: plan, brief, judge, verify the tree. Bulk building goes to
  AGY (`~/fleet/bin/agy-run.sh <job> <brief> "<model>" <workdir>`; Neon/Hydrogen/Flash; Argon sparingly)
  and Haiku as Claude Code on Claudebox (`ssh claudebox`, claude 2.1.330). Not Tom's own subagents for
  Haiku. GPT side is down. Routing: `~/fleet/ROUTING.md`. Check `python3 ~/fleet/bin/agy-usage.py` first.
- Claudebox's plan will arrive from Christopher; it outranks NEXT below.
- In flight: nothing of mine running. Vite pid 557753 (127.0.0.1:5199, Christopher's wall/dev page)
  stays up on purpose; stop with `kill 557753`. S8 adb connected 192.168.1.122:5555.
- Pending question to Christopher: may Tom release `session/pwa-sw-shell` to the lab after 10 PM for T668?

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
