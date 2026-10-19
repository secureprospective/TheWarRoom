# TheWarRoom: deep audit (2026-10-19)

**Verdict:** the foundations are sound. It doesn't need a rewrite. The hardening plan
(F1–F15, G1–G9) is still the right spine, but it is not enough by itself. This audit found
**four native-league issues that block real leagues**, two speed problems you can measure today,
a cheap security fix, and one structural problem: the app's main package is too big for a
person to maintain.

Scope: I read the code on `session/code-sweep` (99k Go lines, 35k TypeScript), ran a security
scan on the Go and frontend dependencies, cross-built the app for Windows, measured the running
dev app on Claude-OS, and read the model's fit report.
**Not done:** a hands-on walk through the screens (see §6).

---

## 1. What is solid (keep it)

- **MFL key:**
  - It is kept in the OS keyring and is hidden in every log line and error message.
  - Requests that carry the key never follow redirects.
  - A crash while setting the key is caught, so the key never reaches a log.
- **SQL:** every query passes its values as parameters; nothing is built by pasting strings together.
- **Links that open the browser:** each one is checked before it opens.
- **Frontend:**
  - no raw HTML injection and no `eval`
  - strict TypeScript, no `any`
  - 225 accessibility labels
  - no clickable `div`s
- **Locks:** taken one after another, never nested, so I found no deadlock risk.
- **Startup:** 147 ms (measured). Background work stays off the window's main thread.
- **Native ledger:** revisions, retry keys and corrections that never edit history. Authorization
  falls back to the league's own records.
- **Shipped app's dependencies:** no known vulnerabilities (`pnpm audit --prod` is clean).
  Go modules are current.

---

## 2. New findings, ranked (not in the hardening plan)

### Native league: must fix before real people play

| # | Finding | Evidence | Fix |
|---|---|---|---|
| **N1** | **A bid placed after an auction's 24 hours have run out is accepted and restarts the clock.** The rulebook ends the auction at 24 hours; the snipe rule covers hours 20–24. The bid check looks at the snipe window but never asks "is this auction already over?". With the clock only running while the app is open (G9), an auction left unprocessed overnight can be stolen. | `commissioner/bid_execute.go:91–131` (place/raise: no closed check); `bid_due.go:46`; `nativecheck/bids.go:307`; Rulebook §Sniping | Refuse place/raise once `standingAt + stand` has passed. Write the failing test first. |
| **N2** | **Your league identity comes from the data folder's path plus your user id.** Restore a backup onto a new machine, a new username or a new folder, and you are no longer the commissioner of your own league. Plan phase 6's gate ("restore on a clean profile reproduces the league") would fail. | `commissioner_app.go:17–22`; the creator is stored as that hash (`commissioner/creation.go:159`) | Create a random installation id once, store it in the database, and back it up with the data. Add a commissioner recovery path. Decide this before phase 6. |
| **N3** | **A native league runs on one computer.** Other owners cannot join from their own machines. Today a "real" native league means the commissioner enters every owner's moves. This is a fact about the product, not a bug, but it decides what "real people running a native league" can mean this season. | G8/G9; one local principal | Your call: (a) commissioner-run leagues for now, or (b) bring the client/server decision forward. |
| **N4** | **Native owners have no screens.** About 20 owner actions are built and tested but nothing in the UI reaches them: setting a lineup, bidding, trading, waivers, the lineup check, standings and results. | the IPC methods with no frontend caller (list in the appendix) | Move this from the backlog into a phase before any native league. |

### Speed: measured on the running app (Claude-OS, 3 h 09 m up)

