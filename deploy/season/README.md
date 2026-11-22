# Season server swap

Saturday-night swap of the Season server (Alpine, OpenRC, 192.168.1.162). Rules: CLAUDE.md "Environments".
Window: Saturday 18:00–23:59 America/Chicago, and only with Christopher's OK each time.
A real swap always needs `--christopher-ok`; a dry run needs it only outside the window (else exit 3).
Every run prints the exact command to approve.

- Plan only, sends nothing: `deploy/season/season-swap.sh <version-dir-name> --dry-run`
- Real swap, after his yes: `deploy/season/season-swap.sh <version-dir-name> --christopher-ok`
- `<version-dir-name>` is a folder under `/opt/twr/versions/`, for example `v51`.

Steps: back up `/var/lib/twr` to `/var/backups/twr`; check the version folder exists; read the current
link; point `/opt/twr/current` at the new version; restart `twr`; health check `curl` on port 5199.
It deletes nothing. If a step fails after the link flip, it prints the rollback command.

Exit codes: 0 done or dry run · 1 step failed · 2 bad arguments · 3 refused by guard (nothing sent).
Test (offline): `python3 -m unittest discover -s deploy/season -v`

Before the first real swap: confirm port 5199 is the Season server's port (it comes from the Beelink
wall URL), and that `curl` is installed on the server (Alpine: `apk add curl`).
