# RESUME — TheWarRoom alpha 0.10 (Claudebox head brain), Mon 2026-11-16
Updated about 1:15 PM CT (third compaction today). On "we are back": do his instruction first, else section 10 item 1.

## 1. WHAT WE ARE DOING
Proving alpha 0.10 so Christopher gets a real server + Linux client to test, with dev work continuing beside it.
Claudebox is head brain: GPT lanes (Sol, Terra, Luna) on the Beelink do the heavy work; Claude subagents only when
GPT is out. Repo: Beelink `chris@192.168.1.191:~/work/TheWarRoom` (main checkout is HIS; worktrees only).
Lab: 192.168.1.182 (from the Beelink: `ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@192.168.1.182`).
CT 258 = stress/sweep server (0.10.0 installed, league loaded). Never touch CT 257 (dev), 110 (gateway), 256.

## 2. DONE LOOKS LIKE (docs/roadmap/RELEASE-0.10.md, 15 lines)
| Line | State |
|---|---|
| 1 tag + CI + Linux package | MET (v0.10.0: verify green, Linux package built) |
| 2 both say 0.10.0 | Server MET on CT 258 (`Server build 0.10.0`). Desktop not checked. |
| 3–5 real server | NOT STARTED (waits on stress/security) |
| 6–7 real client | NOT STARTED |
| 8–11 game day | proven on CT 257 older build only |
| 12–13 update path | IN FLIGHT: Terra lane a010u launches 2:17 PM CT (see 3) |
| 14 stress | NOT PASSED (see 4) |
| 15 security sweep | NOT PASSED: 10 pass, 1 fail, 3 not proven (see 4) |

