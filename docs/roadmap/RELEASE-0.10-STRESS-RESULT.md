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
