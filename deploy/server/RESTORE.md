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
     (hidden as you type; capital or small letters both work).
4. It downloads the newest backup, unlocks it, checks it and installs it. This takes a
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

## What the tools do (for Christopher)

`twr-restore` stops the fresh service and passes the five answers to
`twr-server restore` over stdin, running as the service user. That command refuses a
target database that already holds accounts or league entries. It lists `sets/` newest
first, streams each set through `offsite.Restore` into a new private folder, and checks
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
