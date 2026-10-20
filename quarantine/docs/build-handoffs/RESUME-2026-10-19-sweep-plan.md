# TheWarRoom — RESUME (2026-10-19, after the clean sweep and the hardening plan)

1. **WHAT:** Christopher's cleanup week before any new direction (real people want native leagues).
   - The clean sweep is done.
   - The store and command layers are mapped, reviewed and planned.
   - **The single plan is `docs/build-handoffs/Hardening_Plan_2026-10.md`. Read it first.** It holds
     every ruling, finding (F1–F15), phase (0–12), ETA and backlog item.
   - Evidence for phases 8–12: `docs/build-handoffs/Command_Protocol_Audit_2026-10.md` (gaps G1–G9).
   - Worktree: Beelink `~/work/TheWarRoom/.worktrees/code-sweep`, branch `session/code-sweep`.
   - Never touch the main checkout `~/work/TheWarRoom`.
2. **AGENTS:** none running.
   - Sol 6.1's store review is finished: `~/fleet/runs/sol61-warroom-store-review/stdout.log`.
   - The Sol research launcher is `~/fleet/bin/launch-sol61-super.sh <brief>`.
3. **STATUS**

   | Item | State |
   |---|---|
   | Sweep code (`8b04e2bb`, flaky-test fix included) + test timeout 60m (`84ff9e24`) | verify PASSED, on GitHub |
   | Docs commits `66c402b6`, `03d6acde`, `bc36254b`, `2088030a`, `3045ad7e` | committed locally; push of `66c402b6` mid-verify; the rest not pushed |
   | Merge of `session/code-sweep` | waiting for Christopher's go (plan phase 0) |
   | Filing check | PASS |

4. **IN FLIGHT:** a background `git push` (`~/logs/TheWarRoom/sweep/push.sh`), started about 20:00 UTC.
   - Its pre-push `make verify` takes about 45 min.
   - Check: `tail -c 200 ~/logs/TheWarRoom/sweep/push.log`. It ends with `PUSH_EXIT=N`.
   - When it's done, run `push.sh` again (the same script) to push the remaining docs commits.
5. **ALSO RUNNING:** CT105 `python3 -m http.server 8765 --bind 192.168.1.105`, serving `/root/warroom-map/`.
   - Pages: `index.html` (codebase diagram), `Hardening_Plan_2026-10.md`, `Command_Protocol_Audit_2026-10.md`.
   - Stop it when Christopher is done: find the PID with `ss -ltnp | grep 8765`, then `kill` it.
6. **REFUTED / DO NOT REDO:**
   - Don't trim tests by line count; make them faster (ruling).
   - Don't split out `native.db` now (client/server is deferred).
   - The observations secondary indexes are used by queries (Sol), so benchmark before touching them.
   - `Manifest.Resolve`, `LoadCrosswalk`, `GetCrosswalkReport`, `model.Measures`, `Params.Values` and
     `params.DefaultSet` all have real callers. Keep them.
7. **DECISIONS:** all are in the plan's "Rulings so far" (1–7). Don't relitigate them.
8. **NEXT, IN ORDER:**
   1. Christopher has "more things to cover in this session". Wait for his next topic.
   2. Push the remaining docs commits once the current push finishes.
   3. When he says go: squash-merge `session/code-sweep` (plan phase 0).
   4. Then plan phase 1, the impact map.
9. **ENV:**
   - `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOMEMLIMIT=1500MiB GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false"`
   - Run one golangci-lint or verify at a time.
   - Use `scp`, and `ssh -f setsid` for long jobs.
   - Never run the production binary on the Beelink.
   - Claude-OS is reached from the Beelink with `ssh claudeos`.
10. **HONEST:**
    - The plan is agreed but nothing in it is built.
    - The ETAs are estimates: about 7–8 days for phases 1–7 and 9–13 for phases 8–11.
    - Nothing from the sweep has been walked in the live app on Claude-OS.

## Update (2026-10-19, evening)
- Deep audit and plan v3 (blocks A–E) are in `Hardening_Plan_2026-10.md`.
- **Block A** (fast push) is on `session/fast-push` (stacked on `session/code-sweep`):
  - parallel root tests
  - changed-only pre-push
  - GitHub CI with the full race gate
- **Timings (measured):** root race tests 40 min → 10–11 min; root tests without race 2 min; a full-gate push 15 min.
- **Merge:** a no-click merge once CI is green (Christopher's ruling). Merge the sweep first (it holds the docs).
