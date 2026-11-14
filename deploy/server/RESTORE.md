# Replacing a lost league server, and other incidents

This is for the commissioner or the deputy (the deputy holds the sealed copy of the
restore key). Follow it step by step. You do not need to type anything that is not
written here. If a step shows an error, stop and call Christopher; nothing you have
done up to that point damages the backups.

Goal: the league is back within **four hours**, missing at most the last **fifteen
minutes** of changes.

## Which section do I need?

| What happened | Go to |
|---|---|
| The league server machine is dead, destroyed or stolen, and will not come back | [A. Replace the server](#a-replace-the-server-on-the-spare-machine) |
| One league is damaged or was changed by mistake, and the server is fine | [C. Restore one league](#c-restore-one-league) |
| A league should leave this server | [D. Delete a league](#d-delete-a-league) |
| A league should move to another TheWarRoom server | [E. Move a league to another server](#e-move-a-league-to-another-server) |
| Someone may be using another person's sign-in (lost laptop, leaked password, strange changes) | [B. Sign everyone out](#b-sign-everyone-out) |
| The internet, Cloudflare or the power is down, but the machine is fine | Wait for it to come back. Do not replace the server. |
| The server is running but says changes are paused | Look at the dashboard and do what it says ([CHECKLIST.md](CHECKLIST.md)). Do not replace the server. |

## Before you ever need this (Christopher, once)

Keep these ready, and walk through section A on a practice spare once a year:

- A spare machine with Proxmox installed, its firewall turned on, the isolated league
  VLAN available, and its clock synchronised ([README.md](README.md) prerequisites).
- On that spare, in `/root/twr-release/`: the current release bundle
  (`twr-server-alpine-amd64.tar.gz`) and `proxmox.sh` from the same release.
- A sealed envelope for the deputy holding: this page printed; the restore key (the
  line beginning `AGE-SECRET-KEY-1`, from `warroom-restore-key.txt`); the Cloudflare
  sign-in; the spare's Proxmox sign-in; the league hostname (for example
  `league.example.org`), the Cloudflare zone, the R2 bucket name and its S3 endpoint
  hostname; and a free container number and the VLAN number for the spare.

## Time budget

These are planning figures, not measured on real Proxmox hardware yet:

| Step | Allow |
|---|---|
| 1. Make sure the old server is gone, open the envelope | 15 minutes |
| 2. Two new Cloudflare keys | 30 minutes |
| 3. Install the new server on the spare | 45 minutes |
| 4. Restore the league | 20 minutes |
| 5. Check the report, first backup finishes | 10 minutes |
| 6. Tell the owners | 10 minutes |
| Spare time for problems | about 1 hour 50 minutes |

## A. Replace the server on the spare machine

### 1. Make sure the old server is really gone

If there is any chance the old machine comes back (power returns, someone plugs it
in), unplug its network cable or power now. Two league servers must never run at once.
The installer checks this too: it refuses to take over the league's web address while
the old server is still connected to Cloudflare.

### 2. Make two new keys in Cloudflare

Sign in to the Cloudflare dashboard.

**a. A new backup key.** The old one was on the lost machine, so replace it.

1. Open **Storage & databases → R2 object storage → Overview → Manage R2 API tokens**.
2. Find the token named `WarRoom backups` and **delete** it.
3. Choose **Create Account API token**. Name it `WarRoom backups`. Select **Object Read
   & Write**, then **Apply to specific buckets only**, and select only the league's
   backup bucket. Leave the expiry empty.
4. Create it. Write down the **Access Key ID** and the **Secret Access Key** on paper.
   The secret is shown only once.

**b. A one-time installer key.**

1. Open **My Profile → API Tokens → Create Token → Create Custom Token**.
2. Name it `WarRoom installer`. Add three permissions: **Account → Cloudflare Tunnel →
   Edit**, **Zone → DNS → Edit**, **Zone → Zone → Read**. Under zone resources, choose
   the league's zone only.
3. Create it and copy the token. You will delete it again in step 7.

### 3. Install the new server on the spare

1. Sign in to the spare's Proxmox web page. Select the machine (the node) on the left,
   then **Shell**.
2. Type this and press Enter:

   ```sh
   bash /root/twr-release/proxmox.sh /root/twr-release/twr-server-alpine-amd64.tar.gz
   ```

3. Answer the questions with the values from the envelope. Accept the suggested
   answers in brackets by pressing Enter. For the public hostname, give the **same**
   league hostname as before. For **Replacing a lost league server at that hostname?**
   answer `yes`. Paste the installer key from step 2b when asked (it stays hidden).
4. Wait. It ends with `Installed: https://…` after about ten to thirty minutes. The
   message about moving the league from the desktop app does not apply: you restore
   instead, in the next step.

If it says **the old server is still connected**, go back to step 1. If it says the
hostname **does not point to a TheWarRoom server**, stop and call Christopher.

### 4. Restore the league

1. Still in the Shell, type `pct enter` followed by the container number you chose,
   for example `pct enter 120`, and press Enter.
2. Type this and press Enter:

   ```sh
   sh /usr/local/libexec/twr-restore
   ```

3. It asks, in order:
   - the R2 S3 endpoint (hostname only, from the envelope);
   - the backup bucket name;
   - the **new** Access Key ID from step 2a;
   - the **new** Secret Access Key from step 2a (hidden as you type);
   - the restore key from the envelope, the whole line beginning `AGE-SECRET-KEY-1`
     (hidden as you type; capital or small letters both work);
   - optionally a date (YYYY-MM-DD) to go back to an older backup, for example from
     before a bad change. Press Enter to use the newest backup.
4. It downloads the newest backup (or the newest saved on or before your date), unlocks it, checks it and installs it. This takes a
   few minutes. The restore key is only held in memory and is never saved on the server.

If it says **the restore key does not unlock this backup**, check the key letter by
letter and run step 4 again. If it says **this server already holds a league**, you are
on the wrong container: stop and call Christopher.

### 5. Check the report

A good restore prints lines like these:

```
Unlocked sets/20261025T140500.000000000Z.tar.gz.age, saved 2026-10-25 14:05:00 UTC.
League history matches the checkpoint saved with it.
League history: 1234 entries, fingerprint 3f9a…
Sign-in history: 56 entries, fingerprint 77c1…
Evidence files: 89, fingerprint a01e…
League "Legacy NFL" (native.legacy): 1200 entries, replay fingerprint 5d2b…
Signed out all 32 accounts: earlier sign-ins, helper tokens, printed recovery codes and unused invitations no longer work.
Restore installed. …
```

- Take a photo of the report and send it to Christopher. The fingerprints let him
  compare the restored league against his own records.
- The **saved** time is how current the league is. Anything changed after it is lost.
- A line beginning `Skipping` is fine when an `Unlocked` line follows. It means an
  empty or damaged backup was passed over in favour of an older good one.

Wait about five minutes. Until the first backup of the restored league is uploaded
and checked, the server pauses changes and sign-ins with a plain explanation. Type
`sh /usr/local/libexec/twr-status` and press Enter: when **Backups** reads OK, the
league is open again. Its **Last restore** line keeps the saved time for later.

### 6. Tell the owners

Send something like this to the league:

> The league server was replaced after a hardware failure. Please sign in again with
> your password and authenticator. Your old printed recovery codes no longer work; if
> you lose your authenticator, ask the commissioner to reset your account. Anything
> you changed after 14:05 UTC on Sunday may be missing, so please check your roster
> and redo it.

Use the **saved** time from the report.

### 7. Tidy up

- Put the restore key back in its envelope and reseal it. Nothing needs deleting on the
  server: the key was never saved there.
- In Cloudflare, delete the `WarRoom installer` token (**My Profile → API Tokens**).
- If the installer printed **Delete the old Tunnel**, open **Zero Trust → Networks →
  Tunnels** and delete the tunnel it named.
- If the old machine was **stolen** rather than broken, tell Christopher. Its disk held
  the league, password hashes and authenticator secrets (see [STORAGE.md](STORAGE.md));
  owners should change their league passwords.

## B. Sign everyone out

Use this when a sign-in may be in the wrong hands. Everyone's devices and helper
tokens stop working at once; passwords, authenticators, printed recovery codes and the
league itself stay as they are.

1. In the server's Proxmox web page, select the machine, then **Shell**, then type
   `pct enter` and the league container's number.
2. Type this and press Enter:

   ```sh
   sh /usr/local/libexec/twr-reset-sign-ins
   ```

3. Type a short reason, for example `lost laptop`. It is kept in the sign-in history.
4. It prints `Signed out every device and helper on … accounts.` The league is back
   within a minute. Tell the owners to sign in again.

If one person's password or authenticator is lost, use that person's printed recovery
code, or the console account recovery in
[cmd/twr-server/README.md](../../cmd/twr-server/README.md), instead.

## C. Restore one league

Use this when one league needs to go back to its last backup (or an older day) and the
server itself is fine. Only that league changes: every other league, and the sign-ins of
everyone not in that league, stay exactly as they are.

1. In the server's Proxmox web page, select the machine, then **Shell**, then type
   `pct enter` and the league container's number.
2. Type this and press Enter:

   ```sh
   sh /usr/local/libexec/twr-restore-league
   ```

3. Type the league's id, then the restore key from the sealed copy (it does not show as
   you type), then press Enter for the newest backup or type a day as `YYYY-MM-DD`.
4. It prints `League … is restored from its backup saved …` and how many owners were
   signed out. The league is back within a minute. Tell that league's owners to sign in
   again. The league as it was before the restore is kept for 30 days.

## D. Delete a league

1. Open the container's console as in C, then type:

   ```sh
   sh /usr/local/libexec/twr-delete-league
   ```

2. Type the league's id, then `DELETE` and the id again to confirm.
3. It prints `League … is deleted from this server` and the day its file is removed. Until
   that day (30 days) the file is kept in `/var/lib/twr/archive`; call Christopher if it
   must come back. Other leagues are unchanged.

## E. Move a league to another server

The league travels as its own file. Sign-ins stay on each server, so on the new server every
owner, and the commissioner, makes a new sign-in from their own one-time claim link; their
franchise, roster, contracts and history are exactly as they were.

1. **Old server:** open its container's console as in C, then type
   `sh /usr/local/libexec/twr-move-league`, the league's id, and `1` to start the move.
   It prints the **move file** and the **move code**. From now on the league can be looked
   at on the old server but not changed.
2. **Copy the move file** to the new server's container. On the Proxmox host's **Shell**:
   `pct pull <old container> <move file> /root/league.twrleague`, then
   `pct push <new container> /root/league.twrleague /root/league.twrleague`, then
   `rm /root/league.twrleague`. (If the new server is on another machine, copy the file
   there first, the same way you copy any file.)
3. **New server:** in its container's console type
   `sh /usr/local/libexec/twr-receive-league`, then `/root/league.twrleague`, then the move
   code (it does not show as you type). It prints one claim link per person. Send each
   person only their own link; each link works once.
4. **Old server:** run `sh /usr/local/libexec/twr-move-league` again, the league's id, and
   `2` to finish. The league leaves the old server; its file is kept 30 days in
   `/var/lib/twr/archive`, then the server deletes it. Delete the move file from
   `/var/lib/twr/outgoing` and from the new container's `/root`.

Changed your mind before step 4? Choose `3` instead: the league can change on the old server
again. Do not then use that move file on any server.

## What the tools do (for Christopher)

`twr-move-league` runs `twr-server export-league`, `finish-move` or `cancel-move` with the
service stopped. Export writes `leagues/<league>/moving.json` and a move file: a tar of the
league file (`VACUUM INTO`), its pinned football version and a manifest, gzipped and sealed with
age to a one-time X25519 key that is printed as the move code and kept nowhere. While
`moving.json` exists every write guard of that league refuses. `twr-receive-league` runs
`twr-server import-league`: it opens the file only with the code, accepts only those three kinds
of entry, checks the league file names its league and its audit chain verifies, puts its football
version and players into the reference file, installs the league file and issues one seat claim
per seated person and per commissioner without a franchise (`league_seat_claims` in
`accounts.db`, digests only). A claim, through the normal join page, makes a new account under
the identity the league already names and spends the claim in the same transaction.


`twr-restore-league` stops the service and passes the server's own bucket settings, the
restore key and the day to `twr-server restore-league` over stdin, as the service user. It
picks that league's newest usable set under `leagues/<league>/` (on or before the day),
puts back any football versions or players it needs into the reference file, signs out every
owner named in the league as restored or as it was (sign-in generation, authenticator proof
and an audit row `league_restored_sign_out`), archives the old file for 30 days, and only then
puts the restored file in place. `twr-delete-league` runs `twr-server delete-league`: the
league's folder moves to `archive/<league>@<time>` and its directory row and join link go.
The running server deletes archived leagues older than 30 days at start and daily.


`twr-restore` stops the fresh service and passes the six answers to
`twr-server restore` over stdin, running as the service user. That command refuses a
target database that already holds accounts or league entries. It lists `sets/` newest
first (dropping sets saved after the optional `before` date, unopened), streams each set through `offsite.Restore` into a new private folder, and checks
the staged database against the `checkpoint.json` saved inside the same set. A set with
no league (from a fresh install backed up by mistake), a damaged set or a checkpoint
mismatch is skipped with a printed reason; a key that unlocks nothing stops at once.
In the accepted copy it records both audit heads and `history.RecoveryDigests` (evidence
bodies, and each league's replayed ledger), then calls `ResetAfterRestore`: a new auth
epoch (ends sessions, helper tokens and second-factor proofs), every printed recovery
code and unredeemed invitation removed, and one sign-in history entry per account. It
recomputes the league head and digests and refuses if the reset changed them. Only
then does the script install `server.db`, the restored `authenticator.key` and new
`backup.json` (old R2 token replaced, public recipient derived from the restore key),
remove the old WAL files and start the service. The fresh install's TLS key, Tunnel
pairing and release preview registry are kept. Any failure leaves the fresh install
running unchanged and deletes the decrypted working folder.

`proxmox.sh` asks whether a lost server is being replaced. If so, `cloudflare.sh` accepts
an occupied hostname only when it is a single CNAME to a Tunnel named `twr-…` in the same
account that has no connections (status down or inactive) or is already deleted. It
repoints that record to the new Tunnel in place, then deletes the old Tunnel.

`twr-reset-sign-ins` stops the service, runs `twr-server reset-sign-ins` as the service
user with the reason on stdin (new auth epoch, second-factor proofs cleared, one history
entry per account) and starts the service again, even if the reset failed.

Evidence: `TestSpareHostRestoreDrill` (cmd/twr-server) builds a league with an owner,
authenticator, printed codes, a session and evidence; backs it up; uses a printed code
after the backup; backs up a mistaken empty spare; deletes the old host; restores onto
a fresh install; and checks that the empty set is skipped, a wrong key is refused,
league head, evidence and replay fingerprints match, the old session and both used and
unused printed codes are refused, and the owner signs in again with the restored
authenticator. It times the software path against the four-hour limit.
`TestRestoreRefusesServerHoldingLeague`, `TestSignInResetAndRestoreReset` and
`TestSpareHostListsNewestSetAndNamesWrongKey` cover refusal, the two resets and R2
listing. `test-alpine.sh` runs the installed scripts: a bad restore key leaves state
unchanged and the service restarted, the incident reset keeps files owned by the
service user, and the simulated Cloudflare API covers takeover and its three refusals.
A timed drill on a real Proxmox spare with real Cloudflare and R2 has not been run.
