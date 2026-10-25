# Daily and away checklist

For the commissioner and the deputy. The daily look takes two minutes and changes
nothing in the league. The league server never sends emails, texts or alerts to
anyone, so looking is the only way to know it is well.

## How to look at the league server

1. If you are away from home, turn on the WireGuard VPN on your laptop first. The
   Proxmox page only opens at home or over the VPN. That is on purpose: nobody else
   can see this page.
2. Open the Proxmox web page and sign in.
3. Select the machine (the node) on the left, then **Shell**.
4. Type this, with the league container number from the envelope instead of `120`,
   and press Enter:

   ```sh
   pct exec 120 -- sh /usr/local/libexec/twr-status
   ```

5. Read the lines. Each one starts with **OK**, **PROBLEM** or **NOTE**:

| Line | What it tells you |
|---|---|
| League server | The league program is running on the machine. |
| Tunnel | Owners can reach the league from the internet, through Cloudflare. |
| Backups | The newest backup is saved and checked off-site, so at most 15 minutes of changes could ever be lost. |
| History | Nobody has changed the league or sign-in history behind the app's back. |
| Clock | The time is right, so bid and waiver deadlines run on time. |
| Disk | There is room on the machine for the league and its backups. |
| Last restore, Last sign-everyone-out (NOTE) | When those last happened and why. Nothing to do. |

The last line says **Everything is working. Nothing to do.** or how many problems
there are.

## Every day

1. Look at the league server, as above.
2. If the last line says everything is working, you are done.
3. If a line says **PROBLEM**, do what the **What to do** line under it says. Deal with
   problems from the top down: the first one often causes the others. You will only
   ever be asked to do one of three things:
   - **Look again** after a few minutes. Most problems clear by themselves.
   - **Reboot the container.** On the Proxmox page, select the league container (its
     number) on the left, choose **Reboot** at the top right and confirm. Wait five
     minutes, then look again. Nothing is lost by rebooting.
   - **Call Christopher.** Say which line shows PROBLEM and read out its words, or take
     a photo of the screen.

Do not replace the server, restore a backup, sign everyone out or delete anything
because of the dashboard. Those are in [RESTORE.md](RESTORE.md), only for a machine
that is gone or a sign-in that is in the wrong hands.

## Before Christopher goes away (Christopher, about 15 minutes)

1. Look at the league server together. The deputy does it, start to finish,
   while you watch. Every line should read OK.
2. Check that the deputy's VPN opens the Proxmox page from outside the house (for
   example over a phone hotspot).
3. Check the sealed envelope ([RESTORE.md](RESTORE.md)) is complete. It holds the
   league container number this checklist needs.
4. If the R2 backup key was created with an expiry date, make sure it is not during the
   trip ([BACKUPS.md](BACKUPS.md)).
5. Tell the deputy how to reach you while you are away.

## While Christopher is away (the deputy)

- Look at the league server once a day, as above.
- If owners say they cannot get into the league, look at it then too.
- When you call Christopher, the dashboard's words tell him what he needs. You do not
  need to understand them.

## When nobody can look (both away, or no internet)

Nothing needs doing. The league keeps running. If backups stop, the server pauses
league changes by itself before more than 15 minutes could be lost, and resumes by
itself once a backup is checked again. Look at the league server when you are back.

## What the dashboard reads (for Christopher)

`twr-status` runs `twr-server status` as the service user. It never dials anything
outside the container: the server's private socket `/var/lib/twr/status.sock` (owner-
only, inside the service's own folder, a separate listener from the HTTPS origin that
the Tunnel forwards to), the origin itself at `127.0.0.1:8443` (verified with the
installed origin certificate, the same readiness probe the installer uses, plus its
expiry), cloudflared's `/ready` at `127.0.0.1:20241`, the kernel clock state, and free
space in `/var/lib/twr`. The socket reports what only the running server knows: live
off-site coverage and the newest backup error from `offsite.Manager.Status`, both audit
chains verified afresh, and the newest restore and incident reset read back from the
sign-in history (slice 37 records them there; the restore reason carries the saved
time of the backup it came from). Details in
[internal/serverstatus/README.md](../../internal/serverstatus/README.md).
