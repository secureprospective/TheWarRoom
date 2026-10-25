# Server status dashboard

The league server's health, read at the CT console by the commissioner or deputy
([CHECKLIST.md](../../deploy/server/CHECKLIST.md)). The console is reached only through
the Proxmox web page, which is LAN/VPN-only, so the dashboard is VPN-only without any
new listener or firewall rule. Nothing here is routed through the Tunnel, and nothing
sends a message: no email, push, webhook or alert service. Someone looks.

Flow: `twr-status` (root, CT console) → `twr-server status` as the service user →
`Console.Checks` → `Render`.

| Line | Where it comes from |
|---|---|
| League server | `Console`: GET an unknown route on the origin (`127.0.0.1:8443`) with the installed origin certificate and the Tunnel peer header, expecting the application's 404, the installer's own readiness probe; also warns 30 days before that certificate expires. |
| Tunnel | `Console`: cloudflared's `/ready` on `127.0.0.1:20241` with at least one edge connection. |
| Backups | `Source`: live `offsite.Manager.Status`: write coverage, saved time, newest failure. Unconfigured backups read as a problem, never as healthy. |
| History | `Source`: both audit chains (league, sign-in) verified afresh on each look. |
| Clock | `Console`: the kernel NTP state the due worker trusts (`leagueservice.HostClock`). |
| Disk | `Console`: space available to the service user in `/var/lib/twr`; a problem below 1 GB, the room backups stage in. |
| Last restore, Last sign-everyone-out | `Source`: the newest `restored_sign_ins_reset` and `sign_ins_reset` entries in the sign-in history (`accounts.Store.LastResets`). |

`Source` is served by the running hosted server on `-status-socket`, an owner-only
(0600) Unix socket inside the service's own 0700 folder, a separate listener from the
public HTTPS handler. It answers one route, `GET /v1/status`, with `Report` JSON, and
holds what only the server knows. `Console` checks what is visible from the container
even when the server is down, so a stopped server reads as problems, not a blank
screen.

Details may hold private operator facts (an R2 error naming the bucket, entry counts):
that is acceptable only because the report never leaves the container. `Validate`
refuses any non-loopback address and the HTTP clients ignore proxy settings, so the
dashboard cannot be turned into an outbound notifier.

Evidence: `TestFaultsShowOnPrivateDashboardOnly` (cmd/twr-server) injects each fault and
checks the dashboard, the public handler and every dial. `TestBackupStatesAndResetNotes`
covers the backup states, notes and the all-clear summary. `deploy/server/test-alpine.sh`
runs the installed `twr-status` in disposable Alpine/OpenRC: healthy server, real faults
(no backups, no Tunnel), a stopped server and the root-only refusal.