## 3. IN FLIGHT (survives compaction; all on the Beelink)
| Job | Check alive | Output | On result |
|---|---|---|---|
| Timer → Terra lane `a010c` (CPU diagnosis + perf PR), brief `~/fleet/briefs/alpha010/cpu1.md`, fires 14:13 CT | `pgrep -af "sleep 11700"` | `~/fleet/runs/pulse-a010c/{stdout.log,completion.txt}`, sessions/ mtime | Review PR; hand merge cmd (he authorized merging today's PRs — ask again for new ones) |
| Timer → Terra lane `a010u` (0.10.1 upgrade path, lines 12–13), brief `upgrade1.md`, fires 14:17 CT | `pgrep -af "sleep 4380"` | `~/fleet/runs/pulse-a010u/` | Score lines 12–13. If it says release-number row needed: re-cut decision to Christopher |
| GPT resets about 14:09 CT. If lanes start before reset they fail at start → relaunch with `~/fleet/bin/launch-pulse-lane.sh terra <lane> <brief> high` | | | |
| PR #297 (lab storm + sweep tools, head 789115453) is **DIRTY** (conflicts with #296 on main) | `gh pr view 297` | | Rebase `session/pulse-a010t` on main (worktree `.worktrees/pulse-a010t`), push, merge when green |
| PR #299 (Linux-only desktop CI, head ef373ffae) CI running | `gh pr checks 299` | | Merge when green: `gh pr merge 299 --squash --match-head-commit <head>` |
Watch with a background Bash loop (10-min max, re-arm). Launch lanes with `ssh -f … 'nohup setsid … &'`.

## 4. RESULTS (docs not in repo yet: Beelink `~/fleet/briefs/alpha010/pending-docs/`)
- `RELEASE-0.10-STRESS-RESULT.md`: 3 h, 80 devices, CT 258. Gates 1,3,4,5 PASS (0 lost/0 doubled of 1,605; back in
  7.6–21.3 s; integrity 15/15). Gate 2 FAIL (14,091 502s). Gate 6 FAIL as written (sign-ins/changes pause after
  every restart until first backup). Gate 7 not measured. Gate 8 inconclusive (+12%, sampler gap). Screens p50 1.6 s.
  Raw: lab `/root/stress-0.10.0-run2/`, Beelink `~/fleet/runs/a010-result/`. Storm (gates 9–11) not run: do after speed fix.
- `RELEASE-0.10-SECURITY-RESULT.md`: fail = check 9 (Cloudflare edge serves plain HTTP and TLS 1.2 on `twr-` hosts).
  Not proven: 6 (secret list), 11 (stalled request), 12 (needs co-commish test account). Check 5 PASSED (he rebooted CT 258 18:52 UTC).
- Also pending: `RELEASE-0.10.md` (line 15 + Linux-only/PWA note), `RELEASE-0.10-STRESS.md`, `RELEASE-0.10-SECURITY.md`
  — the last three already landed via #296 except the Linux-only/PWA note. Next docs PR: both RESULT docs + RELEASE-0.10.md + rewrite RELEASE-0.11.md for PWA.

## 5. ARTIFACTS
- Tags: `v0.10.0` = 44a663173; `v0.10.1` = 55be58b03 (#296 stresscheck/docs + #298 503-busy patch). Pushed by me with his go-ahead.
- Main: 55be58b03. Merged today: #295, #296 (cbf52e989), #298 (55be58b03).
- Lab STATE (secrets, never print): `/root/twr-lab/a010s/state.json` (0600). Lab tools copied to `/root/twr-lab/a010s/`.
- Lab R2 restore key for CT 258 backups: only in `.worktrees/pulse-a010s/.scratch/` — do NOT remove that worktree until CT 258 is gone.

## 6. CURRENT PROBLEMS (hypotheses, not conclusions)
- 502s at 80 devices: CPU ~96% of 2 cores, requests exceed the 5 s budget, write deadline also 5 s → reply lost.
  #298 fixes the "lost reply" part only. What burns CPU is unknown: admission benchmark is only 0.11 ms, so likely
  WAITING (lock / single DB connection) — unproven; a010c lane tests it.
- Memory tight: container peak 1,067 of 1,074 MB.
- 0.10.1 has same internal release number (8) as 0.10.0; `twr-upgrade` may refuse it — unknown, a010u checks.

## 7. REFUTED — DO NOT RETEST
- League-load conflict on fresh CT = order of steps (server ran first, fetched its own player IDs), not a bug. Load into empty storage works (18 min).
- `value.read` 409 = F51 player values not on the server ("player records and fitted parameters are not available on this host"); skipped with `--skip-reads value.read`.
- Beelink guard "writes to a protected path" = false match on `>` + `~/.ssh` in one line; use the ssh alias `.scratch/ssh-lab.conf`.
- Mac CI failures = GitHub had no macOS runner (moot now: Linux only).
- Sweep check 3/4 "FAIL" = tool bugs (`ss` missing in container; LAN peer test). Hand checks pass.

## 8. DECISIONS (Christopher, today)
- First release alpha 0.10; 0.10.x = one-PR patch; **0.11 = the PWA (web app incl. mobile), with the commissioner
  Update button folded in**; 0.20 = full field. 0.10 itself does not change today.
- Desktop is **Linux only**; Windows and macOS return when it makes sense (no users yet). Ignore Mac CI.
- He plans to use his last Fable 5.2 credits on the PWA build this week.
- Cloudflare fix for check 9: rule scoped to War Room host names only (he picked "recommended"). He applies it by hand (token can't).
- Stress: load is the gate, speed is baseline. Security sweep required before "permanent".
- He authorized merging #296/#297/#298/#299 and cutting 0.10.1.
- Lanes: keep model/effort lean ("dont over size the model or effort level"). Sol medium, Terra high, Haiku for mechanical collection.

## 9. LEDGER
Committed: main (above). Backbone memory commits 8f908ec, 8a8121f, e6fb573 (no-mac/PWA, PWA=0.11).
Not in repo: the pending docs in section 4 (next docs PR). This RESUME: branch `session/a010-docs`.

## 10. NEXT ACTIONS
1. Check #299 CI → merge. Rebase #297 (`.worktrees/pulse-a010t`) on main, push, merge when green.
2. After 14:13/14:17 CT confirm lanes a010c and a010u started (`ls ~/fleet/runs/pulse-a010c/sessions`); arm watchers.
3. a010u result → score lines 12–13; if release row needed, bring re-cut decision to him.
4. Open docs PR: both RESULT docs + RELEASE-0.10.md note + RELEASE-0.11.md rewritten around PWA + Update button.
5. a010c result → review perf PR; then re-run stress (and storm gates 9–11) on the faster build.
6. Re-run sweep check 9 after he deploys the Cloudflare rule.
7. Tonight after 7:15 PM CT game: week 10 points vs MFL on CT 257 (claudeos `~/twr-d2/tools/w9cmp.py` pattern).
8. Later: real server + client (lines 3–7); rotate CT 257 keys (T621); reap CT 258 (he runs `twr-down`).

## 11. NEEDS CHRISTOPHER
- Cloudflare WAF custom rule `twr https and tls13 only`, Block:
  `(starts_with(http.host, "twr-") and (not ssl or cf.tls_version ne "TLSv1.3"))` — written from memory; unverified.
- Later: `twr-down` for CT 258 (agents cannot remove CTs).

## 12. ENVIRONMENT
- Times in Central (machines UTC, CT = UTC−6). Never print `/etc/twr/*`. Commits over ssh: `GOTOOLCHAIN=go1.26.11`,
  per-worktree `GOLANGCI_LINT_CACHE`; gh at `$HOME/vendor/gh/bin`. Agent hook blocks `pct reboot` etc.
- Filing check FAIL only on Bee's dangling symlinks in `~/scratch/bee-resume-v14-*` (not mine; left).

## 13. HONEST STATUS
0.10.0 is not fit to be permanent: overloaded at 80 devices and sign-ins pause after restarts. Nothing tested at
a realistic 32–40 devices. Update path unproven. Speed fix not yet identified. Earliest real-server stand-up: after a
perf fix + re-run stress, realistically not today.