| # | Finding | Evidence | Fix |
|---|---|---|---|
| **P1** | **Sitting idle, the app uses 22% of a CPU core and 877 MB of memory.** The busy threads are Go's own, not the window's. The likely cause: the native clock wakes **every second** and re-reads every native league's whole ledger twice (waivers, then bids). That is F12's cost multiplied by 86,400 a day. | 41 m 33 s of CPU in 3 h 09 m; 6.25 s per 30 s at rest; `native_waivers_app.go:61` (1 s ticker) → `ProcessDueWaivers` / `ProcessDueBids` → `nativeRecords` | Sleep until the next due time and wake when something is written. Confirm the cause with a CPU profile first. |
| **P2** | **The hourly refresh redoes heavy work that hasn't changed:** the crosswalk takes about 30 s every hour (125,017 links, same result each time), the archive read 17 s, the model fit 7 s, the engine run 10 s. | Claude-OS log, 10:21 → 13:21 | Skip a step when its inputs are unchanged, keyed by input hash. |
| **P3** | **One refresh lock can freeze the main screens.** The board (`TargetSnapshot`), What-if and the trade builder all wait on the lock that the refresh holds while it talks to MFL. The refresh can take up to 2 minutes, and MFL's slow-down replies back off for up to 60 s. | `target_app.go:26`, `refresh_app.go:84`, `refresh_app.go:21`, `mfl/client.go:58–66` | Screens read the last published snapshot, and the refresh publishes in one step. This fits with F1/F2 in phase 4. |
| **P4** | **F15 is visible on disk:** the dev `whatif-dev.db` is **84 MB**; the live one is 1.8 MB. | `ls` on Claude-OS | Already phase 7. This is the evidence. |

### Security

| # | Finding | Evidence | Fix |
|---|---|---|---|
| **S1** | **Go 1.26.4 has 14 known vulnerabilities in code paths the app actually runs** (net/http, TLS, URL parsing). All are fixed in Go 1.26.9. | `govulncheck ./...`: "Your code is affected by 14 vulnerabilities from the Go standard library" | Move to Go 1.26.9 and add `govulncheck` to `make verify`. About 1 hour. |
| **S2** | **The frontend build tools are 2–3 years old** (Vite 3, Vitest 0.25, TypeScript 4.6): 27 advisories, 3 of them critical. **All are in development tools; none ship in the app.** The risk is the Beelink while `wails dev` is running (any web page could send requests to the dev server). | `pnpm audit` 27 / `--prod` 0 | Upgrade Vite, Vitest and TypeScript as one step and re-run the bundle budget. |
| **S3** | Smaller defence-in-depth gaps:<br>- no Content-Security-Policy in the webview<br>- the MFL host is taken from MFL's own reply and not pinned to `*.myfantasyleague.com`, so the API key would go to whatever host that reply names<br>- MFL response bodies are read with no size limit<br>- database files are 0644 (the 0750 folder protects them today; a server install would not) | `frontend/index.html`, `mfl/client.go:262–290`, `client.go:116`, `ls -l ~/.config/TheWarRoom` | Half a day for all four. |

### Reaching other owners (the alpha promise)

| # | Finding | Evidence | Fix |
|---|---|---|---|
| **R1** | **The app doesn't build for Windows,** and it has never been built for macOS. Your league mates are likely on one of those. | `GOOS=windows go build .` → `undefined: syscall.Flock` (`instance_lock.go:18`). Everything else compiles. | Platform-specific lock files, then a Windows and macOS build. |
| **R2** | **No CI.** The only gate is the pre-push hook on the Beelink. There is no `.github/`. | repo root | A GitHub Actions job for lint, test and the Windows, macOS and Linux builds. It also takes load off the Beelink. |
| **R3** | No update path for other owners (no installer or self-update). | Makefile `release` only tags | Decide this with R1, before the alpha. |

### Human maintainability

