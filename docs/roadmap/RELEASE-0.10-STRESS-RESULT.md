# Alpha 0.10 — stress test result (server 0.10.0)
**Run:** 2026-11-16, 15:32–18:42 UTC, lab container CT 258 (2 cores, 1 GiB, no swap), server build 0.10.0
(commit 44a663173), 32 owners / 80 devices through the public tunnel address. `value.read` skipped (player
values, F51, is not in 0.10). Evidence: lab `/root/stress-0.10.0-run2/`, Beelink `~/fleet/runs/a010-result/`.
**Verdict: NOT PASSED.** Durability held. The server is overloaded at 80 devices.

| Gate | Result | Evidence |
|---|---|---|
| 1 No crash or out-of-memory kill outside the 15 scripted stops | PASS | 0 OOM kills; no panic lines. One extra server start at 18:42:33 UTC, at the end-of-run check (not confirmed as scripted). |
| 2 No 502 while the server is up | **FAIL** | 14,091 answers were 502; 11,600 of them outside the scripted stop windows. First at 15:32:59 UTC. |
| 3 Every confirmed action lands exactly once | PASS | 1,605 confirmed, 1,605 in the ledger, 0 lost, 0 doubled; 0/0 at each of the 15 per-stop checks. |
| 4 Back within 2 minutes after each forced stop | PASS | All 15: down 7.6–21.3 s. |
| 5 Database integrity after every restart | PASS | `integrity ok` 15 of 15 and at the final check. |
| 6 Backups keep completing; changes never paused for lack of a backup | **FAIL as written** | Backups completed throughout. But after every restart the status showed changes and sign-ins paused until the first backup was checked ("none since the server started"). Length of the pause not measured. |
| 7 Live scores never older than two poll intervals | NOT MEASURED | No game window was replayed in this run. |
| 8 Memory does not climb steadily | INCONCLUSIVE | First steady phase 573 MB; last 10 minutes 642 MB (+12%, limit 10%); 237 MB at rest after load. Sampler lost 16:53–17:42 UTC. Peak server memory 879 MB; container peak 1,067 MB of a 1,074 MB limit. |

## Answers by status (whole run)
| Kind | 200 | 502 | 503 | 409 | other |
|---|---:|---:|---:|---:|---:|
| Screen reads | 48,984 | 7,097 | 10,033 | 221 | 19 |
| Confirmed clicks | 1,605 | 5,978 | 512 | 193 | 1 |
| Previews | 1,799 | 1,016 | 187 | 13 | 2 |

## Speed baseline at 80 devices (successful answers; budget is 200 ms screen / 100 ms click)
| Kind | p50 | p95 | p99 |
|---|---:|---:|---:|
| Screen reads | 1.6 s | 4.5 s | 5.1 s |
| Previews | 2.3 s | 4.8 s | 5.1 s |
| Confirmed clicks | 3.1 s | 5.0 s | 5.2 s |

Not measured at 8, 32 or 64 devices: the tool is fixed at 80.

## Findings to fix
1. Overload at 80 devices: both cores at about 96%; requests run out of their 5 s budget and the reply is lost
   (502). PR #298 (0.10.1) turns these into a clean 503 with Retry-After. It does not make the server faster.
2. What uses the CPU is not yet known (a lane is set to profile it).
3. Sign-ins and changes pause after every restart until the first backup is checked.
4. A fresh install cannot load the league once the server has run (it fetches its own player list first). The
   README now warns; the installer should do it in the right order.
5. Every refusal is the same generic 409 text; the reason is only in the server log.
6. The memory sampler dies at the first container stop (lab tool).
7. Request storm (gates 9–11) not run: the server already refuses at 80 devices. Run it after the speed fix.

## Follow-up: 0.10.1 scripted-stop 502 attribution (2026-11-17)

**Run:** server build `0.10.1-11-gc4d38df6c` (`c4d38df6c`), 32 owners / 80 devices, 15 scripted `kill9`,
container-stop and service-restart cycles. Evidence: `~/fleet/runs/pulse-lab2/RESULT.md`; lab OUTDIR
`/root/stress-main-1116` (read-only review).

- The public-path observers recorded 1,282 HTTP request 502s plus 1,200 SSE-open 502s (2,482 total),
  all inside the broad scripted-cycle markers and none outside. One HTTP 520 was also inside. The 502 body
  was `error code: 502`. The run's 15 recoveries took 11.16–24.62 seconds and reached the public endpoint.
- These 502s came from the Cloudflare Tunnel/edge proxy path when its configured origin was unavailable,
  not from a running `twr-server`. The lab client used the public hostname; the cycle script explicitly
  SIGKILLs the server, stops the whole container, or stops the service. During a SIGKILL or container stop
  there is no Go process capable of writing an HTTP response. For service stops, `main` cancels the hosted
  context and `ServeHosted` closes the HTTP server. `cloudflared.initd` intentionally leaves the Tunnel
  running across a server restart, and the installed ingress forwards directly to `https://127.0.0.1:8443`.
  The captured report has no Cloudflare-Ray or connector request log, so it identifies the Tunnel boundary,
  not which Cloudflare-side component synthesized each error body.
- The same run recorded 19,192 request 503s. The broader header observer saw 20,569 503s: 19,284 carried
  `Retry-After: 2`; 1,285 did not. The missing-header responses included the server's backup-paused message
  and clustered around restarts. The run predates main commit `4f76130f7` (#316), which routes the server's
  503 refusal paths through `writeUnavailable`. Current source and `TestWriteUnavailableSetsRetryAfter`
  cover that application response, but the new public behavior has not been stress-tested on that commit.

**Disposition — proposal pending Christopher's approval; the existing gate is unchanged here.** An absent
origin cannot return `503` or `Retry-After`; achieving that would require a separately surviving proxy or an
edge fallback, not a Go shutdown handler or an origin setting. For line 14, separate planned-stop availability
from the “no gateway errors while up” gate: keep the 20-second pre-stop burst in the up-server measurement;
start the excluded interval at the actual stop command and end it only after a known origin-ready response;
record 502/520s inside that interval separately and judge them against the existing recovery-time gate. Once
ready, any 502/520 remains a failure. Count origin-generated 503s separately and require every one to carry
`Retry-After`; rerun this check on a build containing #316 before claiming that header gate passes.
