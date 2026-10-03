# RESUME — TheWarRoom Stage 2 built, live gate pending (2026-10-03)

Stage 1 is merged. Stage 2 (league truth) is built and committed on
`session/stage2-league-truth`, and is waiting for its Claude-OS live gate. This file replaces
`RESUME-2026-10-03-stage1-gate.md` as the starting point.

## 1. What we are doing
- **Plan:** `docs/build-handoffs/Core_Build_Plan_2026-10.md`. Stage 2's scope and design are the
  L1–L5 table above its gate.
- **Repo:** `~/work/TheWarRoom` on the Beelink, branch `session/stage2-league-truth`, cut from
  `main` at `7e63557`. Unpushed; push with `git push -u origin session/stage2-league-truth`.
- **Go:** `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB`
- **Gates:** `make lint`, `make verify` (the pre-push hook). Bloat baseline: comment 19,
  provenance 7, tiny files 27, dupl 38.

## 2. Christopher's direction this session (do not relitigate)
- Stage 1: merge approved (1A). The real `~/.config/TheWarRoom/history.db` from the accidental
  production launch is **kept** (2A).
- **Wireframe, not blood and guts.** Transactions, what-if and the refresh plumbing are
  placeholders until the core (Stages 3–8) makes them relevant. Clean the architecture only; build
  no features. Saved as memory `warroom-periphery-is-placeholder-core-first`.
- **After the core, power rankings (M2) are how the core gets polished:** human judgment (his and
  others') on M2 shows what is and isn't calibrated right.
- Product answers (to be built at placeholder depth): one what-if sandbox; what-if kept across a
  refresh; refresh at launch plus a button; Christopher launches the first fresh build himself.

## 3. Ledger
| Ref | What |
|---|---|
| `main` `7e63557` | Stage 1, PR #4 rebase-merged. The pre-rebase tip is tag `archive/session/stage1-measure-store` (`d09d972`); hashes in the Stage 1 docs resolve through it |
| `d09b735` | Stage 2 scope and design (L1–L5) |
| `1b7f44b` | Stage 2 build: `state.Mirror` plus `leagueView`, `RefreshLeague`, `league.Discover`, `rulebook.Sync`, `whatif.db`, `SeasonYear` deleted, the INJURED_RESERVE fix, discovery cache (15-minute TTL), the rail button |
| (next) | This resume |

Hermes tasks: T368 checked (Stage 1 merged); **T369** is open for Stage 2.

## 4. Stage 2 as built
- **`state.Mirror`** is in `thewarroom.db` as a single row, `league_mirror` (season, sha256, JSON
  snapshot).
  - It serves `state.Reader`, which M1, M2, the inspector and scouting all read through.
  - Cap is salaries (taxi and IR at the league's percentages) plus salary adjustments.
- **`RefreshLeague`:**
  1. `league.Discover` reads the season from the league history under id 14432.
  2. `rulebook.Sync` writes a new version only when MFL's config changed, which brings in the
     franchise names.
  3. It fetches rosters (`normalize.Contract`, which needs no players list) and salary
     adjustments (signed money).
  4. `Mirror.Replace` writes only when the snapshot changed.
- **When it runs:**
  - An empty mirror is filled during startup.
  - Otherwise the refresh runs from `OnDomReady` (never in `-probe`), and from "Refresh from MFL"
    under LEGACY in the rail.
- **What-if:** the state store and coordinator live in `whatif.db`, seeded from the mirror. The
  m4, transactions, feed, calendar and standings/schedule caches read it.
- **Not built (placeholder; documented in L3/L4):**
  - a Reset button;
  - the "built on MFL as of" label;
  - a phase derived from MFL;
  - picking up a season rollover before the next launch.
- **Evidence from the dev probe:**
  - First refresh: 32 s. MFL takes about 5 s per request; there are no 429s.
  - Live league: 1,450 rostered players (1,313 ROSTER, 83 TAXI_SQUAD, 54 INJURED_RESERVE). The old
    database held 831.
  - The second launch was ready in 84 ms.

## 5. In flight
- **Claude-OS VM is running** and sits at the lightdm greeter, waiting for Christopher to log in.
  - Check it: `virsh -c qemu:///session list --all`; `ssh claudeos 'DISPLAY=:0 xdpyinfo >/dev/null && echo ok'`.
  - The production build `v0.5.0-130-g1b7f44b` is already at `claudeos:~/twr-gate/stage2/thewarroom`.
  - `claudeos:~/.config/TheWarRoom/` still holds the Stage 1 gate data. Move it aside before the
    gate, so the run starts from an empty config folder like Christopher's fresh launch.
- Nothing else is running.

## 6. Next actions, in order
1. Wait for Christopher to log in to Claude-OS. Then move
   `~/.config/TheWarRoom/*` on claudeos to `~/twr-gate/stage1-config/`.
2. Launch the app: `ssh claudeos 'cd ~/twr-gate/stage2 && DISPLAY=:0 setsid -f ./thewarroom > run1.out 2>&1 < /dev/null'`.
   Expect about 35 s of first refresh before the window, then check:
   - the board, Power, Transact and Trade all show the 32 MFL franchise names (screenshots);
   - pressing "Refresh from MFL" twice ends at "up to date · 2026";
   - `league_mirror` holds one row, and `rulebook_versions` doesn't grow on repeat refreshes;
   - cap for 3 franchises from the mirror equals Σ salary (taxi and IR at their %, currently 100)
     plus adjustments, computed independently from the archived rosters and salaryAdjustments
     bodies in history.db. Send Christopher those 3 to spot-check against MFL's own cap screen.
3. Save evidence to `~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage2-2026-10-03/`, then
   shut Claude-OS down.
4. Record the gate in the plan under Stage 2, push, open the PR, and get Christopher's merge
   go-ahead.
5. **L5:** after the merge, archive the Beelink's two test databases, `thewarroom.db` and
   `thewarroom-dev.db` with their `-shm`/`-wal`/`.lock`/backup files, to
   `~/archive/2026-10-warroom-dbs/`.
   - Log the move in `~/MOVED.md`, `~/archive/MANIFEST.md` and CT105
     `/root/.claude/backbone/context.md`.
   - Keep `history.db`, and leave `history-dev.db` alone.
   - Then Christopher launches the new build (`build/bin/thewarroom`), on his own desktop, himself.
6. Update T369, then go to Stage 3 (the crosswalk), which is core work.

## 7. Lessons (do not relearn)
- **Never run `build/bin/thewarroom` on the Beelink.** Check stamps with `strings`.
- pre-commit sees untracked files, so a stage lands as one commit.
- Any float summed over a Go map's range is non-deterministic in its last bits.
- MFL is about 5 s per request from here. Fewer requests is the only lever.
- `-probe` must not start background work: `domReady` owns the launch refresh.