| # | Finding | Evidence | Fix |
|---|---|---|---|
| **M1** | **`package main` holds 24.6k lines in 241 files.** The `App` struct has about 100 fields and 20+ mutexes. The `*_reading.go` files hold real presentation logic, despite the comment saying "business logic lives in the engine… never here". This is the biggest obstacle for a human maintainer, and the reason the root tests take 40 minutes (one package can't be tested in parallel). | `app.go:43–162`; `wc` | Move the readings into `internal/readings/<area>` packages and break `App` into feature services. Do it inside phase 2 (faster tests): it is the same work. |
| **M2** | The frontend has no linter, so React-hooks mistakes (stale data) go uncaught. | `frontend/package.json` | Add ESLint with only the react-hooks rule. |
| **M3** | Shutdown waits for the week workers but not the other background jobs (facts compile, pending-trade refresh) before closing the databases. | `app.go:349–363` | Track every worker in one WaitGroup. |

### Does the engine deliver its promise?

| # | Finding | Evidence |
|---|---|---|
| **E1** | **Honest scoreboard:**<br>- **The talent ranking clearly beats** "last season × age" (RB 0.76 vs 0.53, TE 0.79 vs 0.56).<br>- **The value ranking roughly ties it** (QB 0.70 vs 0.72, RB 0.70 vs 0.71, WR 0.75 vs 0.75).<br>- **Kickers: the model is worse** (0.07 vs 0.46).<br>The holdout is one season. The edge today is talent and rookies, not dynasty value. | `docs/fit/Fit_Report.md` "Against today's board" |
| **E2** | I couldn't find a kicker fallback to the simpler rule, even though the fit says the model loses there. **Check this before trusting kicker values.** | `internal/model` has a single `"K"` case (`data.go:245`) |

---

## 3. Changes to the hardening plan

| Where | Add | Days |
|---|---|---|
| New **phase 1b**: security quick wins | S1, S3, govulncheck in verify | 0.5 |
| Phase 2 (faster tests) | M1: split `package main`; M2 | +2–3 |
| Phase 4 (stop wrong state) | N1 (late bids), P3 (screens read the published snapshot), M3 | +1 |
| Phase 6 (backups), first | N2: stored installation id and commissioner recovery | +0.5 |
| Phase 7 (fast reads) | P1 (clock wakes on due), P2 (skip unchanged hourly work), CPU profile before and after | +1 |
| New **phase 7b**: native owner screens | N4, built from the command catalog once phase 10 lands, or a plain version now | 2–3 |
| New **phase 7c**: other platforms and CI | R1, R2, S2; R3 decision | 1.5–2 |
| New **UX walk** | A full click-through of every screen on a test VM (§6) | 0.5 + fixes |
| **Your decision** | N3: commissioner-run native leagues now, or the client/server decision brought forward | — |
| Check | E2 kicker fallback | 0.25 |

**New total:** about 15–19 working days for phases 1–7c (was 7–8), then phases 8–12 as planned.

## 4. Recommended order

1. Phase 0: merge the sweep.
2. Phase 1b: the security quick wins (an hour of real work, and it removes 14 live vulnerabilities).
3. Phase 1: the impact map, now covering these findings too.
4. **Phase 4 + N1 and phase 6 + N2 before any native league starts.**
5. The rest as listed.

## 5. What would change these conclusions

- P1's cause is inferred from the code and the thread spread. A CPU profile could point somewhere else.
- N1 assumes the rulebook's "24 hours" also applies to native leagues. If native leagues are meant
  to differ, it becomes a rules setting instead.

## 6. Not covered: the UX walk

- **Why I didn't do it:**
  - The Claude-OS desktop had your browser open, so I didn't drive the app there.
  - One screenshot I took before I saw that caught your browser window. I deleted it straight away
    from CT105, the Beelink and Claude-OS, and didn't look further.
- **What's left:** "user friendly" needs a real click-through of every screen. Plan it as its own
  step on a VM nobody is using.
- **What the code shows:** error messages are plain English, there are loading and fallback
  states, and there are accessibility labels. That is a good sign, but it isn't a walk-through.

## Appendix: owner IPC methods with no screen

TargetNativeBidAction, TargetNativeCapCheck, TargetNativeContractQuote,
TargetNativeFranchiseSchedule, TargetNativeHouseRules, TargetNativeLineup, TargetNativeLineupCheck,
TargetNativeLineupSet, TargetNativePlayerScore, TargetNativeReserveCheck, TargetNativeResults,
TargetNativeRosterCheck, TargetNativeScorePreview, TargetNativeStandings, TargetNativeTradeAction,
TargetNativeWaiverAction, TargetNativeWeek, plus TargetCheckMoves, TargetCorrections,
TargetLeagueCalendar, TargetLeagueSchedule, TargetMarket, TargetSeason.
GetCrosswalkReport and LoadCrosswalk are kept on purpose (they have real callers).
