# TheWarRoom — RESUME (compact-safe, 2026-10-22)

## 1. What we are doing
Hardening session on TheWarRoom: **#1 security, #2 hyper-minimal code**, no new features, method before the thing.
- Active repo: Beelink `chris@192.168.1.191:~/work/TheWarRoom` (checkout is on `session/league-history-pwa` — DO NOT TOUCH).
- **Our worktree:** `~/work/TheWarRoom/.worktrees/hardening-method`, branch `session/hardening-method`, cut from `origin/main` @ `d3dabe7a`.
- CT105 copy `/mnt/storage/claudebox/projects/TheWarRoom` is stale (July) — ignore.

## 2. Agents + harnesses
- **Sol 6.1 builds, Claude reviews, Christopher owns rulings / live test / merge.**
- Launchers (Beelink): `~/fleet/bin/launch-sol61-medium.sh <brief>` and `launch-sol61-super.sh <brief>` (= high). Run dir `~/fleet/runs/sol61-<brief-name>/` → `stdout.log`, `completion.txt`, `sessions/`. Launch with `ssh -f`.
- Briefs: `~/fleet/briefs/`. Today's: `sol61-warroom-hardening-plan-2026-10-22.md` (research), `sol61-warroom-plan-review-2026-10-22.md` (review, verdict NOT READY → fixed).
- Read-only snapshot of main for agents: `~/fleet/runs/warroom-main-snap-2026-10-22` (REF d3dabe7a).
- Go: `/usr/local/go/bin` (prepend to PATH). Fresh worktree needs `pnpm install && pnpm build` in `frontend/` before push/verify (frontend/dist).

## 3. Gates / status
| Item | State |
|---|---|
| Sol research pass | done, verified by Claude |
| Codebase one-page map | done, facts corrected after Sol review |
| Session plan page | done, Sol's 6 changes folded in |
| Sol high review of plan | done; all fact errors confirmed + fixed |
| Method written into Hardening Plan + rule docs | **written, NOT committed, `make verify` NOT run** |
| B0 baseline / B1 inventory / B2 / B3 | not started |

## 4. Artifacts
- `~/Downloads/warroom-one-page.html` — codebase map (Brave: file:///home/chris/Downloads/…)
- `~/Downloads/warroom-session-plan.html` — session plan
- `~/Downloads/hardening-method.diff` — 166-line diff of the doc changes
- Sol outputs: `~/fleet/runs/sol61-sol61-warroom-hardening-plan-2026-10-22/stdout.log` (20 KB), `~/fleet/runs/sol61-sol61-warroom-plan-review-2026-10-22/stdout.log` (22 KB)

## 5. Verified facts (main @ d3dabe7a) — use these, not older docs
- Go prod 104,588 lines / 634 files; Go tests 84,354 / 479; frontend hand-written TS/TSX 21,854 + CSS 2,479; generated 23,208 (wailsjs 16,682 + `registry/*.gen.ts` 6,526).
- `internal/composition`: 6,938 prod vs 25,145 test (3.6:1) — dedup watch, cut LAST, each cut names surviving test.
- 133 bound methods / 15 APIs (`testdata/bindings/manifest.txt`; `Bind:` at `internal/composition/run.go:68`, list 110–127). 23 have no frontend caller (20 `commissioner.Target*Native*`/Target*, `control.TargetSeason`, `market.TargetMarket`, `platform.TargetCheckMoves`) — **KEEP, catalog them**.
- Go catalog exists: `internal/app/catalog/command.go` (`Unset` = not publishable). Extend it; never a second list.
- DB: 3 files (thewarroom/history/whatif, `-dev.db` for dev builds) in `~/.config/TheWarRoom` (0700). 51 prod tables, 6 named indexes. **foreign_keys ON, synchronous FULL, recursive_triggers ON** (`internal/db/pools.go:24`), WAL, 1 writer + ≤10 readers.
- F-findings: F1–F8, F10, F11 fixed in source; F12–F14 partial; **F9, F15 open**.
- No inbound listener, no os/exec in prod. Ways in: IPC, 4 CLI flags, imported files.
- Deep slice for this session: gunzip copies at `internal/archive/archive.go:188` and `internal/store/history/archive.go:161`, both unbounded `io.ReadAll`.
- Other confirmed candidates (later): non-MFL clients follow redirects (only `mfl/client.go` sets CheckRedirect); archive import via `os.DirFS` no symlink guard (`internal/app/sources/archive.go:30`); keyring late-write (`internal/mflkey/store.go:53`); CI lacks pnpm audit + gitleaks; Vite ^3.0.7 / Vitest 0.25.8.

## 6. Refuted — do not retest
- "Foreign keys never switched on" — FALSE, they are ON.
- "72 tables" — FALSE, 51 production.
- "F1–F15 are current" — most are fixed (see §5).
- "Hand-written total 135,447" — FALSE, 128,921 (registries are generated).
- "IPC is the only way in" — FALSE (flags, files).

## 7. Decisions (Christopher, 2026-10-22)
- D1 Method lives INSIDE `Hardening_Plan_2026-10.md` (no fourth plan).
- D2 Quality contract + architecture review govern; Agent Codex reference only.
- D3 Sol builds, Claude reviews.
- D4 23 uncalled methods stay → CLI catalog rows.
- D5 Product is for regular people: commands in plain words; `domain.verb` grammar re-judged at Phase 8 panel.
- D6 Hygiene: delete spent harnesses (trace callers first); walkable tree; lean front matter/index → feeds CLI.
- D7 Living Database Growth Catalog (`docs/data-layer/Database_Growth_Catalog.md`, to write).
- D8 Sol's 6 changes approved: no shrink quota · one deep slice · rule docs first · throwaway DB profile for live test · one final build for tests/review/live · extend existing catalog.
- D9 Christopher: "ready" — execute after compaction.

## 8. Ledger state
Uncommitted in the worktree (written, approved as plan, awaiting verify): `docs/build-handoffs/Hardening_Plan_2026-10.md` (Method section, status, phase 10 gate, live-test line), `AGENTS.md`, `CLAUDE.md`, new `docs/standards/{quality-contract,architecture-review}.md`. Only `HANDOFF.md` (this resume) is committed on the branch. Nothing pushed.

## 9. Next actions, in order
1. `cd ~/work/TheWarRoom/.worktrees/hardening-method`; build frontend (`cd frontend && pnpm install && pnpm build`), then `PATH=/usr/local/go/bin:$PATH make verify`. Report result.
2. Commit the doc changes (`docs: hardening method + rule docs; quality contract governs [AI-assisted]`). Do not push until Christopher says.
3. B0: record baseline (gate results, counts in §5, inherited failures named, not fixed).
4. B1: write Sol brief → 133-method inventory on `internal/app/catalog` (plain-language columns, 23 marked keep) + deep trace of the two gunzip paths + seed Database Growth Catalog with §5 F-statuses. Launch Sol (medium unless told), verify output against code.
5. B2: bring Christopher one fix choice: single capped gunzip, cap set from real archive body sizes (measure on a COPY of history.db).
6. B3: Sol builds on branch; Claude reviews diff + callers; same final build for tests/review/live; Christopher live-tests on a throwaway profile.

## 10. Relay notes
- ssh `chris@192.168.1.191` works from CT105. Use `ssh -f` for Sol launches; no `pkill -f` over ssh.
- Artifacts for Christopher: scp to Beelink `~/Downloads`, give `file://` path for Brave.
- Merge style is squash; Christopher pushes main himself.

## 11. Honest status
Plan is agreed; no code changed. `make verify` on the doc branch has not run. Archive size cap is unknown until measured on a DB copy.
