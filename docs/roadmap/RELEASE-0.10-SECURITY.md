# Alpha 0.10 — security sweep (line 15 of RELEASE-0.10.md)
**Ruled by Christopher 2026-11-16:** the server is not a permanent fixture on the host until this sweep passes.
Standing rule: security is proportionate. This is a hobby league, but the server must never be the weak point on
the home network, and nothing here changes the Proxmox host.

The protections below already exist in the code and installer. The repo's own docs say they have never been proven
on a real install (`deploy/server/README.md` 130–141). The sweep is that proof. Run it against an installed 0.10.0
server from three places. Every line must pass.

## From the container console
1. The server runs as the non-root `twr` user, and `/var/lib/twr` is mode 0700.
2. The database, TLS key and authenticator key are mode 0600; tunnel credentials are 0640. List names and modes
   only. Never print contents.
3. `ss -lntup` shows only 127.0.0.1:8443 (app) and 127.0.0.1:20241 (tunnel readiness). No SSH, nothing on the LAN address.
4. From inside the container, a connection to another machine on the home LAN is refused, and so is IPv6.
   DNS to 1.1.1.1 and outbound HTTPS still work.
5. After a container reboot, both services return by themselves and `twr-status` shows zero problems.
6. After the stress run, `server.log` holds no passwords, enrollment or recovery codes, bearer tokens or key material
   (search for the test accounts' known values; report counts only).

## From another machine on the home LAN
7. Ports 22, 8443 and 20241 on the container's address are all unreachable.

## From the public address
8. Unknown paths and anything like `/admin`, `/metrics`, `/status`, `/healthz` return 404.
9. Plain HTTP is not served, and TLS 1.2 or older is refused.
10. More than 10 sign-in attempts in a minute from one address get a 429 with Retry-After; a correct sign-in works
    again after the wait.
11. A request body over 256 KiB and oversized headers are refused; a slow, stalled request is dropped.
12. An owner account and a co-commissioner account each try a commissioner-only action: refused, and nothing changes.
13. Without signing in, no league data comes back from any `/v1/` read (only sign-in, join and recovery answer).
14. A cross-site form post to `/join` is rejected.

## Recorded, not a gate
- The sign-in throttle is per address and held in memory. Note what happens across a restart.
- Lifetimes of each token type.
- A list of every outside host the server calls.

Results go in `docs/roadmap/RELEASE-0.10-SECURITY-RESULT.md`: pass/fail per line with the evidence.
Any fail is fixed in a 0.10.x patch and that line re-run.
