# TheWarRoom — RESUME (2026-10-19 evening: audit done, plan v3, block A pushed)

1. **WHAT:** the cleanup week. The deep audit is done and folded into the plan as v3 (blocks A–E, test in blocks).
   Block A, fast push, is built and pushed.
   - Beelink worktrees:
     - `~/work/TheWarRoom/.worktrees/code-sweep` (`session/code-sweep`, the sweep plus docs)
     - `.worktrees/fast-push` (`session/fast-push`, stacked on code-sweep)
   - Never touch the main checkout.
2. **DOCS (on fast-push, all pushed except the last commit):**
   - `docs/build-handoffs/Deep_Audit_2026-10-19.md`
   - `Audit_Recommendations_2026-10-19.md`
   - `Hardening_Plan_2026-10.md` (v2 + v3 appended)
   - Sol's review: Beelink `~/fleet/runs/sol61-warroom-audit-triage/stdout.log`
   - Medium-thinking launcher: `~/fleet/bin/launch-sol61-medium.sh`
3. **IN FLIGHT:** a push of `25ea0343` (the CI pnpm fix). It changes the build files, so it runs the full gate (~15 min).
   - Started ~15:05 local.
   - Check: `tail -c 300 ~/logs/TheWarRoom/fastpush/push2.log` (ends `PUSH_EXIT=N`).
   - Then: `gh run list -R secureprospective/TheWarRoom -b session/fast-push -L 1`.
   - CI failed twice before the fix: pnpm/action-setup could not find a pnpm version, because `packageManager` is only in `frontend/package.json`.
4. **BLOCK A STATUS:**

   | Item | State |
   |---|---|
   | Root tests parallel (t.Parallel; 4 global-state files serial); subtests parallel where they build their own fixture | done; 3 race runs, the last shuffled, clean |
   | `mflkey.NewMemory` (per-test key store; go-keyring mock is an unguarded global) | done |
   | `scripts/prepush.sh` (docs → nothing; Go → lint + shuffled non-race tests on changed packages + importers; frontend → its checks; build files → full `make verify`) | done |
   | `.github/workflows/verify.yml` (full make verify incl. race) | waiting on the green run |

   Measured:
   - root race 40 min → 10–11 min
   - root non-race 2 min
   - full-gate push 15 min
   - docs push 1 s
   - Go push ~3–4 min (estimated from its parts)
5. **REFUTED / DO NOT REDO:**
   - Under 5 min with `-race` locally is impossible: the root tests total 4,224 CPU-seconds, about 9 min on 8 cores. Race lives in CI.
   - Don't add new files under 40 lines: the bloat ratchet (tiny_files) fails.
   - Pushing from a heredoc with an apostrophe inside ssh '...' breaks: scp a file instead.
6. **RULINGS (Christopher, 2026-10-19):**
   - GitHub Actions CI: yes.
   - No-click merges for phases with no visible change: yes.
   - Push under 10 min, under 5 preferred.
   - Test in blocks, not per phase. Cut test overlap with evidence; stop writing too many tests.
   - Goal-OS sandbox parked (Hermes PARKED). He is weighing bare metal for ClaudeBox.
   - Memory: `project_warroom_fast_push_rulings.md`.
7. **OPEN QUESTIONS FOR CHRISTOPHER:**
   - The go to merge the sweep (phase 0).
   - Native leagues this season: commissioner-run on one machine, or bring client/server forward (ruling 8 / A16).
8. **NEXT, IN ORDER:**
   1. Wait for Christopher's hardware inventory (the ClaudeBox bare-metal idea). Size it against builds, VMs, Bee/Sol and live gates.
   2. Check the push and CI. On green, and once he gives the sweep go:
      - squash-merge code-sweep first, then fast-push
      - the fast-push merge needs no click
   3. Then block B, starting with the test-overlap measurement (per-test coverage).
9. **ENV:**
   - `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOMEMLIMIT=1500MiB GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false"`
   - gh is at `~/.local/bin/gh`. Use `ssh -f setsid` for long jobs.
   - The Claude-OS desktop is in use by Christopher (browser): don't screenshot or drive it.
10. **HONEST:**
    - CI has not passed yet.
    - The Go-code push time is not yet measured end to end.
    - Nothing from the plan beyond block A is built.
    - The CT105 page server on :8765 is still up (stop it when he's done).
