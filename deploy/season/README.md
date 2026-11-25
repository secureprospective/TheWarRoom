# Season server swap

Saturday-night swap of the Season server (Alpine, OpenRC, 192.168.1.162). Rules: CLAUDE.md "Environments".
Window: Saturday 18:00–23:59 America/Chicago, and only with Christopher's OK each time.
A real swap always needs `--christopher-ok`; a dry run needs it only outside the window (else exit 3).
Every run prints the exact command to approve.

- Plan only, sends nothing: `deploy/season/season-swap.sh <version-dir-name> --dry-run`
- Real swap, after his yes: `deploy/season/season-swap.sh <version-dir-name> --christopher-ok`
- `<version-dir-name>` is a folder under `/opt/twr/versions/`, for example `v51`.

Original flow (now stopped for a consistent backup): back up `/var/lib/twr` to `/var/backups/twr`; check the version folder exists; read the current
link; point `/opt/twr/current` at the new version; start `twr`; health check `curl` on port 5199.
It deletes nothing. If a step fails after the link flip, it prints the rollback command.

Exit codes: 0 done or dry run · 1 step failed · 2 bad arguments · 3 refused by guard (nothing sent).
Test (offline): `python3 -m unittest discover -s deploy/season -v`

Before the first real swap: confirm port 5199 is the Season server's port (it comes from the Beelink
wall URL), and that `curl` is installed on the server (Alpine: `apk add curl`).

## Rollback: data and version together

Use the exact rollback command printed by the swap, with a separate Christopher approval:
`deploy/season/season-swap.sh <swap-stamp> --rollback --christopher-ok`.
Add `--dry-run` to print the operation without contacting the server.
The stamp identifies `/var/backups/twr/twr-data-<swap-stamp>.tgz` and its `.record` file.
Keep both; the record binds the archive checksum, previous version and swapped-in version.
Old backups without records cannot use this procedure.

The swap now stops `twr` before taking its backup, so SQLite files are copied at rest.
Rollback checks the record, archive and previous folder before stopping anything. It stages
that archive, stops `twr`, moves current data to `/var/lib/twr.before-rollback-<swap-stamp>`,
restores the backup and previous version link, then starts `twr` and checks HTTP health.
It does not delete either data copy or the archive. Run only one swap/rollback at a time.
If a command fails, stop and inspect: the service may remain stopped; do not improvise
or delete saved data. HTTP success does not prove league recovery: verify sign-in and a
league read before accepting the rollback. This is upgrade recovery, not first-install recovery.

Offline integration tests use overridden `SEASON_APP_ROOT`, `SEASON_DATA_DIR` and
`SEASON_BACKUP_DIR` with fake SSH, OpenRC and curl; defaults remain the Season paths.
`make test-deploy` runs these and the existing swap guard tests without contacting a host.
