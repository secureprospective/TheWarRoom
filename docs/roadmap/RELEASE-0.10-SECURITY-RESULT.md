# Alpha 0.10 — security sweep result (server 0.10.0)
**Run:** 2026-11-16 about 19:00 UTC, lab container CT 258, `deploy/lab/security-sweep.sh` (PR #297) plus hand
checks. Raw output: lab `/root/stress-0.10.0-run2/security-sweep.txt`.
**Verdict: NOT PASSED.** 10 pass, 1 fail, 3 not yet proven.

| # | Check | Result | Evidence |
|---|---|---|---|
| 1 | Runs as `twr`; data folder 0700 | PASS | process UID twr; `/var/lib/twr` twr 0700 |
| 2 | Key and database file modes | PASS | database, TLS key, authenticator key 0600; tunnel credentials 0640 |
| 3 | Only loopback listeners | PASS (by hand) | TCP: 127.0.0.1:8443 and 127.0.0.1:20241 only. No SSH. The tool failed this line because `ss` is not installed in the container; checked with `netstat`. UDP sockets are the tunnel's outbound ones and DHCP. |
| 4 | Cannot reach the home network; internet works | PASS (by hand) | From inside: Beelink, router, lab host and Proxmox host all refused; 1.1.1.1:443 and DNS work; no global IPv6 route. The tool mis-scored this line. |
| 5 | Comes back by itself after a container restart | PASS | Christopher restarted CT 258 at 18:52 UTC; 90 seconds later `twr-status` showed all 8 lines OK, "Everything is working. Nothing to do." |
| 6 | No secrets in `server.log` after the stress run | NOT PROVEN | 0 matches for the secrets the tool knows; the list of test secrets is not confirmed complete. |
| 7 | Ports 22, 8443, 20241 closed from the LAN | PASS | all three refused |
| 8 | Unknown and admin-style paths 404 | PASS | five paths, all 404 |
| 9 | No plain HTTP; TLS 1.2 refused | **FAIL** | `http://` serves the page (200, no redirect) and a TLS 1.2 connection gets 200. Both are answered at the Cloudflare edge, not by the server (which is TLS 1.3 only on loopback). Fix is a Cloudflare setting for these host names: Always Use HTTPS and minimum TLS 1.3. |
| 10 | Sign-in throttle | PASS | 10 wrong tries 401, then 429; correct sign-in 200 after the wait |
| 11 | Oversized body and headers refused; stalled request dropped | PARTLY | 300 KiB body 400; 16 KiB header 431; stalled upload not proven dropped by the server (client timed out at 60 s) |
| 12 | Owner and co-commissioner cannot do commissioner-only actions | NOT RUN | needs a co-commissioner test account |
| 13 | No league data without sign-in | PASS | four reads, all 401 |
| 14 | Cross-site post to `/join` rejected | PASS | 400 |

## Recorded
- Sign-in throttle is held in memory: a server restart clears it.
- Token lifetimes: device 7 days idle / 30 days absolute; agent read 30 days, execute 24 hours; enrollment 48
  hours; recent-auth 5 minutes; join links do not expire (revocable); recovery codes single-use, no expiry.
- Outside hosts: Cloudflare, DNS 1.1.1.1, ESPN and Sleeper live feeds, GitHub (nflverse stats), R2, release hosts.

## Tool fixes needed (lab script)
Use `netstat` when `ss` is missing; fix the IPv6 route test; check the LAN peer from the host before scoring.
