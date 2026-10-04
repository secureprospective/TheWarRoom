# RESUME — TheWarRoom Stages 2 and 3 merged; next is the Week-9 checkpoint (2026-10-03)

Stage 2 (league truth) and Stage 3 (crosswalk) both passed their live gates and are merged. This
file replaces `RESUME-2026-10-03-stage2.md` as the starting point.

## 1. What we are doing
- **Plan:** `docs/build-handoffs/Core_Build_Plan_2026-10.md`. The next item is the **Week-9
  checkpoint (R3)**, then Stage 4 (signals into the store, one at a time).
- **Repo:** `~/work/TheWarRoom` on the Beelink. `main` is `f6db51f`. The working branch is
  `session/week9-checkpoint`, cut from it; it holds one uncommitted-then-committed change (see §3).
- **Go:** `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB`
- **Gates:** `make lint`, `make verify` (the pre-push hook). Bloat baseline: comment 19,
  provenance 7, tiny files 27, dupl 38.
- **App for Christopher:** `build/bin/thewarroom` is built from `main`, `v0.5.0-138-gf6db51f`.

## 2. Christopher's direction (do not relitigate)
- **Claude-OS stays running.** Never shut it down after a gate: "stop turning it off, its light
  weight and not in anyones way." Memory `leave-claude-os-running`; the project CLAUDE.md is
  updated on this branch. He said it will be up when we come back from compaction.
- Auto-login on Claude-OS: asked three times, never answered. Leave it alone; don't ask again
  unless a gate is blocked by the login screen.
- Wireframe, not blood and guts, for the periphery (transactions, what-if, refresh plumbing).
  The core (Stages 3–8) gets the real work. Memory `warroom-periphery-is-placeholder-core-first`.
- After the core, power rankings (M2) are how the core gets polished, through human judgment.
- MFL counts every salary adjustment its export lists, including those stamped 2023–2025:
  Denver is $123.91 on MFL's own cap screen, which is the app's figure.
- Merges are on his go-ahead after a passing live gate. Stages 2 and 3 both got it.

## 3. Ledger
| Ref | What |
|---|---|
| `0af02fb` on main | Stage 2 merged (PR #5). Pre-rebase tip: tag `archive/session/stage2-league-truth` |
| `f6db51f` on main | Stage 3 merged (PR #6). Pre-rebase tip: tag `archive/session/stage3-crosswalk` |
| `session/week9-checkpoint` | CLAUDE.md: Claude-OS stays running; this resume |

Hermes tasks: T369 and T370 checked (Stage 2 merged; Christopher launched the fresh build: the
app opened instantly and the rosters loaded in about 5 s). T371 (Stage 3) is checked on merge.

## 4. What Stages 2 and 3 built
- **Stage 2:** `state.Mirror` (MFL's league, one row in `thewarroom.db`); `RefreshLeague` (launch
  and a rail button); the season from MFL; what-if in `whatif.db`, seeded from the mirror
  including salary adjustments; one `ready` gate for every IPC method. Its live gate caught three
  bugs, all fixed: early IPC calls, MFL shuffling its scoring-rule blocks, and two caps per team.
- **Stage 3:** DynastyProcess links every id it carries into `history.player_ids` (124,906 links,
  19 id types), with guards for shared ids and reused low MFL ids. Control → Crosswalk shows the
  match report. ScoreLeague links from the crosswalk it already fetches. Live: rostered 1,444 of
  1,450 matched; free agents 744 of 796.
- **L5 done:** the Beelink's test databases are in `~/archive/2026-10-warroom-dbs/`, logged in
  MOVED.md, MANIFEST.md and CT105's context.md. `history.db` was kept.

## 5. In flight
Nothing is running. Claude-OS was shut down after the Stage 3 gate (before the new rule);
Christopher will have it up after compaction. Its `~/.config/TheWarRoom/` holds the Stage 3 gate
data (the Stage 2 final-gate databases plus the crosswalk); earlier gate configs are in
`~/twr-gate/` on it.

## 6. Next actions, in order
1. Check that Claude-OS is up: `virsh -c qemu:///session list --all`; `ssh claudeos who`.
2. Commit this branch's CLAUDE.md change with this resume if it isn't already (`git log -1`).
3. **Week-9 checkpoint (R3).** The plan: "Stages 0–3, then today's engine recomputed through
   `scoring_runs` on fresh data. M1 and M2 show their known limits: points-based, scouting capped,
   rookies at zero. Deliver before the league's Week 9 trade deadline."
   - Find the league's real Week 9 trade deadline from MFL (the league export or calendar). The
     date seen in the app on Claude-OS is what-if test data, not MFL's.
   - Check that M1 and M2 label their known limits on screen; fix any missing label.
   - Have Christopher launch `build/bin/thewarroom` (`v0.5.0-138-gf6db51f`) and press Score
     League, so the board is recomputed on his fresh data. Hand it to him as numbered steps.
   - Record the checkpoint in the plan.
4. Then Stage 4: signals into the store, one at a time (R10). Read its section in the plan first.

## 7. Lessons (do not relearn)
- **Never run `build/bin/thewarroom` on the Beelink.** Check stamps with `strings`.
- `go build ./` in the repo root writes a `TheWarRoom` binary there. Build to `/tmp`, or delete it.
- On Linux, Wails runs OnStartup alongside the page load, and OnDomReady on its message loop.
  IPC calls go through `ready()`; never block OnDomReady.
- MFL shuffles array order between requests (the rules export). Canonicalize before comparing.
- MFL reuses low player ids (team units 151–782, commissioner-created 08xx). Never trust an id
  join on its own; check the name.
- SQLite `LIKE` is case-insensitive: `TYPE=players` also matches `TYPE=playerScores`.
- `/tmp` on the Beelink is a 14 GB tmpfs in RAM. If it fills, tool output is lost; write output to
  `~/scratch` and read it from there. CT105 filled it once tonight
  (`~/fleet/inbox/ASK-claudebox-tmpfs-full-p45neg-2026-10-03.md`).
- Any float summed over a Go map's range is non-deterministic in its last bits.
- MFL is about 5 s per request from the Beelink at times; fewer requests is the only lever.

## 8. Evidence
- Stage 2 gate: `~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage2-2026-10-03/`
- Stage 3 gate: `~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage3-2026-10-03/`
  (`analysis-inputs/` holds the DynastyProcess CSV and the MFL player list the design was
  measured on.)
